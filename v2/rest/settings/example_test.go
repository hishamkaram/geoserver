package settings_test

import (
	"context"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
)

// ExampleClient_Get fetches the singleton global-settings document.
func ExampleClient_Get() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	s, err := c.Settings.Get(context.Background())
	if err != nil {
		return
	}
	fmt.Printf("charset=%s decimals=%d\n",
		s.Global.Settings.Charset, s.Global.Settings.NumDecimals)
}

// ExampleClient_Update changes the verbose-exceptions flag and the
// feature-type cache size. Settings is a merge-patch endpoint —
// fetch, mutate, and put back.
func ExampleClient_Update() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	s, err := c.Settings.Get(context.Background())
	if err != nil {
		return
	}
	s.Global.Settings.VerboseExceptions = true
	s.Global.FeatureTypeCacheSize = 1000

	_ = c.Settings.Update(context.Background(), s)
}
