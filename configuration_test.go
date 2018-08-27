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
}

func TestReloadConfigration(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	success, err := gsCatalog.ReloadConfigration()
	assert.True(t, success)
	assert.Nil(t, err)
}
func TestConfigurationServiceImplemet(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	CatalogType := reflect.TypeOf((*ConfigurationService)(nil)).Elem()
	check := gsCatalog.Implements(CatalogType)
	assert.True(t, check)
}
