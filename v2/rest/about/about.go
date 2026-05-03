// Package about is the v2 sub-client for the GeoServer
// /rest/about/version resource. It surfaces a health-check (Ping) and
// the GeoServer + dependency version document.
package about

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

// Client is the v2 about sub-client.
//
//	if err := c.About.Ping(ctx); err == nil { /* GeoServer is up */ }
//	v, _ := c.About.Version(ctx); fmt.Println(v.Resource[0].Version)
type Client struct {
	core Core
}

// New constructs the sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// VersionInfo wraps the resource list returned by /rest/about/version.
// Each entry is a versioned component (GeoServer itself, GeoTools,
// GeoWebCache, etc.).
type VersionInfo struct {
	Resource []Resource `json:"resource,omitempty"`
}

// Resource is one component in [VersionInfo]. Wire shape uses the
// XML-as-JSON `@name` attribute for the component name.
type Resource struct {
	Name           string `json:"@name,omitempty"`
	Version        string `json:"Version,omitempty"`
	BuildTimestamp string `json:"Build-Timestamp,omitempty"`
	GitRevision    string `json:"Git-Revision,omitempty"`
}

// Ping issues a GET against /rest/about/version and returns nil if
// GeoServer responded with 2xx. Useful for liveness probes from
// orchestration layers.
//
// Returns a *APIError with the underlying status if the server
// answered with a non-2xx, or a transport error if the request never
// reached the server.
func (c *Client) Ping(ctx context.Context) error {
	const op = "About.Ping"
	u, err := c.core.URL("rest", "about", "version")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodGet, u, nil, nil, nil)
}

// Version fetches the full /rest/about/version document — a list of
// component versions (GeoServer core, GeoTools, GeoWebCache, etc.)
// with build timestamps and git revisions.
//
// Use this for richer diagnostics; for a simple "is it up" check,
// [Client.Ping] is cheaper since it discards the body.
func (c *Client) Version(ctx context.Context) (*VersionInfo, error) {
	const op = "About.Version"
	u, err := c.core.URL("rest", "about", "version")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp struct {
		About VersionInfo `json:"about"`
	}
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.About, nil
}
