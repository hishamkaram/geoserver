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
	// DoRaw sends a non-JSON-encoded request body and lets the caller set
	// Content-Type and Accept explicitly. Used by [WorkspaceClient.UploadFile]
	// and [WorkspaceClient.HarvestGranule] to PUT/POST raster bytes
	// (GeoTIFF, image-mosaic zip, granule blob) to the `/file`, `/url`,
	// or `/external` sub-resources.
	DoRaw(ctx context.Context, op, method, requestURL string, body io.Reader, contentType, accept string, query map[string]string) error
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

// UploadFile publishes a file-backed coverage store by uploading the
// raster contents (or pointing at a remote URL or a server-local path).
//
// The endpoint shape is `PUT /workspaces/{ws}/coveragestores/{name}/{method}[.{ext}]`
// where method is one of `file`, `url`, `external`:
//
//   - [UploadMethodFile] (default): body is the raster bytes (a single
//     GeoTIFF, or a zip containing a mosaic + index files). Default
//     Content-Type `application/zip`.
//   - [UploadMethodURL]: body is a URL string the server fetches.
//     Default Content-Type `text/plain`.
//   - [UploadMethodExternal]: body is a server-local filesystem path
//     string. No file transfer happens. Default Content-Type `text/plain`.
//
// Documented `opts.Extension` values for coverage stores: `geotiff`,
// `worldimage`, `imagemosaic`. Other values are accepted by GeoServer
// at the wire level but not officially supported.
//
// If `opts.Update` is non-empty, it's sent as the `update` query parameter.
//
// To add a new granule to an existing image mosaic without
// reconfiguring the whole store, use [WorkspaceClient.HarvestGranule].
func (c *WorkspaceClient) UploadFile(ctx context.Context, name string, body io.Reader, opts UploadOptions) error {
	return c.putFile(ctx, "CoverageStores.UploadFile", http.MethodPut, name, body, opts)
}

// HarvestGranule appends a new granule to an existing structured
// (image-mosaic) coverage store without reconfiguring the store.
//
// The endpoint shape is `POST /workspaces/{ws}/coveragestores/{name}/{method}[.{ext}]`
// — same URL as [WorkspaceClient.UploadFile] but `POST` instead of
// `PUT`. The body is the granule's raster bytes (typically a single
// GeoTIFF whose extent fits inside the mosaic's coverage envelope).
//
// Use `opts.Method = UploadMethodExternal` and pass a server-local
// path in the body if the granule is already on the server's
// filesystem (avoids transferring large rasters across HTTP).
func (c *WorkspaceClient) HarvestGranule(ctx context.Context, name string, body io.Reader, opts UploadOptions) error {
	return c.putFile(ctx, "CoverageStores.HarvestGranule", http.MethodPost, name, body, opts)
}

// putFile is the shared HTTP plumbing for [WorkspaceClient.UploadFile]
// (PUT, replaces store) and [WorkspaceClient.HarvestGranule] (POST,
// appends granule).
func (c *WorkspaceClient) putFile(ctx context.Context, op, httpMethod, name string, body io.Reader, opts UploadOptions) error {
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if body == nil {
		return errors.New(op + ": nil body")
	}

	method := opts.Method
	if method == "" {
		method = UploadMethodFile
	}
	switch method {
	case UploadMethodFile, UploadMethodURL, UploadMethodExternal:
	default:
		return fmt.Errorf("%s: invalid Method %q (want file / url / external)", op, method)
	}

	segment := string(method)
	if opts.Extension != "" {
		segment = segment + "." + opts.Extension
	}

	u, err := c.core.URL("rest", "workspaces", c.workspace, "coveragestores", name, segment)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	contentType := opts.ContentType
	if contentType == "" {
		switch method {
		case UploadMethodURL, UploadMethodExternal:
			contentType = "text/plain"
		default:
			contentType = "application/zip"
		}
	}

	var query map[string]string
	if opts.Update != "" {
		query = map[string]string{"update": opts.Update}
	}

	return c.core.DoRaw(ctx, op, httpMethod, u, body, contentType, "*/*", query)
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
