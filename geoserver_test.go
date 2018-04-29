package geoserver

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	var gsCatalog GeoServer
	file, _ := filepath.Abs("../geoserver/test_sample/config.yml")
	geoserver, err := gsCatalog.LoadConfig(file)
	assert.NotNil(t, geoserver)
	assert.Nil(t, err)
	//test 	if can't find yaml
	file, _ = filepath.Abs("")
	geoserver, err = gsCatalog.LoadConfig(file)
	assert.Nil(t, geoserver)
	assert.NotNil(t, err)
	file, _ = filepath.Abs("../geoserver/test_sample/config.err.yml")
	geoserver, err = gsCatalog.LoadConfig(file)
	assert.Nil(t, geoserver)
	assert.NotNil(t, err)
}
func TestSetLogger(t *testing.T) {
	var gsCatalog GeoServer
	gsCatalog.SetLogger()
	assert.NotNil(t, gsCatalog.logger)
}
func TestGetGeoserverRequest(t *testing.T) {
	gsCatalog := GetCatalog("", "", "")
	request := gsCatalog.GetGeoserverRequest("", getMethod, jsonType, bytes.NewBuffer(make([]byte, 0, 0)), jsonType)
	assert.NotNil(t, request)
}
