package geoserver

import (
	"reflect"
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
