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
	coverageStoresFail, errFail := gsCatalog.GetCoverageStores("dummy")
	assert.Nil(t, coverageStoresFail)
	assert.NotNil(t, errFail)

}
func TestGetCoverageStore(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	coverageStore, err := gsCatalog.GetCoverageStore("nurc", "arcGridSample")
	assert.NotNil(t, coverageStore)
	assert.Nil(t, err)
	coverageStoreFail, errFail := gsCatalog.GetCoverageStore("nurc", "dummy")
	assert.Nil(t, coverageStoreFail)
	assert.NotNil(t, errFail)

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
	createdFail, errFail := gsCatalog.CreateCoverageStore("dummy", CoverageStore{})
	assert.False(t, createdFail)
	assert.NotNil(t, errFail)
}

func TestHDeleteCoverageStore(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	deleted, err := gsCatalog.DeleteCoverageStore("nurc", "worldImageSample", true)
	assert.True(t, deleted)
	assert.Nil(t, err)
	deletedFail, errFail := gsCatalog.DeleteCoverageStore("nurc_dummy", "worldImageSample_dummy", true)
	assert.False(t, deletedFail)
	assert.NotNil(t, errFail)
}
func TestUpdateCoverageStore(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	modified, err := gsCatalog.UpdateCoverageStore("sf", CoverageStore{
		Name:        "sfdem",
		Description: "Updated",
	})
	assert.True(t, modified)
	assert.Nil(t, err)
	modifiedFail, errFail := gsCatalog.UpdateCoverageStore("sf_dummy", CoverageStore{
		Name:        "sfdem",
		Description: "Updated",
	})
	assert.False(t, modifiedFail)
	assert.NotNil(t, errFail)
}

func TestGeoserverImplemetCoverageService(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	CoverageStoresServiceType := reflect.TypeOf((*CoverageStoresService)(nil)).Elem()
	check := gsCatalog.Implements(CoverageStoresServiceType)
	assert.True(t, check)
}
