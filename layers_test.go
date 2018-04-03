package geoserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetshpFiledsName(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	storename := gsCatalog.GetshpFiledsName("hisham.zip")
	assert.Equal(t, storename, "hisham")
}
