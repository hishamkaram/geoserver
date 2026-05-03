package acl

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// ServicesClient operates on the /rest/security/acl/services
// endpoint. Service ACLs control which roles can invoke a given OWS
// operation (e.g. WMS GetMap, WFS GetFeature, WCS GetCoverage).
//
// The wire form is a JSON object whose keys are dotted-pair rule
// strings ("service.operation") and values are comma-separated role
// lists. "*" stands for any value.
type ServicesClient struct {
	core Core
}

// List returns every registered service ACL rule.
func (c *ServicesClient) List(ctx context.Context, _ ListOptions) ([]ServiceRule, error) {
	const op = "ACL.Services.List"
	u, err := c.core.URL("rest", "security", "acl", "services")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	var raw map[string]string
	if err := c.core.Do(ctx, op, http.MethodGet, u, nil, nil, &raw); err != nil {
		return nil, err
	}
	rules := make([]ServiceRule, 0, len(raw))
	for ruleStr, rolesStr := range raw {
		r, parseErr := DecodeServiceRule(ruleStr, rolesStr)
		if parseErr != nil {
			return nil, fmt.Errorf("%s: %w", op, parseErr)
		}
		rules = append(rules, r)
	}
	return rules, nil
}

// Add registers a new service ACL rule. GeoServer returns 200 OK
// (not 201). Adding a rule that already exists is rejected with 409
// Conflict — use [ServicesClient.Update] in that case.
func (c *ServicesClient) Add(ctx context.Context, rule ServiceRule) error {
	const op = "ACL.Services.Add"
	if rule.Service == "" && rule.Operation == "" {
		return errors.New(op + ": empty Service and Operation (Encode would default both to '*'; spell that out explicitly if intended)")
	}
	u, err := c.core.URL("rest", "security", "acl", "services")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	ruleStr, rolesStr := rule.Encode()
	body := map[string]string{ruleStr: rolesStr}
	return c.core.Do(ctx, op, http.MethodPost, u, body, nil, nil)
}

// Update edits an existing service ACL rule's role list. GeoServer's
// PUT semantics require the rule to already exist; modifying a
// non-existent rule fails with 409 Conflict (use [ServicesClient.Add]
// to create instead).
func (c *ServicesClient) Update(ctx context.Context, rule ServiceRule) error {
	const op = "ACL.Services.Update"
	if rule.Service == "" && rule.Operation == "" {
		return errors.New(op + ": empty Service and Operation")
	}
	u, err := c.core.URL("rest", "security", "acl", "services")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	ruleStr, rolesStr := rule.Encode()
	body := map[string]string{ruleStr: rolesStr}
	return c.core.Do(ctx, op, http.MethodPut, u, body, nil, nil)
}

// Delete removes a service ACL rule. The rule is identified by its
// encoded form ("service.operation"); the role list is irrelevant.
func (c *ServicesClient) Delete(ctx context.Context, rule ServiceRule) error {
	const op = "ACL.Services.Delete"
	if rule.Service == "" && rule.Operation == "" {
		return errors.New(op + ": empty Service and Operation")
	}
	ruleStr, _ := rule.Encode()
	u, err := c.core.URL("rest", "security", "acl", "services", ruleStr)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return c.core.Do(ctx, op, http.MethodDelete, u, nil, nil, nil)
}
