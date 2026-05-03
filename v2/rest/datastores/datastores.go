package datastores

import (
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
}

// Client is the v2 datastores sub-client. It is a thin entry point —
// every CRUD operation is workspace-scoped, so callers obtain a
// [*WorkspaceClient] via [Client.InWorkspace] and operate from there:
//
//	c.Datastores.InWorkspace("topp").Get(ctx, "states_shp")
//	c.Datastores.InWorkspace("topp").List(ctx, datastores.ListOptions{})
//
// Construct via the parent [*geoserver.Client] (do not call [New]
// directly outside the root package's wiring).
type Client struct {
	core Core
}

// New constructs the sub-client. Used by the root [*geoserver.Client]
// wiring; library users access the same instance via `c.Datastores`.
func New(core Core) *Client {
	return &Client{core: core}
}

// InWorkspace returns a workspace-scoped view of the datastores client.
// All methods on the returned [*WorkspaceClient] operate on datastores
// under workspaceName.
//
// workspaceName is not validated here — empty / invalid names surface
// from the per-method calls so callers don't have to handle a second
// error return on the fluent path.
func (c *Client) InWorkspace(workspaceName string) *WorkspaceClient {
	return &WorkspaceClient{core: c.core, workspace: workspaceName}
}

// WorkspaceClient is a workspace-scoped datastore sub-client returned by
// [Client.InWorkspace]. Its methods all act on datastores under the
// captured workspace.
type WorkspaceClient struct {
	core      Core
	workspace string
}

// Workspace returns the workspace name this client is scoped to.
func (c *WorkspaceClient) Workspace() string { return c.workspace }

// List returns every datastore configured under the scoped workspace.
//
// The list endpoint returns a thin shape (typically just Name on each
// entry); fetch a full datastore document with [WorkspaceClient.Get].
//
// Returns a *APIError wrapping ErrNotFound if the workspace itself does
// not exist.
//
// Handles GeoServer's empty-collection wire quirk: an empty datastore
// list comes back as `{"dataStores":""}` (a bare string) rather than
// `{"dataStores":{"dataStore":[]}}`. Both shapes are accepted; the
// empty form returns a nil slice.
func (c *WorkspaceClient) List(ctx context.Context, _ ListOptions) ([]Datastore, error) {
	const op = "Datastores.List"
	if c.workspace == "" {
		return nil, errors.New(op + ": empty workspace name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "datastores")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var envelope struct {
		DataStores json.RawMessage `json:"dataStores"`
	}
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.DataStores) == 0 || string(envelope.DataStores) == "null" || envelope.DataStores[0] == '"' {
		// Empty-collection: GeoServer returns `{"dataStores":""}`.
		return nil, nil
	}
	var inner struct {
		DataStore []Datastore `json:"dataStore"`
	}
	if err := json.Unmarshal(envelope.DataStores, &inner); err != nil {
		return nil, fmt.Errorf("%s: decode datastores list: %w", op, err)
	}
	return inner.DataStore, nil
}

// Iter returns a [iter.Seq2] over the datastore list. Useful when
// callers want range-over-func ergonomics; today the underlying endpoint
// is a single-shot list, so the iterator yields each entry from a
// single fetch.
func (c *WorkspaceClient) Iter(ctx context.Context, opts ListOptions) iter.Seq2[Datastore, error] {
	return func(yield func(Datastore, error) bool) {
		ds, err := c.List(ctx, opts)
		if err != nil {
			yield(Datastore{}, err)
			return
		}
		for _, d := range ds {
			if !yield(d, nil) {
				return
			}
		}
	}
}

// Get fetches the full datastore document for the given name. Returns
// a *APIError wrapping ErrNotFound if no such datastore (or workspace)
// exists.
func (c *WorkspaceClient) Get(ctx context.Context, name string) (*Datastore, error) {
	const op = "Datastores.Get"
	if c.workspace == "" {
		return nil, errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "datastores", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp detailResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.DataStore, nil
}

// Create registers a new datastore under the scoped workspace using the
// payload produced by conn.
//
// Returns nil on success. Returns a *APIError wrapping ErrConflict if a
// datastore with the same name already exists, or ErrNotFound if the
// workspace itself doesn't exist.
//
// For the common cases use the [PostGIS] or [JNDI] convenience types;
// for other drivers, supply a [Datastore] directly via [Raw].
func (c *WorkspaceClient) Create(ctx context.Context, conn Connector) error {
	const op = "Datastores.Create"
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if conn == nil {
		return errors.New(op + ": nil connector")
	}
	store := conn.Datastore()
	if store.Name == "" {
		return errors.New(op + ": empty datastore Name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "datastores")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := createRequest{DataStore: store}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Update modifies a datastore via PUT-as-merge-patch. Pointer fields on
// patch let callers distinguish "field absent" from "field set to false
// / empty string".
//
// Note: GeoServer replaces the entire `connectionParameters` block on
// PUT — to change a single parameter, [WorkspaceClient.Get] the full
// document, mutate the entries you need, and put the whole block back
// in patch.ConnectionParameters.
func (c *WorkspaceClient) Update(ctx context.Context, name string, patch *Patch) error {
	const op = "Datastores.Update"
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if patch == nil {
		return errors.New(op + ": nil patch")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "datastores", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := struct {
		DataStore Patch `json:"dataStore"`
	}{DataStore: *patch}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// Delete removes a datastore. With opts.Recurse=true, also removes all
// contained feature types and layers (a non-empty datastore is rejected
// without Recurse).
func (c *WorkspaceClient) Delete(ctx context.Context, name string, opts DeleteOptions) error {
	const op = "Datastores.Delete"
	if c.workspace == "" {
		return errors.New(op + ": empty workspace name")
	}
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "workspaces", c.workspace, "datastores", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	query := map[string]string{"recurse": strconv.FormatBool(opts.Recurse)}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, query, nil)
}
