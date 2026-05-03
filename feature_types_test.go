//go:build integration
// +build integration

package geoserver

import (
	"path/filepath"
	"reflect"
	"strings"
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
	zippedShapefile := filepath.Join(mustPkgDir(t, gsCatalog), "testdata", "museum_nyc.zip")
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

// TestGetFeatureTypeList_Available verifies that a freshly-created PostGIS
// datastore reports its tables under the `available` listing (tables present
// in the DB but not yet configured as feature types in GeoServer).
func TestGetFeatureTypeList_Available(t *testing.T) {
	before()
	const ws = "ftlist_ws"
	const ds = "ftlist_pg"

	if _, err := gsCatalog.CreateWorkspace(ws); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("preconditions: create workspace: %v", err)
	}
	t.Cleanup(func() {
		_, _ = gsCatalog.DeleteWorkspace(ws, true)
	})

	conn := DatastoreConnection{
		Name:   ds,
		Host:   "postgis",
		Port:   5432,
		Type:   "postgis",
		DBName: "gis",
		DBUser: "golang",
		DBPass: "golang",
	}
	created, err := gsCatalog.CreateDatastore(conn, ws)
	if !created || err != nil {
		t.Fatalf("preconditions: create datastore: created=%v err=%v", created, err)
	}

	available, err := gsCatalog.GetFeatureTypeList(ws, ds, FeatureTypeListAvailable)
	assert.NoError(t, err)
	// The seeded `public.lbldyt` table should be in the available list.
	found := false
	for _, n := range available {
		if n == "lbldyt" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected seeded `lbldyt` in available list, got %v", available)

	// configured should be empty (nothing has been published yet).
	configured, err := gsCatalog.GetFeatureTypeList(ws, ds, FeatureTypeListConfigured)
	assert.NoError(t, err)
	assert.Empty(t, configured)
}

// TestCreateFeatureType creates a workspace + PostGIS datastore (against the
// compose-managed PostGIS host `postgis:5432`, DB `gis`) then registers a
// feature type pointing at the seeded `public.lbldyt` table.
func TestCreateFeatureType(t *testing.T) {
	before()
	const ws = "ft_create_ws"
	const ds = "ft_create_pg"
	const ftName = "lbldyt"

	if _, err := gsCatalog.CreateWorkspace(ws); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("preconditions: create workspace: %v", err)
	}
	t.Cleanup(func() {
		_, _ = gsCatalog.DeleteWorkspace(ws, true)
	})

	conn := DatastoreConnection{
		Name:   ds,
		Host:   "postgis",
		Port:   5432,
		Type:   "postgis",
		DBName: "gis",
		DBUser: "golang",
		DBPass: "golang",
	}
	created, err := gsCatalog.CreateDatastore(conn, ws)
	if !created || err != nil {
		t.Fatalf("preconditions: create postgis datastore: created=%v err=%v", created, err)
	}

	ft := &FeatureType{
		Name:       ftName,
		NativeName: ftName,
		Title:      "lbldyt sample",
		Srs:        "EPSG:4326",
	}
	created, err = gsCatalog.CreateFeatureType(ws, ds, ft)
	assert.NoError(t, err)
	assert.True(t, created)

	got, err := gsCatalog.GetFeatureType(ws, ds, ftName)
	assert.NoError(t, err)
	assert.NotNil(t, got)
}
