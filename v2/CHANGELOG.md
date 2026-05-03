# Changelog — v2

All notable changes to `github.com/hishamkaram/geoserver/v2` are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). v2 ships independently of v1 (separate go.mod, separate `v2.x.y` tags).

## [Unreleased]

## [2.0.0-alpha.2] — 2026-05-03

Second alpha. Closes the last v1-parity gap (WMS GetCapabilities + system reload/cache reset), adds full pkg.go.dev godoc Example_* coverage across every sub-client, and refreshes the public-facing docs (READMEs, ROADMAP). No breaking changes from `alpha.1`; existing callers can `go get @v2.0.0-alpha.2` and recompile. Public API may still refine before `v2.0.0` — no production guarantees yet.

### Added

- **`v2/ows/wms/` package** — port of v1's `wms/` package. Same XML type tree (Capabilities, Service, Capability, Layer, Style, BoundingBox, …) so callers move with no shape changes. New free function `wms.ParseCapabilities(io.Reader)` (v2 idiom — `io.Reader` instead of `[]byte`; the deprecated v1 `ParseCapabilities` no-`E` variant that swallowed errors is gone). New sub-client `c.WMS` with `GetCapabilities(ctx, opts)` — global by default, `c.WMS.InWorkspace(ws)` for a workspace-scoped capabilities view. `GetCapabilitiesOptions` carries Version (default "1.1.1") and an optional UpdateSequence cache token.
- **`v2/rest/system/` package** — port of v1's `configuration.go`. New sub-client `c.System` with `Reload(ctx)` (`POST /rest/reload`) and `ResetCache(ctx)` (`POST /rest/reset`). The v1 typo'd method names (`ReloadConfigration` / `RestConfigrationCache`) are dropped; v2 uses the corrected spelling.
- **`internal/transport.DoXML`** — XML-decoding equivalent of `DoJSON` with a 32 MiB body cap (`DoJSON`'s 8 KiB cap is too small for real WMS capabilities documents). The error path still uses the 8 KiB cap so an oversized error body can't blow up.

### Added (tests)

- `v2/ows/wms/wms_test.go` — fixture-based parse tests, httptest tests for global + workspace scope, version + updatesequence query handling, and 404 / 500 → sentinel mapping.
- `v2/ows/wms/wms_integration_test.go` — real GeoServer assertions (capabilities document is non-empty for the global scope and a fresh empty workspace).
- `v2/rest/system/system_test.go` — httptest tests for Reload + ResetCache happy path, plus 401 / 403 / 500 → sentinel mapping.
- `v2/rest/system/system_integration_test.go` — real GeoServer Reload + ResetCache (idempotent, safe to repeat).
- Godoc `Example_*` for both new packages.

### Added (godoc)

- Godoc `Example_*` functions for the remaining 10 sub-clients (layers, layergroups, featuretypes, coverages, coveragestores, namespaces, settings, security, acl, about) so every public sub-client renders an inline usage demo on `pkg.go.dev`. Examples without `// Output:` comments compile-check via `go test` but don't execute, so they stay green without a live GeoServer (PR #62, post-`v2.0.0-alpha.1`).

### Changed (docs)

- `v2/README.md` banner: "v2 in development" → "`v2.0.0-alpha.1` is published" with a `go get` install one-liner and prerelease disclaimer (PR #61, post-`v2.0.0-alpha.1`).
- `v2/README.md` banner refreshed again for full v1 parity (WMS + system landed); the "WMS deferred" disclaimer is gone.
- Root `README.md` Roadmap mirrors the v2 banner state.
- `ROADMAP.md` checkpoints refreshed: `v2.0.0-alpha.1` marked tagged; new milestones for System endpoints, OWS clients (1/3 = wms), Migration guide, and v1 parity at master all marked complete; forward milestones added for `v2.0.0-alpha.2` retag, OWS clients 2/3 + 3/3 (wfs/wcs), `v2.0.0-beta.1`, `v2.0.0`.

### Branch protection (server-side)

- `master`'s required-status-checks list now includes `Unit tests v2 (Go 1.25)` (the original 6 contexts plus the v2 unit job — every PR must pass v2 unit tests before merging). No code change; recorded here for the maintainer audit trail.

## [2.0.0-alpha.1] — 2026-05-03

First public preview of v2. Surface is wide (workspaces, datastores, feature types, coverage stores, coverages, layers, layer groups, styles, namespaces, settings, about, security, ACL) and exercised by both unit and real-GeoServer integration suites on 2.27.4 LTS and 2.28.0 stable. Public API may still change before `v2.0.0` based on early-adopter feedback. No production guarantees.

### Added

- Initial scaffold. `*Client` immutable constructor (`New`) with functional options (`WithHTTPClient`, `WithTransport`, `WithBasicAuth`, `WithBearerToken`, `WithLogger`, `WithUserAgent`, `WithTimeout`, `WithHeader`).
- Single error type `*APIError` with package sentinels: `ErrBadRequest`, `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrMethodNotAllowed`, `ErrConflict`, `ErrUnsupportedMediaType`, `ErrRateLimited`, `ErrServerError`, `ErrBadGateway`, `ErrServiceUnavailable`, `ErrGatewayTimeout`. Match via `errors.Is` and `errors.As`.
- `internal/transport/` package: `BuildURL` (PathEscape + RawPath preservation, ported from v1.1's bug-fixed algorithm), `DoJSON` (single chokepoint for REST calls), `AuthRoundTripper` and `HeaderRoundTripper` (auth and User-Agent attached via the transport stack rather than per-request).
- `rest/workspaces/` reference sub-client: `List`, `Iter` (`iter.Seq2`), `Get`, `Create`, `Update`, `Delete`. httptest unit tests cover 2xx happy paths and 401/404/409/500 sentinel mapping plus the URL-escaping regression guard.
- `rest/datastores/` sub-client (workspace-scoped): `c.Datastores.InWorkspace(ws)` returns a `*WorkspaceClient` exposing `List`, `Iter`, `Get`, `Create`, `Update`, `Delete`. Convenience connectors `PostGIS` and `JNDI` produce the wire-format payload; arbitrary drivers can be supplied via `Raw(Datastore)`. This is the reference for every other workspace-scoped resource (feature types, coverages, layers, …).
- `rest/featuretypes/` sub-client (workspace+datastore-scoped): `c.FeatureTypes.InWorkspace(ws).InDatastore(ds)` returns a `*DatastoreClient` exposing `List`, `Iter`, `Get`, `Create`, `Update`, `Delete`, plus `Discover` (which lists tables in the underlying datastore not yet published — `DiscoverAvailable` / `DiscoverAvailableWithGeometry` / `DiscoverAll`). The `CRS` type carries forward v1's custom marshal/unmarshal handling both the object form (`{"@class":"projected","$":"EPSG:4326"}`) and the bare-string form. Reference for the 2-level hierarchical pattern future v2 resources (coverages under coverage stores) will follow.
- `rest/coveragestores/` sub-client (workspace-scoped): the raster-side analogue of datastores. `c.CoverageStores.InWorkspace(ws)` returns a `*WorkspaceClient` exposing `List`, `Iter`, `Get`, `Create`, `Update`, `Delete`. Coverage stores are simpler than datastores — no connection-parameters complexity, just `URL` + `Type` (e.g., `GeoTIFF`, `ImageMosaic`).
- `rest/coverages/` sub-client (workspace+coverage-store-scoped): `c.Coverages.InWorkspace(ws).InCoverageStore(cs)` returns a `*CoverageStoreClient` exposing `List`, `Iter`, `Get`, `Create`, `Update`, `Delete`, plus `Discover` (lists native coverages in the store; `DiscoverAvailable` / `DiscoverAll`, default `DiscoverAll` since most coverage stores expose a single coverage that's already configured). Carries the same `CRS` / `BoundingBox` / `Keywords` types as featuretypes — these are duplicated for now and may be extracted into a shared package in a follow-up PR.
- `rest/layers/` sub-client (workspace-scoped): `c.Layers.InWorkspace(ws)` returns a `*WorkspaceClient` exposing `List`, `Iter`, `Get`, `Update`, `Delete`. There is no `Create` — layers are auto-created as a side-effect of publishing a feature type or coverage; manage them through this client after publish. `Layer` carries `DefaultStyle`, `Styles`, `Resource` (back-reference to feature type / coverage), and `Attribution`.
- `rest/layergroups/` sub-client (workspace-scoped): `c.LayerGroups.InWorkspace(ws)` returns a `*WorkspaceClient` exposing `List`, `Iter`, `Get`, `Create`, `Update`, `Delete`. The `Published` and `Styles` types carry custom `UnmarshalJSON` to handle GeoServer's wire-format quirks: a single published layer comes back as an object (not a 1-element array), and the per-member `style` array is mixed string-or-object (string for "default style", object for explicit assignment). Both shapes round-trip without panicking.
- `rest/styles/` sub-client (dual-scope): `c.Styles` is the global-scope client; `c.Styles.InWorkspace(ws)` returns a fresh workspace-scoped client (the original is unaffected). Surface: `List`, `Iter`, `Get`, `Create`, `UploadSLD`, `Update`, `Delete`. Carries forward GeoServer's wire-format quirks: empty `{"styles":""}` collection accepted as nil; workspace-scoped `POST /styles` automatically uses `Accept: */*` to dodge the "No such style handler: format = application/json" 500. `UploadSLD` sends the SLD body via PUT with content-type `application/vnd.ogc.sld+xml` (overridable via `UploadOptions.Format` for SE 1.1 / GeoCSS).
- `internal/transport.DoRaw` for non-JSON request bodies (powers `UploadSLD`; reusable for future shapefile-zip and GeoTIFF uploads). The `Request` struct gains `RawBody io.Reader` + `ContentType string` fields used by `buildHTTPRequest`. `coreAdapter` exposes a matching `DoRaw` method for sub-clients to use through the resource-package `Core` interface.
- `rest/namespaces/` sub-client (flat global): `c.Namespaces` exposes `List`, `Iter`, `Get`, `Create`, `Update`, `Delete`. Mirrors the workspace surface, paired with the auto-created namespace each workspace gets.
- `rest/settings/` sub-client (singleton): `c.Settings` exposes `Get` / `Update` against `/rest/settings`. Full nested document — `ServiceSettings` (charset / online-resource / contact), `JAI` tunables (with `JAIExt` operations), `CoverageAccess` thread-pool tuning. `Contact` and `JAIExt` types tolerate GeoServer's empty-string wire form (`"contact":""`, `"jaiext":""`) via custom `UnmarshalJSON`.
- `rest/about/` sub-client (read-only health): `c.About.Ping` issues a cheap liveness probe; `c.About.Version` returns the full component version document (GeoServer core, GeoTools, GeoWebCache build timestamps + git revisions).
- `rest/security/` sub-client (users + groups + roles): `c.Security.Users()` / `c.Security.UsersInService(name)` for the user/group-service-scoped users client; same for groups; `c.Security.Roles` (always global) for role CRUD plus user-role assignment (`AssignToUser` / `UnassignFromUser` / `ForUser`). Decodes both GeoServer 2.28+ (`{"roles":[]}`, `{"groups":[]}`) and older 2.x (`{"roleNames":[]}`, `{"groupNames":[]}`) response shapes via a unified `nameListResponse` helper. Empty service name resolves to `DefaultService` ("default").
- `rest/acl/` sub-client (layer ACLs): `c.ACL.Layers()` returns a `*LayersClient` exposing `List`, `Add`, `Delete`. The typed `Rule{Workspace, Layer, Operation, Roles}` round-trips with the GeoServer wire format `"workspace.layer.op" → "role1,role2"` via `Rule.Encode` / `DecodeRule`. Empty fields default to `*` (any); empty Roles encodes to `*`. Operation values are typed (`OpRead` / `OpWrite` / `OpAdmin`). Service-level and catalog-level ACL endpoints can be added under `c.ACL.Services()` / `c.ACL.Catalog()` in follow-up PRs without breaking the existing surface.

### Refactored

- Shared GIS wire-format types (`CRS` with custom Marshal/Unmarshal, `BoundingBox`, `NativeBoundingBox`, `LatLonBoundingBox`, `Keywords`) extracted to `internal/wire`. `featuretypes.CRS` and `coverages.CRS` (and the related types) are now type aliases for `wire.X`, sharing underlying type identity — values can flow between the two sub-packages without conversion. Public API surface unchanged; users keep accessing the types through the sub-package they already use.

### Fixed

- **`about.Resource.Version`** decodes both string ("2.28.0") and number (`34`) JSON wire forms — GeoTools reports a bare integer in some releases.
- **`layergroups.Styles`** unmarshal handles all four GeoServer wire shapes for the per-member style list: bare string (no overrides), `{"style":""}` (single member, default), `{"style":{...}}` (single member, explicit), and `{"style":[...]}` (multi-member array). Previously only the array form decoded correctly; single-member groups failed with a parse error.
- **`namespaces.Namespace`** unmarshal accepts both wire shapes — the list endpoint returns `{"name": ..., "href": ...}` per entry while the detail endpoint returns `{"prefix": ..., "uri": ..., "isolated": ...}`. The list-shape `name` is coerced into `Prefix` so list results are usable.
- **`datastores.List` empty-collection** — accepts the bare-string `{"dataStores":""}` GeoServer 2.28+ returns for an empty collection (carries forward the v1 issue #22 fix into v2). The same envelope handling is applied to `featuretypes` Discover where applicable.

### Added (tests)

- **Integration test suite** under `v2/rest/<resource>/<resource>_integration_test.go` (build tag `//go:build integration`). Covers workspaces, datastores, featuretypes, coveragestores, coverages, layers, layergroups, styles, namespaces, settings, security, ACL, and about against a real GeoServer 2.27 / 2.28 + PostGIS compose stack. Shared helpers in `internal/testenv` build a v2 client from env vars and synthesize unique resource names per-test for parallel safety.
- New CI step `Run v2 integration tests` runs the suite alongside the v1 integration tests against both `GeoServer 2.27.4` and `GeoServer 2.28.0` matrix legs; both legs must pass for PR merge.
- New Makefile target `make test-v2-integration` for local runs against `make compose-up`.

### Added (godoc)

- `v2/example_test.go`, `v2/rest/workspaces/example_test.go`, `v2/rest/datastores/example_test.go`, `v2/rest/styles/example_test.go` — godoc `Example_*` functions that render under each public symbol on `pkg.go.dev`. Examples without `// Output:` comments compile-check via `go test` but don't execute, so they stay green without a live GeoServer.
