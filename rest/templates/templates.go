package templates

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
}

// Client is the v2 templates sub-client. It is fluently scoped from
// the global root via [Client.InWorkspace], [Client.InDatastore],
// [Client.InFeatureType], [Client.InCoverageStore], [Client.InCoverage].
type Client struct {
	core Core

	// scopePath is the URL prefix segments for this scope, NOT
	// including "rest" or the trailing "templates". Concrete examples:
	//   global       -> nil
	//   workspace    -> ["workspaces", ws]
	//   datastore    -> ["workspaces", ws, "datastores", ds]
	//   featuretype  -> ["workspaces", ws, "datastores", ds, "featuretypes", ft]
	//   coveragestore-> ["workspaces", ws, "coveragestores", cs]
	//   coverage     -> ["workspaces", ws, "coveragestores", cs, "coverages", cov]
	scopePath []string
}

// New constructs the templates sub-client at the global scope.
func New(core Core) *Client {
	return &Client{core: core}
}

// InWorkspace narrows the scope to the named workspace's
// /workspaces/{ws}/templates endpoint.
func (c *Client) InWorkspace(workspace string) *Client {
	return c.with("workspaces", workspace)
}

// InDatastore narrows further to /workspaces/{ws}/datastores/{ds}/templates.
// Must be called on a workspace-scoped client (after [Client.InWorkspace]).
// On other scopes the result is undefined; the SDK does not validate.
func (c *Client) InDatastore(datastore string) *Client {
	return c.with("datastores", datastore)
}

// InFeatureType narrows to
// /workspaces/{ws}/datastores/{ds}/featuretypes/{ft}/templates.
// Must be called on a datastore-scoped client.
func (c *Client) InFeatureType(featureType string) *Client {
	return c.with("featuretypes", featureType)
}

// InCoverageStore narrows to /workspaces/{ws}/coveragestores/{cs}/templates.
// Must be called on a workspace-scoped client.
func (c *Client) InCoverageStore(coverageStore string) *Client {
	return c.with("coveragestores", coverageStore)
}

// InCoverage narrows to
// /workspaces/{ws}/coveragestores/{cs}/coverages/{cov}/templates.
// Must be called on a coverage-store-scoped client.
func (c *Client) InCoverage(coverage string) *Client {
	return c.with("coverages", coverage)
}

// with returns a copy of the client with extra path segments
// appended. The original client is unchanged so chains can fork.
func (c *Client) with(seg ...string) *Client {
	next := make([]string, 0, len(c.scopePath)+len(seg))
	next = append(next, c.scopePath...)
	next = append(next, seg...)
	return &Client{core: c.core, scopePath: next}
}

// listURL builds the URL for the scope's templates list endpoint.
func (c *Client) listURL() (string, error) {
	parts := make([]string, 0, 2+len(c.scopePath))
	parts = append(parts, "rest")
	parts = append(parts, c.scopePath...)
	parts = append(parts, "templates")
	return c.core.URL(parts...)
}

// templateURL builds the URL for one template within this scope.
// name is normalized via [ensureFTL].
func (c *Client) templateURL(name string) (string, error) {
	parts := make([]string, 0, 3+len(c.scopePath))
	parts = append(parts, "rest")
	parts = append(parts, c.scopePath...)
	parts = append(parts, "templates", ensureFTL(name))
	return c.core.URL(parts...)
}

// List returns every template registered at this scope.
func (c *Client) List(ctx context.Context) ([]TemplateRef, error) {
	const op = "Templates.List"
	u, err := c.listURL()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	// Append .json so GeoServer returns the typed JSON envelope
	// instead of the default HTML page.
	u += ".json"
	var raw templatesWire
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &raw); err != nil {
		return nil, err
	}
	return raw.Infos.Info, nil
}

// Get returns the body of one template at this scope as a string.
// FTL templates are typically small text files; bodies larger than
// a few hundred KB are unusual.
//
// Returns [ErrNotFound]-wrapped error if the template does not exist
// at this scope. Note that GeoServer's template lookup walks from
// most-specific to global at request time — this method ONLY checks
// the exact scope, so a "not found" here doesn't mean the template
// is unavailable to the layer at all.
func (c *Client) Get(ctx context.Context, name string) (string, error) {
	const op = "Templates.Get"
	if name == "" {
		return "", errors.New(op + ": empty name")
	}
	u, err := c.templateURL(name)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	body, _, err := c.core.DoStream(ctx, op, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = body.Close() }()
	b, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("%s: read body: %w", op, err)
	}
	return string(b), nil
}

// Put creates or overwrites a template at this scope. body is
// the FTL source bytes; contentType defaults to "text/plain".
//
// GeoServer returns 201 Created for new templates and 200 OK for
// overwrites; this method treats both as success.
func (c *Client) Put(ctx context.Context, name string, body io.Reader) error {
	const op = "Templates.Put"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	if body == nil {
		body = strings.NewReader("")
	}
	u, err := c.templateURL(name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.DoRaw(ctx, op, http.MethodPut, u, body, "text/plain", "*/*", nil)
}

// PutString is a convenience wrapper around [Client.Put] for the
// common case of a string-literal FTL body.
func (c *Client) PutString(ctx context.Context, name, body string) error {
	return c.Put(ctx, name, strings.NewReader(body))
}

// Delete removes a template at this scope.
func (c *Client) Delete(ctx context.Context, name string) error {
	const op = "Templates.Delete"
	if name == "" {
		return errors.New(op + ": empty name")
	}
	u, err := c.templateURL(name)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
