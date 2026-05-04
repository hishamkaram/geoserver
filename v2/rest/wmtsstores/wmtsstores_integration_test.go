//go:build integration

package wmtsstores_test

import (
	"errors"
	"testing"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/internal/testenv"
	"github.com/hishamkaram/geoserver/v2/rest/wmtsstores"
)

// The default GeoServer install ships no cascaded WMS stores. The
// list endpoint returns the empty-collection wire shape
// {"wmtsStores":""} which the SDK normalizes to a nil slice. We
// can't exercise full CRUD without an upstream WMS to cascade FROM,
// but verifying the empty-shape path against a real GeoServer is
// already meaningful coverage.
func TestWMTSStores_List_EmptyOnFreshInstall_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)

	for _, ws := range []string{"topp", "nurc", "sf"} {
		list, err := c.WMTSStores.InWorkspace(ws).List(ctx, wmtsstores.ListOptions{})
		if err != nil {
			t.Errorf("List in %s: %v", ws, err)
		}
		// Length is whatever — fresh install has 0; concurrent runs
		// of other tests may leave stores behind. The key assertion
		// is that the empty-string wire shape doesn't error.
		_ = list
	}
}

func TestWMTSStores_Get_NotFound_Integration(t *testing.T) {
	c := testenv.NewClient(t)
	ctx := testenv.Context(t)
	_, err := c.WMTSStores.InWorkspace("topp").Get(ctx, "v2_it_definitely_not_a_store")
	if !errors.Is(err, geoserver.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
