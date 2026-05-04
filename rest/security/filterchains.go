package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// FilterChain binds a set of URL patterns to an ordered list of
// authentication filters. Requests matching the chain's URL pattern
// run through its filters in order.
//
// On the JSON wire, GeoServer emits filter-chain attributes as
// "@"-prefixed keys (XML-attribute style). The SDK normalizes those
// into typed fields and round-trips identically.
type FilterChain struct {
	Name                     string `json:"@name,omitempty"`
	ClassName                string `json:"@class,omitempty"`
	Path                     string `json:"@path,omitempty"`
	Disabled                 bool   `json:"@disabled,omitempty"`
	AllowSessionCreation     bool   `json:"@allowSessionCreation,omitempty"`
	SSL                      bool   `json:"@ssl,omitempty"`
	MatchHTTPMethod          bool   `json:"@matchHTTPMethod,omitempty"`
	InterceptorName          string `json:"@interceptorName,omitempty"`
	ExceptionTranslationName string `json:"@exceptionTranslationName,omitempty"`
	HTTPMethods              string `json:"@httpMethods,omitempty"`
	RoleFilterName           string `json:"@roleFilterName,omitempty"`

	// Filters is the ordered list of [AuthFilter] names that
	// requests matching this chain run through. On the wire,
	// GeoServer collapses single-element arrays to a scalar
	// string — handled transparently by the custom unmarshal.
	Filters []string `json:"filter,omitempty"`
}

// UnmarshalJSON tolerates the single-element collapse on the
// `filter` field: GeoServer returns it as a JSON array when there
// are 2+ filters and as a bare string when there's exactly 1.
func (fc *FilterChain) UnmarshalJSON(b []byte) error {
	type alias FilterChain
	var raw struct {
		alias
		Filters json.RawMessage `json:"filter"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*fc = FilterChain(raw.alias)
	if len(raw.Filters) == 0 || string(raw.Filters) == "null" {
		return nil
	}
	switch raw.Filters[0] {
	case '[':
		return json.Unmarshal(raw.Filters, &fc.Filters)
	case '"':
		var s string
		if err := json.Unmarshal(raw.Filters, &s); err != nil {
			return err
		}
		fc.Filters = []string{s}
		return nil
	default:
		return fmt.Errorf("security: unexpected JSON shape for filter: %s", raw.Filters)
	}
}

// FilterChainsClient operates on /rest/security/filterchain.
type FilterChainsClient struct {
	core Core
}

// UpdateFilterChainOptions controls Update behavior.
type UpdateFilterChainOptions struct {
	// Position, if non-nil, moves the chain to this 0-based index
	// in the chain ordering while updating its body.
	Position *int
}

// filterChainsListWire decodes the list shape:
// `{"filterchain":{"filters":[{...}, ...]}}`.
type filterChainsListWire struct {
	FilterChain struct {
		Filters []FilterChain `json:"filters"`
	} `json:"filterchain"`
}

// filterChainSingleWire is the per-chain shape for Get / Create / Update:
// `{"filters":{...}}` (note the field name is "filters" — GeoServer's
// REST controller chose the same plural element name as in the list,
// even though it's a single chain here).
type filterChainSingleWire struct {
	Filters FilterChain `json:"filters"`
}

// filterChainOrderWire is the body for SetOrder.
type filterChainOrderWire struct {
	Order []string `json:"order"`
}

// List returns every configured filter chain in active order.
func (c *FilterChainsClient) List(ctx context.Context) ([]FilterChain, error) {
	const op = "Security.FilterChains.List"
	u, err := c.core.URL("rest", "security", "filterchain")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	// Append .json so GeoServer skips its default HTML rendering.
	u += ".json"
	var wrap filterChainsListWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &wrap); err != nil {
		return nil, err
	}
	return wrap.FilterChain.Filters, nil
}

// Get returns one filter chain by name.
func (c *FilterChainsClient) Get(ctx context.Context, name string) (*FilterChain, error) {
	const op = "Security.FilterChains.Get"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "security", "filterchain", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	u += ".json"
	var wrap filterChainSingleWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &wrap); err != nil {
		return nil, err
	}
	return &wrap.Filters, nil
}

// Create registers a new filter chain.
func (c *FilterChainsClient) Create(ctx context.Context, fc *FilterChain) error {
	const op = "Security.FilterChains.Create"
	if fc == nil {
		return errors.New(op + ": nil chain")
	}
	if fc.Name == "" || fc.ClassName == "" || fc.Path == "" {
		return errors.New(op + ": Name, ClassName, and Path are required")
	}
	u, err := c.core.URL("rest", "security", "filterchain")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := filterChainSingleWire{Filters: *fc}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Update replaces the chain at name with the supplied body and
// optionally moves it in the ordering.
func (c *FilterChainsClient) Update(ctx context.Context, name string, fc *FilterChain, opts UpdateFilterChainOptions) error {
	const op = "Security.FilterChains.Update"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if fc == nil {
		return errors.New(op + ": nil chain")
	}
	u, err := c.core.URL("rest", "security", "filterchain", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	var query map[string]string
	if opts.Position != nil {
		query = map[string]string{"position": strconv.Itoa(*opts.Position)}
	}
	body := filterChainSingleWire{Filters: *fc}
	return c.core.Do(ctx, op, http.MethodPut, u, body, query, nil)
}

// Delete removes the named filter chain.
func (c *FilterChainsClient) Delete(ctx context.Context, name string) error {
	const op = "Security.FilterChains.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "security", "filterchain", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}

// SetOrder replaces the chain ordering. Names not in the list keep
// their current configuration but are not consulted at request time
// until re-added.
func (c *FilterChainsClient) SetOrder(ctx context.Context, names []string) error {
	const op = "Security.FilterChains.SetOrder"
	if len(names) == 0 {
		return errors.New(op + ": names list must be non-empty")
	}
	u, err := c.core.URL("rest", "security", "filterchain", "order")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := filterChainOrderWire{Order: names}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}
