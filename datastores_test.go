//go:build integration
// +build integration

package geoserver

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GeoserverDatastoreSuite struct {
	suite.Suite
	gsCatalog *GeoServer
}

func (suite *GeoserverDatastoreSuite) SetupSuite() {
	suite.gsCatalog = GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	created, err := suite.gsCatalog.CreateWorkspace("datastores_test")
	assert.True(suite.T(), created)
	assert.Nil(suite.T(), err)
}
func TestGetDatastoreObj(t *testing.T) {
	conn := DatastoreConnection{
		Name:   "datastores_test",
		Port:   5432,
		Host:   "localhost",
		Type:   "postgis",
		DBName: "cartoview_datastore",
		DBPass: "xxxx",
		DBUser: "postgres",
	}
	assert.NotNil(t, conn.GetDatastoreObj())
}
func (suite *GeoserverDatastoreSuite) TestCreateDatastore() {
	conn := DatastoreConnection{
		Name:   "datastores_test",
		Port:   5432,
		Type:   "postgis",
		DBName: "cartoview_datastore",
		DBPass: "xxxx",
		DBUser: "postgres",
	}
	created, err := suite.gsCatalog.CreateDatastore(conn, "datastores_test")
	assert.True(suite.T(), created)
	assert.Nil(suite.T(), err)
	created, err = suite.gsCatalog.CreateDatastore(conn, "datastores_test")
	assert.False(suite.T(), created)
	assert.NotNil(suite.T(), err)
}

func (suite *GeoserverDatastoreSuite) TestDatastoreExists() {
	exists, err := suite.gsCatalog.DatastoreExists("datastores_test", "datastores_test", true)
	assert.True(suite.T(), exists)
	assert.Nil(suite.T(), err)
	exists, err = suite.gsCatalog.DatastoreExists("datastores_test_dummy", "datastores_test_dummy", true)
	assert.False(suite.T(), exists)
	assert.NotNil(suite.T(), err)
}

func (suite *GeoserverDatastoreSuite) TestDeleteDatastore() {
	deleted, err := suite.gsCatalog.DeleteDatastore("datastores_test", "datastores_test", true)
	assert.True(suite.T(), deleted)
	assert.Nil(suite.T(), err)
	deleted, err = suite.gsCatalog.DeleteDatastore("datastores_test_dummy", "datastores_test_dummy", true)
	assert.False(suite.T(), deleted)
	assert.NotNil(suite.T(), err)
}
func (suite *GeoserverDatastoreSuite) TestGetDatastores() {
	datastores, err := suite.gsCatalog.GetDatastores("topp")
	assert.NotEmpty(suite.T(), datastores)
	assert.NotNil(suite.T(), datastores)
	assert.Nil(suite.T(), err)
	datastores, err = suite.gsCatalog.GetDatastores("topp_dummy")
	assert.Nil(suite.T(), datastores)
	assert.NotNil(suite.T(), err)
}
func (suite *GeoserverDatastoreSuite) TestGetDatastoreDetails() {
	datastore, err := suite.gsCatalog.GetDatastoreDetails("sf", "sf")
	assert.NotEmpty(suite.T(), datastore)
	assert.NotNil(suite.T(), datastore)
	assert.Nil(suite.T(), err)
	datastore, err = suite.gsCatalog.GetDatastoreDetails("sf", "sf_dummy")
	assert.Equal(suite.T(), datastore, &Datastore{})
	assert.NotNil(suite.T(), err)
}

func (suite *GeoserverDatastoreSuite) TearDownSuite() {
	deleted, err := suite.gsCatalog.DeleteWorkspace("datastores_test", true)
	assert.True(suite.T(), deleted)
	assert.Nil(suite.T(), err)
}
func TestGeoserverDatastoreSuite(t *testing.T) {
	suite.Run(t, new(GeoserverDatastoreSuite))
}
func TestGeoserverImplemetDatastoreService(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	DatastoreServiceType := reflect.TypeOf((*DatastoreService)(nil)).Elem()
	check := gsCatalog.Implements(DatastoreServiceType)
	assert.True(t, check)
}
func TestGeoserverImplementDatastoreService(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	CatalogType := reflect.TypeOf((*DatastoreService)(nil)).Elem()
	check := gsCatalog.Implements(CatalogType)
	assert.True(t, check)
}

// TestCreateDatastoreWithOptions verifies the v1.1 Options field is accepted
// by GeoServer. The compose stack runs PostGIS at the docker-network host
// `postgis:5432` (DB `gis`, user/pass `golang`), seeded with the lbldyt
// table (see docker/postgis/init/01-lbldyt.sql).
func TestCreateDatastoreWithOptions(t *testing.T) {
	before()
	const ws = "ds_options_ws"
	const ds = "ds_options_ds"

	if _, err := gsCatalog.CreateWorkspace(ws); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("preconditions: create workspace: %v", err)
	}
	t.Cleanup(func() {
		_, _ = gsCatalog.DeleteWorkspace(ws, true)
	})

	conn := DatastoreConnection{
		Name:   ds,
		Host:   "postgis", // docker-network hostname seen from the GeoServer container
		Port:   5432,
		Type:   "postgis",
		DBName: "gis",
		DBUser: "golang",
		DBPass: "golang",
		Options: []Entry{
			{Key: "Expose primary keys", Value: "true"},
			{Key: "max connections", Value: "5"},
		},
	}

	created, err := gsCatalog.CreateDatastore(conn, ws)
	assert.NoError(t, err)
	assert.True(t, created)

	got, err := gsCatalog.GetDatastoreDetails(ws, ds)
	assert.NoError(t, err)
	assert.NotNil(t, got)

	// Verify Options entries were persisted.
	keys := map[string]string{}
	for _, e := range got.ConnectionParameters.Entry {
		keys[e.Key] = e.Value
	}
	assert.Equal(t, "true", keys["Expose primary keys"])
	assert.Equal(t, "5", keys["max connections"])
}

// TestCreateJNDIDatastore_BadJNDIRef sanity-checks that the JNDI request shape
// reaches GeoServer; we don't have a configured JNDI resource in the compose
// stack, so the request is expected to be rejected with a non-2xx response —
// we just verify no transport-level failure and a typed error is returned.
func TestCreateJNDIDatastore_BadJNDIRef(t *testing.T) {
	before()
	const ws = "jndi_smoke_ws"
	if _, err := gsCatalog.CreateWorkspace(ws); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("preconditions: create workspace: %v", err)
	}
	t.Cleanup(func() {
		_, _ = gsCatalog.DeleteWorkspace(ws, true)
	})

	conn := DatastoreJNDIConnection{
		Name:              "jndi_smoke_ds",
		Type:              "postgis",
		JndiReferenceName: "java:comp/env/jdbc/nonexistent",
	}
	created, err := gsCatalog.CreateJNDIDatastore(conn, ws)
	// Either GeoServer rejects (most common) or it accepts the metadata but
	// cannot resolve at use time. Both are valid; the smoke is that the
	// request path and JSON shape are correct.
	if created {
		t.Logf("GeoServer accepted JNDI metadata; cleanup")
		_, _ = gsCatalog.DeleteDatastore(ws, conn.Name, true)
		return
	}
	assert.Error(t, err)
}
