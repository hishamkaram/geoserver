package geoserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GeoserverDatastoreSuite struct {
	suite.Suite
	gsCatalog *GeoServer
}

func (suite *GeoserverDatastoreSuite) SetupSuite() {
	suite.gsCatalog = GetCatalog("http://geoserver:8080/geoserver/", "admin", "geoserver")
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

func (suite *GeoserverDatastoreSuite) TearDownSuite() {
	deleted, err := suite.gsCatalog.DeleteWorkspace("datastores_test", true)
	assert.True(suite.T(), deleted)
	assert.Nil(suite.T(), err)
}
func TestGeoserverDatastoreSuite(t *testing.T) {
	suite.Run(t, new(GeoserverDatastoreSuite))
}
