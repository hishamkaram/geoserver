//go:build integration

package layers_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/datastores"
	"github.com/hishamkaram/geoserver/v2/rest/featuretypes"
	"github.com/hishamkaram/geoserver/v2/rest/layers"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

// Tests the layer auto-creation flow: publish a feature type and verify
// the matching layer exists, can be read, updated, and deleted.
func TestLayers_AfterPublish_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	wsName := testenv.UniqueName(t, "ws")
	dsName := testenv.UniqueName(t, "ds")
	ftName := testenv.UniqueName(t, "ft")

	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: wsName}); err != nil {
		t.Fatalf("Create workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Workspaces.Delete(ctx, wsName, workspaces.DeleteOptions{Recurse: true})
	})

	if err := c.Datastores.InWorkspace(wsName).Create(ctx, datastores.PostGIS{
		Name:     dsName,
		Host:     testenv.DBHost,
		Port:     testenv.DBPort,
		Database: testenv.DBName,
		User:     testenv.DBUser,
		Password: testenv.DBPass,
	}); err != nil {
		t.Fatalf("Create datastore: %v", err)
	}
	if err := c.FeatureTypes.InWorkspace(wsName).InDatastore(dsName).Create(ctx, &featuretypes.FeatureType{
		Name: ftName, NativeName: "lbldyt", SRS: "EPSG:4326", Enabled: true,
	}); err != nil {
		t.Fatalf("Create feature type: %v", err)
	}

	// The layer should be auto-created with the same name as the feature type.
	layer, err := c.Layers.InWorkspace(wsName).Get(ctx, ftName)
	if err != nil {
		t.Fatalf("Get layer: %v", err)
	}
	if layer.Name != ftName {
		t.Fatalf("Layer.Name = %q, want %q", layer.Name, ftName)
	}
	if layer.DefaultStyle == nil {
		t.Errorf("DefaultStyle should be auto-assigned: %+v", layer)
	}
	if layer.Resource == nil {
		t.Errorf("Resource ref should be set: %+v", layer)
	}

	// List — confirm presence.
	all, err := c.Layers.InWorkspace(wsName).List(ctx, layers.ListOptions{})
	if err != nil {
		t.Fatalf("List layers: %v", err)
	}
	found := false
	for _, l := range all {
		if l.Name == ftName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("layer %q not found in workspace List: %+v", ftName, all)
	}

	// Update — flip Queryable.
	layer.Queryable = true
	if err := c.Layers.InWorkspace(wsName).Update(ctx, ftName, layer); err != nil {
		t.Fatalf("Update layer: %v", err)
	}

	// AddStyle — attach a built-in alternate style (default GeoServer
	// install ships with `line`, `point`, `polygon`, `raster`, etc.).
	if err := c.Layers.InWorkspace(wsName).AddStyle(ctx, ftName, "line",
		layers.AddStyleOptions{}); err != nil {
		t.Fatalf("AddStyle line: %v", err)
	}

	// ListStyles — confirm the new alternate is present.
	got, err := c.Layers.InWorkspace(wsName).ListStyles(ctx, ftName)
	if err != nil {
		t.Fatalf("ListStyles: %v", err)
	}
	foundLine := false
	for _, s := range got {
		if s.Name == "line" {
			foundLine = true
			break
		}
	}
	if !foundLine {
		t.Errorf("style %q not present in ListStyles after AddStyle: %+v", "line", got)
	}

	// AddStyle with Default=true — promotes the new style to the
	// layer's default style atomically. Verify by re-reading the layer.
	if err := c.Layers.InWorkspace(wsName).AddStyle(ctx, ftName, "point",
		layers.AddStyleOptions{Default: true}); err != nil {
		t.Fatalf("AddStyle point default: %v", err)
	}
	updated, err := c.Layers.InWorkspace(wsName).Get(ctx, ftName)
	if err != nil {
		t.Fatalf("Get after AddStyle default: %v", err)
	}
	if updated.DefaultStyle == nil || updated.DefaultStyle.Name != "point" {
		t.Errorf("DefaultStyle = %+v, want name=point", updated.DefaultStyle)
	}

	// Delete with Recurse (drops the underlying feature type too).
	if err := c.Layers.InWorkspace(wsName).Delete(ctx, ftName, layers.DeleteOptions{Recurse: true}); err != nil {
		t.Fatalf("Delete layer: %v", err)
	}
	if _, err := c.Layers.InWorkspace(wsName).Get(ctx, ftName); !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after Delete, got %v", err)
	}
}
