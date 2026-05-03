package wfs

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	DoXML(ctx context.Context, op, method, requestURL string, query map[string]string, out any) error
}

// Client is the v2 WFS sub-client. The current surface covers
// [Client.GetCapabilities]; [Client.InWorkspace] returns a
// workspace-scoped view that issues `/{workspace}/wfs` rather than
// the global `/wfs`.
//
//	caps, err := c.WFS.GetCapabilities(ctx, wfs.GetCapabilitiesOptions{})
//	caps, err := c.WFS.InWorkspace("topp").GetCapabilities(ctx, wfs.GetCapabilitiesOptions{})
//
// Construct via the parent [*geoserver.Client]; do not call [New]
// directly outside the root package's wiring.
type Client struct {
	core      Core
	workspace string
}

// New constructs the global-scope WFS sub-client.
func New(core Core) *Client { return &Client{core: core} }

// InWorkspace returns a fresh WFS client scoped to the given
// workspace. The original (global-scope) client is unaffected.
func (c *Client) InWorkspace(workspace string) *Client {
	return &Client{core: c.core, workspace: workspace}
}

// Workspace returns the workspace name this client is scoped to,
// or "" for the global scope.
func (c *Client) Workspace() string { return c.workspace }

// IsGlobal reports whether this client operates against the global
// `/wfs` endpoint (true) or a workspace-scoped one (false).
func (c *Client) IsGlobal() bool { return c.workspace == "" }

// GetCapabilitiesOptions controls a [Client.GetCapabilities] call.
// All fields are optional.
type GetCapabilitiesOptions struct {
	// Version is the WFS protocol version requested. Default
	// "2.0.0" — GeoServer's modern default. Supported values for
	// this type tree are "1.1.0" and "2.0.0"; the wire format is
	// the same root element (`<WFS_Capabilities>`) for both.
	Version string

	// UpdateSequence is an optional cache-coordination token.
	UpdateSequence string
}

// GetCapabilities fetches the WFS GetCapabilities XML document and
// parses it into a [*Capabilities]. On a 4xx/5xx response, returns a
// *APIError wrapping the appropriate sentinel.
func (c *Client) GetCapabilities(ctx context.Context, opts GetCapabilitiesOptions) (*Capabilities, error) {
	const op = "WFS.GetCapabilities"

	parts := []string{}
	if c.workspace != "" {
		parts = append(parts, c.workspace)
	}
	parts = append(parts, "wfs")
	u, err := c.core.URL(parts...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	version := opts.Version
	if version == "" {
		version = "2.0.0"
	}
	query := map[string]string{
		"service": "wfs",
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

// ParseCapabilities reads a WFS GetCapabilities XML document from r
// and decodes it into a [*Capabilities]. Useful for parsing a
// document fetched out-of-band (saved fixture, custom transport).
// Returns a typed parse error on malformed input.
func ParseCapabilities(r io.Reader) (*Capabilities, error) {
	if r == nil {
		return nil, errors.New("wfs: ParseCapabilities: nil reader")
	}
	var caps Capabilities
	if err := xml.NewDecoder(r).Decode(&caps); err != nil {
		return nil, fmt.Errorf("wfs: parse capabilities: %w", err)
	}
	return &caps, nil
}

// DescribeFeatureTypeOptions controls a [Client.DescribeFeatureType]
// call. TypeNames is required (the prefixed feature-type names to
// describe); a single call may request multiple types.
type DescribeFeatureTypeOptions struct {
	// TypeNames is the list of prefixed feature-type names to
	// describe (e.g., []string{"topp:states"}).
	TypeNames []string

	// Version is the WFS protocol version requested. Default
	// "2.0.0". The XSD response shape is broadly compatible across
	// versions; the type tree decodes both 1.1.0 and 2.0.0 output.
	Version string
}

// DescribeFeatureType fetches the XSD schema describing one or more
// published feature types and parses it into a [*FeatureSchema].
// Use [FeatureSchema.Attributes] for the flat attribute list.
//
// Returns a *APIError wrapping the appropriate sentinel on a 4xx/5xx
// response. An empty TypeNames list returns the schema for every
// published feature type — that response can be large.
func (c *Client) DescribeFeatureType(ctx context.Context, opts DescribeFeatureTypeOptions) (*FeatureSchema, error) {
	const op = "WFS.DescribeFeatureType"

	parts := []string{}
	if c.workspace != "" {
		parts = append(parts, c.workspace)
	}
	parts = append(parts, "wfs")
	u, err := c.core.URL(parts...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	version := opts.Version
	if version == "" {
		version = "2.0.0"
	}
	query := map[string]string{
		"service": "wfs",
		"version": version,
		"request": "DescribeFeatureType",
	}
	if len(opts.TypeNames) > 0 {
		// WFS 2.0 expects "typeNames" (plural, comma-separated);
		// 1.1.0 expects "typeName". Send both to keep callers
		// version-agnostic — GeoServer ignores the irrelevant one.
		joined := strings.Join(opts.TypeNames, ",")
		query["typeNames"] = joined
		query["typeName"] = joined
	}

	var schema FeatureSchema
	if err := c.core.DoXML(ctx, op, http.MethodGet, u, query, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// ParseFeatureSchema reads a WFS DescribeFeatureType XSD document
// from r and decodes it into a [*FeatureSchema]. Useful for parsing
// a document fetched out-of-band.
func ParseFeatureSchema(r io.Reader) (*FeatureSchema, error) {
	if r == nil {
		return nil, errors.New("wfs: ParseFeatureSchema: nil reader")
	}
	var schema FeatureSchema
	if err := xml.NewDecoder(r).Decode(&schema); err != nil {
		return nil, fmt.Errorf("wfs: parse feature schema: %w", err)
	}
	return &schema, nil
}
