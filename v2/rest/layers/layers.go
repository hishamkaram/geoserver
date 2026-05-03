package layers

import (
	"context"
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
