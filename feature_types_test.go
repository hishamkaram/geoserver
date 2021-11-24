package geoserver

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testZippedShapeFile = filepath.Join(gsCatalog.getGoGeoserverPackageDir(), "testdata", "museum_nyc.zip")
	testWorkspace       = "someNonExistentWorkspace"
	testDatastore       = "someNonExistentDatastore"
)

func featureTypePrecondition(t *testing.T) {
	_, err := gsCatalog.CreateWorkspace(testWorkspace)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		assert.Fail(t, "can't create workspace as a precondition for FeatureTypes test")
	}
	uploaded, err := gsCatalog.UploadShapeFile(testZippedShapeFile, testWorkspace, testDatastore)
	if !uploaded || err != nil {
		assert.Fail(t, "can't upload shapefile as a precondition for FeatureTypes test")
	}
}

func featureTypePostcondition() {
	_, _ = gsCatalog.DeleteWorkspace(testWorkspace, true)
}

func TestGetFeatureTypes(t *testing.T) {
	before()

	//precondition
	featureTypePrecondition(t)
	defer func() {
		featureTypePostcondition()
	}()

	featureTypes, err := gsCatalog.GetFeatureTypes(testWorkspace, testDatastore)
	assert.NotNil(t, featureTypes)
	assert.NotEmpty(t, featureTypes)
	assert.Nil(t, err)
	featureTypes, err = gsCatalog.GetFeatureTypes("sf_dummy", "sf_dummy")
	assert.Nil(t, featureTypes)
	assert.NotNil(t, err)
}

func TestGetFeatureType(t *testing.T) {
	before()
	featureTypePrecondition(t)
	defer func() {
		featureTypePostcondition()
	}()
	featureType, err := gsCatalog.GetFeatureType(testWorkspace, testDatastore, "museum_nyc")
	assert.NotNil(t, featureType)
	assert.NotEmpty(t, featureType)
	assert.Nil(t, err)
	featureType, err = gsCatalog.GetFeatureType(testWorkspace, testDatastore, "wrongFeatureType")
	assert.Nil(t, featureType)
	assert.NotNil(t, err)
}

func TestDeleteFeatureType(t *testing.T) {
	before()

	featureTypePrecondition(t)
	defer func() {
		featureTypePostcondition()
	}()

	deleted, err := gsCatalog.DeleteFeatureType(testWorkspace, testDatastore, "museum_nyc", true)
	assert.True(t, deleted)
	assert.Nil(t, err)
	deleted, err = gsCatalog.DeleteFeatureType(testWorkspace, testDatastore, "wrongFeatureType", true)
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

func TestGeoserverImplementFeatureTypeService(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	FeatureTypeServiceType := reflect.TypeOf((*FeatureTypeService)(nil)).Elem()
	check := gsCatalog.Implements(FeatureTypeServiceType)
	assert.True(t, check)
}
