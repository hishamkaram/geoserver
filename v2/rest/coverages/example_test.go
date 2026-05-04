package coverages_test

import (
	"context"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/coverages"
)

// ExampleClient_InWorkspace shows the 2-level scoping. Coverages live
// under workspace + coverage store; chain through both before reaching
// CRUD methods.
func ExampleClient_InWorkspace() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	cs := c.Coverages.InWorkspace("nurc").InCoverageStore("world_dem")
	_ = cs // cs.Get / cs.List / cs.Create / cs.Update / cs.Delete / cs.Discover
}

// ExampleCoverageStoreClient_Discover lists native coverages in the
// store. Default mode (DiscoverAll) returns configured + available
// — most coverage stores expose a single coverage that's already
// configured, so DiscoverAll is the typical default.
func ExampleCoverageStoreClient_Discover() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	cs := c.Coverages.InWorkspace("nurc").InCoverageStore("world_dem")
	names, err := cs.Discover(context.Background(), coverages.DiscoverOptions{})
	if err != nil {
		return
	}
	for _, n := range names {
		fmt.Println(n)
	}
}

// ExampleCoverageStoreClient_Create publishes a native coverage from
// the store. NativeCoverageName must match a name returned by
// [CoverageStoreClient.Discover].
func ExampleCoverageStoreClient_Create() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Coverages.InWorkspace("nurc").InCoverageStore("world_dem").Create(context.Background(),
		&coverages.Coverage{
			Name:               "world_dem",
			NativeCoverageName: "world_dem",
			Title:              "World DEM",
			SRS:                "EPSG:4326",
		})
}

// ExampleGranulesClient_List walks the granules of a structured
// (image-mosaic) coverage. Each granule corresponds to one
// underlying raster file participating in the mosaic.
func ExampleGranulesClient_List() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	g := c.Coverages.InWorkspace("nurc").InCoverageStore("mosaic").Granules("mosaic")
	list, err := g.List(context.Background(), coverages.ListGranulesOptions{Limit: 100})
	if err != nil {
		return
	}
	for _, gr := range list {
		loc, _ := gr.Properties["location"].(string)
		fmt.Printf("%s -> %s\n", gr.ID, loc)
	}
}

// ExampleGranulesClient_DeleteByFilter removes every granule whose
// location matches a CQL pattern, leaving the underlying files on
// disk (Purge=PurgeNone). Pass Filter:"INCLUDE" to delete every
// granule deliberately.
func ExampleGranulesClient_DeleteByFilter() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Coverages.InWorkspace("nurc").InCoverageStore("mosaic").Granules("mosaic").
		DeleteByFilter(context.Background(), coverages.DeleteGranulesOptions{
			Filter: "ingestion < '2020-01-01'",
			Purge:  coverages.PurgeNone,
		})
}
