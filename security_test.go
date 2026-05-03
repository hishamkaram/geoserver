//go:build integration
// +build integration

package geoserver

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurity_GetUsers(t *testing.T) {
	before()
	users, err := gsCatalog.GetUsers("")
	assert.NoError(t, err)
	// Default GeoServer always has at least the admin user.
	assert.NotEmpty(t, users)
	_, err = gsCatalog.GetUsers("nonExistentService")
	assert.Error(t, err)
}

func TestSecurity_GetGroups(t *testing.T) {
	before()
	groups, err := gsCatalog.GetGroups("")
	assert.NoError(t, err)
	// Out of the box, no groups are defined — assertion is on no-error, not count.
	_ = groups
	_, err = gsCatalog.GetGroups("nonExistentService")
	assert.Error(t, err)
}

func TestSecurity_GetRoles(t *testing.T) {
	before()
	roles, err := gsCatalog.GetRoles()
	assert.NoError(t, err)
	assert.NotEmpty(t, roles, "default GeoServer should have at least ROLE_ADMINISTRATOR")
}

func TestSecurity_GetUserRoles_Admin(t *testing.T) {
	before()
	roles, err := gsCatalog.GetUserRoles("admin")
	assert.NoError(t, err)
	assert.NotEmpty(t, roles)
}

func TestSecurity_CreateDeleteUser(t *testing.T) {
	before()
	const user = "secTestUser"

	// Cleanup from any previous failed run.
	_, _ = gsCatalog.DeleteUser(user, "")

	created, err := gsCatalog.CreateUser(user, "p@ss", "")
	assert.NoError(t, err)
	assert.True(t, created)

	// Duplicate create should fail.
	created, err = gsCatalog.CreateUser(user, "p@ss", "")
	assert.False(t, created)
	assert.Error(t, err)

	deleted, err := gsCatalog.DeleteUser(user, "")
	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestSecurity_CreateDeleteGroup(t *testing.T) {
	before()
	const group = "secTestGroup"

	_, _ = gsCatalog.DeleteGroup(group, "")

	created, err := gsCatalog.CreateGroup(group, "")
	assert.NoError(t, err)
	assert.True(t, created)

	deleted, err := gsCatalog.DeleteGroup(group, "")
	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestSecurity_CreateDeleteRole(t *testing.T) {
	before()
	const role = "secTestRole"

	_, _ = gsCatalog.DeleteRole(role)

	created, err := gsCatalog.CreateRole(role)
	assert.NoError(t, err)
	assert.True(t, created)

	created, err = gsCatalog.CreateRole(role)
	assert.False(t, created)
	assert.Error(t, err)

	deleted, err := gsCatalog.DeleteRole(role)
	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestSecurity_AddDeleteUserRole(t *testing.T) {
	before()
	const role = "secTestRole2"
	const user = "secTestUser2"

	// Pre-conditions: role + user must exist (GeoServer 2.27+ requires user
	// to exist for the role association; 2.28 may accept either way).
	if _, err := gsCatalog.CreateRole(role); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("preconditions: create role: %v", err)
	}
	if _, err := gsCatalog.CreateUser(user, "p@ss", ""); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("preconditions: create user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = gsCatalog.DeleteUserRole(role, user)
		_, _ = gsCatalog.DeleteUser(user, "")
		_, _ = gsCatalog.DeleteRole(role)
	})

	added, err := gsCatalog.AddUserRole(role, user)
	assert.NoError(t, err)
	assert.True(t, added)

	// Adding the same association again is idempotent.
	added, err = gsCatalog.AddUserRole(role, user)
	assert.NoError(t, err)
	assert.True(t, added)

	deleted, err := gsCatalog.DeleteUserRole(role, user)
	assert.NoError(t, err)
	assert.True(t, deleted)
}
