package workspaces

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

// Client is the v2 workspaces sub-client. Construct via the parent
// [*geoserver.Client] (do not call [New] directly outside the root
// package's wiring).
type Client struct {
	core Core
}

// New constructs the sub-client. Used by the root [*geoserver.Client]
// wiring; library users access the same instance via
// `c.Workspaces`.
func New(core Core) *Client {
	return &Client{core: core}
}

// List returns every workspace currently configured on the server.
func (c *Client) List(ctx context.Context, _ ListOptions) ([]Workspace, error) {
	const op = "Workspaces.List"
	u, err := c.core.URL("rest", "workspaces")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp listResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Workspaces.Workspace, nil
}

// Iter returns a [iter.Seq2] over the workspace list. Useful when
// callers want range-over-func ergonomics; today the underlying
// endpoint is a single-shot list, so the iterator yields each entry
// from a single fetch. Future paginating endpoints in the v2 surface
// (e.g., layers) follow the same shape.
func (c *Client) Iter(ctx context.Context, opts ListOptions) iter.Seq2[Workspace, error] {
	return func(yield func(Workspace, error) bool) {
		ws, err := c.List(ctx, opts)
		if err != nil {
			yield(Workspace{}, err)
			return
		}
		for _, w := range ws {
			if !yield(w, nil) {
				return
			}
		}
	}
}

// Get fetches the workspace with the given name. Returns a *APIError
// wrapping ErrNotFound if no such workspace exists.
func (c *Client) Get(ctx context.Context, name string) (*Workspace, error) {
	const op = "Workspaces.Get"
	if name == "" {
		return nil, errors.New("Workspaces.Get: empty name")
	}
	u, err := c.core.URL("rest", "workspaces", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp detailResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Workspace, nil
}

// (No Exists method on the v2 surface. Idiomatic Go: call Get and match
// the error against the package sentinel.
//
//	_, err := c.Workspaces.Get(ctx, name)
//	if errors.Is(err, geoserver.ErrNotFound) {
//	    // doesn't exist
//	}
//
// This is the same pattern used by encoding/json and database/sql, and
// avoids the (bool, error) shape that requires callers to disambiguate
// "false, nil = doesn't exist" from "false, err = real failure".)

// Create registers a new workspace.
//
// Returns nil on success. Returns a *APIError wrapping ErrConflict if
// the workspace already exists.
func (c *Client) Create(ctx context.Context, ws *Workspace) error {
	const op = "Workspaces.Create"
	if ws == nil {
		return errors.New("Workspaces.Create: nil workspace")
	}
	if ws.Name == "" {
		return errors.New("Workspaces.Create: empty Name")
	}
	u, err := c.core.URL("rest", "workspaces")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := createRequest{Workspace: *ws}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Update modifies a workspace via PUT-as-merge-patch. Pointer fields on
// patch let callers distinguish "field absent" from "field set to
// false / empty string".
func (c *Client) Update(ctx context.Context, name string, patch *WorkspacePatch) error {
	const op = "Workspaces.Update"
	if name == "" {
		return errors.New("Workspaces.Update: empty name")
	}
	if patch == nil {
		return errors.New("Workspaces.Update: nil patch")
	}
	u, err := c.core.URL("rest", "workspaces", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := struct {
		Workspace WorkspacePatch `json:"workspace"`
	}{Workspace: *patch}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// Delete removes a workspace. With opts.Recurse=true, also removes all
// contained datastores, coverage stores, layer groups, and
// feature/coverage definitions.
func (c *Client) Delete(ctx context.Context, name string, opts DeleteOptions) error {
	const op = "Workspaces.Delete"
	if name == "" {
		return errors.New("Workspaces.Delete: empty name")
	}
	u, err := c.core.URL("rest", "workspaces", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	query := map[string]string{"recurse": strconv.FormatBool(opts.Recurse)}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, query, nil)
}
