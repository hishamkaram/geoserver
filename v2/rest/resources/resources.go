package resources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Core is the plumbing the sub-client needs from the parent [*Client].
type Core interface {
	URL(parts ...string) (string, error)
	Do(ctx context.Context, op string, method, requestURL string, body any, query map[string]string, out any) error
	DoRaw(ctx context.Context, op, method, requestURL string, body io.Reader, contentType, accept string, query map[string]string) error
	DoStream(ctx context.Context, op string, method, requestURL string, query map[string]string) (io.ReadCloser, int, error)
	// SynthesizeError surfaces a package-sentinel error (via the
	// parent's *APIError) for wire responses that are 2xx but
	// semantically failures — see Resource API's "type=undefined"
	// for missing paths.
	SynthesizeError(op, method, requestURL string, statusCode int, bodyHint string) error
}

// Client is the v2 Resource API sub-client. Construct via [New].
//
//	body, err := c.Resources.Get(ctx, "styles/default_point.sld")
//	if err != nil { return err }
//	defer body.Close()
//
//	dir, err := c.Resources.List(ctx, "styles")
//	for _, child := range dir.Children { ... }
//
//	_ = c.Resources.Put(ctx, "templates/foo.ftl", strings.NewReader(template))
//	_ = c.Resources.Move(ctx, "templates/foo.ftl", "templates/bar.ftl")
//	_ = c.Resources.Delete(ctx, "templates/bar.ftl")
type Client struct {
	core Core
}

// New constructs the resource sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// pathSegments builds the URL path for /rest/resource/<path>. The
// {pathToResource} parameter on the upstream API is treated as a
// raw multi-segment path; this helper splits the user-supplied
// path on "/" and returns each segment so the URL helper can
// path-escape per segment without collapsing the slashes.
//
// An empty path resolves to /rest/resource — the top-level data
// directory listing, which is allowed by the upstream API.
func pathSegments(path string) []string {
	parts := splitPath(path)
	out := make([]string, 0, 2+len(parts))
	out = append(out, "rest", "resource")
	out = append(out, parts...)
	return out
}

// Get streams the contents of a regular-file resource at path. The
// caller owns the returned [io.ReadCloser] and must close it.
//
// Returns [ErrNotFound]-wrapped error if the path does not exist.
// Calling Get on a directory will return GeoServer's directory
// listing (HTML/JSON depending on Accept) — use [Client.List]
// for typed directory access instead.
func (c *Client) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	const op = "Resources.Get"
	u, err := c.core.URL(pathSegments(path)...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	body, _, err := c.core.DoStream(ctx, op, http.MethodGet, u, map[string]string{
		"operation": "default",
	})
	return body, err
}

// Stat returns metadata for the resource at path — the bare
// [Metadata] (no children listed even if path is a directory). For
// the full directory listing including children, use [Client.List].
//
// Wire-quirk: GeoServer's operation=metadata endpoint returns
// 200 OK with type="undefined" for non-existent paths (instead of
// 404). This method translates that into an [ErrNotFound]-bearing
// error so callers can match with errors.Is(err, geoserver.ErrNotFound).
func (c *Client) Stat(ctx context.Context, path string) (*Metadata, error) {
	const op = "Resources.Stat"
	u, err := c.core.URL(pathSegments(path)...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var raw resourceMetadataWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, map[string]string{
		"operation": "metadata",
		"format":    "json",
	}, &raw); err != nil {
		return nil, err
	}
	r := raw.ResourceMetadata
	if r.Type == TypeUndefined {
		return nil, c.core.SynthesizeError(op, http.MethodGet, u, http.StatusNotFound,
			fmt.Sprintf("resource %q not found (GeoServer reported type=undefined)", path))
	}
	return &Metadata{
		Name:         r.Name,
		ParentPath:   r.Parent.Path,
		LastModified: r.LastModified,
		Type:         r.Type,
	}, nil
}

// List returns the directory listing for path. path must reference
// a directory — calling List on a regular file returns an error.
//
// To distinguish "is this a file or a directory?" without fetching
// either form, use [Client.Stat] and inspect [Metadata.Type].
func (c *Client) List(ctx context.Context, path string) (*Directory, error) {
	const op = "Resources.List"
	u, err := c.core.URL(pathSegments(path)...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var raw resourceDirectoryWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, map[string]string{
		"operation": "default",
		"format":    "json",
	}, &raw); err != nil {
		return nil, err
	}
	d := raw.ResourceDirectory
	dir := &Directory{
		Metadata: Metadata{
			Name:         d.Name,
			ParentPath:   d.Parent.Path,
			LastModified: d.LastModified,
			Type:         TypeDirectory,
		},
	}
	if len(d.Children.Child) > 0 {
		dir.Children = make([]Child, 0, len(d.Children.Child))
		for _, c := range d.Children.Child {
			dir.Children = append(dir.Children, Child{
				Name:     c.Name,
				Href:     c.Link.Href,
				MimeType: c.Link.Type,
			})
		}
	}
	return dir, nil
}

// Exists reports whether the resource at path exists, and if it
// does, whether it is a regular file or a directory. A non-existent
// resource is reported as (false, "", nil) — not as an error.
//
// Implemented as a [Client.Stat] under the hood; if you also need
// the full metadata, prefer Stat directly.
func (c *Client) Exists(ctx context.Context, path string) (bool, Type, error) {
	meta, err := c.Stat(ctx, path)
	if err != nil {
		// errNotFound is matched at the package level to avoid
		// hard-coding the parent-package sentinel here. We rely on
		// the [Core.Do] contract that 404 maps to errors.Is(err,
		// ErrNotFound) — checked via the package-level helper.
		if isNotFound(err) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, meta.Type, nil
}

// Put writes a regular-file resource at path. body is uploaded as
// the resource's bytes; contentType, if non-empty, is sent in the
// Content-Type header (otherwise GeoServer guesses from the URL
// extension and the bytes themselves).
//
// Creates the resource if absent and overwrites it if present.
// Per the upstream API, intermediate directories are created on
// the fly. PUT is not supported on directories.
//
// Returns 200 if an existing resource was overwritten and 201 if a
// new resource was created; either is treated as success here. The
// caller does not see the distinction.
func (c *Client) Put(ctx context.Context, path string, body io.Reader, contentType string) error {
	const op = "Resources.Put"
	if path == "" || path == "/" {
		return errors.New(op + ": path must reference a non-root resource")
	}
	u, err := c.core.URL(pathSegments(path)...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.DoRaw(ctx, op, http.MethodPut, u, body, contentType, "*/*", map[string]string{
		"operation": "default",
	})
}

// Move relocates a resource from srcPath to dstPath. Equivalent to
// the upstream PUT /resource/{dstPath}?operation=move with the
// source path as the request body.
//
// Returns [ErrNotFound] if srcPath does not exist; 405 (mapped to
// [ErrMethodNotAllowed]) if dstPath references a directory copy
// (not allowed by upstream).
func (c *Client) Move(ctx context.Context, srcPath, dstPath string) error {
	return c.relocate(ctx, "Resources.Move", srcPath, dstPath, "move")
}

// Copy duplicates a resource from srcPath to dstPath. Equivalent to
// the upstream PUT /resource/{dstPath}?operation=copy with the
// source path as the request body.
//
// Per the upstream API, copy is NOT supported on directories — use
// move + write a new copy if you need to duplicate a directory tree.
//
// Returns [ErrNotFound] if srcPath does not exist.
func (c *Client) Copy(ctx context.Context, srcPath, dstPath string) error {
	return c.relocate(ctx, "Resources.Copy", srcPath, dstPath, "copy")
}

// relocate is the shared implementation behind Move and Copy.
func (c *Client) relocate(ctx context.Context, op, srcPath, dstPath, operation string) error {
	if srcPath == "" || srcPath == "/" {
		return errors.New(op + ": srcPath must reference a non-root resource")
	}
	if dstPath == "" || dstPath == "/" {
		return errors.New(op + ": dstPath must reference a non-root resource")
	}
	u, err := c.core.URL(pathSegments(dstPath)...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.DoRaw(ctx, op, http.MethodPut, u,
		strings.NewReader(strings.TrimPrefix(srcPath, "/")),
		"text/plain", "*/*",
		map[string]string{"operation": operation})
}

// Delete removes the resource at path. Recursive — passing a
// directory removes the directory and all descendants.
//
// Returns [ErrNotFound] if the path does not exist.
func (c *Client) Delete(ctx context.Context, path string) error {
	const op = "Resources.Delete"
	if path == "" || path == "/" {
		return errors.New(op + ": refusing to DELETE the root resource directory")
	}
	u, err := c.core.URL(pathSegments(path)...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
