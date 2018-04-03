package geoserver

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCatalog(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	assert.NotNil(t, gsCatalog)
}
func TestCatalogImplemet(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	CatalogType := reflect.TypeOf((*Catalog)(nil)).Elem()
	check := gsCatalog.Implements(CatalogType)
	assert.True(t, check)
}
