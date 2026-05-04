//go:build integration

package gwc_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/gwc"
)

func TestGWC_Layers_List_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	got, err := c.GWC.Layers().List(ctx)
	if err != nil {
		t.Fatalf("Layers.List: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected at least one cached layer in default install")
	}
}

func TestGWC_Layers_Get_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// topp:states is shipped with the default GeoServer install.
	cfg, err := c.GWC.Layers().Get(ctx, "topp:states")
	if err != nil {
		t.Fatalf("Layers.Get: %v", err)
	}
	if cfg.Name != "topp:states" {
		t.Errorf("Name = %q", cfg.Name)
	}
	if cfg.MimeFormats == nil || len(cfg.MimeFormats.String) == 0 {
		t.Errorf("MimeFormats empty: %+v", cfg.MimeFormats)
	}
	if cfg.GridSubsets == nil || len(cfg.GridSubsets.GridSubset) == 0 {
		t.Errorf("GridSubsets empty: %+v", cfg.GridSubsets)
	}
}

func TestGWC_Layers_Get_NotFound_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Wire-quirk: GWC returns 500 (with body "Unknown layer: ...")
	// for unknown layers rather than 404. Both ErrNotFound (mocked
	// path) and ErrServerError (real path) surface as a non-nil
	// typed error, so we accept either; the unit test covers the
	// strict 404 → ErrNotFound mapping separately.
	_, err := c.GWC.Layers().Get(ctx, "definitely:not_a_layer_zz")
	if err == nil {
		t.Fatalf("expected error for unknown layer")
	}
	if !errors.Is(err, geoserver.ErrNotFound) && !errors.Is(err, geoserver.ErrServerError) {
		t.Fatalf("err = %v, want ErrNotFound or ErrServerError", err)
	}
}

func TestGWC_Seed_Truncate_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Truncate is the safest seed op for an integration test —
	// no actual tile generation, just cache invalidation.
	err := c.GWC.Seed().Submit(ctx, "topp:states", &gwc.SeedRequest{
		SRS:         gwc.SRS{Number: 4326},
		ZoomStart:   0,
		ZoomStop:    2,
		Format:      "image/png",
		Type:        gwc.OpTruncate,
		ThreadCount: 1,
		GridSetID:   "EPSG:4326",
		Bounds: &gwc.Bounds{Coords: gwc.BoundsCoords{
			Double: []float64{-180, -90, 180, 90},
		}},
	})
	if err != nil {
		t.Fatalf("Seed.Submit truncate: %v", err)
	}

	// Status query should succeed (may or may not list the just-
	// submitted task — truncate is fast).
	if _, err := c.GWC.Seed().StatusAll(ctx); err != nil {
		t.Fatalf("Seed.StatusAll: %v", err)
	}
	if _, err := c.GWC.Seed().Status(ctx, "topp:states"); err != nil {
		t.Fatalf("Seed.Status: %v", err)
	}
}

func TestGWC_DiskQuota_GetUpdate_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	original, err := c.GWC.DiskQuota().Get(ctx)
	if err != nil {
		t.Fatalf("DiskQuota.Get: %v", err)
	}
	t.Cleanup(func() {
		_ = c.GWC.DiskQuota().Update(ctx, original)
	})

	if original.GlobalExpirationPolicyName == "" {
		t.Errorf("GlobalExpirationPolicyName empty: %+v", original)
	}

	// Round-trip: flip the enabled flag, write, read back, restore.
	modified := *original
	modified.Enabled = !original.Enabled
	if err := c.GWC.DiskQuota().Update(ctx, &modified); err != nil {
		t.Fatalf("DiskQuota.Update: %v", err)
	}
	after, err := c.GWC.DiskQuota().Get(ctx)
	if err != nil {
		t.Fatalf("DiskQuota.Get after update: %v", err)
	}
	if after.Enabled != modified.Enabled {
		t.Errorf("Enabled didn't round-trip: got %v, want %v", after.Enabled, modified.Enabled)
	}
}
