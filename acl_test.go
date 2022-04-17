package geoserver

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestGeoServer_GetLayersAclRules(t *testing.T) {
	test_before(t)
	rules, err := gsCatalog.GetLayersAclRules()
	assert.Nil(t, err)
	assert.NotEqual(t, len(rules), 0)
}

func TestGeoServer_AddDeleteLayersAclRule(t *testing.T) {
	test_before(t)

	aclRule := AclRule{
		Workspace: "someNonExistentWorkspace",
		Layer:     "*",
		Operation: AclOpRead,
		Roles:     []string{"someNonExistentRole"},
	}

	_, err := gsCatalog.CreateWorkspace(aclRule.Workspace)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		assert.Fail(t, "can't create workspace as a precondition for AddLayersAclRule test")
	}

	_, err = gsCatalog.CreateRole(aclRule.Roles[0])
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		assert.Fail(t, "can't create rule as a precondition for AddLayersAclRule test")
	}

	defer func() {
		_, _ = gsCatalog.DeleteRole(aclRule.Roles[0])
		_, _ = gsCatalog.DeleteWorkspace(aclRule.Workspace, true)
	}()

	_, err = gsCatalog.DeleteLayersAclRule(aclRule)

	done, err := gsCatalog.AddLayersAclRule(aclRule)
	assert.Nil(t, err)
	assert.True(t, done)

	done, err = gsCatalog.DeleteLayersAclRule(aclRule)
	assert.Nil(t, err)
	assert.True(t, done)

}
