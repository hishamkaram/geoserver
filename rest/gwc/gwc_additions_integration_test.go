//go:build integration

package gwc_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/gwc"
)

func TestGWC_Global_GetUpdate_RoundTrip_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	original, err := c.GWC.Global().Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if original.Identifier == "" {
		t.Errorf("expected non-empty Identifier, got %+v", original)
	}

	// Restore on cleanup so subsequent tests / dev sessions are unaffected.
	t.Cleanup(func() {
		_ = c.GWC.Global().Update(ctx, original)
	})

	// Toggle backendTimeout to a recognizably-different value.
	target := *original
	if target.BackendTimeout == 240 {
		target.BackendTimeout = 180
	} else {
		target.BackendTimeout = 240
	}
	if err := c.GWC.Global().Update(ctx, &target); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := c.GWC.Global().Get(ctx)
	if err != nil {
		t.Fatalf("Get after Update: %v", err)
	}
	if got.BackendTimeout != target.BackendTimeout {
		t.Errorf("after Update BackendTimeout = %d, want %d", got.BackendTimeout, target.BackendTimeout)
	}
}

func TestGWC_Gridsets_List_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	got, err := c.GWC.Gridsets().List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// Stock GeoServer ships the EPSG:4326 + WebMercatorQuad gridsets
	// and dozens of UTM tilings. Sanity check is just "non-empty".
	if len(got) == 0 {
		t.Fatal("expected at least one gridset, got empty list")
	}
	// Every name should be non-empty.
	for _, n := range got {
		if n == "" {
			t.Fatalf("empty name in list: %v", got)
		}
	}
}

func TestGWC_Gridsets_Get_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	g, err := c.GWC.Gridsets().Get(ctx, "EPSG:4326")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if g.Name != "EPSG:4326" {
		t.Errorf("Name = %q, want %q", g.Name, "EPSG:4326")
	}
	if g.SRS.Number != 4326 {
		t.Errorf("SRS.Number = %d, want 4326", g.SRS.Number)
	}
	if len(g.Extent.Coords) != 4 {
		t.Errorf("Extent.Coords len = %d, want 4", len(g.Extent.Coords))
	}
}

func TestGWC_Gridsets_Get_NotFound_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	_, err := c.GWC.Gridsets().Get(ctx, "definitely-does-not-exist")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// GeoServer may answer 404 or 500 depending on version; accept either.
	if !errors.Is(err, geoserver.ErrNotFound) && !errors.Is(err, geoserver.ErrServerError) {
		t.Errorf("expected ErrNotFound or ErrServerError, got %v", err)
	}
}

func TestGWC_MassTruncate_Capabilities_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	caps, err := c.GWC.MassTruncate().Capabilities(ctx)
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	// Stock GeoServer 2.x exposes all four documented operations.
	want := map[gwc.MassTruncateRequestType]bool{
		gwc.TruncateLayer:      false,
		gwc.TruncateParameters: false,
		gwc.TruncateOrphans:    false,
		gwc.TruncateExtent:     false,
	}
	for _, c := range caps {
		want[c] = true
	}
	for k, seen := range want {
		if !seen {
			t.Errorf("expected %q in capabilities, got %v", k, caps)
		}
	}
}

// TestGWC_MassTruncate_TruncateLayer exercises the wire-quirk path
// (text/xml content-type required) against a stock-bundled layer.
// Truncating an already-empty cache is idempotent.
//
// TruncateOrphans / TruncateParameters / TruncateExtent are covered
// by unit tests but not exercised here — TruncateOrphans rejects an
// empty body on GeoServer 2.28.0 ("layerName is null"); the others
// fail on a fresh stack because there are no cached tiles to truncate
// (the truncate-extent / truncate-parameters paths short-circuit
// when the cache is empty).
func TestGWC_MassTruncate_TruncateLayer_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Use sf:archsites — bundled in the default sample data and
	// always cacheable. Truncating an already-empty cache is a no-op.
	if err := c.GWC.MassTruncate().TruncateLayer(ctx, "sf:archsites"); err != nil {
		t.Fatalf("TruncateLayer: %v", err)
	}
}
