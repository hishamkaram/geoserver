//go:build integration
// +build integration

package geoserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetError(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	err := gsCatalog.GetError(0, []byte("Custom Error"))
	assert.NotNil(t, err)
	err = gsCatalog.GetError(403, []byte("Custom Error"))
	assert.NotNil(t, err)
}
func TestSerializeStruct(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	resource := Resource{Class: "Test", Href: "http://localhost:8080/geoserver/", Name: "Test1"}
	json, err := gsCatalog.SerializeStruct(&resource)
	assert.NotEmpty(t, json)
	assert.Nil(t, err)
	resource2 := make(chan int)
	json, err = gsCatalog.SerializeStruct(&resource2)
	assert.Empty(t, json)
	assert.NotNil(t, err)
}
func TestDoRequest(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	// v1.1: DoRequest with an unsupported method now logs and returns
	// (nil, 0) rather than recovering from a panic and returning the
	// panic message bytes.
	responseText, statusCode := gsCatalog.DoRequest(HTTPRequest{Method: "dummy_method",
		Accept: jsonType,
		URL:    "http://localhost:8080/geoserver/"})
	assert.Equal(t, 0, statusCode)
	assert.Nil(t, responseText)
	// Real GET hits the WFS endpoint; we expect a non-zero status code.
	responseText, statusCode = gsCatalog.DoRequest(HTTPRequest{Method: getMethod,
		Accept: jsonType,
		URL:    "http://localhost:8080/geoserver/wfs"})
	assert.NotEqual(t, 0, statusCode)
	assert.NotNil(t, responseText)
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
	json = []byte(`<xml/>`)
	resource = Resource{}
	err = gsCatalog.DeSerializeJSON(json, &resource)
	assert.Empty(t, resource)
	assert.NotNil(t, err)

}
func TestGetGoGeoserverPackageDir(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	geoserverPath, err := gsCatalog.getGoGeoserverPackageDir()
	assert.NoError(t, err)
	assert.NotEmpty(t, geoserverPath)
}
func TestParseURLL(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	targetURL := gsCatalog.ParseURL("rest", "workspaces")
	assert.NotNil(t, targetURL)
	assert.NotEmpty(t, targetURL)
	assert.Equal(t, targetURL, "http://localhost:8080/geoserver/rest/workspaces")
	gsCatalog = GetCatalog("://htto://localhost:8080/geoserver/", "admin", "geoserver")
	targetURL = gsCatalog.ParseURL("rest", "workspaces")
	assert.Empty(t, targetURL)
}
func BenchmarkParseURL(b *testing.B) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	for i := 0; i < b.N; i++ {
		gsCatalog.ParseURL("rest", "workspaces")
	}
}
func BenchmarkIsEmpty(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsEmpty(struct{}{})
	}
}
