package geoserver

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateWorkspace(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	created, err := gsCatalog.CreateWorkspace("golang_workspace_test")
	assert.True(t, created)
	assert.Nil(t, err)
	gsCatalog = GetCatalog("http://localhost:8080/geoserver_dummy/", "admin", "geoserver")
	created, err = gsCatalog.CreateWorkspace("golang_workspace_test_dummy")
	assert.False(t, created)
	assert.NotNil(t, err)
}

func TestWorkspaceExists(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	exists, err := gsCatalog.WorkspaceExists("golang_workspace_test")
	assert.True(t, exists)
	assert.Nil(t, err)
	exists, err = gsCatalog.WorkspaceExists("golang_workspace_test_dummy")
	assert.False(t, exists)
	assert.NotNil(t, err)
}
func TestGetWorkspace(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	workspace, err := gsCatalog.GetWorkspace("cite")
	assert.NotNil(t, workspace)
	assert.Nil(t, err)
	workspace, err = gsCatalog.GetWorkspace("golang_workspace_test_dummy")
	assert.Nil(t, workspace)
	assert.NotNil(t, err)
}
func TestGetWorkspaces(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	workspaces, err := gsCatalog.GetWorkspaces()
	assert.Nil(t, err)
	assert.False(t, IsEmpty(workspaces))
	assert.NotNil(t, workspaces)
	gsCatalog = GetCatalog("http://localhost:8080/geoserver13/", "admin", "geoserver")
	workspaces, err = gsCatalog.GetWorkspaces()
	assert.NotNil(t, err)
	assert.Nil(t, workspaces)
}
func TestDeleteWorkspace(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	deleted, err := gsCatalog.DeleteWorkspace("golang_workspace_test", true)
	assert.True(t, deleted)
	assert.Nil(t, err)
	deleted, err = gsCatalog.DeleteWorkspace("golang_workspace_test_dummy", true)
	assert.False(t, deleted)
	assert.NotNil(t, err)
}
func TestGeoserverImplemetWorkspaceService(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	WorkspaceServiceType := reflect.TypeOf((*WorkspaceService)(nil)).Elem()
	check := gsCatalog.Implements(WorkspaceServiceType)
	assert.True(t, check)
}
