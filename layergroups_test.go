package geoserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLayerGroups(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	layerGroups, err := gsCatalog.GetLayerGroups("")
	assert.NotNil(t, layerGroups)
	assert.True(t, (len(layerGroups) > 0))
	assert.Nil(t, err)
	gsCatalog = GetCatalog("http://localhost:8080/geoserver_dummy/", "admin", "geoserver")
	layersGroupsFail, groupsErr := gsCatalog.GetLayerGroups("nurc_dummy")
	assert.Nil(t, layersGroupsFail)
	assert.True(t, (len(layersGroupsFail) == 0))
	assert.NotNil(t, groupsErr)
}

func TestGetLayerGroup(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	layerGroup, err := gsCatalog.GetLayerGroup("", "tiger-ny")
	assert.NotNil(t, layerGroup)
	assert.Nil(t, err)
	layerGroupFail, layerGroupErr := gsCatalog.GetLayerGroup("", "dummy_layer_group")
	assert.Equal(t, layerGroupFail, &LayerGroup{})
	assert.NotNil(t, layerGroupErr)
}
