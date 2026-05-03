//go:build integration

package datastores_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/datastores"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

func TestDatastores_PostGIS_CRUD_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	wsName := testenv.UniqueName(t, "ws")
	dsName := testenv.UniqueName(t, "ds")

	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: wsName}); err != nil {
		t.Fatalf("Create workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Workspaces.Delete(ctx, wsName, workspaces.DeleteOptions{Recurse: true})
	})

	ws := c.Datastores.InWorkspace(wsName)

	// Empty-collection regression: an empty workspace must return a nil
	// slice (not a parse error). This is the GeoServer 2.28+ wire quirk
	// fixed in #55 / GitHub issue #22.
	empty, err := ws.List(ctx, datastores.ListOptions{})
	if err != nil {
		t.Fatalf("List on empty workspace: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty datastore list, got %+v", empty)
	}

	// Create — connect to the compose-stack PostGIS by service name.
	if err := ws.Create(ctx, datastores.PostGIS{
		Name:     dsName,
		Host:     testenv.DBHost,
		Port:     testenv.DBPort,
		Database: testenv.DBName,
		User:     testenv.DBUser,
		Password: testenv.DBPass,
	}); err != nil {
		t.Fatalf("Create datastore: %v", err)
	}

	// Get — full document with connection params.
	got, err := ws.Get(ctx, dsName)
	if err != nil {
		t.Fatalf("Get datastore: %v", err)
	}
	if got.Name != dsName {
		t.Fatalf("Name = %q, want %q", got.Name, dsName)
	}
	if got.Type == "" {
		t.Errorf("Type empty in returned datastore: %+v", got)
	}
	if got.Workspace == nil || got.Workspace.Name != wsName {
		t.Errorf("Workspace ref = %+v, want Name=%q", got.Workspace, wsName)
	}

	// List should now include the new datastore.
	all, err := ws.List(ctx, datastores.ListOptions{})
	if err != nil {
		t.Fatalf("List populated: %v", err)
	}
	found := false
	for _, d := range all {
		if d.Name == dsName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("datastore %q not found in List: %+v", dsName, all)
	}

	// Duplicate Create must produce an error. GeoServer 2.28's wire
	// behavior here is buggy: instead of returning 409 Conflict it
	// returns 500 with body "Store … already exists in workspace …".
	// Both ErrConflict (mocked path) and ErrServerError (real path)
	// surface as a non-nil typed error, so we accept either; the
	// unit test covers the strict 409 → ErrConflict mapping
	// separately.
	err = ws.Create(ctx, datastores.PostGIS{
		Name: dsName, Host: testenv.DBHost, Port: testenv.DBPort,
		Database: testenv.DBName, User: testenv.DBUser, Password: testenv.DBPass,
	})
	if err == nil {
		t.Fatalf("expected error on duplicate Create, got nil")
	}
	if !errors.Is(err, geoserver.ErrConflict) && !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("expected ErrConflict or ErrServerError on duplicate, got %v", err)
	}

	// Delete.
	if err := ws.Delete(ctx, dsName, datastores.DeleteOptions{Recurse: true}); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify gone.
	_, err = ws.Get(ctx, dsName)
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after Delete, got %v", err)
	}

	// Empty list again.
	leftover, err := ws.List(ctx, datastores.ListOptions{})
	if err != nil {
		t.Fatalf("List after Delete: %v", err)
	}
	if len(leftover) != 0 {
		t.Fatalf("expected empty after Delete, got %+v", leftover)
	}
}
