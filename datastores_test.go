package geoserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GeoserverCatalogSuite struct {
	suite.Suite
	gsCatalog *GeoServer
}

func (suite *GeoserverCatalogSuite) SetupSuite() {
	suite.gsCatalog = GetCatalog("http://localhost:8080/geoserver13/", "admin", "geoserver")
	created, err := suite.gsCatalog.CreateWorkspace("datastores_test")
	assert.True(suite.T(), created)
	assert.Nil(suite.T(), err)
}
func (suite *GeoserverCatalogSuite) TestCreateDatastore() {
	created, err := suite.gsCatalog.CreateDatastore(DatastoreConnection{
		Name:   "datastores_test",
		Port:   5432,
		Type:   "postgis",
		DBName: "cartoview_datastore",
		DBPass: "clogic",
		DBUser: "hishamkaram",
	}, "datastores_test")
	assert.True(suite.T(), created)
	assert.Nil(suite.T(), err)
}

func (suite *GeoserverCatalogSuite) TestDatastoreExists() {
	exists, err := suite.gsCatalog.DatastoreExists("datastores_test", "datastores_test", true)
	assert.True(suite.T(), exists)
	assert.Nil(suite.T(), err)
}

func (suite *GeoserverCatalogSuite) TestDeleteDatastore() {
	deleted, err := suite.gsCatalog.DeleteDatastore("datastores_test", "datastores_test", true)
	assert.True(suite.T(), deleted)
	assert.Nil(suite.T(), err)
}

func (suite *GeoserverCatalogSuite) TearDownSuite() {
	suite.gsCatalog = GetCatalog("http://localhost:8080/geoserver13/", "admin", "geoserver")
	deleted, err := suite.gsCatalog.DeleteWorkspace("datastores_test", true)
	assert.True(suite.T(), deleted)
	assert.Nil(suite.T(), err)
}
func TestGeoserverCatalogSuite(t *testing.T) {
	suite.Run(t, new(GeoserverCatalogSuite))
}
