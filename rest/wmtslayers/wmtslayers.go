// Package wmtslayers is the v2 sub-client for GeoServer's cascaded
// WMS layers — published references to remote WMS server layers
// served through the local GeoServer.
//
// Endpoints live at
// /rest/workspaces/{ws}/wmtsstores/{store}/wmtslayers (canonical) and
// /rest/workspaces/{ws}/wmtslayers (workspace-scoped shortcut).
package wmtslayers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strconv"

	"github.com/hishamkaram/geoserver/v2/internal/wire"
)

type (
	// BoundingBox — see [wire.BoundingBox].
	BoundingBox = wire.BoundingBox
	// NativeBoundingBox — see [wire.NativeBoundingBox].
	NativeBoundingBox = wire.NativeBoundingBox
	// LatLonBoundingBox — see [wire.LatLonBoundingBox].
	LatLonBoundingBox = wire.LatLonBoundingBox
	// Keywords — see [wire.Keywords].
	Keywords = wire.Keywords
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
}

// StoreRef points at a WMS store (response-only).
type StoreRef struct {
	Name string `json:"name,omitempty"`
}

// WMTSLayer is one cascaded WMTS layer published by a [WMTSStore].
//
// Required fields on Create: Name, NativeName.
type WMTSLayer struct {
	Name              string             `json:"name,omitempty"`
	NativeName        string             `json:"nativeName,omitempty"`
	Title             string             `json:"title,omitempty"`
	Abstract          string             `json:"abstract,omitempty"`
	Description       string             `json:"description,omitempty"`
	Keywords          *Keywords          `json:"keywords,omitempty"`
	NativeCRS         string             `json:"nativeCRS,omitempty"`
	SRS               string             `json:"srs,omitempty"`
	NativeBoundingBox *NativeBoundingBox `json:"nativeBoundingBox,omitempty"`
	LatLonBoundingBox *LatLonBoundingBox `json:"latLonBoundingBox,omitempty"`
	ProjectionPolicy  string             `json:"projectionPolicy,omitempty"`
	Enabled           bool               `json:"enabled,omitempty"`
	Store             *StoreRef          `json:"store,omitempty"`
	ForcedRemoteStyle string             `json:"forcedRemoteStyle,omitempty"`
	PreferredFormat   string             `json:"preferredFormat,omitempty"`
	MinScale          float64            `json:"minScale,omitempty"`
	MaxScale          float64            `json:"maxScale,omitempty"`
}

// MarshalJSON wraps the layer in GeoServer's `{"wmtsLayer":{...}}`
// envelope used by POST/PUT.
func (l WMTSLayer) MarshalJSON() ([]byte, error) {
	type alias WMTSLayer
	return json.Marshal(map[string]alias{"wmtsLayer": alias(l)})
}

// UnmarshalJSON accepts both the wrapped form (`{"wmtsLayer":{...}}`)
// and a flat object.
func (l *WMTSLayer) UnmarshalJSON(b []byte) error {
	type alias WMTSLayer
	var wrapped struct {
		WMTSLayer *alias `json:"wmtsLayer"`
	}
	if err := json.Unmarshal(b, &wrapped); err == nil && wrapped.WMTSLayer != nil {
		*l = WMTSLayer(*wrapped.WMTSLayer)
		return nil
	}
	var flat alias
	if err := json.Unmarshal(b, &flat); err != nil {
		return err
	}
	*l = WMTSLayer(flat)
	return nil
}

// Ref is a `{name, href}` reference returned by list endpoints.
type Ref struct {
	Name string `json:"name"`
	Href string `json:"href"`
}

type listWire struct {
	WMTSLayers json.RawMessage `json:"wmtsLayers"`
}

// Client is the cascaded WMTS layers sub-client.
type Client struct {
	core Core
}

// New constructs the sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// InWorkspace narrows scope to one workspace.
func (c *Client) InWorkspace(workspace string) *WorkspaceClient {
	return &WorkspaceClient{core: c.core, workspace: workspace}
}

// WorkspaceClient operates on /workspaces/{ws}/wmtslayers — the
// workspace-wide list across all WMS stores in the workspace.
type WorkspaceClient struct {
	core      Core
	workspace string
}

// Workspace returns the workspace bound to this client.
func (c *WorkspaceClient) Workspace() string { return c.workspace }

// InStore narrows further to one WMS store, exposing the canonical
// /workspaces/{ws}/wmtsstores/{store}/wmtslayers path.
func (c *WorkspaceClient) InStore(store string) *StoreClient {
	return &StoreClient{core: c.core, workspace: c.workspace, store: store}
}

// ListOptions controls listing behavior.
type ListOptions struct{}

// DeleteOptions controls deletion behavior.
type DeleteOptions struct {
	Recurse bool
}

// CreateOptions controls listing behavior for creates.
type CreateOptions struct{}

// List returns every cascaded WMTS layer in the workspace.
func (c *WorkspaceClient) List(ctx context.Context, _ ListOptions) ([]Ref, error) {
	const op = "WMTSLayers.List"
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtslayers")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return c.decodeList(ctx, op, u)
}

// Get returns one workspace-scoped layer by name.
func (c *WorkspaceClient) Get(ctx context.Context, name string) (*WMTSLayer, error) {
	const op = "WMTSLayers.Get"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtslayers", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var l WMTSLayer
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// Delete removes a layer at the workspace scope.
func (c *WorkspaceClient) Delete(ctx context.Context, name string, opts DeleteOptions) error {
	const op = "WMTSLayers.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtslayers", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	var query map[string]string
	if opts.Recurse {
		query = map[string]string{"recurse": strconv.FormatBool(true)}
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, query, nil)
}

func (c *WorkspaceClient) decodeList(ctx context.Context, op, u string) ([]Ref, error) {
	var wrap listWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.WMTSLayers) == 0 || wrap.WMTSLayers[0] == '"' {
		return nil, nil
	}
	var inner struct {
		WMTSLayer []Ref `json:"wmtsLayer"`
	}
	if err := json.Unmarshal(wrap.WMTSLayers, &inner); err != nil {
		return nil, fmt.Errorf("%s: decode list: %w", op, err)
	}
	return inner.WMTSLayer, nil
}

// StoreClient is store-scoped CRUD —
// /workspaces/{ws}/wmtsstores/{store}/wmtslayers — the canonical
// surface for creating cascaded layers (Create requires a store
// parent).
type StoreClient struct {
	core      Core
	workspace string
	store     string
}

// Workspace returns the workspace bound to this client.
func (c *StoreClient) Workspace() string { return c.workspace }

// Store returns the WMS store bound to this client.
func (c *StoreClient) Store() string { return c.store }

// List returns every cascaded WMTS layer under this store.
func (c *StoreClient) List(ctx context.Context, _ ListOptions) ([]Ref, error) {
	const op = "WMTSLayers.Store.List"
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtsstores", c.store, "wmtslayers")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return decodeListAt(ctx, op, c.core, u)
}

// Iter is the range-over-func pagination helper. The wmtslayers
// endpoint doesn't paginate; this is a single-page Seq2.
func (c *StoreClient) Iter(ctx context.Context, opts ListOptions) iter.Seq2[Ref, error] {
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

// Get returns one cascaded WMTS layer under this store.
func (c *StoreClient) Get(ctx context.Context, name string) (*WMTSLayer, error) {
	const op = "WMTSLayers.Store.Get"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtsstores", c.store, "wmtslayers", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var l WMTSLayer
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// Create publishes a cascaded WMTS layer. Required: Name, NativeName.
func (c *StoreClient) Create(ctx context.Context, layer *WMTSLayer) error {
	const op = "WMTSLayers.Store.Create"
	if layer == nil {
		return errors.New(op + ": nil layer")
	}
	if layer.Name == "" || layer.NativeName == "" {
		return errors.New(op + ": Name and NativeName are required")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtsstores", c.store, "wmtslayers")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, layer, nil, nil)
}

// Update replaces the layer at name.
func (c *StoreClient) Update(ctx context.Context, name string, layer *WMTSLayer) error {
	const op = "WMTSLayers.Store.Update"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if layer == nil {
		return errors.New(op + ": nil layer")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtsstores", c.store, "wmtslayers", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPut, u, layer, nil, nil)
}

// Delete removes a layer.
func (c *StoreClient) Delete(ctx context.Context, name string, opts DeleteOptions) error {
	const op = "WMTSLayers.Store.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "wmtsstores", c.store, "wmtslayers", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	var query map[string]string
	if opts.Recurse {
		query = map[string]string{"recurse": strconv.FormatBool(true)}
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, query, nil)
}

func decodeListAt(ctx context.Context, op string, core Core, u string) ([]Ref, error) {
	var wrap listWire
	if err := core.Do(ctx, op, http.MethodGet, u, nil, nil, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.WMTSLayers) == 0 || wrap.WMTSLayers[0] == '"' {
		return nil, nil
	}
	var inner struct {
		WMTSLayer []Ref `json:"wmtsLayer"`
	}
	if err := json.Unmarshal(wrap.WMTSLayers, &inner); err != nil {
		return nil, fmt.Errorf("%s: decode list: %w", op, err)
	}
	return inner.WMTSLayer, nil
}
