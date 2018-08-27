package geoserver

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestConfigrationCache(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	success, err := gsCatalog.RestConfigrationCache()
	assert.True(t, success)
	assert.Nil(t, err)
	gsCatalog = GetCatalog("http://localhost:8080/geoserver/dummy_rest", "admin", "geoserver")
	successF, errF := gsCatalog.RestConfigrationCache()
	assert.False(t, successF)
	assert.NotNil(t, errF)
}

func TestReloadConfigration(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	success, err := gsCatalog.ReloadConfigration()
	assert.True(t, success)
	assert.Nil(t, err)
	gsCatalog = GetCatalog("http://localhost:8080/geoserver/dummy_rest", "admin", "geoserver")
	successF, errF := gsCatalog.ReloadConfigration()
	assert.False(t, successF)
	assert.NotNil(t, errF)
}
func TestConfigurationServiceImplemet(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	CatalogType := reflect.TypeOf((*ConfigurationService)(nil)).Elem()
	check := gsCatalog.Implements(CatalogType)
	assert.True(t, check)
}
