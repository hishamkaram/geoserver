package gwc_test

import (
	"context"
	"errors"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/gwc"
)

// ExampleClient_Layers lists every layer GeoWebCache is configured
// to cache.
func ExampleClient_Layers() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	names, err := c.GWC.Layers().List(context.Background())
	if err != nil {
		return
	}
	for _, n := range names {
		fmt.Println(n)
	}
}

// ExampleLayersClient_Get reads the per-layer cache configuration —
// gridsets, MIME types, expire times, parameter filters.
func ExampleLayersClient_Get() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	cfg, err := c.GWC.Layers().Get(context.Background(), "topp:states")
	if err != nil {
		return
	}
	fmt.Printf("%s enabled=%v\n", cfg.Name, cfg.Enabled)
	if cfg.MimeFormats != nil {
		for _, m := range cfg.MimeFormats.String {
			fmt.Println(" ", m)
		}
	}
}

// ExampleSeedClient_Submit invalidates the cached tiles for a layer
// after a data update — the daily-driver workflow for any production
// deployment serving WMS via tiles.
//
// The call is asynchronous: Submit returns immediately and the task
// runs in the background. Use [SeedClient.Status] to poll progress.
func ExampleSeedClient_Submit() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	err := c.GWC.Seed().Submit(context.Background(), "topp:states", &gwc.SeedRequest{
		SRS:         gwc.SRS{Number: 4326},
		ZoomStart:   0,
		ZoomStop:    8,
		Format:      "image/png",
		Type:        gwc.OpTruncate,
		ThreadCount: 2,
		GridSetID:   "EPSG:4326",
		Bounds: &gwc.Bounds{Coords: gwc.BoundsCoords{
			Double: []float64{-180, -90, 180, 90},
		}},
	})
	if errors.Is(err, geoserver.ErrNotFound) {
		fmt.Println("layer doesn't exist")
	}
}

// ExampleSeedClient_StatusAll polls progress on every running
// seed/reseed/truncate task across all layers.
func ExampleSeedClient_StatusAll() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	st, err := c.GWC.Seed().StatusAll(context.Background())
	if err != nil {
		return
	}
	for _, t := range st.Tasks {
		fmt.Printf("task %d: %d/%d tiles, %ds left, status=%s\n",
			t.TaskID, t.TilesProcessed, t.TotalTiles, t.RemainingSeconds, t.Status)
	}
}

// ExampleDiskQuotaClient_Get reads the disk-quota policy controlling
// LFU/LRU eviction and the maximum disk usage for the tile cache.
func ExampleDiskQuotaClient_Get() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	dq, err := c.GWC.DiskQuota().Get(context.Background())
	if err != nil {
		return
	}
	fmt.Printf("enabled=%v policy=%s", dq.Enabled, dq.GlobalExpirationPolicyName)
	if dq.GlobalQuota != nil {
		fmt.Printf(" cap=%d bytes", dq.GlobalQuota.Bytes)
	}
	fmt.Println()
}
