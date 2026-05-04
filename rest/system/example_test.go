package system_test

import (
	"context"
	"errors"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
)

// Example_reload reloads GeoServer's catalog and configuration from
// disk. Use after an out-of-band edit to the data directory or after
// rolling out new plugins.
func Example_reload() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	if err := c.System.Reload(context.Background()); err != nil {
		switch {
		case errors.Is(err, geoserver.ErrUnauthorized):
			fmt.Println("auth required")
		case errors.Is(err, geoserver.ErrForbidden):
			fmt.Println("admin role required")
		}
	}
}

// Example_resetCache invalidates store / raster / schema caches
// without a full Reload — useful after an upstream schema migration
// when configuration on disk hasn't changed.
func Example_resetCache() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.System.ResetCache(context.Background())
}
