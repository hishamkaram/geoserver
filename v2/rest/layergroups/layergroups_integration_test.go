//go:build integration

package layergroups_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/datastores"
	"github.com/hishamkaram/geoserver/v2/rest/featuretypes"
	"github.com/hishamkaram/geoserver/v2/rest/layergroups"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

func TestLayerGroups_CRUD_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	wsName := testenv.UniqueName(t, "ws")
	dsName := testenv.UniqueName(t, "ds")
	ftName := testenv.UniqueName(t, "ft")
	lgName := testenv.UniqueName(t, "lg")

	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: wsName}); err != nil {
		t.Fatalf("Create workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Workspaces.Delete(ctx, wsName, workspaces.DeleteOptions{Recurse: true})
	})

	// Build a single layer to compose the group from.
	if err := c.Datastores.InWorkspace(wsName).Create(ctx, datastores.PostGIS{
		Name: dsName, Host: testenv.DBHost, Port: testenv.DBPort,
		Database: testenv.DBName, User: testenv.DBUser, Password: testenv.DBPass,
	}); err != nil {
		t.Fatalf("Create datastore: %v", err)
	}
	if err := c.FeatureTypes.InWorkspace(wsName).InDatastore(dsName).Create(ctx, &featuretypes.FeatureType{
		Name: ftName, NativeName: "lbldyt", SRS: "EPSG:4326", Enabled: true,
	}); err != nil {
		t.Fatalf("Create feature type: %v", err)
	}

	// Group reference must be qualified ("workspace:layer").
	qualifiedLayer := wsName + ":" + ftName

	if err := c.LayerGroups.InWorkspace(wsName).Create(ctx, &layergroups.LayerGroup{
		Name: lgName,
		Mode: "SINGLE",
		Publishables: layergroups.Publishables{
			Published: layergroups.Published{
				{Type: "layer", Name: qualifiedLayer},
			},
		},
	}); err != nil {
		t.Fatalf("Create layer group: %v", err)
	}

	// Get back — verify the single-member shape (Published unmarshals as
	// object, not 1-element array, on read).
	got, err := c.LayerGroups.InWorkspace(wsName).Get(ctx, lgName)
	if err != nil {
		t.Fatalf("Get layer group: %v", err)
	}
	if got.Name != lgName {
		t.Fatalf("Name = %q", got.Name)
	}
	if len(got.Publishables.Published) != 1 {
		t.Fatalf("expected one published member, got %+v", got.Publishables.Published)
	}

	// Update — change Title.
	got.Title = "Updated Title"
	if err := c.LayerGroups.InWorkspace(wsName).Update(ctx, lgName, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = c.LayerGroups.InWorkspace(wsName).Get(ctx, lgName)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.Title != "Updated Title" {
		t.Errorf("Title = %q, want Updated Title", got.Title)
	}

	// Delete — no recurse param (LayerGroup delete doesn't accept it;
	// underlying layer is unaffected).
	if err := c.LayerGroups.InWorkspace(wsName).Delete(ctx, lgName); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := c.LayerGroups.InWorkspace(wsName).Get(ctx, lgName); !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after Delete, got %v", err)
	}
}
