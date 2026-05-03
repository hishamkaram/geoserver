package coveragestores_test

import (
	"context"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/coveragestores"
)

// ExampleClient_InWorkspace returns a workspace-scoped coverage-stores
// client. Coverage stores are the raster-side analogue of datastores —
// each one points at a raster source (GeoTIFF, ImageMosaic, ArcSDE,
// etc.) and holds zero or more coverages.
func ExampleClient_InWorkspace() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.CoverageStores.InWorkspace("nurc")
}

// ExampleWorkspaceClient_Create publishes a GeoTIFF as a coverage
// store. URL points at a file inside the GeoServer data directory
// (or any URL the JVM can resolve).
func ExampleWorkspaceClient_Create() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.CoverageStores.InWorkspace("nurc").Create(context.Background(),
		&coveragestores.CoverageStore{
			Name:    "world_dem",
			URL:     "file:data/coverages/world_dem.tif",
			Type:    "GeoTIFF",
			Enabled: true,
		})
}

// ExampleWorkspaceClient_Update flips the Enabled flag without
// touching the URL or type. The pointer fields on
// [coveragestores.Patch] distinguish "leave alone" (nil) from
// "set to zero value" (&value).
func ExampleWorkspaceClient_Update() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	enabled := false
	_ = c.CoverageStores.InWorkspace("nurc").Update(context.Background(), "world_dem",
		&coveragestores.Patch{Enabled: &enabled})
}
