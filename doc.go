// Package geoserver is a Go client for the GeoServer REST API.
//
// Use [GetCatalog] (or [New], introduced in v1.1) to construct a *GeoServer
// instance, then call its methods to manage workspaces, datastores, layers,
// styles, coverages, and more.
//
// Concurrency: a *GeoServer constructed via [New] is safe for concurrent
// reads (i.e. concurrent calls to its methods); however, mutating its
// exported fields after construction is NOT safe. A v2 redesign with
// private fields and an immutable client is planned.
//
// Supported GeoServer versions: 2.27 (LTS) and 2.28 (current stable).
// GeoServer 3.0 support is tracked for v2.
package geoserver
