// Package acl is the v2 sub-client for the GeoServer access-control-list
// endpoints under /rest/security/acl. The four covered surfaces are:
//
//   - [LayersClient] (`c.ACL.Layers()`) — per-workspace.layer permissions.
//     Rule key shape: "workspace.layer.op", e.g. "topp.states.w".
//   - [ServicesClient] (`c.ACL.Services()`) — per-OWS-operation permissions.
//     Rule key shape: "service.operation", e.g. "wms.GetMap" or "*.*".
//   - [RESTClient] (`c.ACL.REST()`) — per-REST-pattern permissions.
//     Rule key shape: "<URL Ant pattern>:<HTTP methods>", e.g. "/**:GET".
//   - [CatalogClient] (`c.ACL.Catalog()`) — singleton catalog mode
//     (HIDE / MIXED / CHALLENGE) plus a configuration reload endpoint.
//
// Layers/Services/REST share a common rule wire format: a JSON object
// whose keys are the rule string and values are comma-separated role
// lists ("*" for any role). The typed encode / decode helpers
// ([Rule.Encode] / [DecodeRule], [ServiceRule.Encode] / [DecodeServiceRule],
// [RESTRule.Encode] / [DecodeRESTRule]) round-trip those.
package acl

import (
	"errors"
	"fmt"
	"strings"
)

// Operation is a GeoServer ACL operation kind. GeoServer encodes
// these as single letters in the rule string.
type Operation string

// Recognized ACL operations.
const (
	// OpRead permits read / WMS GetMap / WFS GetFeature.
	OpRead Operation = "r"
	// OpWrite permits WFS-T transactions and modifying the resource.
	OpWrite Operation = "w"
	// OpAdmin permits configuring the resource (deleting it,
	// changing its publishability, etc.).
	OpAdmin Operation = "a"
)

// Rule is a layer ACL rule.
//
// Workspace and Layer take an entity name or "*" for any entity.
// Roles is the list of role names allowed to perform the operation;
// an empty list means "any role" (encoded on the wire as "*").
//
// The wire format is a dotted triple ("workspace.layer.op"), so the
// workspace and layer names cannot contain dots — GeoServer rejects
// such rules.
type Rule struct {
	Workspace string
	Layer     string
	Operation Operation
	Roles     []string
}

// Encode converts the Rule into the wire-format pair (rule, roles)
// GeoServer's REST API expects:
//
//	("workspace.layer.op", "role1,role2")
//
// Empty Workspace / Layer default to "*"; empty Operation defaults to
// [OpRead]; empty Roles defaults to "*".
func (r Rule) Encode() (rule, roles string) {
	ws := r.Workspace
	if ws == "" {
		ws = "*"
	}
	layer := r.Layer
	if layer == "" {
		layer = "*"
	}
	op := r.Operation
	if op == "" {
		op = OpRead
	}
	rs := r.Roles
	if len(rs) == 0 {
		rs = []string{"*"}
	}
	return fmt.Sprintf("%s.%s.%s", ws, layer, op), strings.Join(rs, ",")
}

// DecodeRule parses GeoServer's wire format back into a [Rule].
// rule must be of the form "workspace.layer.op"; rolesStr is a
// comma-separated list (or "*" / "" for "any role").
func DecodeRule(rule, rolesStr string) (Rule, error) {
	parts := strings.Split(rule, ".")
	if len(parts) != 3 {
		return Rule{}, errors.New("acl: rule string must be 'workspace.layer.op'")
	}
	r := Rule{
		Workspace: parts[0],
		Layer:     parts[1],
		Operation: Operation(parts[2]),
	}
	if rolesStr != "" && rolesStr != "*" {
		r.Roles = strings.Split(rolesStr, ",")
	}
	return r, nil
}

// ListOptions controls listing behavior. Currently empty.
type ListOptions struct{}

// ServiceRule is an OWS-operation ACL rule. The wire form is the
// dotted pair "service.operation" mapped to a comma-separated role
// list. Service is e.g. "wms" / "wfs" / "wcs" / "*" (any service).
// Operation is the OWS request name, e.g. "GetMap" / "GetFeature" /
// "*" (any operation).
type ServiceRule struct {
	Service   string
	Operation string
	Roles     []string
}

// Encode converts the ServiceRule into the wire-format pair
// (rule, roles). Empty Service and Operation default to "*"; empty
// Roles defaults to "*".
func (r ServiceRule) Encode() (rule, roles string) {
	s := r.Service
	if s == "" {
		s = "*"
	}
	op := r.Operation
	if op == "" {
		op = "*"
	}
	rs := r.Roles
	if len(rs) == 0 {
		rs = []string{"*"}
	}
	return fmt.Sprintf("%s.%s", s, op), strings.Join(rs, ",")
}

// DecodeServiceRule parses GeoServer's wire format back into a
// [ServiceRule]. rule must be of the form "service.operation".
func DecodeServiceRule(rule, rolesStr string) (ServiceRule, error) {
	parts := strings.Split(rule, ".")
	if len(parts) != 2 {
		return ServiceRule{}, errors.New("acl: service rule must be 'service.operation'")
	}
	r := ServiceRule{Service: parts[0], Operation: parts[1]}
	if rolesStr != "" && rolesStr != "*" {
		r.Roles = strings.Split(rolesStr, ",")
	}
	return r, nil
}

// RESTRule is a per-REST-request ACL rule.
//
// The wire form for REST rules is "<pattern>:<methods>" where
// <pattern> is a Spring URL Ant pattern (e.g. "/**", "/rest/workspaces/**")
// and <methods> is a comma-separated list of HTTP methods (e.g.
// "GET", "POST,PUT,DELETE", or "*" for any method). The path-segment
// form documented for DELETE uses ";" as the separator instead of
// ":"; both [DecodeRESTRule] forms are accepted on input. Encode
// emits the canonical body form using ":" since that is what GET/POST
// bodies use.
//
// Slashes inside the pattern are URL-percent-encoded automatically
// when the rule is used as a path segment for DELETE — the
// [Core.URL] helper applies url.PathEscape per segment.
type RESTRule struct {
	Pattern string
	Methods []string
	Roles   []string
}

// Encode converts the RESTRule into the wire-format pair
// (rule, roles). Empty Pattern defaults to "/**"; empty Methods
// defaults to "*"; empty Roles defaults to "*".
func (r RESTRule) Encode() (rule, roles string) {
	p := r.Pattern
	if p == "" {
		p = "/**"
	}
	m := r.Methods
	if len(m) == 0 {
		m = []string{"*"}
	}
	rs := r.Roles
	if len(rs) == 0 {
		rs = []string{"*"}
	}
	return p + ":" + strings.Join(m, ","), strings.Join(rs, ",")
}

// EncodePathSegment is like [RESTRule.Encode] but emits the
// ";"-separator form GeoServer requires when the rule is embedded
// in a URL path segment (used by DELETE).
func (r RESTRule) EncodePathSegment() string {
	rule, _ := r.Encode()
	// Swap the FIRST ":" (between pattern and methods) for ";".
	if i := strings.Index(rule, ":"); i >= 0 {
		return rule[:i] + ";" + rule[i+1:]
	}
	return rule
}

// DecodeRESTRule parses GeoServer's wire format back into a
// [RESTRule]. Accepts both ":" (body form) and ";" (URL form) as
// the separator between the URL pattern and the methods list.
func DecodeRESTRule(rule, rolesStr string) (RESTRule, error) {
	sep := strings.IndexAny(rule, ":;")
	if sep < 0 {
		return RESTRule{}, errors.New("acl: REST rule must be 'pattern:methods' or 'pattern;methods'")
	}
	r := RESTRule{
		Pattern: rule[:sep],
		Methods: strings.Split(rule[sep+1:], ","),
	}
	if rolesStr != "" && rolesStr != "*" {
		r.Roles = strings.Split(rolesStr, ",")
	}
	return r, nil
}

// CatalogMode controls how GeoServer advertises secured layers and
// behaves when a secured layer is accessed without the necessary
// privileges. See the GeoServer Security manual for semantics.
type CatalogMode string

// Recognized catalog modes.
const (
	// CatalogModeHide makes secured resources invisible to
	// unauthenticated users (omitted from capabilities, 404 on
	// direct access).
	CatalogModeHide CatalogMode = "HIDE"
	// CatalogModeMixed leaves secured resources visible in
	// capabilities but returns 401/403 on direct access without
	// privileges.
	CatalogModeMixed CatalogMode = "MIXED"
	// CatalogModeChallenge forces a 401 challenge with auth headers
	// on every secured resource access.
	CatalogModeChallenge CatalogMode = "CHALLENGE"
)
