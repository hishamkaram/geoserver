package geoserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateWorkspace(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver13/", "admin", "geoserver")
	created, err := gsCatalog.CreateWorkspace("golang_workspace_test")
	assert.True(t, created)
	assert.Nil(t, err)
}

func TestWorkspaceExists(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver13/", "admin", "geoserver")
	exists, err := gsCatalog.WorkspaceExists("golang_workspace_test")
	assert.True(t, exists)
	assert.Nil(t, err)
}
func TestGetWorkspaces(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver13/", "admin", "geoserver")
	workspaces, err := gsCatalog.GetWorkspaces()
	assert.Nil(t, err)
	assert.False(t, IsEmpty(workspaces))
	assert.NotNil(t, workspaces)
}
func TestDeleteWorkspace(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver13/", "admin", "geoserver")
	deleted, err := gsCatalog.DeleteWorkspace("golang_workspace_test", true)
	assert.True(t, deleted)
	assert.Nil(t, err)
}
