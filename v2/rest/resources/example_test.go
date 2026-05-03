package resources_test

import (
	"context"
	"fmt"
	"io"
	"strings"

	geoserver "github.com/hishamkaram/geoserver/v2"
)

// ExampleClient_Get streams a default style file from a fresh
// GeoServer install and prints the first line.
func ExampleClient_Get() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	body, err := c.Resources.Get(context.Background(), "styles/default_point.sld")
	if err != nil {
		return
	}
	defer body.Close()

	contents, _ := io.ReadAll(body)
	if i := strings.IndexByte(string(contents), '\n'); i > 0 {
		fmt.Println(string(contents[:i]))
	}
}

// ExampleClient_List dumps the names of every default style.
func ExampleClient_List() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	dir, err := c.Resources.List(context.Background(), "styles")
	if err != nil {
		return
	}
	for _, child := range dir.Children {
		fmt.Println(child.Name)
	}
}

// ExampleClient_Put uploads an FTL template that customizes WMS
// GetFeatureInfo HTML output for a layer.
func ExampleClient_Put() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	template := `<table>
<#list features as feature>
  <tr><td>${feature.name}</td><td>${feature.label}</td></tr>
</#list>
</table>
`
	_ = c.Resources.Put(context.Background(),
		"workspaces/topp/featuretypes_states/content.ftl",
		strings.NewReader(template),
		"text/plain")
}

// ExampleClient_Delete removes a single resource.
func ExampleClient_Delete() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Resources.Delete(context.Background(),
		"workspaces/topp/featuretypes_states/content.ftl")
}
