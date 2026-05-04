// Package fonts is the v2 sub-client for the GeoServer
// /rest/fonts endpoint. It returns the list of font families the
// JVM exposes to GeoServer's labelling pipeline (SLD TextSymbolizer,
// WMS GetMap labelling). Useful as a sanity check before publishing
// styles that reference specific fonts — typos surface as silent
// label-rendering fallbacks rather than errors otherwise.
package fonts

import (
	"context"
	"fmt"
	"net/http"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
}

// Client is the v2 fonts sub-client.
type Client struct {
	core Core
}

// New constructs the fonts sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// fontsResponse matches the GeoServer wire shape `{"fonts":[...]}`.
type fontsResponse struct {
	Fonts []string `json:"fonts"`
}

// List returns every font family GeoServer can use for SLD labelling.
// The list is sourced from the JVM's `GraphicsEnvironment` plus any
// font files dropped into the data directory's `styles/` subdirectory,
// so the result reflects whatever is on the server's classpath at
// the time of the call. Order is server-defined (typically
// alphabetical for system fonts).
func (c *Client) List(ctx context.Context) ([]string, error) {
	const op = "Fonts.List"
	u, err := c.core.URL("rest", "fonts")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp fontsResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Fonts, nil
}
