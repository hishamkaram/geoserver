package styles_test

import (
	"context"
	"os"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/styles"
)

// ExampleClient_InWorkspace returns a workspace-scoped styles client.
// Without InWorkspace the client operates against the global
// /rest/styles endpoint.
func ExampleClient_InWorkspace() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	global := c.Styles                     // operates on /rest/styles
	scoped := c.Styles.InWorkspace("topp") // operates on /rest/workspaces/topp/styles

	_, _ = global, scoped
}

// ExampleClient_Create registers a workspace-scoped style metadata
// document. Follow with [Client.UploadSLD] to attach the SLD body.
//
// The workspace-scoped POST endpoint requires Accept: */* (a GeoServer
// REST quirk) — v2 applies this automatically; the global path uses
// the default Accept.
func ExampleClient_Create() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Styles.InWorkspace("topp").Create(context.Background(), &styles.Style{
		Name: "my-polygon",
	})
}

// ExampleClient_UploadSLD uploads or replaces the SLD body for an
// existing style. Default Content-Type is "application/vnd.ogc.sld+xml"
// (SLD 1.0); override via [styles.UploadOptions.Format] for SE 1.1 or
// GeoCSS.
func ExampleClient_UploadSLD() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	f, err := os.Open("polygon.sld")
	if err != nil {
		return
	}
	defer f.Close()

	_ = c.Styles.InWorkspace("topp").UploadSLD(context.Background(),
		"my-polygon", f, styles.UploadOptions{})
}
