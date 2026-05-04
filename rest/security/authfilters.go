package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// AuthFilter describes one authentication filter — a single auth
// step (anonymous, basic, form, rememberme, oidc, …) that filter
// chains compose into per-URL auth pipelines.
//
// Like [AuthProvider], the wire shape is heterogeneous: the Java
// filter class drives extra fields. The SDK normalizes the typed
// core (Name, ClassName) and keeps the rest in [AuthFilter.Extras].
type AuthFilter struct {
	// Name is the catalog name of the filter (e.g. "anonymous",
	// "basic", "form").
	Name string `json:"name,omitempty"`

	// ClassName is the FQN of the GeoServer Java class implementing
	// the filter (e.g.
	// "org.geoserver.security.filter.GeoServerAnonymousAuthenticationFilter").
	ClassName string `json:"className,omitempty"`

	// Extras carries the filter-specific config not represented in
	// the typed fields (e.g. an OIDC filter's clientId/clientSecret).
	Extras map[string]interface{} `json:"-"`
}

// MarshalJSON serializes the filter as a flat object — typed fields
// + Extras. Conflicts resolved in favor of typed fields.
func (f AuthFilter) MarshalJSON() ([]byte, error) {
	out := map[string]interface{}{}
	for k, v := range f.Extras {
		out[k] = v
	}
	if f.Name != "" {
		out["name"] = f.Name
	}
	if f.ClassName != "" {
		out["className"] = f.ClassName
	}
	return json.Marshal(out)
}

// UnmarshalJSON parses either a flat filter object or GeoServer's
// class-name-keyed envelope (e.g.
// `{"o.g.s.config.AnonymousAuthenticationFilterConfig":{...}}`).
// Typed fields land in the typed slots and everything else
// accumulates in Extras.
func (f *AuthFilter) UnmarshalJSON(b []byte) error {
	if inner, ok := unwrapClassEnvelope(b); ok {
		b = inner
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		switch k {
		case "name":
			_ = json.Unmarshal(v, &f.Name)
		case "className":
			_ = json.Unmarshal(v, &f.ClassName)
		default:
			if f.Extras == nil {
				f.Extras = map[string]interface{}{}
			}
			var anyVal interface{}
			if err := json.Unmarshal(v, &anyVal); err == nil {
				f.Extras[k] = anyVal
			}
		}
	}
	return nil
}

// AuthFilterRef is one entry in the auth-filters listing.
type AuthFilterRef struct {
	Name string `json:"name"`
	Href string `json:"href"`
}

// AuthFiltersClient operates on /rest/security/authfilters.
type AuthFiltersClient struct {
	core Core
}

// authFiltersListWire decodes the list-shape envelope:
// `{"authfilters":{"authfilter":[{name, href}, ...]}}`.
type authFiltersListWire struct {
	AuthFilters struct {
		AuthFilter []AuthFilterRef `json:"authfilter"`
	} `json:"authfilters"`
}

// List returns every configured auth filter (just names + hrefs;
// fetch detail via [AuthFiltersClient.Get]).
func (c *AuthFiltersClient) List(ctx context.Context) ([]AuthFilterRef, error) {
	const op = "Security.AuthFilters.List"
	u, err := c.core.URL("rest", "security", "authfilters")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var wrap authFiltersListWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &wrap); err != nil {
		return nil, err
	}
	return wrap.AuthFilters.AuthFilter, nil
}

// Get returns one filter by name.
//
// Wire-quirk: GeoServer's auth-filters endpoint returns `200 OK`
// with body `{"null":""}` for filter names that don't exist instead
// of `404 Not Found`. This method detects the empty-result wire
// shape (Name field comes back blank after a successful response)
// and surfaces it as an [ErrNotFound]-bearing error so callers can
// match with errors.Is(err, geoserver.ErrNotFound).
func (c *AuthFiltersClient) Get(ctx context.Context, name string) (*AuthFilter, error) {
	const op = "Security.AuthFilters.Get"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "security", "authfilters", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var f AuthFilter
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &f); err != nil {
		return nil, err
	}
	if f.Name == "" {
		return nil, c.core.SynthesizeError(op, http.MethodGet, u, http.StatusNotFound,
			fmt.Sprintf("auth filter %q not found (GeoServer returned empty/null body)", name))
	}
	return &f, nil
}

// Create registers a new filter. Name and ClassName are required.
func (c *AuthFiltersClient) Create(ctx context.Context, f *AuthFilter) error {
	const op = "Security.AuthFilters.Create"
	if f == nil {
		return errors.New(op + ": nil filter")
	}
	if f.Name == "" || f.ClassName == "" {
		return errors.New(op + ": Name and ClassName are required")
	}
	u, err := c.core.URL("rest", "security", "authfilters")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, f, nil, nil)
}

// Update replaces the filter at name with the supplied body.
func (c *AuthFiltersClient) Update(ctx context.Context, name string, f *AuthFilter) error {
	const op = "Security.AuthFilters.Update"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if f == nil {
		return errors.New(op + ": nil filter")
	}
	u, err := c.core.URL("rest", "security", "authfilters", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPut, u, f, nil, nil)
}

// Delete removes the named filter. Filters referenced by an active
// filter chain cannot be deleted (server returns 4xx).
func (c *AuthFiltersClient) Delete(ctx context.Context, name string) error {
	const op = "Security.AuthFilters.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "security", "authfilters", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
