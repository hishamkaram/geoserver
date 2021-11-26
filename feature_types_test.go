package geoserver

import (
	"encoding/json"
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
	test_before(t)

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
	test_before(t)
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
	test_before(t)

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

func TestCreateFeatureType(t *testing.T) {
	test_before(t)

	_, err := gsCatalog.CreateWorkspace(testWorkspace)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		assert.Fail(t, "can't create workspace as a precondition for CreateFeatureTypes test")
	}
	defer func() {
		_, _ = gsCatalog.DeleteWorkspace(testWorkspace, true)
	}()

	featureType := FeatureType{}

	err = json.Unmarshal(featureTypeCreatingSample(), &featureType)
	assert.Nil(t, err)

	p := testConfig.Postgres

	conn := DatastoreConnection{
		Name:    testDatastore,
		Port:    p.Port,
		Host:    p.Host,
		Type:    p.Type,
		DBName:  p.DBName,
		DBPass:  p.DBPass,
		DBUser:  p.DBUser,
		Options: p.Options,
	}

	defer func() {
		_, _ = gsCatalog.DeleteDatastore(testWorkspace, testDatastore, true)
	}()

	created, err := gsCatalog.CreateDatastore(conn, testWorkspace)
	if !created || err != nil {
		assert.Fail(t, "Can't create datastore as precondition to CreateFeatureType")
		return
	}

	created, err = gsCatalog.CreateFeatureType(testWorkspace, testDatastore, &featureType)
	assert.True(t, created)
	assert.Nil(t, err)

	created, err = gsCatalog.CreateFeatureType(testWorkspace, testDatastore, &featureType)
	assert.True(t, created)
	assert.Nil(t, err)

}

func featureTypeCreatingSample() []byte {
	return []byte(`
	{
		"name": "polygon",
		"title": "polygon",
		"srs": "EPSG:4326",
		"nativeBoundingBox": {
		  "minx": -180,
		  "maxx": 180,
		  "miny": -90,
		  "maxy": 90,
		  "crs": "EPSG:4326"
		},
		"latLonBoundingBox": {
		  "minx": -180,
		  "maxx": 180,
		  "miny": -90,
		  "maxy": 90,
		  "crs": "EPSG:4326"
		},
		"projectionPolicy": "FORCE_DECLARED",
		"serviceConfiguration": false,
		"simpleConversionEnabled": false,
		"padWithZeros": false,
		"forcedDecimal": false,
		"overridingServiceSRS": false,
		"skipNumberMatched": false,
		"circularArcPresent": false,
		"attributes": {
		  "attribute": [
			{
			  "name": "group",
			  "maxOccurs": 1,
			  "nillable": true,
			  "binding": "java.lang.Integer"
			},
			{
			  "name": "name",
			  "maxOccurs": 1,
              "length": 100,
			  "nillable": true,
			  "binding": "java.lang.String"
			},
			{
			  "name": "descr",
			  "maxOccurs": 1,
			  "length": 256,
			  "nillable": true,
			  "binding": "java.lang.String"
			},
			{
			  "name": "style",
			  "maxOccurs": 1,
			  "length": 256,
			  "nillable": true,
			  "binding": "java.lang.String"
			},
			{
			  "name": "options",
			  "maxOccurs": 1,
			  "nillable": true,
			  "binding": "java.lang.String"
			},
			{
			  "name": "geom",
			  "maxOccurs": 1,
			  "nillable": true,
			  "binding": "org.locationtech.jts.geom.MultiPolygon"
			},
			{
			  "name": "created",
			  "maxOccurs": 1,
			  "nillable": true,
			  "binding": "java.sql.Timestamp"
			}
		  ]
		}
  	}`)
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
