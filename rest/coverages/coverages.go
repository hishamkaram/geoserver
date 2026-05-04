package coverages

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

// Client is the v2 coverages sub-client. Coverages live two levels
// deep in GeoServer's REST hierarchy (workspace → coverage store →
// coverage), and the SDK reflects that:
//
//	c.Coverages.InWorkspace("ne").InCoverageStore("states_tiff").Get(ctx, "states")
//
// Construct via the parent [*geoserver.Client] (do not call [New]
// directly outside the root package's wiring).
type Client struct {
	core Core
}

// New constructs the sub-client. Used by the root [*geoserver.Client]
// wiring; library users access the same instance via `c.Coverages`.
func New(core Core) *Client {
	return &Client{core: core}
}

// InWorkspace returns the workspace-scoped intermediate client.
func (c *Client) InWorkspace(workspace string) *WorkspaceClient {
	return &WorkspaceClient{core: c.core, workspace: workspace}
}

// WorkspaceClient is the workspace-scoped intermediate. Drill further
// to a coverage store with [WorkspaceClient.InCoverageStore].
type WorkspaceClient struct {
	core      Core
	workspace string
}

// Workspace returns the workspace name this client is scoped to.
func (c *WorkspaceClient) Workspace() string { return c.workspace }

// InCoverageStore returns the operating [*CoverageStoreClient] scoped
// to the given coverage store inside this workspace.
func (c *WorkspaceClient) InCoverageStore(store string) *CoverageStoreClient {
	return &CoverageStoreClient{core: c.core, workspace: c.workspace, store: store}
}

// CoverageStoreClient is the workspace+coverage-store-scoped coverage
// client. All CRUD methods live here.
type CoverageStoreClient struct {
	core      Core
	workspace string
	store     string
}

// Workspace returns the workspace name this client is scoped to.
func (c *CoverageStoreClient) Workspace() string { return c.workspace }

// CoverageStore returns the coverage-store name this client is scoped to.
func (c *CoverageStoreClient) CoverageStore() string { return c.store }

// List returns the configured coverages under the scoped coverage
// store.
//
// List entries are sparsely populated by GeoServer — typically only
// Name is set. Use [CoverageStoreClient.Get] for a fully-populated
// [Coverage].
func (c *CoverageStoreClient) List(ctx context.Context, _ ListOptions) ([]Coverage, error) {
	const op = "Coverages.List"
	if err := c.checkScope(op); err != nil {
		return nil, err
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "coveragestores", c.store, "coverages")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp listResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Coverages.Coverage, nil
}

// Iter returns a [iter.Seq2] over the configured coverage list.
func (c *CoverageStoreClient) Iter(ctx context.Context, opts ListOptions) iter.Seq2[Coverage, error] {
	return func(yield func(Coverage, error) bool) {
		covs, err := c.List(ctx, opts)
		if err != nil {
			yield(Coverage{}, err)
			return
		}
		for _, cov := range covs {
			if !yield(cov, nil) {
				return
			}
		}
	}
}

// Get fetches the full coverage document for the given name.
func (c *CoverageStoreClient) Get(ctx context.Context, name string) (*Coverage, error) {
	const op = "Coverages.Get"
	if err := c.checkScope(op); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "coveragestores", c.store, "coverages", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp detailResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Coverage, nil
}

// Create publishes a new coverage from the underlying coverage store.
//
// The minimum payload is Name + NativeCoverageName: GeoServer derives
// the rest (CRS, bounding box, native format) from the source raster.
// Additional fields override the auto-derived values.
//
// Returns nil on success. Returns a *APIError wrapping ErrConflict if a
// coverage with the same name already exists.
func (c *CoverageStoreClient) Create(ctx context.Context, cov *Coverage) error {
	const op = "Coverages.Create"
	if err := c.checkScope(op); err != nil {
		return err
	}
	if cov == nil {
		return errors.New(op + ": nil coverage")
	}
	if cov.Name == "" {
		return errors.New(op + ": empty Name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "coveragestores", c.store, "coverages")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := createRequest{Coverage: *cov}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Update modifies a coverage via PUT-as-merge-patch. The whole
// [Coverage] is sent; fields with zero values that have `omitempty`
// JSON tags are dropped from the wire form.
//
// For partial edits, fetch the current document with
// [CoverageStoreClient.Get], mutate the fields you need, and PUT the
// result back.
func (c *CoverageStoreClient) Update(ctx context.Context, name string, cov *Coverage) error {
	const op = "Coverages.Update"
	if err := c.checkScope(op); err != nil {
		return err
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if cov == nil {
		return errors.New(op + ": nil coverage")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "coveragestores", c.store, "coverages", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := createRequest{Coverage: *cov}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// Delete removes a coverage. With opts.Recurse=true, also removes the
// layer that exposes it.
func (c *CoverageStoreClient) Delete(ctx context.Context, name string, opts DeleteOptions) error {
	const op = "Coverages.Delete"
	if err := c.checkScope(op); err != nil {
		return err
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "coveragestores", c.store, "coverages", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	query := map[string]string{"recurse": strconv.FormatBool(opts.Recurse)}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, query, nil)
}

// Discover lists native coverage names that exist in the underlying
// coverage store. The default mode (zero value) is [DiscoverAll], which
// returns configured plus available coverages (the more useful default
// for the raster flow — most coverage stores expose a single coverage
// that is already configured, so "available" alone often returns
// nothing).
func (c *CoverageStoreClient) Discover(ctx context.Context, opts DiscoverOptions) ([]string, error) {
	const op = "Coverages.Discover"
	if err := c.checkScope(op); err != nil {
		return nil, err
	}
	kind := opts.Kind
	if kind == "" {
		kind = DiscoverAll
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "coveragestores", c.store, "coverages")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp discoverResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, map[string]string{"list": string(kind)}, &resp); err != nil {
		return nil, err
	}
	return resp.List.String, nil
}

func (c *CoverageStoreClient) checkScope(op string) error {
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if c.store == "" {
		return errors.New(op + ": empty coverage store name")
	}
	return nil
}
