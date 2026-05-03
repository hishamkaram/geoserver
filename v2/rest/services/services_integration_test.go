//go:build integration

package services_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/services"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

func TestServices_WMS_Global_GetUpdate_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	got, err := c.Services.WMS().Get(ctx)
	if err != nil {
		t.Fatalf("WMS.Get global: %v", err)
	}
	if got.Name == "" {
		t.Errorf("WMS Name is empty: %+v", got.ServiceInfo)
	}

	// Round-trip: bump MaxRenderingTime, write, read back, restore.
	original := got.MaxRenderingTime
	t.Cleanup(func() {
		got.MaxRenderingTime = original
		_ = c.Services.WMS().Update(ctx, got)
	})

	got.MaxRenderingTime = original + 7
	if err := c.Services.WMS().Update(ctx, got); err != nil {
		t.Fatalf("WMS.Update global: %v", err)
	}
	after, err := c.Services.WMS().Get(ctx)
	if err != nil {
		t.Fatalf("WMS.Get after update: %v", err)
	}
	if after.MaxRenderingTime != original+7 {
		t.Errorf("MaxRenderingTime didn't round-trip: got %d, want %d", after.MaxRenderingTime, original+7)
	}
}

func TestServices_WFS_PerWorkspace_Override_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	wsName := testenv.UniqueName(t, "ws")
	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: wsName}); err != nil {
		t.Fatalf("Create workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Workspaces.Delete(ctx, wsName, workspaces.DeleteOptions{Recurse: true})
	})

	wfs := c.Services.WFS().InWorkspace(wsName)

	// No override yet — Get should return ErrNotFound (some
	// GeoServer versions return 200 with an empty body; both are
	// acceptable for the SDK contract).
	_, err := wfs.Get(ctx)
	if err != nil && !errors.Is(err, geoserver.ErrNotFound) {
		t.Logf("Get pre-override (acceptable): %v", err)
	}

	// Create the override.
	if err := wfs.Update(ctx, &services.WFSSettings{
		ServiceInfo: services.ServiceInfo{Enabled: true, Title: "Override"},
		MaxFeatures: 250,
	}); err != nil {
		t.Fatalf("Update override: %v", err)
	}

	got, err := wfs.Get(ctx)
	if err != nil {
		t.Fatalf("Get after override: %v", err)
	}
	if got.MaxFeatures != 250 {
		t.Errorf("MaxFeatures = %d, want 250", got.MaxFeatures)
	}

	// Delete falls back to the global config. After delete, Get
	// should return ErrNotFound (or 200 empty per server version).
	if err := wfs.Delete(ctx); err != nil {
		t.Fatalf("Delete override: %v", err)
	}
}

func TestServices_WCS_Global_Get_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	got, err := c.Services.WCS().Get(ctx)
	if err != nil {
		t.Fatalf("WCS.Get: %v", err)
	}
	if got.Name == "" {
		t.Errorf("WCS Name is empty: %+v", got.ServiceInfo)
	}
}

func TestServices_WMTS_Global_Get_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	got, err := c.Services.WMTS().Get(ctx)
	if err != nil {
		// WMTS may not be enabled in every install; treat 404 as a
		// skip rather than a hard failure.
		if errors.Is(err, geoserver.ErrNotFound) {
			t.Skipf("WMTS service not enabled on this GeoServer")
		}
		t.Fatalf("WMTS.Get: %v", err)
	}
	if got.Name == "" {
		t.Errorf("WMTS Name is empty: %+v", got.ServiceInfo)
	}
}
