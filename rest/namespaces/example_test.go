package namespaces_test

import (
	"context"
	"errors"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/namespaces"
)

// ExampleClient_Get fetches a namespace by prefix. Each workspace gets
// an auto-created namespace with the same prefix; explicit Create is
// only needed for namespaces that don't follow that pattern.
func ExampleClient_Get() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	ns, err := c.Namespaces.Get(context.Background(), "topp")
	if errors.Is(err, geoserver.ErrNotFound) {
		return
	}
	if err == nil {
		fmt.Printf("%s -> %s\n", ns.Prefix, ns.URI)
	}
}

// ExampleClient_Iter ranges over every namespace.
func ExampleClient_Iter() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	for ns, err := range c.Namespaces.Iter(context.Background(), namespaces.ListOptions{}) {
		if err != nil {
			return
		}
		fmt.Println(ns.Prefix)
	}
}

// ExampleClient_Create registers a namespace with a non-standard URI
// (e.g., for an external schema or OGC service binding).
func ExampleClient_Create() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.Namespaces.Create(context.Background(), &namespaces.Namespace{
		Prefix: "acme",
		URI:    "https://schemas.acme.example/v1",
	})
}
