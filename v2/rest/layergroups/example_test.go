package layergroups_test

import (
	"context"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/layergroups"
)

// ExampleClient_InWorkspace returns a workspace-scoped layer-groups
// client.
func ExampleClient_InWorkspace() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.LayerGroups.InWorkspace("topp")
}

// ExampleWorkspaceClient_Iter ranges over every layer group in a
// workspace.
func ExampleWorkspaceClient_Iter() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	for g, err := range c.LayerGroups.InWorkspace("topp").Iter(context.Background(), layergroups.ListOptions{}) {
		if err != nil {
			return
		}
		fmt.Printf("%s — %d members\n", g.Name, len(g.Publishables.Published))
	}
}

// ExampleWorkspaceClient_Create bundles two layers into a SINGLE-mode
// layer group. The mixed string/object Styles wire form is handled
// automatically — pass an empty [layergroups.Styles] to default each
// member's style.
func ExampleWorkspaceClient_Create() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.LayerGroups.InWorkspace("topp").Create(context.Background(), &layergroups.LayerGroup{
		Name: "boundaries",
		Mode: "SINGLE",
		Publishables: layergroups.Publishables{Published: layergroups.Published{
			{Type: "layer", Name: "topp:states"},
			{Type: "layer", Name: "topp:counties"},
		}},
	})
}
