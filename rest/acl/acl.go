package acl

import (
	"context"
	"errors"
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

// Client is the v2 ACL sub-client. It exposes four nested sub-clients:
// [Client.Layers], [Client.Services], [Client.REST], and [Client.Catalog].
//
//	rules, _ := c.ACL.Layers().List(ctx, acl.ListOptions{})
//	_ = c.ACL.Services().Add(ctx, acl.ServiceRule{
//	    Service: "wms", Operation: "GetMap", Roles: []string{"ROLE_USER"},
//	})
//	mode, _ := c.ACL.Catalog().Get(ctx)
type Client struct {
	core Core
}

// New constructs the ACL sub-client.
func New(core Core) *Client {
	return &Client{core: core}
}

// Layers returns the layer-ACL sub-client. Layer ACLs control which
// roles can read / write / administer a given workspace.layer entity.
func (c *Client) Layers() *LayersClient {
	return &LayersClient{core: c.core}
}

// Services returns the service-ACL sub-client. Service ACLs control
// which roles can invoke a given OWS operation (e.g. WMS GetMap, WFS
// GetFeature). Rule key shape is "service.operation".
func (c *Client) Services() *ServicesClient {
	return &ServicesClient{core: c.core}
}

// REST returns the REST-ACL sub-client. REST ACLs control which
// roles can hit a given URL pattern with a given set of HTTP methods.
// Rule key shape is "<URL Ant pattern>:<HTTP methods>".
func (c *Client) REST() *RESTClient {
	return &RESTClient{core: c.core}
}

// Catalog returns the catalog-mode sub-client. The catalog mode
// (HIDE / MIXED / CHALLENGE) controls how GeoServer advertises
// secured layers in capabilities documents and how it responds to
// access without the required privileges. The same client also
// exposes the configuration reload endpoint.
func (c *Client) Catalog() *CatalogClient {
	return &CatalogClient{core: c.core}
}

// LayersClient operates on the /rest/security/acl/layers endpoint.
type LayersClient struct {
	core Core
}

// List returns every registered layer ACL rule.
//
// The wire response is a JSON object whose keys are dotted-triple
// rule strings and values are comma-separated role lists; this method
// decodes both into typed [Rule] values.
func (c *LayersClient) List(ctx context.Context, _ ListOptions) ([]Rule, error) {
	const op = "ACL.Layers.List"
	u, err := c.core.URL("rest", "security", "acl", "layers")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var raw map[string]string
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &raw); err != nil {
		return nil, err
	}
	rules := make([]Rule, 0, len(raw))
	for ruleStr, rolesStr := range raw {
		r, parseErr := DecodeRule(ruleStr, rolesStr)
		if parseErr != nil {
			return nil, fmt.Errorf("%s: %w", op, parseErr)
		}
		rules = append(rules, r)
	}
	return rules, nil
}

// Add registers a new layer ACL rule.
//
// GeoServer returns 200 OK (not 201 Created) for ACL additions.
// Adding a rule that already exists is rejected with 409.
func (c *LayersClient) Add(ctx context.Context, rule Rule) error {
	const op = "ACL.Layers.Add"
	if rule.Operation == "" {
		return errors.New(op + ": empty Operation")
	}
	u, err := c.core.URL("rest", "security", "acl", "layers")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	ruleStr, rolesStr := rule.Encode()
	body := map[string]string{ruleStr: rolesStr}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Update edits an existing layer ACL rule's role list. GeoServer's
// PUT semantics require the rule to already exist; modifying a
// non-existent rule fails with 409 Conflict (use [LayersClient.Add]
// to create instead).
func (c *LayersClient) Update(ctx context.Context, rule Rule) error {
	const op = "ACL.Layers.Update"
	if rule.Operation == "" {
		return errors.New(op + ": empty Operation")
	}
	u, err := c.core.URL("rest", "security", "acl", "layers")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	ruleStr, rolesStr := rule.Encode()
	body := map[string]string{ruleStr: rolesStr}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// Delete removes a layer ACL rule. The rule is identified by its
// encoded form ("workspace.layer.op"); the role list is irrelevant
// to the delete path.
func (c *LayersClient) Delete(ctx context.Context, rule Rule) error {
	const op = "ACL.Layers.Delete"
	if rule.Operation == "" {
		return errors.New(op + ": empty Operation")
	}
	ruleStr, _ := rule.Encode()
	u, err := c.core.URL("rest", "security", "acl", "layers", ruleStr)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
