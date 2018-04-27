package geoserver

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFeatrueTypes(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	featureTypes, err := gsCatalog.GetFeatureTypes("sf", "sf")
	assert.NotNil(t, featureTypes)
	assert.NotEmpty(t, featureTypes)
	assert.Nil(t, err)
	featureTypes, err = gsCatalog.GetFeatureTypes("sf_dummy", "sf_dummy")
	assert.Nil(t, featureTypes)
	assert.NotNil(t, err)
}
func TestGetFeatrueType(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	featureType, err := gsCatalog.GetFeatureType("sf", "sf", "bugsites")
	assert.NotNil(t, featureType)
	assert.NotEmpty(t, featureType)
	assert.Nil(t, err)
	featureType, err = gsCatalog.GetFeatureType("sf_dummy", "sf_dummy", "bugsites")
	assert.Nil(t, featureType)
	assert.NotNil(t, err)
}
func TestDeleteFeatureType(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	zippedShapefile := filepath.Join(gsCatalog.getGoGeoserverPackageDir(), "test_sample", "museum_nyc.zip")
	uploaded, err := gsCatalog.UploadShapeFile(zippedShapefile, "featureTypeWorkspace", "")
	assert.True(t, uploaded)
	assert.Nil(t, err)
	deleted, err := gsCatalog.DeleteFeatureType("featureTypeWorkspace", "", "museum_nyc", true)
	assert.True(t, deleted)
	assert.Nil(t, err)
	deleted, err = gsCatalog.DeleteFeatureType("sf_dummy", "s_dummyf", "archsites", true)
	assert.False(t, deleted)
	assert.NotNil(t, err)
}

func TestGeoserverImplemetFeatureTypeService(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	FeatureTypeServiceType := reflect.TypeOf((*FeatureTypeService)(nil)).Elem()
	check := gsCatalog.Implements(FeatureTypeServiceType)
	assert.True(t, check)
}
