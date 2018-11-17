package geoserver

import (
	"encoding/json"
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
	workspaceLayerGroup, workspaceErr := gsCatalog.GetLayerGroup("tiger", "tiger-ny")
	assert.Equal(t, workspaceLayerGroup, &LayerGroup{})
	assert.NotNil(t, workspaceErr)
	layerGroupFail, layerGroupErr := gsCatalog.GetLayerGroup("", "dummy_layer_group")
	assert.Equal(t, layerGroupFail, &LayerGroup{})
	assert.NotNil(t, layerGroupErr)
}
func TestUnmarshalJSON(t *testing.T) {
	data := []byte(`<layerGroups>
	<layerGroup>
	<name>test</name>
	<atom:link xmlns:atom="http://www.w3.org/2005/Atom" rel="alternate" href="http://localhost/geoserver/rest/workspaces/geonode/layergroups/test.xml" type="application/atom+xml"/>
	</layerGroup>
	<layerGroup>
	<name>test22</name>
	<atom:link xmlns:atom="http://www.w3.org/2005/Atom" rel="alternate" href="http://localhost/geoserver/rest/workspaces/geonode/layergroups/test22.xml" type="application/atom+xml"/>
	</layerGroup>
	</layerGroups>`)
	var publishedGroupLayers PublishedGroupLayers
	err := json.Unmarshal(data, &publishedGroupLayers)
	assert.NotNil(t, err)
	singleOneLayerData := []byte(`{
        "@type": "layer",
        "name": "nyc_fatality_neighbourhood_2a3e3916",
        "href": "http://localhost/geoserver/rest/layers/nyc_fatality_neighbourhood_2a3e3916.json"
      }`)
	var singleObj PublishedGroupLayers
	singleErr := json.Unmarshal(singleOneLayerData, &singleObj)
	assert.Nil(t, singleErr)
}
