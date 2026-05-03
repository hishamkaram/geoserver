package datastores_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/datastores"
)

// ExampleClient_InWorkspace returns a workspace-scoped datastore
// client. All datastore operations are workspace-scoped — the root
// [*Client] is just an entry point.
func ExampleClient_InWorkspace() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	ws := c.Datastores.InWorkspace("topp")
	_ = ws // ws.Get / ws.Create / ws.List / ws.Update / ws.Delete
}

// ExampleWorkspaceClient_UploadFile publishes a Shapefile by
// uploading a zipped bundle (the .shp + .shx + .dbf + .prj
// sidecars). For non-PostgreSQL data this is the fast path —
// no need to copy files into the data directory by hand.
func ExampleWorkspaceClient_UploadFile() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	f, err := os.Open("states.zip")
	if err != nil {
		return
	}
	defer f.Close()

	_ = c.Datastores.InWorkspace("topp").UploadFile(context.Background(),
		"states_shp", f, datastores.UploadOptions{Extension: "shp"})
}

// ExampleWorkspaceClient_UploadFile_external registers a
// server-local Shapefile path without transferring bytes — the
// store points at a file already on the GeoServer host.
func ExampleWorkspaceClient_UploadFile_external() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Datastores.InWorkspace("topp").UploadFile(context.Background(),
		"states_shp", strings.NewReader("/srv/geoserver/data/states.shp"),
		datastores.UploadOptions{
			Method:    datastores.UploadMethodExternal,
			Extension: "shp",
		})
}

// ExampleWorkspaceClient_Create_postGIS publishes a PostGIS connection
// using the convenience [datastores.PostGIS] connector. For other
// drivers, supply a [datastores.Datastore] directly via
// [datastores.Raw].
func ExampleWorkspaceClient_Create_postGIS() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	err := c.Datastores.InWorkspace("topp").Create(context.Background(), datastores.PostGIS{
		Name:     "states_pg",
		Host:     "postgis.example.com",
		Port:     5432,
		Database: "gis",
		Schema:   "public",
		User:     "gis_ro",
		Password: "secret",
	})
	if errors.Is(err, geoserver.ErrConflict) {
		fmt.Println("already exists")
	}
}

// ExampleWorkspaceClient_Create_raw publishes a shapefile via the
// [datastores.Raw] adapter — useful for drivers that don't have a
// dedicated convenience type yet.
func ExampleWorkspaceClient_Create_raw() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Datastores.InWorkspace("topp").Create(context.Background(),
		datastores.Raw(datastores.Datastore{
			Name: "states_shp",
			ConnectionParameters: datastores.ConnectionParameters{
				Entry: []datastores.ConnectionEntry{
					{Key: "url", Value: "file:data/shapefiles/states.shp"},
				},
			},
		}))
}

// ExampleWorkspaceClient_Iter ranges over every datastore in a
// workspace.
func ExampleWorkspaceClient_Iter() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	for ds, err := range c.Datastores.InWorkspace("topp").Iter(context.Background(), datastores.ListOptions{}) {
		if err != nil {
			return
		}
		fmt.Printf("%s (%s)\n", ds.Name, ds.Type)
	}
}
