package acl_test

import (
	"context"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/acl"
)

// ExampleClient_Layers returns the layer-ACL sub-client. Future
// service-level and catalog-level ACL endpoints will live alongside
// (e.g., c.ACL.Services(), c.ACL.Catalog()).
func ExampleClient_Layers() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.ACL.Layers()
}

// ExampleLayersClient_Add grants the EDITOR role write access to
// every layer in the topp workspace. Empty Layer defaults to "*"
// (any layer); empty Roles encodes as "*" (any role) — be deliberate.
func ExampleLayersClient_Add() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.ACL.Layers().Add(context.Background(), acl.Rule{
		Workspace: "topp",
		Layer:     "*",
		Operation: acl.OpWrite,
		Roles:     []string{"EDITOR"},
	})
}

// ExampleLayersClient_List dumps every layer ACL rule on the server.
func ExampleLayersClient_List() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	rules, err := c.ACL.Layers().List(context.Background(), acl.ListOptions{})
	if err != nil {
		return
	}
	for _, r := range rules {
		key, roles := r.Encode()
		fmt.Printf("%s -> %s\n", key, roles)
	}
}
