package geoserver

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GeoserverStyleSuite struct {
	suite.Suite
	gsCatalog *GeoServer
}

func (suite *GeoserverStyleSuite) SetupSuite() {
	suite.gsCatalog = GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	created, err := suite.gsCatalog.CreateWorkspace("styles_test")
	assert.True(suite.T(), created)
	assert.Nil(suite.T(), err)
}

func (suite *GeoserverStyleSuite) TestStyles() {
	sldPath, _ := filepath.Abs("test_sample/airports.sld")
	sld, _ := ioutil.ReadFile(sldPath)
	created, uploadErr := suite.gsCatalog.CreateStyle("styles_test", "test_test")
	assert.True(suite.T(), created)
	assert.Nil(suite.T(), uploadErr)
	uploaded, err := suite.gsCatalog.UploadStyle(bytes.NewBuffer(sld), "styles_test", "test_test")
	assert.True(suite.T(), uploaded)
	assert.Nil(suite.T(), err)
	styles, getErr := suite.gsCatalog.GetStyles("styles_test")
	assert.NotEmpty(suite.T(), styles)
	assert.Nil(suite.T(), getErr)
	style, styleErr := suite.gsCatalog.GetStyle("styles_test", "test_test")
	assert.NotEmpty(suite.T(), style)
	assert.Nil(suite.T(), styleErr)
	deleted, deleteErr := suite.gsCatalog.DeleteStyle("styles_test", "test_test", true)
	assert.True(suite.T(), deleted)
	assert.Nil(suite.T(), deleteErr)
}

func (suite *GeoserverStyleSuite) TearDownSuite() {
	suite.gsCatalog = GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	deleted, err := suite.gsCatalog.DeleteWorkspace("styles_test", true)
	assert.True(suite.T(), deleted)
	assert.Nil(suite.T(), err)
}
func TestGeoserverStyleSuite(t *testing.T) {
	suite.Run(t, new(GeoserverStyleSuite))
}
func TestGeoserverImplemetStyleService(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	StyleServiceType := reflect.TypeOf((*StyleService)(nil)).Elem()
	check := gsCatalog.Implements(StyleServiceType)
	assert.True(t, check)
}
