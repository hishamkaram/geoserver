// Package wfstransforms is the v2 sub-client for the GeoServer
// XSLT-based WFS output transforms at /rest/services/wfs/transforms.
//
// Transforms let WFS-T producers register XSLT files that re-shape
// GetFeature output into custom formats (HTML reports, KML,
// site-specific XML schemas, etc.). The endpoint is part of the
// `gs-xslt-wfs` extension and is NOT installed in default GeoServer
// distributions — calls against an unequipped server return 404.
//
// The flow is two-step:
//
//  1. Register the transform metadata via [Client.Create] — name,
//     source MIME type, output MIME type, output format, file
//     extension. Returns 201 Created.
//  2. Upload the XSLT body via [Client.PutXSLT] (PUT with
//     Content-Type: application/xslt+xml). Re-uploading replaces
//     the existing XSLT.
//
// Or, in a single shot, [Client.CreateWithXSLT] POSTs the XSLT
// body directly with the metadata fields encoded as query
// parameters. Either is supported by the upstream API.
package wfstransforms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
	DoRaw(ctx context.Context, op, method, requestURL string, body io.Reader, contentType, accept string, query map[string]string) error
	DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error)
}

// Transform is the metadata registration for an XSLT transform.
type Transform struct {
	Name           string `json:"name,omitempty"`
	SourceFormat   string `json:"sourceFormat,omitempty"`
	OutputFormat   string `json:"outputFormat,omitempty"`
	OutputMimeType string `json:"outputMimeType,omitempty"`
	FileExtension  string `json:"fileExtension,omitempty"`
	XSLT           string `json:"xslt,omitempty"`
}

// MarshalJSON wraps the transform in GeoServer's
// `{"transform":{...}}` envelope used by POST/PUT bodies.
func (t Transform) MarshalJSON() ([]byte, error) {
	type alias Transform
	return json.Marshal(map[string]alias{"transform": alias(t)})
}

// UnmarshalJSON accepts both the wrapped and the flat shape.
func (t *Transform) UnmarshalJSON(b []byte) error {
	type alias Transform
	var wrapped struct {
		Transform *alias `json:"transform"`
	}
	if err := json.Unmarshal(b, &wrapped); err == nil && wrapped.Transform != nil {
		*t = Transform(*wrapped.Transform)
		return nil
	}
	var flat alias
	if err := json.Unmarshal(b, &flat); err != nil {
		return err
	}
	*t = Transform(flat)
	return nil
}

// Ref is one entry in the transforms listing.
type Ref struct {
	Name string `json:"name"`
	Href string `json:"href"`
}

// transformsListWire decodes the list-shape envelope:
// `{"transforms":{"transform":[{...}, ...]}}` or
// `{"transforms":""}` (empty).
type transformsListWire struct {
	Transforms json.RawMessage `json:"transforms"`
}

// Client is the v2 WFS XSLT transforms sub-client.
type Client struct {
	core Core
}

// New constructs the WFS XSLT transforms sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// CreateWithXSLTOptions controls the [Client.CreateWithXSLT]
// single-shot upload form.
type CreateWithXSLTOptions struct {
	// Name (required) is the transform's catalog name.
	Name string
	// SourceFormat (e.g. "text/xml; subtype=gml/2.1.2") is the
	// expected GetFeature output that the XSLT consumes.
	SourceFormat string
	// OutputFormat (e.g. "text/html") is the WFS output format
	// name callers will pass as `outputFormat=` to GetFeature.
	OutputFormat string
	// OutputMimeType (e.g. "text/html") is the Content-Type
	// returned to clients.
	OutputMimeType string
	// FileExtension (e.g. "html") used when the output is
	// served as a download.
	FileExtension string
}

// List returns every registered XSLT transform.
func (c *Client) List(ctx context.Context) ([]Ref, error) {
	const op = "WFSTransforms.List"
	u, err := c.core.URL("rest", "services", "wfs", "transforms")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var wrap transformsListWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Transforms) == 0 || wrap.Transforms[0] == '"' {
		return nil, nil
	}
	var inner struct {
		Transform []Ref `json:"transform"`
	}
	if err := json.Unmarshal(wrap.Transforms, &inner); err != nil {
		return nil, fmt.Errorf("%s: decode list: %w", op, err)
	}
	return inner.Transform, nil
}

// Get returns a transform's metadata.
func (c *Client) Get(ctx context.Context, name string) (*Transform, error) {
	const op = "WFSTransforms.Get"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "services", "wfs", "transforms", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var t Transform
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// GetXSLT streams the XSLT body for one transform. The caller owns
// the returned [io.ReadCloser] and must close it.
func (c *Client) GetXSLT(ctx context.Context, name string) (io.ReadCloser, error) {
	const op = "WFSTransforms.GetXSLT"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "services", "wfs", "transforms", name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	body, _, err := c.core.DoStream(ctx, op, http.MethodGet, u, nil)
	return body, err
}

// Create registers the transform metadata WITHOUT an XSLT body.
// Required: Name, SourceFormat, OutputFormat. Use
// [Client.PutXSLT] afterward to upload the XSLT itself.
func (c *Client) Create(ctx context.Context, transform *Transform) error {
	const op = "WFSTransforms.Create"
	if transform == nil {
		return errors.New(op + ": nil transform")
	}
	if transform.Name == "" || transform.SourceFormat == "" || transform.OutputFormat == "" {
		return errors.New(op + ": Name, SourceFormat, and OutputFormat are required")
	}
	u, err := c.core.URL("rest", "services", "wfs", "transforms")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPost, u, transform, nil, nil)
}

// CreateWithXSLT registers a transform AND uploads its XSLT body in
// a single POST. The XSLT body is sent with
// `Content-Type: application/xslt+xml`; metadata fields are encoded
// as query parameters per the upstream API.
func (c *Client) CreateWithXSLT(ctx context.Context, body io.Reader, opts CreateWithXSLTOptions) error {
	const op = "WFSTransforms.CreateWithXSLT"
	if opts.Name == "" || opts.SourceFormat == "" || opts.OutputFormat == "" {
		return errors.New(op + ": Name, SourceFormat, and OutputFormat are required")
	}
	u, err := c.core.URL("rest", "services", "wfs", "transforms")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	query := map[string]string{
		"name":         opts.Name,
		"sourceFormat": opts.SourceFormat,
		"outputFormat": opts.OutputFormat,
	}
	if opts.OutputMimeType != "" {
		query["outputMimeType"] = opts.OutputMimeType
	}
	if opts.FileExtension != "" {
		query["fileExtension"] = opts.FileExtension
	}
	return c.core.DoRaw(ctx, op, http.MethodPost, u, body, "application/xslt+xml", "*/*", query)
}

// PutXSLT replaces the XSLT body of an existing transform.
func (c *Client) PutXSLT(ctx context.Context, name string, body io.Reader) error {
	const op = "WFSTransforms.PutXSLT"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "services", "wfs", "transforms", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.DoRaw(ctx, op, http.MethodPut, u, body, "application/xslt+xml", "*/*", nil)
}

// Update replaces the transform metadata.
func (c *Client) Update(ctx context.Context, name string, transform *Transform) error {
	const op = "WFSTransforms.Update"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if transform == nil {
		return errors.New(op + ": nil transform")
	}
	u, err := c.core.URL("rest", "services", "wfs", "transforms", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodPut, u, transform, nil, nil)
}

// Delete removes a transform.
func (c *Client) Delete(ctx context.Context, name string) error {
	const op = "WFSTransforms.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL("rest", "services", "wfs", "transforms", name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
