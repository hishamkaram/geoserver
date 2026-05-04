// Package wmtsstores is the v2 sub-client for the GeoServer cascaded
// WMS store endpoint at /rest/workspaces/{ws}/wmtsstores. A WMS store
// references a remote WMS server; cascaded layers under the store
// re-publish that remote WMS through the local GeoServer.
package wmtsstores

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

// WMTSStore is a cascaded WMTS store. Required fields on Create:
// Name, CapabilitiesURL.
type WMTSStore struct {
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

// MarshalJSON wraps the store in GeoServer's `{"wmtsStore":{...}}`
// envelope expected by POST/PUT. See [WMTSStore] doc for the
// rationale on the Password / AuthKey gosec suppression.
func (s WMTSStore) MarshalJSON() ([]byte, error) {
	type alias WMTSStore
	return json.Marshal(map[string]alias{"wmtsStore": alias(s)}) //nolint:gosec // Password/AuthKey are required upstream-WMTS credentials per GeoServer's wmtsStore schema.
}

// UnmarshalJSON accepts both the wrapped form (`{"wmtsStore":{...}}`)
// and a flat object.
func (s *WMTSStore) UnmarshalJSON(b []byte) error {
	type alias WMTSStore
	var wrapped struct {
		WMTSStore *alias `json:"wmtsStore"`
	}
	if err := json.Unmarshal(b, &wrapped); err == nil && wrapped.WMTSStore != nil {
		*s = WMTSStore(*wrapped.WMTSStore)
		return nil
	}
	var flat alias
	if err := json.Unmarshal(b, &flat); err != nil {
		return err
	}
	*s = WMTSStore(flat)
	return nil
}

// Ref is a `{name, href}` reference returned by list endpoints.
type Ref struct {
	Name string `json:"name"`
	Href string `json:"href"`
}

// listWire decodes the list envelope shape:
// `{"wmtsStores":{"wmtsStore":[{name, href}, ...]}}` or
// `{"wmtsStores":""}` (empty).
type listWire struct {
	WMTSStores json.RawMessage `json:"wmtsStores"`
}

// Client is the v2 cascaded WMTS stores sub-client.
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

// List returns every cascaded WMTS store in the workspace.
func (c *WorkspaceClient) List(ctx context.Context, _ ListOptions) ([]Ref, error) {
	const op = "WMTSStores.List"
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtsstores")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var wrap listWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.WMTSStores) == 0 || wrap.WMTSStores[0] == '"' {
		return nil, nil
	}
	var inner struct {
		WMTSStore []Ref `json:"wmtsStore"`
	}
	if err := json.Unmarshal(wrap.WMTSStores, &inner); err != nil {
		return nil, fmt.Errorf("%s: decode list: %w", op, err)
	}
	return inner.WMTSStore, nil
}

// Iter yields refs as a range-over-func iterator. Single-page
// fallback (the GeoServer wmtsstores endpoint does not paginate).
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
func (c *WorkspaceClient) Get(ctx context.Context, name string) (*WMTSStore, error) {
	const op = "WMTSStores.Get"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtsstores", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var s WMTSStore
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Create registers a new store. Required fields: Name, CapabilitiesURL.
func (c *WorkspaceClient) Create(ctx context.Context, store *WMTSStore) error {
	const op = "WMTSStores.Create"
	if store == nil {
		return errors.New(op + ": nil store")
	}
	if store.Name == "" || store.CapabilitiesURL == "" {
		return errors.New(op + ": Name and CapabilitiesURL are required")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtsstores")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, store, nil, nil)
}

// Update replaces the store at name.
func (c *WorkspaceClient) Update(ctx context.Context, name string, store *WMTSStore) error {
	const op = "WMTSStores.Update"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if store == nil {
		return errors.New(op + ": nil store")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtsstores", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPut, u, store, nil, nil)
}

// Delete removes a store. Set DeleteOptions.Recurse to also remove
// any cascaded layers under it.
func (c *WorkspaceClient) Delete(ctx context.Context, name string, opts DeleteOptions) error {
	const op = "WMTSStores.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtsstores", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	var query map[string]string
	if opts.Recurse {
		query = map[string]string{"recurse": strconv.FormatBool(true)}
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, query, nil)
}
