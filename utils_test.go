package geoserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerializeStruct(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	resource := Resource{Class: "Test", Href: "http://localhost:8080/geoserver/", Name: "Test1"}
	json, err := gsCatalog.SerializeStruct(&resource)
	assert.NotEmpty(t, json)
	assert.Nil(t, err)
}
func TestIsEmpty(t *testing.T) {
	emptyStruct := GeoServer{}
	falseVar := false
	emtyString := ""
	assert.True(t, IsEmpty(emptyStruct))
	assert.True(t, IsEmpty(nil))
	assert.True(t, IsEmpty(falseVar))
	assert.True(t, IsEmpty(emtyString))
}

func TestDeSerializeJSON(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	json := []byte(`{"@class":"Test","name":"Test1","href":"http://localhost:8080/geoserver/"}`)
	resource := Resource{}
	err := gsCatalog.DeSerializeJSON(json, &resource)
	assert.NotNil(t, resource)
	assert.NotEmpty(t, resource)
	assert.Nil(t, err)
}
func TestGetGoGeoserverPackageDir(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	geoserverPath := gsCatalog.getGoGeoserverPackageDir()
	assert.NotNil(t, geoserverPath)
	assert.NotEmpty(t, geoserverPath)
}
func TestParseURLL(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	targetURL := gsCatalog.ParseURL("rest", "workspaces")
	assert.NotNil(t, targetURL)
	assert.NotEmpty(t, targetURL)
	assert.Equal(t, targetURL, "http://localhost:8080/geoserver/rest/workspaces")
}
