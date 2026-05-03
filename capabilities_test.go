//go:build integration
// +build integration

package geoserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCapabilities(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	caps, err := gsCatalog.GetCapabilities("")
	assert.Nil(t, err)
	assert.NotNil(t, caps)
	gsCatalog = GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	nilCap, capErr := gsCatalog.GetCapabilities("YouAreLost")
	assert.NotNil(t, capErr)
	assert.Nil(t, nilCap)
}
