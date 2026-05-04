package templates_test

import (
	"context"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	_ "github.com/hishamkaram/geoserver/v2/rest/templates"
)

// ExampleClient_PutString uploads an FTL template that customizes
// WMS GetFeatureInfo HTML output for one feature type. The same
// API works at every scope (global / workspace / datastore / feature
// type / coverage store / coverage).
func ExampleClient_PutString() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	template := `<table>
<#list features as feature>
  <tr><td>${feature.STATE_NAME.value}</td><td>${feature.PERSONS.value}</td></tr>
</#list>
</table>`

	_ = c.Templates.InWorkspace("topp").
		InDatastore("states_pg").
		InFeatureType("states").
		PutString(context.Background(), "content", template)
}

// ExampleClient_List enumerates the templates registered at a scope.
// At every scope the wire shape is the same (a TemplateInfos
// envelope with one TemplateInfo per template).
func ExampleClient_List() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	list, err := c.Templates.InWorkspace("topp").List(context.Background())
	if err != nil {
		return
	}
	for _, ref := range list {
		fmt.Println(ref.Name)
	}
}

// ExampleClient_Delete removes a template at the most-specific
// scope. GeoServer's runtime template lookup then falls back to
// the next-broader scope (e.g. workspace, then global) automatically.
func ExampleClient_Delete() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Templates.InWorkspace("topp").
		InDatastore("states_pg").
		InFeatureType("states").
		Delete(context.Background(), "content")
}
