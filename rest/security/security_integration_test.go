//go:build integration

package security_test

import (
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/security"
)

func TestSecurity_Users_CRUD_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	username := testenv.UniqueName(t, "user")

	t.Cleanup(func() {
		_ = c.Security.Users().Delete(ctx, username)
	})

	if err := c.Security.Users().Create(ctx, &security.User{
		Name: username, Enabled: true, Password: "test-password-123",
	}); err != nil {
		t.Fatalf("Create user: %v", err)
	}

	users, err := c.Security.Users().List(ctx, security.ListOptions{})
	if err != nil {
		t.Fatalf("List users: %v", err)
	}
	found := false
	for _, u := range users {
		if u.Name == username {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("user %q not found in list", username)
	}

	if err := c.Security.Users().Delete(ctx, username); err != nil {
		t.Fatalf("Delete user: %v", err)
	}
}

func TestSecurity_Groups_CRUD_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	groupname := testenv.UniqueName(t, "grp")

	t.Cleanup(func() {
		_ = c.Security.Groups().Delete(ctx, groupname)
	})

	if err := c.Security.Groups().Create(ctx, groupname); err != nil {
		t.Fatalf("Create group: %v", err)
	}

	groups, err := c.Security.Groups().List(ctx, security.ListOptions{})
	if err != nil {
		t.Fatalf("List groups: %v", err)
	}
	found := false
	for _, g := range groups {
		if g.Name == groupname {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("group %q not found in list (cross-version response shape may have failed)", groupname)
	}

	if err := c.Security.Groups().Delete(ctx, groupname); err != nil {
		t.Fatalf("Delete group: %v", err)
	}
}

func TestSecurity_Roles_AssignToUser_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	username := testenv.UniqueName(t, "rolesuser")
	rolename := testenv.UniqueName(t, "ROLE")
	// GeoServer convention: roles are typically uppercase. Our
	// uniqueName lowercases, so prefix with ROLE_ to mirror the
	// convention without breaking the underlying string.

	t.Cleanup(func() {
		_ = c.Security.Roles.UnassignFromUser(ctx, rolename, username)
		_ = c.Security.Roles.Delete(ctx, rolename)
		_ = c.Security.Users().Delete(ctx, username)
	})

	if err := c.Security.Users().Create(ctx, &security.User{
		Name: username, Enabled: true, Password: "test-password-123",
	}); err != nil {
		t.Fatalf("Create user: %v", err)
	}
	if err := c.Security.Roles.Create(ctx, rolename); err != nil {
		t.Fatalf("Create role: %v", err)
	}
	if err := c.Security.Roles.AssignToUser(ctx, rolename, username); err != nil {
		t.Fatalf("AssignToUser: %v", err)
	}

	roles, err := c.Security.Roles.ForUser(ctx, username)
	if err != nil {
		t.Fatalf("ForUser: %v", err)
	}
	hasRole := false
	for _, r := range roles {
		if r == rolename {
			hasRole = true
			break
		}
	}
	if !hasRole {
		t.Fatalf("role %q not in ForUser(%q): %v (cross-version response shape may have failed)",
			rolename, username, roles)
	}

	// Re-assign should be idempotent (200 OK).
	if err := c.Security.Roles.AssignToUser(ctx, rolename, username); err != nil {
		t.Fatalf("AssignToUser idempotent: %v", err)
	}

	if err := c.Security.Roles.UnassignFromUser(ctx, rolename, username); err != nil {
		t.Fatalf("UnassignFromUser: %v", err)
	}
}
