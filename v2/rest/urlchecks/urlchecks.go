// Package urlchecks is the v2 sub-client for the GeoServer URL
// External Access Checks endpoint at /rest/urlchecks. URL checks are
// allow/deny lists for external URLs that GeoServer is permitted to
// fetch — e.g. SLD external graphics, image-mosaic remote rasters,
// cascaded WMS sources. Each check is a regex that incoming external
// URLs are matched against; a request is allowed only when at least
// one check matches.
package urlchecks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
}

// URLCheck is a single URL access rule.
type URLCheck struct {
	// Name is the catalog name of the check (required, unique).
	Name string `json:"name,omitempty"`
	// Description is human-readable text shown in the admin UI.
	Description string `json:"description,omitempty"`
	// Enabled toggles the check on or off without removing it.
	Enabled bool `json:"enabled,omitempty"`
	// Regex is the regular expression external URLs must match
	// (required). Standard Java regex syntax.
	Regex string `json:"regex,omitempty"`
}

// MarshalJSON wraps the URLCheck in GeoServer's required class-name
// envelope (`{"regexUrlCheck":{...}}`) for POST/PUT bodies. Sending
// a flat object is rejected by the server with 500.
func (u URLCheck) MarshalJSON() ([]byte, error) {
	type alias URLCheck
	return json.Marshal(map[string]alias{"regexUrlCheck": alias(u)})
}

// UnmarshalJSON accepts the wrapped form
// (`{"regexUrlCheck":{...}}`) GeoServer returns on GET, falling
// back to a bare object if the wrapper is absent.
func (u *URLCheck) UnmarshalJSON(b []byte) error {
	type alias URLCheck
	// Try wrapped form first.
	var wrapped struct {
		RegexURLCheck *alias `json:"regexUrlCheck"`
	}
	if err := json.Unmarshal(b, &wrapped); err == nil && wrapped.RegexURLCheck != nil {
		*u = URLCheck(*wrapped.RegexURLCheck)
		return nil
	}
	// Fallback: flat decode.
	var flat alias
	if err := json.Unmarshal(b, &flat); err != nil {
		return err
	}
	*u = URLCheck(flat)
	return nil
}

// URLCheckRef is one entry in the URL-checks listing.
type URLCheckRef struct {
	Name string `json:"name"`
	Href string `json:"href"`
}

// Client is the v2 URL-checks sub-client.
type Client struct {
	core Core
}

// New constructs the URL-checks sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// urlChecksListWire decodes the list-shape envelope. The empty
// collection comes back as `{"urlChecks":""}` (bare string instead
// of object) — same wire-quirk pattern as styles, datastores, etc.
type urlChecksListWire struct {
	URLChecks json.RawMessage `json:"urlChecks"`
}

// List returns every configured URL check.
func (c *Client) List(ctx context.Context) ([]URLCheckRef, error) {
	const op = "URLChecks.List"
	u, err := c.core.URL("rest", "urlchecks")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var wrap urlChecksListWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.URLChecks) == 0 || wrap.URLChecks[0] == '"' {
		// Empty-collection wire shape: bare string.
		return nil, nil
	}
	var inner struct {
		URLCheck []URLCheckRef `json:"urlCheck"`
	}
	if err := json.Unmarshal(wrap.URLChecks, &inner); err != nil {
		return nil, fmt.Errorf("%s: decode list: %w", op, err)
	}
	return inner.URLCheck, nil
}

// Get returns one URL check by name.
func (c *Client) Get(ctx context.Context, name string) (*URLCheck, error) {
	const op = "URLChecks.Get"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "urlchecks", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var check URLCheck
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &check); err != nil {
		return nil, err
	}
	return &check, nil
}

// Create registers a new URL check. Name and Regex are required.
func (c *Client) Create(ctx context.Context, check *URLCheck) error {
	const op = "URLChecks.Create"
	if check == nil {
		return errors.New(op + ": nil check")
	}
	if check.Name == "" || check.Regex == "" {
		return errors.New(op + ": Name and Regex are required")
	}
	u, err := c.core.URL("rest", "urlchecks")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, check, nil, nil)
}

// Update replaces the URL check at name with the supplied body.
// Per the upstream API, only changed fields need to be present.
func (c *Client) Update(ctx context.Context, name string, check *URLCheck) error {
	const op = "URLChecks.Update"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if check == nil {
		return errors.New(op + ": nil check")
	}
	u, err := c.core.URL("rest", "urlchecks", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPut, u, check, nil, nil)
}

// Delete removes the named URL check.
func (c *Client) Delete(ctx context.Context, name string) error {
	const op = "URLChecks.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "urlchecks", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
