// publish-postgis is a runnable example: register a PostGIS-backed datastore
// in a GeoServer workspace and publish an existing table as a feature type
// using the v1.1 typed Options field on DatastoreConnection.
//
// Prerequisites (matches `make compose-up`):
//   - GeoServer at http://localhost:8080/geoserver/ (admin / geoserver)
//   - PostGIS reachable from inside the GeoServer container at host
//     "postgis" (compose service name), DB "gis", user/pass "golang"
//   - A table "public.lbldyt" with at least one geometry column. The
//     compose stack seeds this via docker/postgis/init/01-lbldyt.sql.
//
// Run with:
//
//	go run ./examples/publish-postgis
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hishamkaram/geoserver"
)

func main() {
	url := envOr("GEOSERVER_URL", "http://localhost:8080/geoserver/")
	user := envOr("GEOSERVER_USER", "admin")
	pass := envOr("GEOSERVER_PASS", "geoserver")

	gs := geoserver.New(url, user, pass, geoserver.WithTimeout(30*time.Second))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const (
		ws    = "examples_publish_demo"
		ds    = "examples_postgis_ds"
		table = "lbldyt"
	)

	// Cleanup any prior run.
	_, _ = gs.DeleteWorkspaceContext(ctx, ws, true)

	if _, err := gs.CreateWorkspaceContext(ctx, ws); err != nil {
		fatal("create workspace: %v", err)
	}
	fmt.Printf("created workspace %q\n", ws)
	defer func() {
		_, _ = gs.DeleteWorkspaceContext(ctx, ws, true)
		fmt.Printf("cleaned up workspace %q\n", ws)
	}()

	// PostGIS connection. Note: Host is the docker-network hostname seen
	// from the GeoServer container, not localhost. Options carries any
	// extra connection params (e.g., expose primary keys, max connections).
	conn := geoserver.DatastoreConnection{
		Name:   ds,
		Host:   "postgis",
		Port:   5432,
		Type:   "postgis",
		DBName: "gis",
		DBUser: "golang",
		DBPass: "golang",
		Options: []geoserver.Entry{
			{Key: "Expose primary keys", Value: "true"},
			{Key: "max connections", Value: "5"},
		},
	}
	if _, err := gs.CreateDatastoreContext(ctx, conn, ws); err != nil {
		fatal("create datastore: %v", err)
	}
	fmt.Printf("created datastore %q in workspace %q\n", ds, ws)

	// Discover what tables in the datastore are published vs unpublished.
	available, err := gs.GetFeatureTypeListContext(ctx, ws, ds, geoserver.FeatureTypeListAvailable)
	if err != nil {
		fatal("list available feature types: %v", err)
	}
	fmt.Printf("available (unpublished) tables: %v\n", available)

	// Publish the seeded table as a feature type.
	ft := &geoserver.FeatureType{
		Name:       table,
		NativeName: table,
		Title:      "Sample lbldyt feature type",
		Srs:        "EPSG:4326",
	}
	if _, err := gs.CreateFeatureTypeContext(ctx, ws, ds, ft); err != nil {
		fatal("create feature type: %v", err)
	}
	fmt.Printf("published %q as feature type\n", table)

	// Fetch the corresponding layer to confirm GeoServer wired it up.
	layer, err := gs.GetLayerContext(ctx, ws, table)
	if err != nil {
		fatal("fetch layer: %v", err)
	}
	fmt.Printf("layer: name=%q type=%q default-style=%q\n",
		layer.Name, layer.Type, defaultStyleName(layer))
}

func defaultStyleName(layer *geoserver.Layer) string {
	if layer == nil || layer.DefaultStyle == nil {
		return ""
	}
	return layer.DefaultStyle.Name
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
