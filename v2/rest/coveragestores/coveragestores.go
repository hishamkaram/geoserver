package coveragestores

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

// Client is the v2 coverage-stores sub-client. Workspace-scoped — every
// CRUD operation goes through [Client.InWorkspace]:
//
//	c.CoverageStores.InWorkspace("ne").Get(ctx, "states_geotiff")
//
// Construct via the parent [*geoserver.Client] (do not call [New]
// directly outside the root package's wiring).
type Client struct {
	core Core
}

// New constructs the sub-client. Used by the root [*geoserver.Client]
// wiring; library users access the same instance via `c.CoverageStores`.
func New(core Core) *Client {
	return &Client{core: core}
}

// InWorkspace returns a workspace-scoped view of the coverage-stores
// client. All methods on the returned [*WorkspaceClient] operate on
// stores under workspaceName.
func (c *Client) InWorkspace(workspaceName string) *WorkspaceClient {
	return &WorkspaceClient{core: c.core, workspace: workspaceName}
}

// WorkspaceClient is a workspace-scoped coverage-store sub-client
// returned by [Client.InWorkspace].
type WorkspaceClient struct {
	core      Core
	workspace string
}

// Workspace returns the workspace name this client is scoped to.
func (c *WorkspaceClient) Workspace() string { return c.workspace }

// List returns every coverage store configured under the scoped
// workspace. Returns a *APIError wrapping ErrNotFound if the workspace
// itself does not exist.
func (c *WorkspaceClient) List(ctx context.Context, _ ListOptions) ([]CoverageStore, error) {
	const op = "CoverageStores.List"
	if c.workspace == "" {
		return nil, errors.New(op + ": empty workspace name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "coveragestores")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp listResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.CoverageStores.CoverageStore, nil
}

// Iter returns a [iter.Seq2] over the coverage-store list.
func (c *WorkspaceClient) Iter(ctx context.Context, opts ListOptions) iter.Seq2[CoverageStore, error] {
	return func(yield func(CoverageStore, error) bool) {
		stores, err := c.List(ctx, opts)
		if err != nil {
			yield(CoverageStore{}, err)
			return
		}
		for _, s := range stores {
			if !yield(s, nil) {
				return
			}
		}
	}
}

// Get fetches the full coverage-store document for the given name.
func (c *WorkspaceClient) Get(ctx context.Context, name string) (*CoverageStore, error) {
	const op = "CoverageStores.Get"
	if c.workspace == "" {
		return nil, errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "coveragestores", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp detailResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.CoverageStore, nil
}

// Create registers a new coverage store under the scoped workspace.
//
// Returns nil on success. Returns a *APIError wrapping ErrConflict if a
// store with the same name already exists.
func (c *WorkspaceClient) Create(ctx context.Context, store *CoverageStore) error {
	const op = "CoverageStores.Create"
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if store == nil {
		return errors.New(op + ": nil coverage store")
	}
	if store.Name == "" {
		return errors.New(op + ": empty Name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "coveragestores")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := createRequest{CoverageStore: *store}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Update modifies a coverage store via PUT-as-merge-patch. Pointer
// fields on patch let callers distinguish "field absent" from "field
// set to false / empty string".
func (c *WorkspaceClient) Update(ctx context.Context, name string, patch *Patch) error {
	const op = "CoverageStores.Update"
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if patch == nil {
		return errors.New(op + ": nil patch")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "coveragestores", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := struct {
		CoverageStore Patch `json:"coverageStore"`
	}{CoverageStore: *patch}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// Delete removes a coverage store. With opts.Recurse=true, also removes
// all configured coverages and the layers that expose them.
func (c *WorkspaceClient) Delete(ctx context.Context, name string, opts DeleteOptions) error {
	const op = "CoverageStores.Delete"
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "coveragestores", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	query := map[string]string{"recurse": strconv.FormatBool(opts.Recurse)}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, query, nil)
}
