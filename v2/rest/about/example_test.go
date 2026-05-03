package about_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
)

// Example_ping issues a cheap liveness probe. Useful at boot time
// to fail fast if GeoServer isn't reachable, before any resource
// calls.
func Example_ping() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"),
		geoserver.WithTimeout(2*time.Second),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.About.Ping(ctx); err != nil {
		switch {
		case errors.Is(err, geoserver.ErrUnauthorized):
			fmt.Println("up but credentials rejected")
		default:
			fmt.Println("not reachable")
		}
	}
}

// Example_version reads the full component version document. Useful
// for diagnostics and for gating feature paths on a minimum GeoServer
// version.
func Example_version() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	info, err := c.About.Version(context.Background())
	if err != nil {
		return
	}
	for _, r := range info.Resource {
		fmt.Printf("%s = %s\n", r.Name, r.Version)
	}
}
