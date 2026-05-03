//go:build integration

package coverages_test

import (
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/coverages"
)

// Read against the default-install nurc/arcGridSample fixture.
const (
	nurcWorkspace = "nurc"
	nurcStore     = "arcGridSample"
)

func TestCoverages_ReadDefaults_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	cov := c.Coverages.InWorkspace(nurcWorkspace).InCoverageStore(nurcStore)

	all, err := cov.List(ctx, coverages.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) == 0 {
		t.Fatalf("expected at least one coverage in nurc/arcGridSample, got empty")
	}

	// Pick one and Get its full document to exercise the CRS unmarshal.
	first := all[0]
	got, err := cov.Get(ctx, first.Name)
	if err != nil {
		t.Fatalf("Get %q: %v", first.Name, err)
	}
	if got.Name != first.Name {
		t.Fatalf("Name mismatch: got %q want %q", got.Name, first.Name)
	}
	// Native CRS or LatLonBoundingBox should be populated for a real
	// raster — exercises the wire.CRS Unmarshal both shapes path.
	if got.NativeCRS == nil && got.LatLonBoundingBox == nil {
		t.Errorf("expected NativeCRS or LatLonBoundingBox on a real coverage; got %+v", got)
	}
}

func TestCoverages_Discover_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Default mode (DiscoverAll) lists configured + available.
	got, err := c.Coverages.InWorkspace(nurcWorkspace).InCoverageStore(nurcStore).
		Discover(ctx, coverages.DiscoverOptions{})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected Discover(All) to list at least the configured coverage")
	}
}
