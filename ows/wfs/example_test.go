package wfs_test

import (
	"context"
	"fmt"
	"strings"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/ows/wfs"
)

// ExampleClient_GetCapabilities fetches the global WFS capabilities
// document and prints every advertised feature type.
func ExampleClient_GetCapabilities() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	caps, err := c.WFS.GetCapabilities(context.Background(), wfs.GetCapabilitiesOptions{})
	if err != nil {
		return
	}
	fmt.Printf("WFS %s — %s\n", caps.Version, caps.ServiceIdentification.Title)
	for _, ft := range caps.FeatureTypeList.FeatureType {
		fmt.Printf("  - %s (%s)\n", ft.Name, ft.DefaultSRS)
	}
}

// ExampleClient_InWorkspace returns a workspace-scoped capabilities
// view — only feature types under that workspace appear.
func ExampleClient_InWorkspace() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_, _ = c.WFS.InWorkspace("topp").GetCapabilities(context.Background(),
		wfs.GetCapabilitiesOptions{Version: "1.1.0"})
}

// ExampleClient_DescribeFeatureType fetches the XSD schema for a
// published feature type and prints each attribute's name and type.
func ExampleClient_DescribeFeatureType() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	schema, err := c.WFS.DescribeFeatureType(context.Background(),
		wfs.DescribeFeatureTypeOptions{TypeNames: []string{"topp:states"}})
	if err != nil {
		return
	}
	for _, attr := range schema.Attributes("") {
		fmt.Printf("  %s : %s\n", attr.Name, attr.Type)
	}
}

// ExampleParseCapabilities decodes a capabilities document fetched
// out-of-band — useful for parsing a saved fixture or a body from a
// custom transport.
func ExampleParseCapabilities() {
	body := strings.NewReader(`<?xml version="1.0"?>
<wfs:WFS_Capabilities version="2.0.0"
    xmlns:wfs="http://www.opengis.net/wfs/2.0"
    xmlns:ows="http://www.opengis.net/ows/1.1">
  <ows:ServiceIdentification><ows:Title>Demo</ows:Title></ows:ServiceIdentification>
</wfs:WFS_Capabilities>`)

	caps, err := wfs.ParseCapabilities(body)
	if err != nil {
		return
	}
	fmt.Println(caps.ServiceIdentification.Title)
	// Output: Demo
}
