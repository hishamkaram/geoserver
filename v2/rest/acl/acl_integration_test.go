//go:build integration

package acl_test

import (
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/acl"
)

func TestACL_Layers_CRUD_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Use a fake workspace name in the rule — GeoServer accepts ACL
	// rules referring to entities that may or may not exist; the rule
	// is just a permission predicate.
	wsName := testenv.UniqueName(t, "ws")
	layerName := testenv.UniqueName(t, "ly")
	roleName := testenv.UniqueName(t, "ROLE")

	rule := acl.Rule{
		Workspace: wsName,
		Layer:     layerName,
		Operation: acl.OpRead,
		Roles:     []string{roleName},
	}

	t.Cleanup(func() {
		_ = c.ACL.Layers().Delete(ctx, rule)
	})

	if err := c.ACL.Layers().Add(ctx, rule); err != nil {
		t.Fatalf("Add ACL rule: %v", err)
	}

	rules, err := c.ACL.Layers().List(ctx, acl.ListOptions{})
	if err != nil {
		t.Fatalf("List ACL rules: %v", err)
	}
	found := false
	for _, r := range rules {
		if r.Workspace == wsName && r.Layer == layerName && r.Operation == acl.OpRead {
			if len(r.Roles) == 1 && r.Roles[0] == roleName {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("rule for %s.%s.r not found in List", wsName, layerName)
	}

	if err := c.ACL.Layers().Delete(ctx, rule); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Confirm removal.
	rules, err = c.ACL.Layers().List(ctx, acl.ListOptions{})
	if err != nil {
		t.Fatalf("List after Delete: %v", err)
	}
	for _, r := range rules {
		if r.Workspace == wsName && r.Layer == layerName && r.Operation == acl.OpRead {
			t.Fatalf("rule still present after Delete: %+v", r)
		}
	}
}
