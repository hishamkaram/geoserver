package styles

import (
	"bytes"
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
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
	DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error)
	// DoRaw sends a non-JSON body (e.g., SLD XML upload) with explicit
	// Content-Type and Accept. Required for [Client.UploadSLD] and used
	// by [Client.Create] to send the workspace-scoped quirk Accept value.
	DoRaw(ctx context.Context, op, method, requestURL string, body io.Reader, contentType, accept string, query map[string]string) error
}

// Client is the v2 styles sub-client. It is operable directly for the
// global scope, or via [Client.InWorkspace] for a workspace-scoped
// view:
//
//	c.Styles.Get(ctx, "polygon")                       // global
//	c.Styles.InWorkspace("topp").Get(ctx, "states")    // workspace
//
// Construct via the parent [*geoserver.Client] (do not call [New]
// directly outside the root package's wiring).
type Client struct {
	core Core
	// workspace is "" for the global scope.
	workspace string
}

// New constructs the global-scope styles sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// InWorkspace returns a fresh styles client scoped to the given
// workspace. The original (global-scope) client is unaffected.
func (c *Client) InWorkspace(workspace string) *Client {
	return &Client{core: c.core, workspace: workspace}
}

// Workspace returns the workspace name this client is scoped to, or
// "" for the global scope.
func (c *Client) Workspace() string { return c.workspace }

// IsGlobal reports whether this client operates against the global
// /rest/styles endpoint (true) or a workspace-scoped endpoint (false).
func (c *Client) IsGlobal() bool { return c.workspace == "" }

// urlParts builds the URL path parts for the current scope plus extra
// trailing segments.
func (c *Client) urlParts(extra ...string) []string {
	parts := []string{"rest"}
	if c.workspace != "" {
		parts = append(parts, "workspaces", c.workspace)
	}
	parts = append(parts, "styles")
	parts = append(parts, extra...)
	return parts
}

// List returns every style configured under the current scope.
//
// Handles GeoServer's empty-collection wire quirk: an empty styles
// list comes back as `{"styles":""}` (a bare string) rather than
// `{"styles":{"style":[]}}`. Both shapes are accepted; the empty form
// returns a nil slice.
func (c *Client) List(ctx context.Context, _ ListOptions) ([]Style, error) {
	const op = "Styles.List"
	u, err := c.core.URL(c.urlParts()...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var envelope struct {
		Styles json.RawMessage `json:"styles"`
	}
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Styles) == 0 || string(envelope.Styles) == "null" || envelope.Styles[0] == '"' {
		// Empty-collection: GeoServer returns `{"styles":""}`.
		return nil, nil
	}
	var inner struct {
		Style []Style `json:"style"`
	}
	if err := json.Unmarshal(envelope.Styles, &inner); err != nil {
		return nil, fmt.Errorf("%s: decode styles list: %w", op, err)
	}
	return inner.Style, nil
}

// Iter returns a [iter.Seq2] over the styles list.
func (c *Client) Iter(ctx context.Context, opts ListOptions) iter.Seq2[Style, error] {
	return func(yield func(Style, error) bool) {
		ss, err := c.List(ctx, opts)
		if err != nil {
			yield(Style{}, err)
			return
		}
		for _, s := range ss {
			if !yield(s, nil) {
				return
			}
		}
	}
}

// Get fetches the style metadata document for the given name.
func (c *Client) Get(ctx context.Context, name string) (*Style, error) {
	const op = "Styles.Get"
	if name == "" {
		return nil, errors.New(op + ": empty name")
	}
	u, err := c.core.URL(c.urlParts(name)...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var resp detailResponse
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Style, nil
}

// Create registers an empty style — the metadata document only.
// Follow with [Client.UploadSLD] to attach the SLD body.
//
// If style.Filename is empty, "{name}.sld" is used.
//
// The workspace-scoped POST /styles endpoint has a GeoServer quirk: if
// Accept is "application/json" the request is dispatched to a non-
// existent JSON style handler and returns 500. The workaround is to
// send Accept: "*/*" so the request routes to the metadata-creation
// path. v2 applies this automatically on the workspace-scoped path;
// the global path uses the default Accept.
func (c *Client) Create(ctx context.Context, style *Style) error {
	const op = "Styles.Create"
	if style == nil {
		return errors.New(op + ": nil style")
	}
	if style.Name == "" {
		return errors.New(op + ": empty Name")
	}
	if style.Filename == "" {
		style.Filename = style.Name + ".sld"
	}
	u, err := c.core.URL(c.urlParts()...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body, err := json.Marshal(createRequest{Style: *style})
	if err != nil {
		return fmt.Errorf("%s: encode body: %w", op, err)
	}
	accept := ""
	if c.workspace != "" {
		// Workspace-scoped quirk: see GoDoc on Create above.
		accept = "*/*"
	}
	return c.core.DoRaw(ctx, op, http.MethodPost, u,
		bytes.NewReader(body),
		"application/json; charset=utf-8",
		accept,
		nil)
}

// UploadSLD uploads (or replaces) the SLD body for an existing style.
//
// Call [Client.Create] first to register the metadata, then UploadSLD
// to attach the content. Or use [Client.UploadSLDOrCreate] to handle
// both in one call.
//
// The Content-Type defaults to "application/vnd.ogc.sld+xml" (SLD 1.0
// / SE 1.0); override via opts.Format for SE 1.1 or GeoCSS.
func (c *Client) UploadSLD(ctx context.Context, name string, body io.Reader, opts UploadOptions) error {
	const op = "Styles.UploadSLD"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if body == nil {
		return errors.New(op + ": nil body")
	}
	contentType := opts.Format
	if contentType == "" {
		contentType = "application/vnd.ogc.sld+xml"
	}
	u, err := c.core.URL(c.urlParts(name)...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.DoRaw(ctx, op, http.MethodPut, u, body, contentType, "", nil)
}

// Update modifies the style metadata via PUT-as-merge-patch with a
// JSON body. Use this to rename, change Format / LanguageVersion, or
// adjust Filename. To replace the SLD content, use [Client.UploadSLD].
func (c *Client) Update(ctx context.Context, name string, style *Style) error {
	const op = "Styles.Update"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if style == nil {
		return errors.New(op + ": nil style")
	}
	u, err := c.core.URL(c.urlParts(name)...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	body := createRequest{Style: *style}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// Delete removes a style. With opts.Purge=true also removes the
// on-disk SLD file from the GeoServer data directory.
func (c *Client) Delete(ctx context.Context, name string, opts DeleteOptions) error {
	const op = "Styles.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.core.URL(c.urlParts(name)...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	query := map[string]string{"purge": strconv.FormatBool(opts.Purge)}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, query, nil)
}
