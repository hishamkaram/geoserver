package layers_test

import (
	"context"
	"errors"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/layers"
)

// ExampleClient_InWorkspace returns a workspace-scoped layers client.
// Layer operations are always workspace-scoped — the root client is
// just an entry point.
func ExampleClient_InWorkspace() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	ws := c.Layers.InWorkspace("topp")
	_ = ws // ws.Get / ws.List / ws.Update / ws.Delete (no Create)
}

// ExampleWorkspaceClient_Iter ranges over every layer in a workspace.
// There is no Create method — layers are created as a side-effect of
// publishing a feature type (via [featuretypes]) or a coverage (via
// [coverages]); manage them via this client after publish.
func ExampleWorkspaceClient_Iter() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	for l, err := range c.Layers.InWorkspace("topp").Iter(context.Background(), layers.ListOptions{}) {
		if err != nil {
			return
		}
		fmt.Println(l.Name)
	}
}

// ExampleWorkspaceClient_Update reassigns a layer's default style.
// Useful after [styles.Client.Create]+UploadSLD to make the new SLD
// the default rendering for a layer.
func ExampleWorkspaceClient_Update() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	ws := c.Layers.InWorkspace("topp")
	layer, err := ws.Get(context.Background(), "states")
	if errors.Is(err, geoserver.ErrNotFound) {
		return
	}

	layer.DefaultStyle = &layers.Ref{Name: "my-polygon"}
	_ = ws.Update(context.Background(), "states", layer)
}
