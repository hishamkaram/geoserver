//go:build integration

package featuretypes_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/datastores"
	"github.com/hishamkaram/geoserver/v2/rest/featuretypes"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

// The compose stack seeds a "lbldyt" table in the gis database via
// docker/postgis/init/01-lbldyt.sql.
const nativeTable = "lbldyt"

func TestFeatureTypes_Publish_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	wsName := testenv.UniqueName(t, "ws")
	dsName := testenv.UniqueName(t, "ds")
	ftName := testenv.UniqueName(t, "ft")

	// Workspace + datastore.
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

	ft := c.FeatureTypes.InWorkspace(wsName).InDatastore(dsName)

	// Discover should report the seeded lbldyt table as available.
	available, err := ft.Discover(ctx, featuretypes.DiscoverOptions{
		Kind: featuretypes.DiscoverAvailableWithGeometry,
	})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	hasNative := false
	for _, n := range available {
		if n == nativeTable {
			hasNative = true
			break
		}
	}
	if !hasNative {
		t.Fatalf("expected %q in Discover output, got %v", nativeTable, available)
	}

	// Publish.
	if err := ft.Create(ctx, &featuretypes.FeatureType{
		Name:       ftName,
		NativeName: nativeTable,
		SRS:        "EPSG:4326",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("Create feature type: %v", err)
	}

	// Get — full document, attributes populated.
	got, err := ft.Get(ctx, ftName)
	if err != nil {
		t.Fatalf("Get feature type: %v", err)
	}
	if got.Name != ftName || got.NativeName != nativeTable {
		t.Fatalf("FeatureType = %+v", got)
	}
	if got.Attributes == nil || len(got.Attributes.Attribute) == 0 {
		t.Errorf("expected attributes populated from PostGIS introspection: %+v", got.Attributes)
	}

	// List should include it.
	all, err := ft.List(ctx, featuretypes.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, t := range all {
		if t.Name == ftName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("feature type %q not found in List", ftName)
	}

	// Delete with recurse to also drop the layer.
	if err := ft.Delete(ctx, ftName, featuretypes.DeleteOptions{Recurse: true}); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = ft.Get(ctx, ftName)
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after Delete, got %v", err)
	}
}
