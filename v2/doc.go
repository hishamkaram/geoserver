// Package geoserver is a Go client for the GeoServer REST API (v2 line).
//
// # Status
//
// v2 is in development. Only the Workspaces resource is currently
// implemented as a reference; other resources (datastores, styles, layers,
// layergroups, coverages, namespaces, settings, security, ACL, about,
// capabilities, feature types) port in subsequent PRs following the same
// pattern. Until v2 reaches v2.0.0, the v1 line remains the recommended
// import for production use:
//
//	import "github.com/hishamkaram/geoserver"        // v1: stable, full surface
//	import "github.com/hishamkaram/geoserver/v2"     // v2: in development
//
// See ../ROADMAP.md for the v2 milestones and ../docs/migration-v1-to-v2.md
// for the v1 → v2 migration guide.
//
// # Design tenets
//
// v2 is a clean redesign that breaks v1's monolithic *GeoServer surface
// into a sub-client per resource:
//
//   - Immutable [*Client]. All fields private. Configured via functional
//     options at construction; no post-construction mutation. Concurrent
//     use is safe.
//   - Mandatory [context.Context] as first arg on every public method. No
//     Background shims, no twin pairs.
//   - Sub-client pattern. (*Client).Workspaces returns a per-resource
//     client whose methods follow a consistent List / Get / Exists /
//     Create / Update / Delete / Iter shape.
//   - Single error type. Every HTTP error is a [*APIError] wrapping one
//     of the package sentinels ([ErrNotFound], [ErrConflict], …) so
//     errors.Is and errors.As are the supported match styles.
//   - Auth via http.RoundTripper. Basic / bearer auth attaches to the
//     transport layer once at construction; per-call paths don't
//     re-authenticate. Custom RoundTrippers (OpenTelemetry, retry libs,
//     Vault-rotated creds) layer naturally.
//   - Streaming uploads. Resources that accept binary payloads take
//     [io.Reader] and never slurp into memory.
//   - Pagination via [iter.Seq2]. List endpoints expose Iter for
//     range-over-func; non-paginating endpoints fall back to a
//     single-page Seq2.
//   - Zero runtime third-party dependencies. stdlib net/http,
//     encoding/json, encoding/xml, log/slog, context, iter only.
//     Test deps allowed.
//
// # Quick start
//
//	c, err := geoserver.New("http://localhost:8080/geoserver/",
//	    geoserver.WithBasicAuth("admin", "geoserver"),
//	    geoserver.WithTimeout(10*time.Second),
//	)
//	if err != nil {
//	    return err
//	}
//
//	ctx := context.Background()
//	workspaces, err := c.Workspaces.List(ctx, workspaces.ListOptions{})
//	if errors.Is(err, geoserver.ErrUnauthorized) {
//	    // handle bad credentials
//	}
package geoserver
