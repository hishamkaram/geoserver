// Package wmsstores is the v2 sub-client for the GeoServer cascaded
// WMS store endpoint at /rest/workspaces/{ws}/wmsstores. A WMS store
// references a remote WMS server; cascaded layers under the store
// re-publish that remote WMS through the local GeoServer.
package wmsstores

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strconv"
)

// WorkspaceRef is the workspace pointer GeoServer carries back on a
// store response.
type WorkspaceRef struct {
	Name string `json:"name,omitempty"`
}

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
}

// WMSStore is a cascaded WMS store. Required fields on Create:
// Name, CapabilitiesURL.
type WMSStore struct {
	Name            string        `json:"name,omitempty"`
	Type            string        `json:"type,omitempty"`
	Enabled         bool          `json:"enabled,omitempty"`
	Workspace       *WorkspaceRef `json:"workspace,omitempty"`
	Default         bool          `json:"_default,omitempty"`
	CapabilitiesURL string        `json:"capabilitiesURL,omitempty"`
	User            string        `json:"user,omitempty"`
	Password        string        `json:"password,omitempty"`
	HeaderName      string        `json:"headerName,omitempty"`
	HeaderValue     string        `json:"headerValue,omitempty"`
	AuthKey         string        `json:"authKey,omitempty"`
	MaxConnections  int           `json:"maxConnections,omitempty"`
	ReadTimeout     int           `json:"readTimeout,omitempty"`
	ConnectTimeout  int           `json:"connectTimeout,omitempty"`
	UseHTTPConnPool bool          `json:"useHttpConnectionPooling,omitempty"`
}

// MarshalJSON wraps the store in GeoServer's `{"wmsStore":{...}}`
// envelope expected by POST/PUT. The store body legitimately
// includes upstream-WMS basic-auth credentials (Password, AuthKey)
// — these are part of the documented GeoServer config schema, not
// an exfiltration channel, so the gosec credential heuristic is
// suppressed below.
func (s WMSStore) MarshalJSON() ([]byte, error) {
	type alias WMSStore
	return json.Marshal(map[string]alias{"wmsStore": alias(s)}) //nolint:gosec // Password/AuthKey are required upstream-WMS credentials per GeoServer's wmsStore schema.
}

// UnmarshalJSON accepts both the wrapped form (`{"wmsStore":{...}}`)
// and a flat object.
func (s *WMSStore) UnmarshalJSON(b []byte) error {
	type alias WMSStore
	var wrapped struct {
		WMSStore *alias `json:"wmsStore"`
	}
	if err := json.Unmarshal(b, &wrapped); err == nil && wrapped.WMSStore != nil {
		*s = WMSStore(*wrapped.WMSStore)
		return nil
	}
	var flat alias
	if err := json.Unmarshal(b, &flat); err != nil {
		return err
	}
	*s = WMSStore(flat)
	return nil
}

// Ref is a `{name, href}` reference returned by list endpoints.
type Ref struct {
	Name string `json:"name"`
	Href string `json:"href"`
}

// listWire decodes the list envelope shape:
// `{"wmsStores":{"wmsStore":[{name, href}, ...]}}` or
// `{"wmsStores":""}` (empty).
type listWire struct {
	WMSStores json.RawMessage `json:"wmsStores"`
}

// Client is the v2 cascaded WMS stores sub-client.
type Client struct {
	core Core
}

// New constructs the WMS stores sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// InWorkspace narrows scope to one workspace's WMS stores.
func (c *Client) InWorkspace(workspace string) *WorkspaceClient {
	return &WorkspaceClient{core: c.core, workspace: workspace}
}

// WorkspaceClient is workspace-scoped CRUD for WMS stores.
type WorkspaceClient struct {
	core      Core
	workspace string
}

// Workspace returns the workspace bound to this client.
func (c *WorkspaceClient) Workspace() string { return c.workspace }

// ListOptions controls listing behavior. Currently empty.
type ListOptions struct{}

// DeleteOptions controls deletion behavior.
type DeleteOptions struct {
	// Recurse removes any cascaded layers under the store along
	// with the store itself.
	Recurse bool
}

// List returns every cascaded WMS store in the workspace.
func (c *WorkspaceClient) List(ctx context.Context, _ ListOptions) ([]Ref, error) {
	const op = "WMSStores.List"
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmsstores")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var wrap listWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.WMSStores) == 0 || wrap.WMSStores[0] == '"' {
		return nil, nil
	}
	var inner struct {
		WMSStore []Ref `json:"wmsStore"`
	}
	if err := json.Unmarshal(wrap.WMSStores, &inner); err != nil {
		return nil, fmt.Errorf("%s: decode list: %w", op, err)
	}
	return inner.WMSStore, nil
}

// Iter yields refs as a range-over-func iterator. Single-page
// fallback (the GeoServer wmsstores endpoint does not paginate).
func (c *WorkspaceClient) Iter(ctx context.Context, opts ListOptions) iter.Seq2[Ref, error] {
	return func(yield func(Ref, error) bool) {
		refs, err := c.List(ctx, opts)
		if err != nil {
			yield(Ref{}, err)
			return
		}
		for _, r := range refs {
			if !yield(r, nil) {
				return
			}
		}
	}
}

// Get returns one store by name.
func (c *WorkspaceClient) Get(ctx context.Context, name string) (*WMSStore, error) {
	const op = "WMSStores.Get"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmsstores", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var s WMSStore
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Create registers a new store. Required fields: Name, CapabilitiesURL.
func (c *WorkspaceClient) Create(ctx context.Context, store *WMSStore) error {
	const op = "WMSStores.Create"
	if store == nil {
		return errors.New(op + ": nil store")
	}
	if store.Name == "" || store.CapabilitiesURL == "" {
		return errors.New(op + ": Name and CapabilitiesURL are required")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmsstores")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, store, nil, nil)
}

// Update replaces the store at name.
func (c *WorkspaceClient) Update(ctx context.Context, name string, store *WMSStore) error {
	const op = "WMSStores.Update"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if store == nil {
		return errors.New(op + ": nil store")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmsstores", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPut, u, store, nil, nil)
}

// Delete removes a store. Set DeleteOptions.Recurse to also remove
// any cascaded layers under it.
func (c *WorkspaceClient) Delete(ctx context.Context, name string, opts DeleteOptions) error {
	const op = "WMSStores.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmsstores", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	var query map[string]string
	if opts.Recurse {
		query = map[string]string{"recurse": strconv.FormatBool(true)}
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, query, nil)
}
