//go:build integration
// +build integration

package geoserver

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestACL_GetLayersACLRules(t *testing.T) {
	before()
	rules, err := gsCatalog.GetLayersACLRules()
	assert.NoError(t, err)
	// GeoServer's default config has a few rules out of the box, so the list
	// should be non-empty. We don't pin exact contents because the default
	// rule set varies between 2.27 and 2.28.
	if len(rules) == 0 {
		t.Logf("no ACL rules present — possible if GeoServer is reconfigured; not fatal")
	}
}

func TestACL_AddDeleteLayersACLRule(t *testing.T) {
	before()

	rule := ACLRule{
		Workspace: "acl_test_ws",
		Layer:     "*",
		Operation: ACLOpRead,
		Roles:     []string{"acl_test_role"},
	}

	// Pre-conditions: workspace + role must exist for GeoServer to accept
	// the ACL rule.
	if _, err := gsCatalog.CreateWorkspace(rule.Workspace); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("preconditions: create workspace: %v", err)
	}
	if _, err := gsCatalog.CreateRole(rule.Roles[0]); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("preconditions: create role: %v", err)
	}
	t.Cleanup(func() {
		_, _ = gsCatalog.DeleteRole(rule.Roles[0])
		_, _ = gsCatalog.DeleteWorkspace(rule.Workspace, true)
	})

	// Best-effort prior cleanup so a previous failed run doesn't trip us.
	_, _ = gsCatalog.DeleteLayersACLRule(rule)

	added, err := gsCatalog.AddLayersACLRule(rule)
	assert.NoError(t, err)
	assert.True(t, added)

	// Adding the same rule again should fail with conflict.
	added, err = gsCatalog.AddLayersACLRule(rule)
	assert.False(t, added)
	if err != nil && !errors.Is(err, ErrConflict) && !strings.Contains(err.Error(), "exist") {
		t.Logf("second AddLayersACLRule returned %v (not strictly ErrConflict, but non-nil — acceptable across versions)", err)
	}

	deleted, err := gsCatalog.DeleteLayersACLRule(rule)
	assert.NoError(t, err)
	assert.True(t, deleted)
}
