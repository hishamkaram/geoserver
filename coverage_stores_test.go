package geoserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCoverageStores(t *testing.T) {
	gsCatalog := GetCatalog("http://geoserver:8080/geoserver/", "admin", "geoserver")
	coverageStores, err := gsCatalog.GetCoverageStores("nurc")
	assert.NotNil(t, coverageStores)
	assert.Nil(t, err)
}

func TestHDeleteCoverageStore(t *testing.T) {
	gsCatalog := GetCatalog("http://geoserver:8080/geoserver/", "admin", "geoserver")
	deleted, err := gsCatalog.DeleteCoverageStore("nurc", "worldImageSample", true)
	assert.True(t, deleted)
	assert.Nil(t, err)
}
