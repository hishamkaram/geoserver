//go:build integration

package coveragestores_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/coveragestores"
)

// The default GeoServer install ships with a "nurc" workspace containing
// the "arcGridSample" coverage store. Reads against it verify the
// CRUD path against real raster data without our test having to seed a
// fixture.
const (
	nurcWorkspace = "nurc"
	nurcStore     = "arcGridSample"
)

func TestCoverageStores_ReadDefaults_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	cs := c.CoverageStores.InWorkspace(nurcWorkspace)

	all, err := cs.List(ctx, coveragestores.ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) == 0 {
		t.Fatalf("expected default nurc workspace to ship with coverage stores; got empty")
	}

	got, err := cs.Get(ctx, nurcStore)
	if err != nil {
		t.Fatalf("Get %q: %v", nurcStore, err)
	}
	if got.Name != nurcStore {
		t.Fatalf("Name = %q", got.Name)
	}
	if got.Type == "" {
		t.Errorf("Type empty: %+v", got)
	}
}

func TestCoverageStores_NotFound_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	_, err := c.CoverageStores.InWorkspace(nurcWorkspace).Get(ctx, "absolutely-not-a-coverage-store")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
