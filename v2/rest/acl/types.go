// Package acl is the v2 sub-client for the GeoServer access-control-list
// endpoints under /rest/security/acl. The current surface covers
// layer ACL rules; service-level and catalog-level ACL endpoints
// can be added in follow-up PRs without breaking changes.
//
// GeoServer's ACL wire format encodes a rule as the dotted triple
// "workspace.layer.op" mapped to a comma-separated role list (or "*"
// for any role). This package wraps that into a typed [Rule] with
// [Rule.Encode] / [DecodeRule] for round-tripping.
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
