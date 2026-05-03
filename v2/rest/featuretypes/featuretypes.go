package featuretypes

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

// Client is the v2 feature-types sub-client. Feature types live two
// levels deep in GeoServer's REST hierarchy (workspace → datastore →
// feature type), and the SDK reflects that:
//
//	c.FeatureTypes.InWorkspace("topp").InDatastore("states_pg").Get(ctx, "states")
//
// Construct via the parent [*geoserver.Client] (do not call [New]
// directly outside the root package's wiring).
type Client struct {
	core Core
}

// New constructs the sub-client. Used by the root [*geoserver.Client]
// wiring; library users access the same instance via `c.FeatureTypes`.
func New(core Core) *Client {
	return &Client{core: core}
}

// InWorkspace returns the workspace-scoped intermediate client. Drill
// further with [WorkspaceClient.InDatastore] to obtain the operating
// [*DatastoreClient].
func (c *Client) InWorkspace(workspace string) *WorkspaceClient {
	return &WorkspaceClient{core: c.core, workspace: workspace}
}

// WorkspaceClient is the workspace-scoped intermediate. Its only role
// is to return a [*DatastoreClient] via [WorkspaceClient.InDatastore].
// Workspace-only listing across datastores is intentionally not exposed
// in this version; iterate datastores explicitly if you need that view.
type WorkspaceClient struct {
	core      Core
	workspace string
}

// Workspace returns the workspace name this client is scoped to.
func (c *WorkspaceClient) Workspace() string { return c.workspace }

// InDatastore returns the operating [*DatastoreClient] scoped to the
// given datastore inside this workspace.
func (c *WorkspaceClient) InDatastore(datastore string) *DatastoreClient {
	return &DatastoreClient{core: c.core, workspace: c.workspace, datastore: datastore}
}

// DatastoreClient is the workspace+datastore-scoped feature-type client.
// All CRUD methods live here.
type DatastoreClient struct {
	core      Core
	workspace string
	datastore string
}

// Workspace returns the workspace name this client is scoped to.
func (c *DatastoreClient) Workspace() string { return c.workspace }

// Datastore returns the datastore name this client is scoped to.
func (c *DatastoreClient) Datastore() string { return c.datastore }

// List returns the configured feature types under the scoped datastore.
//
// List entries are sparsely populated by GeoServer — typically only
// Name (and an internal Href) is set. Use [DatastoreClient.Get] for a
// fully-populated [FeatureType].
func (c *DatastoreClient) List(ctx context.Context, _ ListOptions) ([]FeatureType, error) {
	const op = "FeatureTypes.List"
	if err := c.checkScope(op); err != nil {
		return nil, err
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "datastores", c.datastore, "featuretypes")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp listResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.FeatureTypes.FeatureType, nil
}

// Iter returns a [iter.Seq2] over the configured feature-type list.
func (c *DatastoreClient) Iter(ctx context.Context, opts ListOptions) iter.Seq2[FeatureType, error] {
	return func(yield func(FeatureType, error) bool) {
		fts, err := c.List(ctx, opts)
		if err != nil {
			yield(FeatureType{}, err)
			return
		}
		for _, ft := range fts {
			if !yield(ft, nil) {
				return
			}
		}
	}
}

// Get fetches the full feature-type document for the given name.
// Returns a *APIError wrapping ErrNotFound if the feature type, the
// datastore, or the workspace does not exist.
func (c *DatastoreClient) Get(ctx context.Context, name string) (*FeatureType, error) {
	const op = "FeatureTypes.Get"
	if err := c.checkScope(op); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "datastores", c.datastore, "featuretypes", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp detailResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.FeatureType, nil
}

// Create publishes a new feature type under the scoped datastore. Only
// valid for database-backed datastores (PostGIS, Oracle, SQL Server,
// etc.). For shapefile / geopackage stores, use the upload-by-file
// path on the parent datastores client (deferred to a later v2 PR).
//
// Returns nil on success. Returns a *APIError wrapping ErrConflict if a
// feature type with the same name already exists.
func (c *DatastoreClient) Create(ctx context.Context, ft *FeatureType) error {
	const op = "FeatureTypes.Create"
	if err := c.checkScope(op); err != nil {
		return err
	}
	if ft == nil {
		return errors.New(op + ": nil feature type")
	}
	if ft.Name == "" {
		return errors.New(op + ": empty Name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "datastores", c.datastore, "featuretypes")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := createRequest{FeatureType: *ft}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Update modifies a feature type via PUT-as-merge-patch. The whole
// [FeatureType] is sent; fields with zero values that have `omitempty`
// JSON tags are dropped from the wire form.
//
// For partial edits, fetch the current document with [DatastoreClient.Get],
// mutate the fields you need, and PUT the result back. GeoServer's PUT
// semantics for feature types are last-write-wins on the fields that
// actually appear in the request body.
func (c *DatastoreClient) Update(ctx context.Context, name string, ft *FeatureType) error {
	const op = "FeatureTypes.Update"
	if err := c.checkScope(op); err != nil {
		return err
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if ft == nil {
		return errors.New(op + ": nil feature type")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "datastores", c.datastore, "featuretypes", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := createRequest{FeatureType: *ft}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// Delete removes a feature type. With opts.Recurse=true, also removes
// the layer that exposes it; without Recurse a feature type with a
// referencing layer is rejected.
func (c *DatastoreClient) Delete(ctx context.Context, name string, opts DeleteOptions) error {
	const op = "FeatureTypes.Delete"
	if err := c.checkScope(op); err != nil {
		return err
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "datastores", c.datastore, "featuretypes", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	query := map[string]string{"recurse": strconv.FormatBool(opts.Recurse)}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, query, nil)
}

// Discover lists feature-type names that exist in the underlying
// datastore but have not yet been configured as GeoServer feature
// types — useful for finding tables to publish.
//
// The default mode (zero value) is [DiscoverAvailable]. Use
// [DiscoverAvailableWithGeometry] to filter to tables with a geometry
// column, or [DiscoverAll] to also include configured names.
func (c *DatastoreClient) Discover(ctx context.Context, opts DiscoverOptions) ([]string, error) {
	const op = "FeatureTypes.Discover"
	if err := c.checkScope(op); err != nil {
		return nil, err
	}
	kind := opts.Kind
	if kind == "" {
		kind = DiscoverAvailable
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "datastores", c.datastore, "featuretypes")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp discoverResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, map[string]string{"list": string(kind)}, &resp); err != nil {
		return nil, err
	}
	return resp.List.String, nil
}

// checkScope validates that the workspace and datastore captured in
// this client are non-empty. Per-method calls invoke this so the fluent
// path (InWorkspace().InDatastore()) doesn't have to surface a second
// error return.
func (c *DatastoreClient) checkScope(op string) error {
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if c.datastore == "" {
		return errors.New(op + ": empty datastore name")
	}
	return nil
}
