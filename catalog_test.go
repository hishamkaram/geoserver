package geoserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCatalog(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	assert.NotNil(t, gsCatalog)
}
