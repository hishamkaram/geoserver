package geoserver

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateNamespace(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	created, err := gsCatalog.CreateNamespace("golang_namespace_test", "http://golang.org")
	assert.True(t, created)
	assert.Nil(t, err)
	gsCatalog = GetCatalog("http://localhost:8080/geoserver_dummy/", "admin", "geoserver")
	created, err = gsCatalog.CreateNamespace("golang_namespace_test_dummy", "http://golang.org")
	assert.False(t, created)
	assert.NotNil(t, err)
}

func TestNamespaceExists(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	exists, err := gsCatalog.NamespaceExists("golang_namespace_test")
	assert.True(t, exists)
	assert.Nil(t, err)
	exists, err = gsCatalog.NamespaceExists("golang_namespace_test_dummy")
	assert.False(t, exists)
	assert.NotNil(t, err)
}
func TestGetNamespace(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	namespace, err := gsCatalog.GetNamespace("cite")
	assert.NotNil(t, namespace)
	assert.Nil(t, err)
	namespace, err = gsCatalog.GetNamespace("golang_namespace_test_dummy")
	assert.True(t, IsEmpty(namespace))
	assert.NotNil(t, err)
}
func TestGetNamespaces(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	namespaces, err := gsCatalog.GetNamespaces()
	assert.Nil(t, err)
	assert.False(t, IsEmpty(namespaces))
	assert.NotNil(t, namespaces)
	gsCatalog = GetCatalog("http://localhost:8080/geoserver13/", "admin", "geoserver")
	namespaces, err = gsCatalog.GetNamespaces()
	assert.NotNil(t, err)
	assert.Nil(t, namespaces)
}
func TestDeleteNamespace(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	deleted, err := gsCatalog.DeleteNamespace("golang_namespace_test")
	assert.True(t, deleted)
	assert.Nil(t, err)
	deleted, err = gsCatalog.DeleteNamespace("golang_namespace_test_dummy")
	assert.False(t, deleted)
	assert.NotNil(t, err)
}
func TestGeoserverImplemetNamespaceService(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	NamespaceServiceType := reflect.TypeOf((*NamespaceService)(nil)).Elem()
	check := gsCatalog.Implements(NamespaceServiceType)
	assert.True(t, check)
}
