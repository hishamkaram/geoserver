package geoserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRunning(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	isRunning, err := gsCatalog.IsRunning()
	assert.True(t, isRunning)
	assert.Nil(t, err)
}
