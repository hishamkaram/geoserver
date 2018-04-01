package geoserver

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	var gsCatalog GeoServer
	file, _ := filepath.Abs("test_sample/config.yml")
	gsCatalog.LoadConfig(file)
	assert.NotNil(t, gsCatalog)
}
func TestSetLogger(t *testing.T) {
	var gsCatalog GeoServer
	gsCatalog.SetLogger()
	assert.NotNil(t, gsCatalog.logger)
}
func TestGetGeoserverRequest(t *testing.T) {
	gsCatalog := GetCatalog("", "", "")
	request, err := gsCatalog.GetGeoserverRequest("", getMethod, jsonType, bytes.NewBuffer(make([]byte, 0, 0)), jsonType)
	assert.Nil(t, err)
	assert.NotNil(t, request)
}
