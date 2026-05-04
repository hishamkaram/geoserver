// Package system is the v2 sub-client for GeoServer's server-management
// endpoints under /rest — currently `POST /rest/reload` and
// `POST /rest/reset`. These are operational primitives, not catalog
// resources, so they live in their own sub-client rather than under
// [about] (which is read-only).
package system

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
	DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error)
}

// Client is the v2 system sub-client.
//
//	if err := c.System.Reload(ctx); err != nil { /* … */ }
//	if err := c.System.ResetCache(ctx); err != nil { /* … */ }
//
// Construct via the parent [*geoserver.Client]; do not call [New]
// directly outside the root package's wiring.
type Client struct {
	core Core
}

// New constructs the sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// Reload reloads the GeoServer catalog and configuration from disk.
// Use this after an out-of-band edit to the data directory or after
// rolling out new GeoServer plugins. Drops in-memory caches and
// reconnects every datastore.
//
// Mapped to `POST /rest/reload`. Requires admin auth.
func (c *Client) Reload(ctx context.Context) error {
	const op = "System.Reload"
	u, err := c.core.URL("rest", "reload")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, nil, nil, nil)
}

// ResetCache resets all store, raster, and schema caches. Use this
// when the underlying data has changed (e.g., a PostGIS schema
// migration) and stale GeoServer caches need to be invalidated
// without a full [Client.Reload].
//
// Mapped to `POST /rest/reset`. Requires admin auth.
func (c *Client) ResetCache(ctx context.Context) error {
	const op = "System.ResetCache"
	u, err := c.core.URL("rest", "reset")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, nil, nil, nil)
}
