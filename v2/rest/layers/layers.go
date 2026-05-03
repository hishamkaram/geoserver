package layers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strconv"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
// Defined here as an interface so this subpackage doesn't import the
// root package (which would create an import cycle since the root
// constructs sub-clients).
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
	DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error)
	// DoRaw sends a non-JSON-decoded request and lets the caller set
	// Content-Type and Accept explicitly. Required for [Client.AddStyle]
	// because GeoServer's POST /layers/{l}/styles returns 201 with an
	// empty body and refuses callers asking for `Accept: application/json`
	// (406 Not Acceptable) — same wire-format quirk as the workspace-
	// scoped POST on /styles. See the comment on [WorkspaceClient.AddStyle].
	DoRaw(ctx context.Context, op, method, requestURL string, body io.Reader, contentType, accept string, query map[string]string) error
}

// Client is the v2 layers sub-client.
//
//	c.Layers.InWorkspace("topp").Get(ctx, "states")
//
// Construct via the parent [*geoserver.Client] (do not call [New]
// directly outside the root package's wiring).
type Client struct {
	core Core
}

// New constructs the sub-client. Used by the root [*geoserver.Client]
// wiring; library users access the same instance via `c.Layers`.
func New(core Core) *Client {
	return &Client{core: core}
}

// InWorkspace returns a workspace-scoped view of the layers client.
func (c *Client) InWorkspace(workspaceName string) *WorkspaceClient {
	return &WorkspaceClient{core: c.core, workspace: workspaceName}
}

// WorkspaceClient is the workspace-scoped layers client.
type WorkspaceClient struct {
	core      Core
	workspace string
}

// Workspace returns the workspace name this client is scoped to.
func (c *WorkspaceClient) Workspace() string { return c.workspace }

// List returns every layer configured under the scoped workspace.
//
// List entries are sparsely populated by GeoServer — typically only
// Name is set. Use [WorkspaceClient.Get] for a fully-populated [Layer].
func (c *WorkspaceClient) List(ctx context.Context, _ ListOptions) ([]Layer, error) {
	const op = "Layers.List"
	if c.workspace == "" {
		return nil, errors.New(op + ": empty workspace name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "layers")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp listResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Layers.Layer, nil
}

// Iter returns a [iter.Seq2] over the layer list.
func (c *WorkspaceClient) Iter(ctx context.Context, opts ListOptions) iter.Seq2[Layer, error] {
	return func(yield func(Layer, error) bool) {
		layers, err := c.List(ctx, opts)
		if err != nil {
			yield(Layer{}, err)
			return
		}
		for _, l := range layers {
			if !yield(l, nil) {
				return
			}
		}
	}
}

// Get fetches the full layer document for the given name.
func (c *WorkspaceClient) Get(ctx context.Context, name string) (*Layer, error) {
	const op = "Layers.Get"
	if c.workspace == "" {
		return nil, errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "layers", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp detailResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Layer, nil
}

// Update modifies a layer via PUT-as-merge-patch. The whole [Layer] is
// sent; fields with zero values that have `omitempty` JSON tags are
// dropped from the wire form.
//
// Common edits: change DefaultStyle, set Queryable=true, attach an
// Attribution. For partial edits, fetch the current document with
// [WorkspaceClient.Get], mutate the fields you need, and PUT the result
// back.
func (c *WorkspaceClient) Update(ctx context.Context, name string, layer *Layer) error {
	const op = "Layers.Update"
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if layer == nil {
		return errors.New(op + ": nil layer")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "layers", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := struct {
		Layer Layer `json:"layer"`
	}{Layer: *layer}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// ListStyles returns the layer's alternative-style list — the styles
// callable through WMS `?styles=<name>` beyond the layer's default
// style. The default style is exposed separately on
// [Layer.DefaultStyle] (read via [WorkspaceClient.Get]); this method
// covers only the additional-styles sub-resource.
//
// An empty list (no alternatives configured) is the common case and
// returns nil with no error.
//
// Note on the wire URL: GeoServer's layer-style sub-resource lives at
// the global `/rest/layers/<workspace>:<layer>/styles` path; the
// workspace-prefixed form (`/rest/workspaces/{ws}/layers/{l}/styles`)
// returns 404. This client keeps the API workspace-scoped (because
// callers naturally think in workspace context) and translates to
// the global qualified-name URL internally. The colon in the
// qualified name is percent-encoded by the URL builder; GeoServer
// accepts both raw and encoded forms.
func (c *WorkspaceClient) ListStyles(ctx context.Context, layer string) ([]Ref, error) {
	const op = "Layers.ListStyles"
	if c.workspace == "" {
		return nil, errors.New(op + ": empty workspace name")
	}
	if layer == "" {
		return nil, errors.New(op + ": empty layer name")
	}
	u, err := c.core.URL("rest", "layers", c.workspace+":"+layer, "styles")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp stylesListResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Styles.Style, nil
}

// AddStyle attaches a style to the layer's alternative-style list.
// With opts.Default=true the call also atomically promotes the new
// style to the layer's default style — equivalent to a separate
// [WorkspaceClient.Update] but in one wire round-trip.
//
// The named style must already exist (registered via
// [styles.Client.Create] either globally or in a workspace).
//
// Removing an alternative style is not exposed as a dedicated method
// because the GeoServer docs do not document a DELETE on this
// sub-resource. Use [WorkspaceClient.Update] with the unwanted
// reference removed from [Layer.Styles] instead.
//
// Wire-format quirks handled here:
//   - URL: see [WorkspaceClient.ListStyles].
//   - Accept header: GeoServer returns 201 with an empty body and
//     refuses callers requesting `Accept: application/json`
//     (responds 406 Not Acceptable). Send `Accept: */*` instead;
//     same workaround that [styles.Client.Create] applies to its
//     workspace-scoped POST.
func (c *WorkspaceClient) AddStyle(ctx context.Context, layer, styleName string, opts AddStyleOptions) error {
	const op = "Layers.AddStyle"
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if layer == "" {
		return errors.New(op + ": empty layer name")
	}
	if styleName == "" {
		return errors.New(op + ": empty styleName")
	}
	u, err := c.core.URL("rest", "layers", c.workspace+":"+layer, "styles")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	bodyJSON, err := json.Marshal(addStyleRequest{Style: addStylePayload{Name: styleName}})
	if err != nil {
		return fmt.Errorf("%s: encode body: %w", op, err)
	}
	var query map[string]string
	if opts.Default {
		query = map[string]string{"default": "true"}
	}
	return c.core.DoRaw(ctx, op, http.MethodPost, u,
		bytes.NewReader(bodyJSON),
		"application/json; charset=utf-8",
		"*/*",
		query)
}

// Delete removes a layer. With opts.Recurse=true, also removes the
// underlying feature type or coverage. Default leaves the data
// resource intact.
func (c *WorkspaceClient) Delete(ctx context.Context, name string, opts DeleteOptions) error {
	const op = "Layers.Delete"
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "layers", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	query := map[string]string{"recurse": strconv.FormatBool(opts.Recurse)}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, query, nil)
}
