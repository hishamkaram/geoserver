package services_test

import (
	"context"
	"errors"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/services"
)

// ExampleClient_WMS reads the global WMS settings and tightens the
// rendering-time cap so a slow style can't monopolize the worker pool.
func ExampleClient_WMS() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	wms, err := c.Services.WMS().Get(context.Background())
	if err != nil {
		return
	}
	wms.MaxRenderingTime = 30 // seconds
	_ = c.Services.WMS().Update(context.Background(), wms)
}

// ExampleWFSClient_InWorkspace caps WFS GetFeature responses for one
// tenant. The per-workspace override doesn't affect the global config
// or other workspaces.
func ExampleWFSClient_InWorkspace() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Services.WFS().InWorkspace("topp").Update(context.Background(),
		&services.WFSSettings{
			ServiceInfo:  services.ServiceInfo{Enabled: true, Title: "topp WFS"},
			MaxFeatures:  10000,
			ServiceLevel: "BASIC",
		})
}

// ExampleWMSWorkspaceClient_Delete removes the per-workspace WMS
// override so the workspace falls back to the global configuration.
func ExampleWMSWorkspaceClient_Delete() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	err := c.Services.WMS().InWorkspace("topp").Delete(context.Background())
	if errors.Is(err, geoserver.ErrNotFound) {
		fmt.Println("no override existed")
	}
}

// ExampleWCSClient sets WCS memory caps to keep one bad request from
// OOM'ing the JVM. Memory values are integers in kilobytes despite
// the upstream OpenAPI YAML's documented (boolean) shape — see the
// WCSSettings doc.
func ExampleWCSClient() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Services.WCS().Update(context.Background(), &services.WCSSettings{
		ServiceInfo:     services.ServiceInfo{Enabled: true},
		MaxInputMemory:  1024 * 1024, // 1 GiB
		MaxOutputMemory: 2048 * 1024, // 2 GiB
	})
}
