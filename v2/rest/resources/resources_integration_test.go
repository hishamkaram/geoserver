//go:build integration

package resources_test

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/resources"
)

// TestResources_Stat_DirectoryAndFile_Integration confirms the wire
// shape against a vanilla GeoServer install. The "styles" directory
// always exists in a fresh data dir; "default_point.sld" is one of
// the default seed styles.
func TestResources_Stat_DirectoryAndFile_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	dirMeta, err := c.Resources.Stat(ctx, "styles")
	if err != nil {
		t.Fatalf("Stat styles dir: %v", err)
	}
	if dirMeta.Name != "styles" {
		t.Errorf("dir Name = %q", dirMeta.Name)
	}
	if dirMeta.Type != resources.TypeDirectory {
		t.Errorf("dir Type = %q, want directory", dirMeta.Type)
	}

	fileMeta, err := c.Resources.Stat(ctx, "styles/default_point.sld")
	if err != nil {
		t.Fatalf("Stat default_point.sld: %v", err)
	}
	if fileMeta.Type != resources.TypeResource {
		t.Errorf("file Type = %q, want resource", fileMeta.Type)
	}
}

func TestResources_List_StylesDir_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	dir, err := c.Resources.List(ctx, "styles")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if dir.Type != resources.TypeDirectory {
		t.Errorf("Type = %q", dir.Type)
	}
	// Vanilla GeoServer ships with multiple default styles; this is
	// a sanity check that we got a populated listing through.
	if len(dir.Children) < 5 {
		t.Errorf("expected ≥5 default styles, got %d", len(dir.Children))
	}
	// At least one expected default — default_point.sld is in every
	// stock distribution.
	found := false
	for _, ch := range dir.Children {
		if ch.Name == "default_point.sld" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("default_point.sld not in listing; children: %v", dir.Children)
	}
}

func TestResources_Get_FileContent_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	body, err := c.Resources.Get(ctx, "styles/default_point.sld")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer body.Close()

	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	// SLD is XML — should start with the XML declaration.
	if !strings.HasPrefix(string(got), "<?xml") {
		head := got
		if len(head) > 80 {
			head = head[:80]
		}
		t.Errorf("expected XML body, first 80 bytes: %q", head)
	}
}

func TestResources_Exists_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	exists, typ, err := c.Resources.Exists(ctx, "styles/default_point.sld")
	if err != nil {
		t.Fatalf("Exists existing: %v", err)
	}
	if !exists || typ != resources.TypeResource {
		t.Errorf("existing file: exists=%v type=%q", exists, typ)
	}

	exists, _, err = c.Resources.Exists(ctx, "styles/this_does_not_exist_"+fmt.Sprint(time.Now().UnixNano())+".sld")
	if err != nil {
		t.Fatalf("Exists missing: %v", err)
	}
	if exists {
		t.Errorf("expected exists=false for missing file")
	}
}

// Full create/read/move/copy/delete flow under a unique temp path.
func TestResources_RoundTrip_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Use a unique sub-directory under the writable workspaces/ tree
	// (which always exists). Filename uses nanosecond uniqueness so
	// repeated runs don't collide.
	stamp := time.Now().UnixNano()
	srcPath := fmt.Sprintf("workspaces/v2_it_%d/test.txt", stamp)
	movedPath := fmt.Sprintf("workspaces/v2_it_%d/test_moved.txt", stamp)
	copiedPath := fmt.Sprintf("workspaces/v2_it_%d/test_copy.txt", stamp)

	// Best-effort cleanup: remove the whole temp directory.
	t.Cleanup(func() {
		_ = c.Resources.Delete(ctx, fmt.Sprintf("workspaces/v2_it_%d", stamp))
	})

	const payload = "hello world"
	if err := c.Resources.Put(ctx, srcPath, strings.NewReader(payload), "text/plain"); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Read it back.
	body, err := c.Resources.Get(ctx, srcPath)
	if err != nil {
		t.Fatalf("Get after Put: %v", err)
	}
	got, _ := io.ReadAll(body)
	body.Close()
	if string(got) != payload {
		t.Errorf("Get content = %q, want %q", got, payload)
	}

	// Move.
	if err := c.Resources.Move(ctx, srcPath, movedPath); err != nil {
		t.Fatalf("Move: %v", err)
	}
	// Source should be gone.
	exists, _, err := c.Resources.Exists(ctx, srcPath)
	if err != nil {
		t.Fatalf("Exists src after Move: %v", err)
	}
	if exists {
		t.Errorf("src %q still exists after Move", srcPath)
	}
	// Destination should be there.
	exists, _, err = c.Resources.Exists(ctx, movedPath)
	if err != nil {
		t.Fatalf("Exists moved after Move: %v", err)
	}
	if !exists {
		t.Errorf("moved %q not present after Move", movedPath)
	}

	// Copy moved → copied.
	if err := c.Resources.Copy(ctx, movedPath, copiedPath); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	for _, p := range []string{movedPath, copiedPath} {
		exists, _, err := c.Resources.Exists(ctx, p)
		if err != nil {
			t.Fatalf("Exists %q after Copy: %v", p, err)
		}
		if !exists {
			t.Errorf("%q missing after Copy", p)
		}
	}

	// Delete both.
	for _, p := range []string{movedPath, copiedPath} {
		if err := c.Resources.Delete(ctx, p); err != nil {
			t.Fatalf("Delete %q: %v", p, err)
		}
	}
	// Confirm both gone.
	for _, p := range []string{movedPath, copiedPath} {
		_, err := c.Resources.Stat(ctx, p)
		if !errors.Is(err, geoserver.ErrNotFound) {
			t.Errorf("Stat %q after Delete: expected ErrNotFound, got %v", p, err)
		}
	}
}
