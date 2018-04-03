package geoserver

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCoverageStores(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	coverageStores, err := gsCatalog.GetCoverageStores("nurc")
	assert.NotNil(t, coverageStores)
	assert.Nil(t, err)
}

func TestHDeleteCoverageStore(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	deleted, err := gsCatalog.DeleteCoverageStore("nurc", "worldImageSample", true)
	assert.True(t, deleted)
	assert.Nil(t, err)
}

func TestGeoserverImplemetCoverageService(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	CoverageStoresServiceType := reflect.TypeOf((*CoverageStoresService)(nil)).Elem()
	check := gsCatalog.Implements(CoverageStoresServiceType)
	assert.True(t, check)
}
