//go:build integration

package coverages_test

import (
	"errors"
	"strings"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/coverages"
)

// The default GeoServer install ships an `nurc:mosaic` image-mosaic
// coverage store with multiple granules backed by global_mosaic_*.png
// rasters. These tests exercise the granule index against that
// fixture without modifying any state — read-only.
const (
	mosaicWorkspace = "nurc"
	mosaicStore     = "mosaic"
	mosaicCoverage  = "mosaic"
)

func TestCoverages_Granules_Schema_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	g := c.Coverages.InWorkspace(mosaicWorkspace).InCoverageStore(mosaicStore).Granules(mosaicCoverage)
	schema, err := g.Schema(ctx)
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if len(schema.Attributes) < 2 {
		t.Errorf("expected >=2 attributes in mosaic granule schema, got %d", len(schema.Attributes))
	}
	// `the_geom` and `location` are present in every default mosaic.
	wantNames := map[string]bool{"the_geom": false, "location": false}
	for _, a := range schema.Attributes {
		if _, ok := wantNames[a.Name]; ok {
			wantNames[a.Name] = true
		}
	}
	for n, found := range wantNames {
		if !found {
			t.Errorf("expected attribute %q in schema; got %+v", n, schema.Attributes)
		}
	}
}

func TestCoverages_Granules_List_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	g := c.Coverages.InWorkspace(mosaicWorkspace).InCoverageStore(mosaicStore).Granules(mosaicCoverage)
	list, err := g.List(ctx, coverages.ListGranulesOptions{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) == 0 {
		t.Fatalf("expected at least 1 granule in default mosaic, got 0")
	}
	// Every granule should have an ID and a non-empty location property.
	for i, gr := range list {
		if gr.ID == "" {
			t.Errorf("granule[%d] missing ID: %+v", i, gr)
		}
		if loc, _ := gr.Properties["location"].(string); loc == "" {
			t.Errorf("granule[%d] missing location property: %+v", i, gr.Properties)
		}
	}
}

func TestCoverages_Granules_List_Filter_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	g := c.Coverages.InWorkspace(mosaicWorkspace).InCoverageStore(mosaicStore).Granules(mosaicCoverage)
	// Filter on the location attribute that ships with the default mosaic.
	list, err := g.List(ctx, coverages.ListGranulesOptions{
		Filter: "location LIKE 'global_mosaic_0%'",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("List with filter: %v", err)
	}
	for _, gr := range list {
		loc, _ := gr.Properties["location"].(string)
		if !strings.HasPrefix(loc, "global_mosaic_0") {
			t.Errorf("filter returned non-matching granule: %+v", gr)
		}
	}
}

func TestCoverages_Granules_Get_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	g := c.Coverages.InWorkspace(mosaicWorkspace).InCoverageStore(mosaicStore).Granules(mosaicCoverage)
	// First find a real granule ID by listing.
	list, err := g.List(ctx, coverages.ListGranulesOptions{Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) == 0 {
		t.Skip("no granules in default mosaic; skipping Get round-trip")
	}
	id := list[0].ID
	gr, err := g.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get %q: %v", id, err)
	}
	if gr == nil {
		t.Fatalf("Get returned nil for known granule %q", id)
	}
	if gr.ID != id {
		t.Errorf("Get ID = %q, want %q", gr.ID, id)
	}
}

func TestCoverages_Granules_Get_NotFound_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	g := c.Coverages.InWorkspace(mosaicWorkspace).InCoverageStore(mosaicStore).Granules(mosaicCoverage)
	gr, err := g.Get(ctx, "definitely_not_a_real_granule_id_zzz")
	// GeoServer 2.27 sometimes 404s and sometimes returns an empty
	// FeatureCollection. Both are acceptable.
	if err != nil {
		if !errors.Is(err, geoserver.ErrNotFound) {
			t.Fatalf("expected ErrNotFound or empty result, got %v", err)
		}
	} else if gr != nil {
		t.Errorf("expected nil granule for missing id, got %+v", gr)
	}
}
