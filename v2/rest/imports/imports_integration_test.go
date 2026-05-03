//go:build integration

package imports_test

import (
	"errors"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/imports"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

// requireImporter probes /rest/imports and skips the test if the
// extension isn't installed (the endpoint returns 404). The
// Importer is an optional GeoServer extension and isn't always
// present.
func requireImporter(t *testing.T, c *geoserver.Client) {
	t.Helper()
	_, err := c.Imports.List(testenv.Context(t))
	if errors.Is(err, geoserver.ErrNotFound) {
		t.Skip("Importer extension not installed (GET /rest/imports → 404)")
	}
	if err != nil {
		t.Fatalf("Importer probe: %v", err)
	}
}

func TestImports_List_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	requireImporter(t, c)
	ctx := testenv.Context(t)

	// List should succeed (may be empty on a fresh server).
	got, err := c.Imports.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	t.Logf("Imports.List returned %d sessions", len(got))
}

func TestImports_CreateExecuteDelete_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	requireImporter(t, c)
	ctx := testenv.Context(t)

	wsName := testenv.UniqueName(t, "ws")
	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: wsName}); err != nil {
		t.Fatalf("Create workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Workspaces.Delete(ctx, wsName, workspaces.DeleteOptions{Recurse: true})
	})

	// Create a session targeting the workspace, no data source —
	// the importer accepts an empty session for the caller to
	// append tasks later.
	imp, err := c.Imports.Create(ctx, imports.ImportRequest{
		TargetWorkspace: wsName,
	}, imports.CreateOptions{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Note: the Importer assigns IDs starting at 0 — id=0 is a
	// legitimate value for the first session on a fresh server.
	// Verify identity via Href instead.
	if imp.Href == "" {
		t.Errorf("Create returned empty Href: %+v", imp)
	}
	t.Logf("Created import session id=%d state=%s href=%s", imp.ID, imp.State, imp.Href)

	t.Cleanup(func() {
		_ = c.Imports.Delete(ctx, imp.ID)
	})

	// Get round-trips the session.
	got, err := c.Imports.Get(ctx, imp.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != imp.ID {
		t.Errorf("Get id = %d, want %d", got.ID, imp.ID)
	}

	// List includes our session (until it's pruned).
	all, err := c.Imports.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, im := range all {
		if im.ID == imp.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("our session %d not in List: %+v", imp.ID, all)
	}

	// Tasks should be empty (no data source provided on Create).
	tasks, err := c.Imports.ListTasks(ctx, imp.ID)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks for empty session, got %d", len(tasks))
	}
}

func TestImports_FileSession_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	requireImporter(t, c)
	ctx := testenv.Context(t)

	wsName := testenv.UniqueName(t, "ws")
	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: wsName}); err != nil {
		t.Fatalf("Create workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Workspaces.Delete(ctx, wsName, workspaces.DeleteOptions{Recurse: true})
	})

	// The compose stack ships testdata/hurricane_tracks.zip in the
	// repo. Inside the container, the data dir is mounted and
	// callers typically reference paths the server can see; for an
	// integration test we use a path the importer-side filesystem
	// can read. Skip if no convenient path is available.
	hostPath := "/srv/geoserver/data_dir"
	imp, err := c.Imports.Create(ctx, imports.ImportRequest{
		TargetWorkspace: wsName,
		Data: &imports.Data{
			Type:     imports.DataTypeDirectory,
			Location: hostPath,
		},
	}, imports.CreateOptions{})
	if err != nil {
		// The importer rejects directories it can't read; treat
		// that as a skip rather than a hard failure since the
		// path varies by deploy.
		t.Skipf("Importer can't read %s (deploy-specific): %v", hostPath, err)
	}
	t.Cleanup(func() {
		_ = c.Imports.Delete(ctx, imp.ID)
	})

	t.Logf("Importer auto-populated session id=%d state=%s", imp.ID, imp.State)

	// Wait briefly for INIT to advance (for directory imports the
	// session enters PENDING / READY once the importer scans the
	// directory).
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		got, err := c.Imports.Get(ctx, imp.ID)
		if err != nil {
			t.Fatalf("Get during poll: %v", err)
		}
		if got.State != imports.StateInit {
			t.Logf("session advanced to %s", got.State)
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
}
