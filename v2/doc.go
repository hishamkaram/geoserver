// Package geoserver is a Go client for the GeoServer REST API (v2 line).
//
// # Status
//
// v2 has full v1 feature parity at master and exceeds it. The
// package covers every "everyone needs it" endpoint group identified
// in the gap analysis: catalog (workspaces, datastores, feature
// types, coverage stores, coverages, layers, layer groups, styles,
// namespaces), settings, system (reload + cache reset), about,
// security (users / groups / roles + layer ACL), per-service OWS
// configuration (WMS / WFS / WCS / WMTS), file-upload publishing on
// stores, layer–style associations, GeoWebCache (per-layer cache
// config + seed/reseed/truncate + diskquota), the Importer
// extension (batch ingest), and the OWS read-only trio
// (GetCapabilities + DescribeFeatureType + DescribeCoverage).
//
// Public API is frozen for review through the v2.0.0-beta.* line —
// breaking changes will not land without a strong reason. The latest
// published preview tag is on the v2.0.0-beta.* line; see the v2 README
// for the current pin. Until v2 reaches v2.0.0, the v1 line remains
// the recommended import for production:
//
//	import "github.com/hishamkaram/geoserver"        // v1: stable, full surface
//	import "github.com/hishamkaram/geoserver/v2"     // v2: preview
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
//   - Sub-client pattern. Public fields like (*Client).Workspaces and
//     (*Client).Datastores expose typed per-resource clients with
//     consistent List / Get / Create / Update / Delete / Iter shapes.
//     Hierarchical resources fluently chain through scope —
//     c.Datastores.InWorkspace("topp"), c.FeatureTypes.InWorkspace(ws).InDatastore(ds).
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
//	wss, err := c.Workspaces.List(ctx, workspaces.ListOptions{})
//	if errors.Is(err, geoserver.ErrUnauthorized) {
//	    // handle bad credentials
//	}
package geoserver
