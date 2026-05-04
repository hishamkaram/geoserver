package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// AuthProvider describes one authentication provider — the bridge
// between GeoServer's catalog auth and an external identity backend
// (UsernamePassword, LDAP, OAuth/OIDC, header-auth, etc.).
//
// The wire shape is heterogeneous: every concrete provider type
// declares its own java config class as the JSON envelope key (e.g.
// "org.geoserver.security.config.UsernamePasswordAuthenticationProviderConfig"
// for a UsernamePassword provider) and its own provider-specific
// fields alongside the common core fields. This SDK normalizes the
// core fields into typed properties and keeps any extras in the
// [AuthProvider.Extras] free-form map. Round-tripping a fetched
// provider back unchanged is supported.
type AuthProvider struct {
	// ID is server-generated; ignore on creates.
	ID string `json:"id,omitempty"`

	// Name is the provider's catalog name (e.g. "default",
	// "corporateLdap"). Required on create.
	Name string `json:"name,omitempty"`

	// ClassName is the FQN of the GeoServer Java class that
	// implements this provider's auth flow (e.g.
	// "org.geoserver.security.auth.UsernamePasswordAuthenticationProvider").
	// Required on create.
	ClassName string `json:"className,omitempty"`

	// UserGroupServiceName names the user/group service this
	// provider authenticates against (typically "default").
	// Required on create.
	UserGroupServiceName string `json:"userGroupServiceName,omitempty"`

	// Extras is provider-specific configuration that doesn't fit
	// the typed core fields — for example an LDAP provider's
	// `serverURL`, `userFormat`, or an OIDC provider's
	// `clientId`/`clientSecret`. Any key not in the typed list
	// above is preserved here on read and re-emitted on write.
	Extras map[string]interface{} `json:"-"`
}

// MarshalJSON serializes the provider as a flat object: typed core
// fields plus all entries from [AuthProvider.Extras]. Conflicts (an
// Extras key matching a typed field) are resolved in favor of the
// typed field.
func (p AuthProvider) MarshalJSON() ([]byte, error) {
	out := map[string]interface{}{}
	for k, v := range p.Extras {
		out[k] = v
	}
	if p.ID != "" {
		out["id"] = p.ID
	}
	if p.Name != "" {
		out["name"] = p.Name
	}
	if p.ClassName != "" {
		out["className"] = p.ClassName
	}
	if p.UserGroupServiceName != "" {
		out["userGroupServiceName"] = p.UserGroupServiceName
	}
	return json.Marshal(out)
}

// UnmarshalJSON parses either a flat provider object or GeoServer's
// class-name-keyed envelope (e.g.
// `{"o.g.s.config.UsernamePasswordAuthenticationProviderConfig":{...}}`).
// Typed core fields land in the typed slots and everything else
// accumulates in [AuthProvider.Extras].
func (p *AuthProvider) UnmarshalJSON(b []byte) error {
	if inner, ok := unwrapClassEnvelope(b); ok {
		b = inner
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		switch k {
		case "id":
			_ = json.Unmarshal(v, &p.ID)
		case "name":
			_ = json.Unmarshal(v, &p.Name)
		case "className":
			_ = json.Unmarshal(v, &p.ClassName)
		case "userGroupServiceName":
			_ = json.Unmarshal(v, &p.UserGroupServiceName)
		default:
			if p.Extras == nil {
				p.Extras = map[string]interface{}{}
			}
			var anyVal interface{}
			if err := json.Unmarshal(v, &anyVal); err == nil {
				p.Extras[k] = anyVal
			}
		}
	}
	return nil
}

// unwrapClassEnvelope detects GeoServer's
// `{"<JavaClassFQN>": {...}}` single-key envelope and returns the
// inner JSON object so a flat-decoder path can finish parsing.
// Heuristic: exactly one top-level key, the key looks like a Java
// FQN (contains a dot), and the value is a JSON object.
func unwrapClassEnvelope(b []byte) ([]byte, bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, false
	}
	if len(raw) != 1 {
		return nil, false
	}
	for k, v := range raw {
		if !strings.Contains(k, ".") {
			return nil, false
		}
		// Strip leading whitespace before checking the first byte.
		trimmed := v
		for len(trimmed) > 0 {
			c := trimmed[0]
			if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
				break
			}
			trimmed = trimmed[1:]
		}
		if len(trimmed) == 0 || trimmed[0] != '{' {
			return nil, false
		}
		return v, true
	}
	return nil, false
}

// AuthProvidersClient operates on /rest/security/authproviders.
type AuthProvidersClient struct {
	core Core
}

// CreateAuthProviderOptions controls Create behavior.
type CreateAuthProviderOptions struct {
	// Position is the 0-based insert index in the active order. Zero
	// (the default) appends at the end.
	Position int
}

// UpdateAuthProviderOptions controls Update behavior.
type UpdateAuthProviderOptions struct {
	// Position, if non-nil, moves the provider to this 0-based
	// index in the active order while updating its body.
	Position *int
}

// authProvidersListWire decodes the list-shape envelope. GeoServer
// returns one of two shapes depending on number of providers:
//
//   - `{"authproviders":[{...}, {...}]}` — array (documented OpenAPI form)
//   - `{"authproviders":{"<className>":{...}}}` — single-element collapse
//     keyed by the concrete provider class. We accept both.
type authProvidersListWire struct {
	AuthProviders json.RawMessage `json:"authproviders"`
}

// authProviderOrderWire is the body shape for SetOrder.
type authProviderOrderWire struct {
	Order []string `json:"order"`
}

// List returns every configured auth provider, in active order.
func (c *AuthProvidersClient) List(ctx context.Context) ([]AuthProvider, error) {
	const op = "Security.AuthProviders.List"
	u, err := c.core.URL("rest", "security", "authproviders")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var wrap authProvidersListWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.AuthProviders) == 0 || string(wrap.AuthProviders) == "null" {
		return nil, nil
	}
	// Try array first.
	if wrap.AuthProviders[0] == '[' {
		var arr []AuthProvider
		if err := json.Unmarshal(wrap.AuthProviders, &arr); err != nil {
			return nil, fmt.Errorf("%s: decode array: %w", op, err)
		}
		return arr, nil
	}
	// Otherwise treat as object keyed by class name.
	var byClass map[string]AuthProvider
	if err := json.Unmarshal(wrap.AuthProviders, &byClass); err != nil {
		return nil, fmt.Errorf("%s: decode map: %w", op, err)
	}
	out := make([]AuthProvider, 0, len(byClass))
	for _, p := range byClass {
		out = append(out, p)
	}
	return out, nil
}

// Get returns one provider by name.
func (c *AuthProvidersClient) Get(ctx context.Context, name string) (*AuthProvider, error) {
	const op = "Security.AuthProviders.Get"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "security", "authproviders", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var p AuthProvider
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Create registers a new provider. Required fields on the input:
// Name, ClassName, UserGroupServiceName.
func (c *AuthProvidersClient) Create(ctx context.Context, p *AuthProvider, opts CreateAuthProviderOptions) error {
	const op = "Security.AuthProviders.Create"
	if p == nil {
		return errors.New(op + ": nil provider")
	}
	if p.Name == "" || p.ClassName == "" || p.UserGroupServiceName == "" {
		return errors.New(op + ": Name, ClassName, and UserGroupServiceName are required")
	}
	u, err := c.core.URL("rest", "security", "authproviders")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	var query map[string]string
	if opts.Position > 0 {
		query = map[string]string{"position": strconv.Itoa(opts.Position)}
	}
	return c.core.Do(ctx, op, http.MethodPost, u, p, query, nil)
}

// Update replaces the provider's body and optionally moves it in the
// active order.
func (c *AuthProvidersClient) Update(ctx context.Context, name string, p *AuthProvider, opts UpdateAuthProviderOptions) error {
	const op = "Security.AuthProviders.Update"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if p == nil {
		return errors.New(op + ": nil provider")
	}
	u, err := c.core.URL("rest", "security", "authproviders", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	var query map[string]string
	if opts.Position != nil {
		query = map[string]string{"position": strconv.Itoa(*opts.Position)}
	}
	return c.core.Do(ctx, op, http.MethodPut, u, p, query, nil)
}

// Delete removes the named provider. Removing a provider also
// removes it from the active order.
func (c *AuthProvidersClient) Delete(ctx context.Context, name string) error {
	const op = "Security.AuthProviders.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "security", "authproviders", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}

// SetOrder replaces the active provider order. Names not in the
// list become inactive (they remain configured but are not consulted
// during auth). Empty list is rejected by GeoServer.
func (c *AuthProvidersClient) SetOrder(ctx context.Context, names []string) error {
	const op = "Security.AuthProviders.SetOrder"
	if len(names) == 0 {
		return errors.New(op + ": names list must be non-empty")
	}
	u, err := c.core.URL("rest", "security", "authproviders", "order")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := authProviderOrderWire{Order: names}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}
