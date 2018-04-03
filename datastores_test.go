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
func (suite *GeoserverDatastoreSuite) TestCreateDatastore() {
	created, err := suite.gsCatalog.CreateDatastore(DatastoreConnection{
		Name:   "datastores_test",
		Port:   5432,
		Type:   "postgis",
		DBName: "cartoview_datastore",
		DBPass: "xxxx",
		DBUser: "postgres",
	}, "datastores_test")
	assert.True(suite.T(), created)
	assert.Nil(suite.T(), err)
}

func (suite *GeoserverDatastoreSuite) TestDatastoreExists() {
	exists, err := suite.gsCatalog.DatastoreExists("datastores_test", "datastores_test", true)
	assert.True(suite.T(), exists)
	assert.Nil(suite.T(), err)
}

func (suite *GeoserverDatastoreSuite) TestDeleteDatastore() {
	deleted, err := suite.gsCatalog.DeleteDatastore("datastores_test", "datastores_test", true)
	assert.True(suite.T(), deleted)
	assert.Nil(suite.T(), err)
}
func (suite *GeoserverDatastoreSuite) TestGetDatastores() {
	datastores, err := suite.gsCatalog.GetDatastores("topp")
	assert.NotEmpty(suite.T(), datastores)
	assert.NotNil(suite.T(), datastores)
	assert.Nil(suite.T(), err)
}
func (suite *GeoserverDatastoreSuite) TestGetDatastoreDetails() {
	datastore, err := suite.gsCatalog.GetDatastoreDetails("sf", "sf")
	assert.NotEmpty(suite.T(), datastore)
	assert.NotNil(suite.T(), datastore)
	assert.Nil(suite.T(), err)
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
