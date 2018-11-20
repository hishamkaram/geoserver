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

func TestCreateLayerGroup(t *testing.T) {
	gsCatalog := GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
	workspace := Resource{Name: ""}
	var proj interface{} = "EPSG:4326"
	layergroup := LayerGroup{Name: "golang",
		Title:     "Go",
		Mode:      "SINGLE",
		Workspace: &workspace,
		Publishables: Publishables{Published: []*GroupPublishableItem{
			{Type: "layer", Name: "tiger:poly_landmarks", Href: "http://localhost:8080/geoserver/rest/workspaces/tiger/layers/poly_landmarks.json"},
		}}, Styles: LayerGroupStyles{Style: []*Resource{
			{Name: "poly_landmarks", Href: "http://localhost:8080/geoserver/rest/styles/poly_landmarks.json"},
		}}, Bounds: NativeBoundingBox{
			BoundingBox: BoundingBox{
				Minx: -74.047185,
				Maxx: -73.90782,
				Miny: 40.679648,
				Maxy: 40.882078},
			Crs: &proj,
		}}
	created, createErr := gsCatalog.CreateLayerGroup("", &layergroup)
	assert.True(t, created)
	assert.Nil(t, createErr)
	gsCatalog = GetCatalog("http://localhost:8080/geoserver_dummy/", "admin", "geoserver")
	createdFail, createErrFail := gsCatalog.CreateLayerGroup("", &layergroup)
	assert.False(t, createdFail)
	assert.NotNil(t, createErrFail)
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
