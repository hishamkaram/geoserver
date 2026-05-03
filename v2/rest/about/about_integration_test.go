//go:build integration

package about_test

import (
	"testing"

	"github.com/hishamkaram/geoserver/v2/internal/testenv"
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
