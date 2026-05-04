//go:build integration

package about_test

import (
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/about"
)

func TestAbout_Ping_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	if err := c.About.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestAbout_Version_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	v, err := c.About.Version(ctx)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if len(v.Resource) == 0 {
		t.Fatalf("expected at least one component, got empty list")
	}
	hasGeoServer := false
	for _, r := range v.Resource {
		if r.Name == "GeoServer" {
			hasGeoServer = true
			if r.Version == "" {
				t.Errorf("GeoServer component has empty Version")
			}
		}
	}
	if !hasGeoServer {
		t.Fatalf("no GeoServer component in Version response: %+v", v.Resource)
	}
}

func TestAbout_Manifests_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	// Unfiltered list: every bundle in the WAR.
	list, err := c.About.Manifests(ctx, about.ListManifestsOptions{})
	if err != nil {
		t.Fatalf("Manifests (no filter): %v", err)
	}
	if len(list) < 50 {
		t.Errorf("expected ≥50 bundles in the manifest, got %d", len(list))
	}
	for _, m := range list[:5] { // spot-check the first few
		if m.Name == "" {
			t.Errorf("manifest entry missing @name: %+v", m)
		}
	}

	// Filter that matches nothing — verify the empty-string
	// wire-quirk path returns nil cleanly (no error).
	empty, err := c.About.Manifests(ctx, about.ListManifestsOptions{
		Manifest: "definitely-no-such-bundle-zzz.*",
	})
	if err != nil {
		t.Fatalf("Manifests (no-match filter): %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected empty list for no-match filter, got %d", len(empty))
	}
}

func TestAbout_SystemStatus_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	metrics, err := c.About.SystemStatus(ctx)
	if err != nil {
		t.Fatalf("SystemStatus: %v", err)
	}
	if len(metrics) == 0 {
		t.Fatalf("expected non-empty metrics list")
	}
	// All recognized categories should be represented even when
	// individual values report "NOT AVAILABLE" (the test stack runs
	// on Linux without OSHI native libs in many cases).
	categories := map[string]bool{}
	for _, m := range metrics {
		categories[m.Category] = true
	}
	for _, c := range []string{"SYSTEM", "CPU", "MEMORY"} {
		if !categories[c] {
			t.Errorf("expected metric category %q, got %v", c, keysOf(categories))
		}
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
