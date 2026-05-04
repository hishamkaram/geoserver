// publish-postgis is a runnable v2 example: end-to-end flow that creates
// a workspace, registers a PostGIS datastore, publishes a feature type,
// and reads the resulting layer back. Demonstrates the hierarchical
// sub-clients (InWorkspace, InDatastore).
//
// Requires the make-compose-up PostGIS stack with the lbldyt table
// pre-loaded (docker/postgis/init/01-lbldyt.sql).
//
//	go run ./v2/examples/publish-postgis
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/datastores"
	"github.com/hishamkaram/geoserver/v2/rest/featuretypes"
	"github.com/hishamkaram/geoserver/v2/rest/layers"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

func main() {
	url := envOr("GEOSERVER_URL", "http://localhost:8080/geoserver/")
	user := envOr("GEOSERVER_USER", "admin")
	pass := envOr("GEOSERVER_PASS", "geoserver")

	c, err := geoserver.New(url,
		geoserver.WithBasicAuth(user, pass),
		geoserver.WithTimeout(30*time.Second),
		geoserver.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))),
	)
	if err != nil {
		fatal("construct client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const (
		workspace = "v2_examples_postgis"
		datastore = "lbldyt_pg"
		// Native table name — must exist in the PostGIS DB.
		nativeTable  = "lbldyt"
		featureType  = "lbldyt"
		dbHost       = "postgis"
		dbName       = "geoserver"
		dbUser       = "postgres"
		dbPass       = "postgres"
		dbPort       = 5432
		expectedAttr = "wkb_geometry"
	)

	// Best-effort cleanup from a previous run, in workspace-recurse order.
	_ = c.Workspaces.Delete(ctx, workspace, workspaces.DeleteOptions{Recurse: true})

	// 1. Workspace.
	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: workspace}); err != nil {
		fatal("create workspace: %v", err)
	}
	fmt.Printf("created workspace %q\n", workspace)

	// 2. PostGIS datastore.
	if err := c.Datastores.InWorkspace(workspace).Create(ctx, datastores.PostGIS{
		Name:     datastore,
		Host:     dbHost,
		Port:     dbPort,
		Database: dbName,
		User:     dbUser,
		Password: dbPass,
	}); err != nil {
		fatal("create datastore: %v", err)
	}
	fmt.Printf("created PostGIS datastore %q in %q\n", datastore, workspace)

	// 3. Discover available tables (should include nativeTable).
	available, err := c.FeatureTypes.InWorkspace(workspace).InDatastore(datastore).
		Discover(ctx, featuretypes.DiscoverOptions{Kind: featuretypes.DiscoverAvailableWithGeometry})
	if err != nil {
		fatal("discover tables: %v", err)
	}
	fmt.Printf("discovered %d available tables: %v\n", len(available), available)

	// 4. Publish the feature type. NativeName must match the DB table.
	if err := c.FeatureTypes.InWorkspace(workspace).InDatastore(datastore).
		Create(ctx, &featuretypes.FeatureType{
			Name:       featureType,
			NativeName: nativeTable,
			SRS:        "EPSG:4326",
			Enabled:    true,
		}); err != nil {
		fatal("publish feature type: %v", err)
	}
	fmt.Printf("published feature type %q (native %q)\n", featureType, nativeTable)

	// 5. Fetch the auto-created layer back.
	layer, err := c.Layers.InWorkspace(workspace).Get(ctx, featureType)
	if err != nil {
		fatal("fetch layer: %v", err)
	}
	fmt.Printf("layer: name=%q type=%q resource=%q queryable=%t\n",
		layer.Name, layer.Type,
		safeName(layer.Resource),
		layer.Queryable)

	// 6. Inspect the feature type document — verify the geometry column was discovered.
	ft, err := c.FeatureTypes.InWorkspace(workspace).InDatastore(datastore).
		Get(ctx, featureType)
	if err != nil {
		fatal("fetch feature type: %v", err)
	}
	hasGeom := false
	if ft.Attributes != nil {
		for _, a := range ft.Attributes.Attribute {
			if a.Name == expectedAttr {
				hasGeom = true
				break
			}
		}
	}
	fmt.Printf("feature type has %s column: %t\n", expectedAttr, hasGeom)

	// 7. Cleanup. Recurse=true through the workspace cleans the whole tree.
	if err := c.Workspaces.Delete(ctx, workspace, workspaces.DeleteOptions{Recurse: true}); err != nil {
		// Don't error out — the example has already done its job.
		fmt.Fprintf(os.Stderr, "warn: cleanup failed: %v\n", err)
	} else {
		fmt.Printf("cleaned up workspace %q\n", workspace)
	}

	// Demonstrate that the datastore is gone.
	_, err = c.Datastores.InWorkspace(workspace).Get(ctx, datastore)
	if errors.Is(err, geoserver.ErrNotFound) {
		fmt.Println("verified datastore was removed by recursive workspace delete")
	}
}

func safeName(r *layers.Ref) string {
	if r == nil {
		return ""
	}
	return r.Name
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
