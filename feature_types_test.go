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
	featureType, err = gsCatalog.GetFeatureType("tiger", "nyc", "poi")
	assert.NotNil(t, featureType)
	assert.NotEmpty(t, featureType)
	assert.Nil(t, err)
	featureType, err = gsCatalog.GetFeatureType("sf_dummy", "sf_dummy", "bugsites")
	assert.Nil(t, featureType)
	assert.NotNil(t, err)
}
func TestDeleteFeatureType(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	zippedShapefile := filepath.Join(gsCatalog.getGoGeoserverPackageDir(), "testdata", "museum_nyc.zip")
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
func TestCRSTypeMarshalJSON(t *testing.T) {
	proj := []byte(`PROJCS["NAD27 / UTM zone 13N",
	GEOGCS["NAD27",
	  DATUM["North American Datum 1927",
		SPHEROID["Clarke 1866", 6378206.4, 294.9786982138982, AUTHORITY["EPSG","7008"]],
		TOWGS84[-4.2, 135.4, 181.9, 0.0, 0.0, 0.0, 0.0],
		AUTHORITY["EPSG","6267"]],
	  PRIMEM["Greenwich", 0.0, AUTHORITY["EPSG","8901"]],
	  UNIT["degree", 0.017453292519943295],
	  AXIS["Geodetic longitude", EAST],
	  AXIS["Geodetic latitude", NORTH],
	  AUTHORITY["EPSG","4267"]],
	PROJECTION["Transverse_Mercator"],
	PARAMETER["central_meridian", -105.0],
	PARAMETER["latitude_of_origin", 0.0],
	PARAMETER["scale_factor", 0.9996],
	PARAMETER["false_easting", 500000.0],
	PARAMETER["false_northing", 0.0],
	UNIT["m", 1.0],
	AXIS["Easting", EAST],
	AXIS["Northing", NORTH],
	AUTHORITY["EPSG","26713"]]}`)
	projected := CRSType{Class: "projected", Value: string(proj)}
	projectedData, err := projected.MarshalJSON()
	assert.Nil(t, err)
	assert.NotNil(t, projectedData)
	strSrs := CRSType{Class: "string", Value: "EPSG:4326"}
	strSrsData, strSrserr := strSrs.MarshalJSON()
	assert.Nil(t, strSrserr)
	assert.NotNil(t, strSrsData)
	var emptySrs CRSType
	emptySrsData, emptySrsErr := emptySrs.MarshalJSON()
	assert.Equal(t, "{}", string(emptySrsData))
	assert.Nil(t, emptySrsErr)

}

func TestGeoserverImplemetFeatureTypeService(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	FeatureTypeServiceType := reflect.TypeOf((*FeatureTypeService)(nil)).Elem()
	check := gsCatalog.Implements(FeatureTypeServiceType)
	assert.True(t, check)
}
