package geoserver

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRunning(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	isRunning, err := gsCatalog.IsRunning()
	assert.True(t, isRunning)
	assert.Nil(t, err)
	gsCatalog = GetCatalog("http://localhost:8080/geoserver_dummy/", "admin", "geoserver")
	isRunning, err = gsCatalog.IsRunning()
	assert.False(t, isRunning)
	assert.NotNil(t, err)
}
func TestGeoserverImplemetAbout(t *testing.T) {
	gsCatalog := reflect.TypeOf(&GeoServer{})
	AboutServiceType := reflect.TypeOf((*AboutService)(nil)).Elem()
	check := gsCatalog.Implements(AboutServiceType)
	assert.True(t, check)
}
