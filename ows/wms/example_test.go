package wms_test

import (
	"context"
	"fmt"
	"strings"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/ows/wms"
)

// ExampleClient_GetCapabilities fetches the global WMS capabilities
// document and prints every advertised top-level layer.
func ExampleClient_GetCapabilities() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	caps, err := c.WMS.GetCapabilities(context.Background(), wms.GetCapabilitiesOptions{})
	if err != nil {
		return
	}
	fmt.Printf("WMS %s — %s\n", caps.Version, caps.Service.Title)
	for _, layer := range caps.Capability.Layer.Layer {
		fmt.Printf("  - %s\n", layer.Title)
	}
}

// ExampleClient_InWorkspace returns a workspace-scoped capabilities
// view — only layers under that workspace appear in the response.
func ExampleClient_InWorkspace() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_, _ = c.WMS.InWorkspace("topp").GetCapabilities(context.Background(),
		wms.GetCapabilitiesOptions{Version: "1.3.0"})
}

// ExampleParseCapabilities decodes a capabilities document fetched
// out-of-band — useful for parsing a saved fixture or a body from a
// custom transport.
func ExampleParseCapabilities() {
	body := strings.NewReader(`<?xml version="1.0"?>
<WMT_MS_Capabilities version="1.1.1">
  <Service><Title>Demo</Title></Service>
  <Capability/>
</WMT_MS_Capabilities>`)

	caps, err := wms.ParseCapabilities(body)
	if err != nil {
		return
	}
	fmt.Println(caps.Service.Title)
	// Output: Demo
}
