package namespaces

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

// Client is the v2 namespaces sub-client. Namespaces are flat under
// /rest/namespaces (no workspace scoping).
//
//	c.Namespaces.Get(ctx, "topp")
//	c.Namespaces.Create(ctx, &namespaces.Namespace{Prefix: "ne", URI: "http://example.com/ne"})
//
// Construct via the parent [*geoserver.Client] (do not call [New]
// directly outside the root package's wiring).
type Client struct {
	core Core
}

// New constructs the sub-client. Used by the root [*geoserver.Client]
// wiring; library users access the same instance via `c.Namespaces`.
func New(core Core) *Client {
	return &Client{core: core}
}

// List returns every namespace configured on the server.
func (c *Client) List(ctx context.Context, _ ListOptions) ([]Namespace, error) {
	const op = "Namespaces.List"
	u, err := c.core.URL("rest", "namespaces")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp listResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Namespaces.Namespace, nil
}

// Iter returns a [iter.Seq2] over the namespace list.
func (c *Client) Iter(ctx context.Context, opts ListOptions) iter.Seq2[Namespace, error] {
	return func(yield func(Namespace, error) bool) {
		ns, err := c.List(ctx, opts)
		if err != nil {
			yield(Namespace{}, err)
			return
		}
		for _, n := range ns {
			if !yield(n, nil) {
				return
			}
		}
	}
}

// Get fetches the namespace with the given prefix.
func (c *Client) Get(ctx context.Context, prefix string) (*Namespace, error) {
	const op = "Namespaces.Get"
	if prefix == "" {
		return nil, errors.New(op + ": empty prefix")
	}
	u, err := c.core.URL("rest", "namespaces", prefix)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp detailResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Namespace, nil
}

// Create registers a new namespace. The Prefix becomes both the
// namespace prefix and the auto-created backing workspace's name.
//
// Returns nil on success. Returns a *APIError wrapping ErrConflict if
// a namespace with the same prefix already exists.
func (c *Client) Create(ctx context.Context, ns *Namespace) error {
	const op = "Namespaces.Create"
	if ns == nil {
		return errors.New(op + ": nil namespace")
	}
	if ns.Prefix == "" {
		return errors.New(op + ": empty Prefix")
	}
	if ns.URI == "" {
		return errors.New(op + ": empty URI")
	}
	u, err := c.core.URL("rest", "namespaces")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := createRequest{Namespace: *ns}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Update modifies a namespace via PUT-as-merge-patch. Use this to
// change the URI (e.g., update XML namespace URL) without recreating
// the namespace.
func (c *Client) Update(ctx context.Context, prefix string, patch *Patch) error {
	const op = "Namespaces.Update"
	if prefix == "" {
		return errors.New(op + ": empty prefix")
	}
	if patch == nil {
		return errors.New(op + ": nil patch")
	}
	u, err := c.core.URL("rest", "namespaces", prefix)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := struct {
		Namespace Patch `json:"namespace"`
	}{Namespace: *patch}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// Delete removes a namespace. The associated workspace is also
// deleted by GeoServer.
func (c *Client) Delete(ctx context.Context, prefix string) error {
	const op = "Namespaces.Delete"
	if prefix == "" {
		return errors.New(op + ": empty prefix")
	}
	u, err := c.core.URL("rest", "namespaces", prefix)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
