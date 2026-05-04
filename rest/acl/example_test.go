package acl_test

import (
	"context"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/acl"
)

// ExampleClient_Layers returns the layer-ACL sub-client.
func ExampleClient_Layers() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.ACL.Layers()
}

// ExampleServicesClient_Add grants the USER role permission to invoke
// WMS GetMap. Use service "*" / operation "*" for global rules.
func ExampleServicesClient_Add() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.ACL.Services().Add(context.Background(), acl.ServiceRule{
		Service:   "wms",
		Operation: "GetMap",
		Roles:     []string{"ROLE_USER"},
	})
}

// ExampleRESTClient_Add restricts mutating REST endpoints under
// /rest/workspaces/** to admins only. Methods are HTTP verbs (or "*"
// for any verb).
func ExampleRESTClient_Add() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.ACL.REST().Add(context.Background(), acl.RESTRule{
		Pattern: "/rest/workspaces/**",
		Methods: []string{"POST", "PUT", "DELETE"},
		Roles:   []string{"ROLE_ADMIN"},
	})
}

// ExampleCatalogClient_Update flips GeoServer's catalog mode to
// CHALLENGE — secured resources stay visible in capabilities but
// every direct access without privileges returns a 401 challenge.
func ExampleCatalogClient_Update() {
	c, _ := geoserver.New("http://localhost:8080/geoserver",
		geoserver.WithBasicAuth("admin", "geoserver"))

	_ = c.ACL.Catalog().Update(context.Background(), acl.CatalogModeChallenge)
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
