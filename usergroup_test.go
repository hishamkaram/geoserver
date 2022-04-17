package geoserver

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestGeoServer_GetUsers(t *testing.T) {
	test_before(t)
	users, err := gsCatalog.GetUsers("")
	assert.Nil(t, err)
	assert.NotEqual(t, len(users), 0)

	users, err = gsCatalog.GetUsers("someNonExistentService")
	assert.NotNil(t, err)
}

func TestGeoServer_GetGroups(t *testing.T) {
	test_before(t)
	users, err := gsCatalog.GetGroups("")
	assert.Nil(t, err)
	//no groups in default configuration
	assert.Equal(t, len(users), 0)

	users, err = gsCatalog.GetGroups("someNonExistentService")
	assert.NotNil(t, err)
}

func TestGeoServer_GetRoles(t *testing.T) {
	test_before(t)
	roles, err := gsCatalog.GetRoles()
	assert.Nil(t, err)
	assert.NotEqual(t, len(roles), 0)
}

func TestGeoServer_GetUserRoles(t *testing.T) {
	test_before(t)
	roles, err := gsCatalog.GetUserRoles("admin")
	assert.Nil(t, err)
	assert.NotEqual(t, len(roles), 0)

	roles, err = gsCatalog.GetUserRoles("someNonExistentUser")
	assert.Nil(t, err)
	assert.Equal(t, len(roles), 0)

}

func TestGeoServer_CreateUser(t *testing.T) {
	test_before(t)

	user := "someNonExistentUser"
	created, err := gsCatalog.CreateUser(user, "any", "")
	assert.Nil(t, err)
	assert.True(t, created)

	defer func() {
		_, _ = gsCatalog.DeleteUser(user, "")
	}()

	created, err = gsCatalog.CreateUser(user, "any", "")
	assert.NotNil(t, err)
	assert.False(t, created)
}

func TestGeoServer_CreateGroup(t *testing.T) {
	test_before(t)
	group := "someNonExistentGroup"

	created, err := gsCatalog.CreateGroup(group, "")
	assert.Nil(t, err)
	assert.True(t, created)

	defer func() {
		_, _ = gsCatalog.DeleteGroup(group, "")
	}()

	created, err = gsCatalog.CreateGroup(group, "")
	assert.NotNil(t, err)
	assert.False(t, created)
}

func TestGeoServer_CreateRole(t *testing.T) {
	test_before(t)

	role := "someNonExistentRole"

	defer func() {
		_, _ = gsCatalog.DeleteRole(role)
	}()

	created, err := gsCatalog.CreateRole(role)
	assert.Nil(t, err)
	assert.True(t, created)

	created, err = gsCatalog.CreateRole(role)
	assert.NotNil(t, err)
	assert.False(t, created)

}

func TestGeoServer_AddUserRole(t *testing.T) {
	test_before(t)

	role := "someNonExistentRole"
	user := "someNonExistentUser"
	created, err := gsCatalog.CreateRole(role)
	if !created || err != nil {
		assert.Fail(t, "can't create a role as a precondition for AddUserRole test")
	}

	defer func() {
		_, _ = gsCatalog.DeleteRole(role)
	}()

	created, err = gsCatalog.AddUserRole(role, user)
	assert.Nil(t, err)
	assert.True(t, created)

	created, err = gsCatalog.AddUserRole(role, user)
	assert.Nil(t, err)
	assert.True(t, created)

	created, err = gsCatalog.AddUserRole(role+"2", user)
	assert.NotNil(t, err)
	assert.False(t, created)
}

func TestGeoServer_DeleteUserRole(t *testing.T) {
	test_before(t)

	role := "someNonExistentRole"
	user := "someNonExistentUser"

	created, err := gsCatalog.CreateRole(role)
	if !created || err != nil {
		assert.Fail(t, "can't create a role as a precondition for DeleteUserRole test")
	}

	_, err = gsCatalog.CreateUser(user, "test", "")
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		assert.Fail(t, "can't create user as a precondition for DeleteUserRole test")
	}

	defer func() {
		_, _ = gsCatalog.DeleteRole(role)
		_, _ = gsCatalog.DeleteUser(user, "")
	}()

	deleted, err := gsCatalog.DeleteUserRole(role, user)
	assert.Nil(t, err)
	assert.True(t, deleted)

	//if user doesn't have the role DeleteUserRole returns success anyway
	deleted, err = gsCatalog.DeleteUserRole(role, user)
	assert.Nil(t, err)
	assert.True(t, deleted)

	deleted, err = gsCatalog.DeleteUserRole(role+"2", user)
	assert.NotNil(t, err)
	assert.False(t, deleted)
}

func TestGeoServer_DeleteRole(t *testing.T) {
	test_before(t)

	created, err := gsCatalog.CreateRole("someNonExistentRole")
	if !created || err != nil {
		assert.Fail(t, "can't create a role as a precondition for DeleteRole test")
	}

	deleted, err := gsCatalog.DeleteRole("someNonExistentRole")
	assert.Nil(t, err)
	assert.True(t, deleted)

	deleted, err = gsCatalog.DeleteRole("someNonExistentRole")
	assert.NotNil(t, err)
	assert.False(t, deleted)
}

func TestGeoServer_DeleteUser(t *testing.T) {
	test_before(t)

	user := "UserDeleteTest"
	_, err := gsCatalog.CreateUser(user, "test", "")
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		assert.Fail(t, "can't create user as a precondition for DeleteUser test")
	}

	deleted, err := gsCatalog.DeleteUser(user, "")
	assert.Nil(t, err)
	assert.True(t, deleted)

	deleted, err = gsCatalog.DeleteUser(user, "")
	assert.NotNil(t, err)
	assert.False(t, deleted)
}

func TestGeoServer_DeleteGroup(t *testing.T) {
	test_before(t)

	group := "GroupDeleteTest"
	_, err := gsCatalog.CreateGroup(group, "")
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		assert.Fail(t, "can't create group as a precondition for DeleteUser test")
	}

	deleted, err := gsCatalog.DeleteGroup(group, "")
	assert.Nil(t, err)
	assert.True(t, deleted)

	deleted, err = gsCatalog.DeleteGroup(group, "")
	assert.NotNil(t, err)
	assert.False(t, deleted)
}
