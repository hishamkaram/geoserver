package layergroups

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
	DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error)
}

// Client is the v2 layer-groups sub-client. Workspace-scoped — every
// CRUD operation goes through [Client.InWorkspace]:
//
//	c.LayerGroups.InWorkspace("topp").Get(ctx, "tasmania")
type Client struct {
	core Core
}

// New constructs the sub-client. Used by the root [*geoserver.Client]
// wiring; library users access the same instance via `c.LayerGroups`.
func New(core Core) *Client {
	return &Client{core: core}
}

// InWorkspace returns a workspace-scoped view of the layer-groups
// client.
func (c *Client) InWorkspace(workspaceName string) *WorkspaceClient {
	return &WorkspaceClient{core: c.core, workspace: workspaceName}
}

// WorkspaceClient is the workspace-scoped layer-groups client.
type WorkspaceClient struct {
	core      Core
	workspace string
}

// Workspace returns the workspace name this client is scoped to.
func (c *WorkspaceClient) Workspace() string { return c.workspace }

// List returns every layer group configured under the scoped
// workspace. List entries are sparsely populated — typically only Name.
func (c *WorkspaceClient) List(ctx context.Context, _ ListOptions) ([]LayerGroup, error) {
	const op = "LayerGroups.List"
	if c.workspace == "" {
		return nil, errors.New(op + ": empty workspace name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "layergroups")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp listResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.LayerGroups.LayerGroup, nil
}

// Iter returns a [iter.Seq2] over the layer-group list.
func (c *WorkspaceClient) Iter(ctx context.Context, opts ListOptions) iter.Seq2[LayerGroup, error] {
	return func(yield func(LayerGroup, error) bool) {
		groups, err := c.List(ctx, opts)
		if err != nil {
			yield(LayerGroup{}, err)
			return
		}
		for _, g := range groups {
			if !yield(g, nil) {
				return
			}
		}
	}
}

// Get fetches the full layer-group document for the given name.
func (c *WorkspaceClient) Get(ctx context.Context, name string) (*LayerGroup, error) {
	const op = "LayerGroups.Get"
	if c.workspace == "" {
		return nil, errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "layergroups", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp detailResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.LayerGroup, nil
}

// Create registers a new layer group under the scoped workspace.
func (c *WorkspaceClient) Create(ctx context.Context, group *LayerGroup) error {
	const op = "LayerGroups.Create"
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if group == nil {
		return errors.New(op + ": nil layer group")
	}
	if group.Name == "" {
		return errors.New(op + ": empty Name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "layergroups")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := createRequest{LayerGroup: *group}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Update modifies a layer group via PUT-as-merge-patch.
func (c *WorkspaceClient) Update(ctx context.Context, name string, group *LayerGroup) error {
	const op = "LayerGroups.Update"
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if group == nil {
		return errors.New(op + ": nil layer group")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "layergroups", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := createRequest{LayerGroup: *group}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// Delete removes a layer group. Note: GeoServer does not support
// `?recurse=` on layer-group delete — the underlying layers are not
// affected; only the group reference is removed.
func (c *WorkspaceClient) Delete(ctx context.Context, name string) error {
	const op = "LayerGroups.Delete"
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "layergroups", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
