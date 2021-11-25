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
	gsCatalog     *GeoServer
	conn          DatastoreConnection
	workspaceName string
	datastoreName string
}

func (suite *GeoserverDatastoreSuite) SetupSuite() {

	test_before(suite.T())

	p := testConfig.Postgres

	suite.workspaceName = testConfig.Geoserver.Workspace
	suite.datastoreName = p.Name

	suite.conn = DatastoreConnection{
		Name:   suite.datastoreName,
		Port:   p.Port,
		Host:   p.Host,
		Type:   p.Type,
		DBName: p.DBName,
		DBPass: p.DBPass,
		DBUser: p.DBUser,
	}

	suite.gsCatalog = GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	created, err := suite.gsCatalog.CreateWorkspace(suite.workspaceName)
	if err != nil && strings.Contains(err.Error(), "already exists") {
		_, err = suite.gsCatalog.DeleteWorkspace(suite.workspaceName, true)
		if err != nil {
			assert.Fail(suite.T(), "can't setup suite")
		}
		created, err = suite.gsCatalog.CreateWorkspace(suite.workspaceName)
	}
	assert.True(suite.T(), created)
	assert.Nil(suite.T(), err)
}

func (suite *GeoserverDatastoreSuite) TearDownSuite() {
	deleted, err := suite.gsCatalog.DeleteWorkspace(suite.workspaceName, true)
	assert.True(suite.T(), deleted)
	assert.Nil(suite.T(), err)
}

func (suite *GeoserverDatastoreSuite) CreateDatastore() (bool, error) {
	return suite.gsCatalog.CreateDatastore(suite.conn, suite.workspaceName)
}

func (suite *GeoserverDatastoreSuite) Test01GetDatastoreObj() {
	assert.NotNil(suite.T(), suite.conn.GetDatastoreObj())
}

func (suite *GeoserverDatastoreSuite) Test02CreateDatastore() {
	created, err := suite.CreateDatastore()
	assert.True(suite.T(), created)
	assert.Nil(suite.T(), err)
	created, err = suite.CreateDatastore()
	assert.False(suite.T(), created)
	assert.NotNil(suite.T(), err)
}

func (suite *GeoserverDatastoreSuite) Test03DatastoreExists() {
	exists, err := suite.gsCatalog.DatastoreExists(suite.workspaceName, suite.datastoreName, true)
	assert.True(suite.T(), exists)
	assert.Nil(suite.T(), err)
	exists, err = suite.gsCatalog.DatastoreExists(suite.workspaceName+"_dummy", suite.datastoreName+"_dummy", true)
	assert.False(suite.T(), exists)
	assert.NotNil(suite.T(), err)
}

func (suite *GeoserverDatastoreSuite) Test04DeleteDatastore() {
	deleted, err := suite.gsCatalog.DeleteDatastore(suite.workspaceName, suite.datastoreName, true)
	assert.True(suite.T(), deleted)
	assert.Nil(suite.T(), err)
	deleted, err = suite.gsCatalog.DeleteDatastore(suite.workspaceName+"_dummy", suite.datastoreName+"_dummy", true)
	assert.False(suite.T(), deleted)
	assert.NotNil(suite.T(), err)
}

func (suite *GeoserverDatastoreSuite) Test05GetDatastores() {
	_, _ = suite.CreateDatastore()
	datastores, err := suite.gsCatalog.GetDatastores(suite.workspaceName)
	assert.NotEmpty(suite.T(), datastores)
	assert.NotNil(suite.T(), datastores)
	assert.Nil(suite.T(), err)
	datastores, err = suite.gsCatalog.GetDatastores(suite.workspaceName + "_wrong")
	assert.Nil(suite.T(), datastores)
	assert.NotNil(suite.T(), err)
}

func (suite *GeoserverDatastoreSuite) Test06GetDatastoreDetails() {
	datastore, err := suite.gsCatalog.GetDatastoreDetails(suite.workspaceName, suite.datastoreName)
	assert.NotEmpty(suite.T(), datastore)
	assert.NotNil(suite.T(), datastore)
	assert.Nil(suite.T(), err)
	datastore, err = suite.gsCatalog.GetDatastoreDetails(suite.workspaceName, suite.datastoreName+"_dummy")
	assert.Equal(suite.T(), datastore, &Datastore{})
	assert.NotNil(suite.T(), err)
}

func TestGeoserverDatastoreSuite(t *testing.T) {
	suite.Run(t, new(GeoserverDatastoreSuite))
}

func TestGeoserverImplementDatastoreService(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	DatastoreServiceType := reflect.TypeOf((*DatastoreService)(nil)).Elem()
	check := gsCatalog.Implements(DatastoreServiceType)
	assert.True(t, check)
}
