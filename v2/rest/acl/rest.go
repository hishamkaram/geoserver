package acl

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// RESTClient operates on the /rest/security/acl/rest endpoint.
// REST ACLs control which roles can hit a given URL pattern with a
// given set of HTTP methods.
//
// The wire form in request/response bodies uses ":" as the separator
// between the URL pattern and the methods list (e.g. "/**:GET");
// the URL-path form used by DELETE uses ";" instead. [DecodeRESTRule]
// accepts both. [RESTRule.Encode] emits the body form; the package
// uses [RESTRule.EncodePathSegment] internally for DELETE.
//
// # Production caveat for REST DELETE
//
// REST ACL rule keys (e.g. "/**:GET", "/rest/workspaces/**:POST,PUT")
// contain characters that GeoServer's HTTP firewall rejects in URL
// paths by default:
//
//   - ";" — Spring Security's StrictHttpFirewall blocks ";" with 500
//     "potentially malicious String". Setting the GeoServer property
//     GEOSERVER_USE_STRICT_FIREWALL=false swaps to the lenient
//     DefaultHttpFirewall and unblocks ";". The dev/test docker stack
//     in this repo sets this; production deployments must do the same.
//   - "/" — both StrictHttpFirewall and DefaultHttpFirewall reject
//     URL-encoded slashes (%2F) with 500 "requestURI cannot contain
//     encoded slash". The fix requires Java-level Spring Security
//     configuration (firewall.setAllowUrlEncodedSlash(true)) which is
//     not exposed via env vars or REST.
//
// As a practical consequence, [RESTClient.Delete] is wired for
// completeness but **does not work against a default GeoServer
// install**. Callers should either:
//
//   - Configure GeoServer's HTTP firewall to allow encoded slashes and
//     semicolons, then this method works as documented.
//   - Use the admin UI or edit security/rest.properties on disk and
//     call [CatalogClient.Reload] to pick up the change.
//   - Replace existing rules via [RESTClient.Update] (which uses PUT
//     to the list endpoint and is unaffected by URL-path quirks).
//
// [RESTClient.Add], [RESTClient.List], and [RESTClient.Update] all
// hit the list endpoint with no rule in the URL path and work
// correctly against a default install.
type RESTClient struct {
	core Core
}

// List returns every registered REST ACL rule.
func (c *RESTClient) List(ctx context.Context, _ ListOptions) ([]RESTRule, error) {
	const op = "ACL.REST.List"
	u, err := c.core.URL("rest", "security", "acl", "rest")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var raw map[string]string
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &raw); err != nil {
		return nil, err
	}
	rules := make([]RESTRule, 0, len(raw))
	for ruleStr, rolesStr := range raw {
		r, parseErr := DecodeRESTRule(ruleStr, rolesStr)
		if parseErr != nil {
			return nil, fmt.Errorf("%s: %w", op, parseErr)
		}
		rules = append(rules, r)
	}
	return rules, nil
}

// Add registers a new REST ACL rule. GeoServer returns 200 OK (not
// 201). Adding a rule that already exists is rejected with 409
// Conflict — use [RESTClient.Update] in that case.
func (c *RESTClient) Add(ctx context.Context, rule RESTRule) error {
	const op = "ACL.REST.Add"
	if rule.Pattern == "" && len(rule.Methods) == 0 {
		return errors.New(op + ": empty Pattern and Methods (Encode would default both to wildcards; spell that out explicitly if intended)")
	}
	u, err := c.core.URL("rest", "security", "acl", "rest")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	ruleStr, rolesStr := rule.Encode()
	body := map[string]string{ruleStr: rolesStr}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Update edits an existing REST ACL rule's role list. GeoServer's
// PUT semantics require the rule to already exist; modifying a
// non-existent rule fails with 409 Conflict (use [RESTClient.Add]
// to create instead).
func (c *RESTClient) Update(ctx context.Context, rule RESTRule) error {
	const op = "ACL.REST.Update"
	if rule.Pattern == "" && len(rule.Methods) == 0 {
		return errors.New(op + ": empty Pattern and Methods")
	}
	u, err := c.core.URL("rest", "security", "acl", "rest")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	ruleStr, rolesStr := rule.Encode()
	body := map[string]string{ruleStr: rolesStr}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// Delete removes a REST ACL rule. The rule is identified by its
// path-segment form "<pattern>;<methods>" (e.g. "/**;GET"); the role
// list is irrelevant.
//
// Wire-quirk: GeoServer requires the slashes inside the pattern, the
// "*" glob characters, and the ";" separator to all be transmitted
// literally — not percent-encoded. Go's [net/url.PathEscape] would
// escape all three, which Tomcat / GeoServer's StrictHttpFirewall
// rejects. This method therefore appends the rule segment manually
// rather than going through the URL helper.
func (c *RESTClient) Delete(ctx context.Context, rule RESTRule) error {
	const op = "ACL.REST.Delete"
	if rule.Pattern == "" && len(rule.Methods) == 0 {
		return errors.New(op + ": empty Pattern and Methods")
	}
	base, err := c.core.URL("rest", "security", "acl", "rest")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	u := base + "/" + rule.EncodePathSegment()
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
