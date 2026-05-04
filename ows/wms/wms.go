package wms

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
// Defined as an interface so this subpackage doesn't import the root
// (which would create an import cycle since the root constructs sub-
// clients).
type Core interface {
	URL(parts ...string) (string, error)
	DoXML(ctx context.Context, op, method, requestURL string, query map[string]string, out any) error
}

// Client is the v2 WMS sub-client. The current surface covers
// [Client.GetCapabilities]; [Client.InWorkspace] returns a
// workspace-scoped view that issues `/ {workspace}/wms` rather than
// the global `/wms`.
//
//	caps, err := c.WMS.GetCapabilities(ctx, wms.GetCapabilitiesOptions{})
//	caps, err := c.WMS.InWorkspace("topp").GetCapabilities(ctx, wms.GetCapabilitiesOptions{})
//
// Construct via the parent [*geoserver.Client]; do not call [New]
// directly outside the root package's wiring.
type Client struct {
	core Core
	// workspace is "" for the global scope.
	workspace string
}

// New constructs the global-scope WMS sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// InWorkspace returns a fresh WMS client scoped to the given
// workspace. Use this when you want only the workspace's layer tree
// in the capabilities document. The original (global-scope) client
// is unaffected.
func (c *Client) InWorkspace(workspace string) *Client {
	return &Client{core: c.core, workspace: workspace}
}

// Workspace returns the workspace name this client is scoped to,
// or "" for the global scope.
func (c *Client) Workspace() string { return c.workspace }

// IsGlobal reports whether this client operates against the global
// `/wms` endpoint (true) or a workspace-scoped one (false).
func (c *Client) IsGlobal() bool { return c.workspace == "" }

// GetCapabilitiesOptions controls a [Client.GetCapabilities] call.
// All fields are optional.
type GetCapabilitiesOptions struct {
	// Version is the WMS protocol version requested. Default
	// "1.1.1" — matches v1's GetCapabilities and is the version
	// this package's [Capabilities] type tree decodes.
	Version string

	// UpdateSequence is an optional cache-coordination token.
	// GeoServer returns 304 / a fresh document depending on the
	// sequence relative to the server's current state. Leave empty
	// to always get the current document.
	UpdateSequence string
}

// GetCapabilities fetches the WMS GetCapabilities XML document and
// parses it into a [*Capabilities]. On a 4xx/5xx response, returns a
// *APIError wrapping the appropriate sentinel.
func (c *Client) GetCapabilities(ctx context.Context, opts GetCapabilitiesOptions) (*Capabilities, error) {
	const op = "WMS.GetCapabilities"

	parts := []string{}
	if c.workspace != "" {
		parts = append(parts, c.workspace)
	}
	parts = append(parts, "wms")
	u, err := c.core.URL(parts...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	version := opts.Version
	if version == "" {
		version = "1.1.1"
	}
	query := map[string]string{
		"service": "wms",
		"version": version,
		"request": "GetCapabilities",
	}
	if opts.UpdateSequence != "" {
		query["updatesequence"] = opts.UpdateSequence
	}

	var caps Capabilities
	if err := c.core.DoXML(ctx, op, http.MethodGet, u, query, &caps); err != nil {
		return nil, err
	}
	return &caps, nil
}

// ParseCapabilities reads a WMS GetCapabilities XML document from r
// and decodes it into a [*Capabilities]. Useful for parsing a
// capabilities document fetched out-of-band — e.g., a saved fixture,
// or a body fetched through a custom transport. Returns a typed parse
// error on malformed input.
func ParseCapabilities(r io.Reader) (*Capabilities, error) {
	if r == nil {
		return nil, errors.New("wms: ParseCapabilities: nil reader")
	}
	var caps Capabilities
	if err := xml.NewDecoder(r).Decode(&caps); err != nil {
		return nil, fmt.Errorf("wms: parse capabilities: %w", err)
	}
	return &caps, nil
}
