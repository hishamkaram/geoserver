package coveragestores_test

import (
	"context"
	"os"
	"strings"

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

// ExampleWorkspaceClient_UploadFile publishes a GeoTIFF by
// uploading the raster bytes — the modern alternative to copying
// the file into the data directory by hand and calling Reload.
func ExampleWorkspaceClient_UploadFile() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	f, err := os.Open("world_dem.tif")
	if err != nil {
		return
	}
	defer f.Close()

	_ = c.CoverageStores.InWorkspace("nurc").UploadFile(context.Background(),
		"world_dem", f, coveragestores.UploadOptions{
			Extension:   "geotiff",
			ContentType: "image/tiff",
		})
}

// ExampleWorkspaceClient_HarvestGranule appends a new granule to an
// existing image-mosaic store. Use UploadMethodExternal with a
// server-local path to avoid transferring large rasters across HTTP.
func ExampleWorkspaceClient_HarvestGranule() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.CoverageStores.InWorkspace("nurc").HarvestGranule(context.Background(),
		"world_mosaic",
		strings.NewReader("/srv/geoserver/granules/2026_05_03.tif"),
		coveragestores.UploadOptions{
			Method:    coveragestores.UploadMethodExternal,
			Extension: "imagemosaic",
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
