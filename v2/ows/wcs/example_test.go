package wcs_test

import (
	"context"
	"fmt"
	"strings"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/ows/wcs"
)

// ExampleClient_GetCapabilities fetches the global WCS capabilities
// document and prints every advertised coverage.
func ExampleClient_GetCapabilities() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	caps, err := c.WCS.GetCapabilities(context.Background(), wcs.GetCapabilitiesOptions{})
	if err != nil {
		return
	}
	fmt.Printf("WCS %s — %s\n", caps.Version, caps.ServiceIdentification.Title)
	for _, cov := range caps.Contents.CoverageSummary {
		fmt.Printf("  - %s\n", cov.CoverageID)
	}
}

// ExampleClient_InWorkspace returns a workspace-scoped capabilities
// view — only coverages under that workspace appear.
func ExampleClient_InWorkspace() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_, _ = c.WCS.InWorkspace("nurc").GetCapabilities(context.Background(),
		wcs.GetCapabilitiesOptions{})
}

// ExampleParseCapabilities decodes a capabilities document fetched
// out-of-band.
func ExampleParseCapabilities() {
	body := strings.NewReader(`<?xml version="1.0"?>
<wcs:Capabilities version="2.0.1"
    xmlns:wcs="http://www.opengis.net/wcs/2.0"
    xmlns:ows="http://www.opengis.net/ows/2.0">
  <ows:ServiceIdentification><ows:Title>Demo</ows:Title></ows:ServiceIdentification>
</wcs:Capabilities>`)

	caps, err := wcs.ParseCapabilities(body)
	if err != nil {
		return
	}
	fmt.Println(caps.ServiceIdentification.Title)
	// Output: Demo
}
