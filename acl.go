package geoserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
)

// ACLOperation is a GeoServer ACL operation kind.
type ACLOperation string

// ACL operation constants. GeoServer uses single-letter codes in the rule
// string ("workspace.layer.r"), where "r" is read, "w" is write, "a" is
// admin.
const (
	ACLOpRead  ACLOperation = "r"
	ACLOpWrite ACLOperation = "w"
	ACLOpAdmin ACLOperation = "a"
)

// ACLRule represents a GeoServer layer ACL rule.
//
// Workspace and Layer take an entity name or "*" for any entity. Operation is
// one of [ACLOpRead], [ACLOpWrite], [ACLOpAdmin]. Roles is the list of role
// names allowed to perform the operation; an empty list means "any role"
// ("*").
type ACLRule struct {
	Workspace string       // workspace name or "*" for any workspace
	Layer     string       // layer name or "*" for any layer
	Operation ACLOperation // r / w / a
	Roles     []string     // allowed roles; empty == "*"
}

// ACLService defines GeoServer ACL operations on layers.
type ACLService interface {
	GetLayersACLRules() (rules []ACLRule, err error)
	AddLayersACLRule(rule ACLRule) (added bool, err error)
	DeleteLayersACLRule(rule ACLRule) (deleted bool, err error)
}

// ACLServiceWithContext is the context-aware sibling of [ACLService].
type ACLServiceWithContext interface {
	GetLayersACLRulesContext(ctx context.Context) (rules []ACLRule, err error)
	AddLayersACLRuleContext(ctx context.Context, rule ACLRule) (added bool, err error)
	DeleteLayersACLRuleContext(ctx context.Context, rule ACLRule) (deleted bool, err error)
}

// ToStrings converts an ACLRule into the wire-format pair (rule, roles)
// GeoServer's REST API expects: ("workspace.layer.op", "role1,role2"). Empty
// fields default to "*" (any).
func (rule ACLRule) ToStrings() (ruleString string, rolesString string) {
	ws := rule.Workspace
	if ws == "" {
		ws = "*"
	}
	layer := rule.Layer
	if layer == "" {
		layer = "*"
	}
	op := rule.Operation
	if op == "" {
		op = ACLOpRead
	}
	roles := rule.Roles
	if len(roles) == 0 {
		roles = []string{"*"}
	}
	return fmt.Sprintf("%s.%s.%s", ws, layer, op), strings.Join(roles, ",")
}

// StringToACLRule parses GeoServer's wire format back into an ACLRule. The
// rule string must be of the form "workspace.layer.op"; roles is a
// comma-separated list.
func StringToACLRule(rule string, roles string) (aclRule ACLRule, err error) {
	parts := strings.Split(rule, ".")
	if len(parts) != 3 {
		return ACLRule{}, errors.New("acl: rule string must be 'workspace.layer.op'")
	}
	aclRule.Workspace = parts[0]
	aclRule.Layer = parts[1]
	aclRule.Operation = ACLOperation(parts[2])
	if roles != "" {
		aclRule.Roles = strings.Split(roles, ",")
	}
	return aclRule, nil
}

// GetLayersACLRules lists all registered layer ACL rules using context.Background.
func (g *GeoServer) GetLayersACLRules() (rules []ACLRule, err error) {
	return g.GetLayersACLRulesContext(context.Background())
}

// GetLayersACLRulesContext is the context-aware variant of [GeoServer.GetLayersACLRules].
func (g *GeoServer) GetLayersACLRulesContext(ctx context.Context) (rules []ACLRule, err error) {
	targetURL := g.ParseURL("rest", "security", "acl", "layers")
	httpRequest := HTTPRequest{
		Method: getMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		return nil, g.GetError(responseCode, response)
	}
	var aclResponse map[string]string
	if err = g.DeSerializeJSON(response, &aclResponse); err != nil {
		return nil, fmt.Errorf("GetLayersACLRules: decode response: %w", err)
	}
	rules = make([]ACLRule, 0, len(aclResponse))
	for ruleStr, rolesStr := range aclResponse {
		parsed, parseErr := StringToACLRule(ruleStr, rolesStr)
		if parseErr != nil {
			return nil, fmt.Errorf("GetLayersACLRules: %w", parseErr)
		}
		rules = append(rules, parsed)
	}
	return rules, nil
}

// AddLayersACLRule adds a layer ACL rule using context.Background.
func (g *GeoServer) AddLayersACLRule(rule ACLRule) (added bool, err error) {
	return g.AddLayersACLRuleContext(context.Background(), rule)
}

// AddLayersACLRuleContext is the context-aware variant of [GeoServer.AddLayersACLRule].
//
// GeoServer returns 200 OK (not 201 Created) for ACL additions.
func (g *GeoServer) AddLayersACLRuleContext(ctx context.Context, rule ACLRule) (added bool, err error) {
	targetURL := g.ParseURL("rest", "security", "acl", "layers")
	ruleStr, rolesStr := rule.ToStrings()
	body := map[string]string{ruleStr: rolesStr}
	data, serErr := g.SerializeStruct(body)
	if serErr != nil {
		return false, fmt.Errorf("AddLayersACLRule: serialize: %w", serErr)
	}
	httpRequest := HTTPRequest{
		Method:   postMethod,
		Accept:   jsonType,
		Data:     bytes.NewBuffer(data),
		DataType: jsonType,
		URL:      targetURL,
		Query:    nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		return false, g.GetError(responseCode, response)
	}
	return true, nil
}

// DeleteLayersACLRule removes a layer ACL rule using context.Background.
func (g *GeoServer) DeleteLayersACLRule(rule ACLRule) (deleted bool, err error) {
	return g.DeleteLayersACLRuleContext(context.Background(), rule)
}

// DeleteLayersACLRuleContext is the context-aware variant of [GeoServer.DeleteLayersACLRule].
func (g *GeoServer) DeleteLayersACLRuleContext(ctx context.Context, rule ACLRule) (deleted bool, err error) {
	ruleStr, _ := rule.ToStrings()
	targetURL := g.ParseURL("rest", "security", "acl", "layers", ruleStr)
	httpRequest := HTTPRequest{
		Method: deleteMethod,
		Accept: jsonType,
		URL:    targetURL,
		Query:  nil,
	}
	response, responseCode := g.DoRequestContext(ctx, httpRequest)
	if responseCode != statusOk {
		g.logger.Warn(string(response))
		return false, g.GetError(responseCode, response)
	}
	return true, nil
}
