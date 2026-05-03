//go:build integration

package workspaces_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

func TestWorkspaces_CRUD_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	name := testenv.UniqueName(t, "ws")

	// Cleanup hook in case the test fails partway through.
	t.Cleanup(func() {
		_ = c.Workspaces.Delete(ctx, name, workspaces.DeleteOptions{Recurse: true})
	})

	// Verify a fresh workspace doesn't exist yet.
	_, err := c.Workspaces.Get(ctx, name)
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for fresh name, got %v", err)
	}

	// Create.
	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: name}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get.
	got, err := c.Workspaces.Get(ctx, name)
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if got.Name != name {
		t.Fatalf("Workspace.Name = %q, want %q", got.Name, name)
	}

	// List should include it.
	all, err := c.Workspaces.List(ctx, workspaces.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, ws := range all {
		if ws.Name == name {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("workspace %q not found in List", name)
	}

	// Conflict on duplicate Create.
	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: name}); !errors.Is(err, geoserver.ErrConflict) {
		t.Fatalf("expected ErrConflict on duplicate Create, got %v", err)
	}

	// Update — flip Isolated.
	isolated := true
	if err := c.Workspaces.Update(ctx, name, &workspaces.WorkspacePatch{Isolated: &isolated}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = c.Workspaces.Get(ctx, name)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if !got.Isolated {
		t.Fatalf("expected Isolated=true after Update, got %+v", got)
	}

	// Delete.
	if err := c.Workspaces.Delete(ctx, name, workspaces.DeleteOptions{Recurse: true}); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify gone.
	_, err = c.Workspaces.Get(ctx, name)
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after Delete, got %v", err)
	}
}

func TestWorkspaces_Iter_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	count := 0
	for ws, err := range c.Workspaces.Iter(ctx, workspaces.ListOptions{}) {
		if err != nil {
			t.Fatalf("Iter error: %v", err)
		}
		if ws.Name == "" {
			t.Errorf("workspace with empty name: %+v", ws)
		}
		count++
	}
	if count == 0 {
		t.Fatalf("expected at least one workspace from Iter (default install ships with several)")
	}
}
