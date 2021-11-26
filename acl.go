package geoserver

import (
	"errors"
	"fmt"
	"strings"
)

type aclOperation string

const (
	AclOpRead  = "r" //acl read operation
	AclOpWrite = "w" //acl write operation
	AclOpAdmin = "a" //acl administer operation
)

// AclRule represents geoserver acl rule,
// Workspace and Layer contains workspace and layer names or "*" for any entity,
// Operation contains on of: AclOpRead, AclOpWrite, AclOpAdmin
// Roles contains array of role names the operation is allowed to
type AclRule struct {
	Workspace string       // workspace name or '*' for any workspace
	Layer     string       // layer name or '*' for any layer
	Operation aclOperation // operation
	Roles     []string     // list of roles allowed to perform the operation, null or empty array equal "*"
}

// ToStrings convert aclRule to string representation a geoserver expects
func (aclRule AclRule) ToStrings() (ruleString string, rolesString string) {
	if aclRule.Workspace == "" {
		aclRule.Workspace = "*"
	}
	if aclRule.Layer == "" {
		aclRule.Layer = "*"
	}
	if aclRule.Operation == "" {
		aclRule.Operation = AclOpRead
	}
	if aclRule.Roles == nil || len(aclRule.Roles) == 0 {
		aclRule.Roles = []string{"*"}
	}

	return fmt.Sprintf("%v.%v.%v", aclRule.Workspace, aclRule.Layer, aclRule.Operation), strings.Join(aclRule.Roles, ",")

}

// StringToAclRule parse and convert aclRule from string representation to AclRule struct
func StringToAclRule(rule string, roles string) (aclRule AclRule, err error) {
	parts := strings.Split(rule, ".")
	if len(parts) != 3 {
		err = errors.New("wrong acl string")
		return
	}
	rolesArr := strings.Split(roles, ",")

	aclRule.Workspace = parts[0]
	aclRule.Layer = parts[1]
	aclRule.Operation = (aclOperation)(parts[2])
	aclRule.Roles = rolesArr

	return
}

// GetLayersAclRules returns array of all registered acl rules for all layers
// err is an error if error occurred else err is nil
func (g *GeoServer) GetLayersAclRules() (rules []AclRule, err error) {
	var aclResponse map[string]string

	targetURL := g.ParseURL("rest", "security", "acl", "layers")

	err = g.requestResource(targetURL, &aclResponse)

	if err != nil {
		return []AclRule{}, err
	}

	rules = make([]AclRule, 0, len(aclResponse))
	for key, value := range aclResponse {
		aclRule, err := StringToAclRule(key, value)
		if err != nil {
			return rules, err
		}
		rules = append(rules, aclRule)
	}
	return
}

// AddLayersAclRule adds acl rule
// err is an error if error occurred else err is nil
func (g *GeoServer) AddLayersAclRule(aclRule AclRule) (done bool, err error) {

	targetURL := g.ParseURL("rest", "security", "acl", "layers")

	ruleString, roleString := aclRule.ToStrings()
	createAclRequest := map[string]string{
		ruleString: roleString,
	}

	return g.createEntity(targetURL, createAclRequest, func(statusCode int, response []byte) error {
		if statusCode != statusOk {
			g.logger.Error(string(response))
			return g.GetError(statusCode, response)
		}
		return nil
	})
}

// DeleteLayersAclRule deletes acl rule
// returns true/false if deleted or not, err is an error if error occurred else err is nil
func (g *GeoServer) DeleteLayersAclRule(aclRule AclRule) (done bool, err error) {
	ruleString, _ := aclRule.ToStrings()
	targetURL := g.ParseURL("rest", "security", "acl", "layers", ruleString)
	return g.deleteEntity(targetURL)
}
