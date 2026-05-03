package featuretypes_test

import (
	"context"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/featuretypes"
)

// ExampleClient_InWorkspace shows the 2-level scoping. Feature types
// live under workspace + datastore, so callers fluently chain through
// both levels before reaching CRUD methods.
func ExampleClient_InWorkspace() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	ds := c.FeatureTypes.InWorkspace("topp").InDatastore("states_pg")
	_ = ds // ds.Get / ds.List / ds.Create / ds.Update / ds.Delete / ds.Discover
}

// ExampleDatastoreClient_Discover lists tables in the underlying
// datastore that haven't yet been published as feature types — the
// typical input to a publish workflow.
func ExampleDatastoreClient_Discover() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	ds := c.FeatureTypes.InWorkspace("topp").InDatastore("states_pg")
	tables, err := ds.Discover(context.Background(), featuretypes.DiscoverOptions{
		Kind: featuretypes.DiscoverAvailableWithGeometry,
	})
	if err != nil {
		return
	}
	for _, t := range tables {
		fmt.Println(t)
	}
}

// ExampleDatastoreClient_Create publishes a PostGIS table as a feature
// type. NativeName must match the underlying table; SRS is required
// for WMS rendering.
func ExampleDatastoreClient_Create() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.FeatureTypes.InWorkspace("topp").InDatastore("states_pg").Create(context.Background(),
		&featuretypes.FeatureType{
			Name:       "states",
			NativeName: "states",
			Title:      "USA States",
			SRS:        "EPSG:4326",
		})
}
