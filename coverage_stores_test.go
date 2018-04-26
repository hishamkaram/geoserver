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
	coverageStores, err = gsCatalog.GetCoverageStores("dummy")
	assert.Nil(t, coverageStores)
	assert.NotNil(t, err)

}
func TestCreateCoverageStores(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	coverageStore := CoverageStore{
		Name:        "sfdem_test",
		Description: "sfdem_test Description",
		Type:        "GeoTIFF",
		URL:         "file:data/sf/sfdem.tif",
		Workspace: &Resource{
			Name: "sf",
			Href: "http://localhost:8080/geoserver/rest/workspaces/sf.json",
		},
		Enabled: true,
	}
	created, err := gsCatalog.CreateCoverageStore("sf", coverageStore)
	assert.True(t, created)
	assert.Nil(t, err)
	created, err = gsCatalog.CreateCoverageStore("dummy", CoverageStore{})
	assert.False(t, created)
	assert.NotNil(t, err)
}

func TestHDeleteCoverageStore(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	deleted, err := gsCatalog.DeleteCoverageStore("nurc", "worldImageSample", true)
	assert.True(t, deleted)
	assert.Nil(t, err)
	deleted, err = gsCatalog.DeleteCoverageStore("nurc_dummy", "worldImageSample_dummy", true)
	assert.False(t, deleted)
	assert.NotNil(t, err)
}

func TestGeoserverImplemetCoverageService(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	CoverageStoresServiceType := reflect.TypeOf((*CoverageStoresService)(nil)).Elem()
	check := gsCatalog.Implements(CoverageStoresServiceType)
	assert.True(t, check)
}
