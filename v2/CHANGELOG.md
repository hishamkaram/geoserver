# Changelog — v2

All notable changes to `github.com/hishamkaram/geoserver/v2` are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). v2 ships independently of v1 (separate go.mod, separate `v2.x.y` tags).

## [Unreleased]

## [2.0.0-beta.3] — 2026-05-04

Third beta. Adds four longer-tail surfaces on top of beta.2's tier-2-complete state — fonts, master/self password rotation, GWC global config + gridsets + mass-truncate, and the monitoring (request audit log) extension. The dev/test docker image now bakes the `gs-monitor` plugin in alongside `gs-importer` so CI exercises the full audit-log surface against real GeoServer 2.27.4 LTS and 2.28.0 stable. No breaking changes from `beta.2`; existing callers can `go get @v2.0.0-beta.3` and recompile. Public API stays frozen for review through the beta line.

### Documented — WFS XSLT transforms (`c.WFSTransforms`) extension status

The `gs-xslt-wfs` extension that the beta.2 `c.WFSTransforms` surface targets was removed from upstream GeoServer in 2.24 and is NOT shipped — neither in stable nor in community / SNAPSHOT channels — for any 2.24+ release, including the supported 2.27 LTS and 2.28 stable lines. The upstream OpenAPI YAML still documents the surface. The package godoc now flags this clearly: the surface is preserved for custom builds and pre-2.24 deployments only; CI verifies the "extension absent → ErrNotFound" path.

### Added — Fonts list

Closes the fonts longer-tail item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). Sanity-check before publishing styles that reference specific fonts — typos would otherwise surface as silent label-rendering fallbacks.

- **`c.Fonts.List(ctx)`** at `/rest/fonts` — returns the list of font families the JVM exposes to GeoServer's SLD labelling pipeline as `[]string`. The result reflects whatever is on the server's classpath at call time (system fonts plus anything dropped into the data directory's `styles/` subdirectory).

### Added — Monitoring (request audit log)

Closes the monitoring longer-tail item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). Read-only access to GeoServer's request audit log, served by the `gs-monitor` extension. The dev/test docker image now bakes the extension in so CI exercises the surface against a real server.

- **`c.Monitor.List(ctx, ListOptions)`** at `/rest/monitor/requests.csv` — returns `[]Request` decoded from the CSV wire form. `Request` covers the daily-driver audit columns (ID / Path / Service / Operation / HTTPMethod / StartTime / EndTime / TotalTime / Status / ResponseStatus / ResponseLength / RemoteAddr / RemoteUser / Resources, …).
- **`c.Monitor.ListRaw(ctx, ListOptions)`** returns the raw CSV `io.ReadCloser` for streaming-pipeline use cases or to access fields not promoted into the typed `Request`.
- **`c.Monitor.Get(ctx, id)`** at `/rest/monitor/requests/{id}.csv` for a single audit entry.
- **`ListOptions`** — `From` / `To` (ISO 8601 timestamps), `Filter` (`attributeName:OP:value`), `Order`, `Offset` / `Count`, `Live` (live-vs-completed), `Fields` (column projection).
- **Wire-quirk:** `fields` parameter accepts only one column today (despite the docs). Multi-column projection trips a 500 `"No such property 'Id,Path' for object Request"`. Fetch all fields and discard client-side until upstream fixes the parser. Documented on `ListOptions.Fields`.

### Docker image — Monitor extension baked in

- **`docker/Dockerfile`** now downloads and installs the `gs-monitor` plugin during the image build. Without it `GET /rest/monitor/requests.csv` returns 404 and the audit-log integration suite would silently skip.

### Added — GeoWebCache: global config, gridsets, mass-truncate

Three new sub-clients on `c.GWC` cover the GWC endpoints not in the original GWC port. All three are universal (work without any GeoServer extension) and integration-test against the dev/test docker stack.

- **`c.GWC.Global().Get(ctx)` / `Update(ctx, *Global)`** at `/gwc/rest/global` — runtime stats toggle, WMTS CITE compliance flag, backend timeout. Wire envelope `{"global":{...}}`; PUT accepts JSON.
- **`c.GWC.Gridsets()`** at `/gwc/rest/gridsets` — `List` (`[]string`), `Get(ctx, name) → *GridSet`, `Delete(ctx, name)`. `GridSet` covers the daily-driver fields (Name, SRS, Extent, Resolutions / Scales / ScaleNames, AlignTopLeft / YCoordinateFirst, MetersPerUnit, TileWidth, TileHeight). Create deferred — the XML wire shape for an arbitrary CRS extent is gnarly; the built-in gridsets (EPSG:4326, WebMercatorQuad, dozens of UTM tilings) cover the common case.
- **`c.GWC.MassTruncate()`** at `/gwc/rest/masstruncate` — `Capabilities`, `TruncateLayer`, `TruncateParameters`, `TruncateOrphans`, `TruncateExtent`. Typed enums for the four documented operation kinds.
- **Wire-quirk:** mass-truncate POST requires `Content-Type: text/xml`; the parser registered under `application/xml` rejects the body with `"Format extension unknown"`. The SDK always sends `text/xml`.

### Added — Master password & self password (security)

Closes the master-password and self-admin-password longer-tail items from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). Two new sub-clients on `c.Security` cover the daily-driver auth-rotation surface.

- **`c.Security.MasterPassword.Get(ctx)`** / **`Update(ctx, oldPwd, newPwd)`** at `/rest/security/masterpw`. The master password unlocks GeoServer's keystore (used for storing connection-string passwords, JKS aliases) — distinct from the admin user's login password. GeoServer exposes the current value via GET (admin-gated) for backup / disaster-recovery flows; treat the returned value with the same care as any other secret.
- **`c.Security.SelfPassword.Change(ctx, newPwd)`** at `/rest/security/self/password`. PUT-only by design (GeoServer responds 405 "You can not request the password!" to GET); the request's auth header proves possession in lieu of an old-password field.
- **Wire-quirk:** master password PUT requires both `oldMasterPassword` and `newMasterPassword`; same-value rotation is rejected with `422 "Cannot change master password"`. Self-password PUT body is just `{"newPassword":"..."}` — no old-password field.

## [2.0.0-beta.2] — 2026-05-04

Second beta. **Closes the original tier-2 gap-analysis backlog from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md)** — eight new sub-clients land on top of beta.1's frozen surface: mosaic granules, FTL templates, auth providers / filters / chains, URL checks, cascaded WMS / WMTS stores + layers, WFS XSLT transforms, manifests + system status, and runtime logging. No breaking changes from `beta.1`; existing callers can `go get @v2.0.0-beta.2` and recompile. Public API stays frozen for review through the beta line — breaking changes will not land without a strong reason.

### Added — Logging configuration

Closes the logging tier-2 item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). Adjust the active log4j profile and stdout-mirror toggle at runtime without bouncing the server — the daily-driver use case for production debugging.

- **`c.Logging.Get(ctx)`** / **`c.Logging.Update(ctx, *Config)`** at `/rest/logging`. `Config` has `Level` (e.g. "DEFAULT_LOGGING", "VERBOSE_LOGGING", "QUIET_LOGGING", "PRODUCTION_LOGGING", "GEOSERVER_DEVELOPER_LOGGING"), `Location` (read-only since GeoServer 3.0), `StdOutLogging`.
- Wire-quirk: PUT bodies use the `{"logging":{...}}` envelope; SDK marshal wraps automatically.

This closes the original tier-2 gap-analysis backlog. The full "everyone needs it" + tier-2 surface from `docs/v2-tier2-gaps.md` is now covered. Remaining future work tracks new GeoServer endpoints as the upstream API grows.

### Added — About: manifests + system status

Closes the manifests-and-system-status tier-2 item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). Two new methods on the existing `c.About` client surface ops / capacity-planning telemetry without leaving the SDK.

- **`c.About.Manifests(ctx, ListManifestsOptions{Manifest, Key, Value})`** at `/rest/about/manifest` — returns `[]ManifestEntry`, one per OSGi bundle / packaged JAR (`~150` on a stock install). `ListManifestsOptions` carries optional regex filters for bundle names, attribute keys, and attribute values. Each entry has a typed `Name` plus a free-form `Fields map[string]json.RawMessage` for the heterogeneous MANIFEST.MF attributes (`Bundle-Version`, `Build-Jdk`, `Implementation-Title`, etc.); helper `(ManifestEntry).String(field)` extracts a string value coercing JSON numbers/bools.
- **`c.About.SystemStatus(ctx)`** at `/rest/about/system-status` — returns `[]SystemMetric`. Each metric is one OS / JVM / GeoServer telemetry point (`OPERATING_SYSTEM`, `CPU_LOAD`, `MEMORY_USED`, `GEOSERVER_THREADS`, …) categorized by `SYSTEM` / `CPU` / `MEMORY` / `SWAP` / `FILE_SYSTEM` / `NETWORK` / `SENSORS` / `GEOSERVER`. `Available=false` + `Value="NOT AVAILABLE"` is common on Linux containers without OSHI native libs; production hosts populate the values.
- **Wire-quirk:** Manifest endpoint defaults to HTML; the SDK appends `.json` to the URL. Empty filter result comes back as `{"about":""}` (bare string instead of object) — normalized to a nil slice. Manifest responses commonly exceed 100 KB; `Manifests` streams the body via `DoStream` rather than buffering through the JSON Do path's 8 KiB cap.

### Added — WFS XSLT transforms

Closes the WFS XSLT transforms tier-2 item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). Transforms let WFS-T producers register XSLT files that re-shape `GetFeature` output into custom formats (HTML reports, KML, site-specific XML schemas, etc.). Endpoints live at `/rest/services/wfs/transforms` under the `gs-xslt-wfs` extension; calls against an unequipped GeoServer return `ErrNotFound`.

- **`c.WFSTransforms`** — `List` / `Get` / `Create` (metadata-only) / `Update` / `Delete`, plus `GetXSLT` / `PutXSLT` for the XSLT body, plus a single-shot `CreateWithXSLT` that POSTs the XSLT body directly with metadata as query parameters (per the upstream API's `application/xslt+xml` content-type path).
- **`Transform`** typed core: `Name`, `SourceFormat`, `OutputFormat`, `OutputMimeType`, `FileExtension`, `XSLT` (the path on disk).
- Wire-quirk: bodies use the `{"transform":{...}}` envelope GeoServer expects on POST/PUT.

The integration test verifies the "extension absent" path against the dev/test docker stack (which doesn't bake in `gs-xslt-wfs`). To run the full CRUD round-trip, install the extension and set `GEOSERVER_HAS_XSLT_WFS=1`.

### Added — Cascaded WMS / WMTS stores and layers

Closes the cascaded-WMS/WMTS tier-2 item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). Cascaded stores reference a remote WMS / WMTS server; cascaded layers re-publish that remote server's layers through the local GeoServer (federation / proxy setups).

- **`c.WMSStores`** at `/rest/workspaces/{ws}/wmsstores` — workspace-scoped CRUD plus `Iter` for the (single-page) listing. `WMSStore` covers the daily-driver fields: `Name`, `Type`, `Enabled`, `CapabilitiesURL`, `User`/`Password`/`AuthKey`/`HeaderName`/`HeaderValue`, `MaxConnections`, `ReadTimeout`, `ConnectTimeout`, `UseHTTPConnPool`.
- **`c.WMSLayers`** at `/rest/workspaces/{ws}/wmsstores/{store}/wmslayers` (canonical) and `/rest/workspaces/{ws}/wmslayers` (cross-store list) — 2-level scoped via `InWorkspace(ws).InStore(s)`. Daily-driver fields: `Name`, `NativeName`, `Title`, `Abstract`, `Keywords`, `NativeCRS`/`SRS`, bbox, `ProjectionPolicy`, `Enabled`, `ForcedRemoteStyle`, `PreferredFormat`, `MinScale`/`MaxScale`.
- **`c.WMTSStores`** + **`c.WMTSLayers`** — parallel surface for cascaded WMTS.
- **Wire shape:** all four packages MarshalJSON wrap in the documented per-type envelope (`{"wmsStore":{...}}` / `{"wmsLayer":{...}}` / `{"wmtsStore":{...}}` / `{"wmtsLayer":{...}}`); UnmarshalJSON accepts both wrapped and flat. List endpoints handle the empty-collection wire shape (`{"wmsStores":""}` etc.).

Integration tests verify the empty-list and 404 paths against the live stack; full CRUD requires an upstream WMS/WMTS server to cascade FROM and isn't exercised in the test stack — the unit tests cover the wire-shape round-trip.

### Added — URL checks (SSRF allow-list)

Closes the URL-checks tier-2 item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). URL External Access Checks are allow/deny lists for external URLs that GeoServer is permitted to fetch (SLD external graphics, image-mosaic remote rasters, cascaded WMS sources). SSRF-conscious deployments use them to constrain off-server URL fetching.

- **`c.URLChecks`** at `/rest/urlchecks` — `List` / `Get` / `Create` / `Update` / `Delete`. Typed core: `Name`, `Description`, `Enabled`, `Regex`.
- **Wire-quirk: POST/PUT bodies require the `regexUrlCheck` class-name envelope.** Flat JSON is rejected with 500. The `URLCheck.MarshalJSON` always wraps; `URLCheck.UnmarshalJSON` accepts both wrapped and flat (so callers don't need to special-case GET responses).
- **Wire-quirk: empty list comes back as `{"urlChecks":""}`** (bare string instead of object) — same empty-collection pattern as styles, datastores, etc. `List` returns `nil` on the empty form.

### Added — Auth providers, auth filters, filter chains

Closes the auth-providers + filter-chains tier-2 item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). Three new sub-clients on `c.Security` cover the security-pluggability surface for multi-IdP deployments.

- **`c.Security.AuthProviders`** — `/security/authproviders`. `List` / `Get` / `Create` / `Update` / `Delete` plus `SetOrder` to replace the active provider order. Supports the four documented core fields (`ID`, `Name`, `ClassName`, `UserGroupServiceName`) plus a free-form `Extras map[string]interface{}` that carries provider-specific config (e.g. an LDAP provider's `serverURL`/`userFormat`, an OIDC provider's `clientId`/`clientSecret`). Custom `MarshalJSON` / `UnmarshalJSON` round-trip the extras flat alongside the typed fields.
- **`c.Security.AuthFilters`** — `/security/authfilters`. `List` / `Get` / `Create` / `Update` / `Delete`. `AuthFilter` mirrors `AuthProvider`'s typed-core + `Extras` shape.
- **`c.Security.FilterChains`** — `/security/filterchain`. `List` / `Get` / `Create` / `Update` / `Delete` plus `SetOrder`. Typed core covers the eleven `@`-prefixed JSON attributes (`@name`, `@class`, `@path`, `@disabled`, `@allowSessionCreation`, `@ssl`, `@matchHTTPMethod`, `@interceptorName`, `@exceptionTranslationName`, `@httpMethods`, `@roleFilterName`) plus the `filter` array of [AuthFilter] names.

### Wire-format quirks (security: auth providers / filters)

Discovered via local integration testing against live GeoServer 2.28.0 and confirmed by reading the upstream `restconfig` controllers:

- **GET responses for individual auth providers / filters use a class-name-keyed envelope** — e.g. `{"o.g.s.config.AnonymousAuthenticationFilterConfig":{...}}` — instead of returning the entity flat. The SDK's `unwrapClassEnvelope` heuristic detects the single-key Java-FQN wrapper and unwraps transparently before flat-decoding.
- **Auth providers' List endpoint returns either an array or a class-keyed map** — `{"authproviders":[...]}` (documented OpenAPI shape) or `{"authproviders":{"<className>":{...}}}` (single-element collapse). `List` accepts both.
- **Auth filters' GET on a missing name returns `200 + {"null":""}`** instead of `404`. `AuthFilters.Get` detects the empty-Name result and synthesizes an `ErrNotFound`-bearing error so callers can use `errors.Is(err, geoserver.ErrNotFound)`. Auth providers and filter chains both correctly return 404.
- **Filter chain `filter` field collapses to a scalar string** when there's exactly one filter. `FilterChain.UnmarshalJSON` accepts both shapes.
- **Filter chain GET / Create / Update use a `{"filters":{...}}` body envelope** — note the field name is "filters" (plural) for a single chain entity. Required for both request and response bodies.

### Added — Templates (FTL) sub-client

Closes the templates tier-2 item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). FreeMarker (FTL) templates customize GetFeatureInfo HTML output, WMS HTML capabilities, and other text outputs; GeoServer scopes them at six nested levels and looks up most-specific to global at request time.

- **`c.Templates`** is the global root. Fluent scoping narrows to `InWorkspace(ws)` → `InDatastore(ds)` → `InFeatureType(ft)`, or `InWorkspace(ws)` → `InCoverageStore(cs)` → `InCoverage(cov)`.
- **`List(ctx)`** returns `[]TemplateRef` decoded from the class-name-wrapped envelope (`{"org.geoserver.rest.catalog.TemplateInfos":{"org.geoserver.rest.catalog.TemplateInfo":[...]}}`).
- **`Get(ctx, name)`** streams the FTL body and returns it as a string.
- **`Put(ctx, name, body io.Reader)`** / **`PutString(ctx, name, body string)`** write or overwrite a template (PUT with `Content-Type: text/plain`).
- **`Delete(ctx, name)`** removes a template at this scope.
- **`.ftl` suffix is auto-normalized.** `Get(ctx, "foo")` and `Get(ctx, "foo.ftl")` both target the same resource. List returns names with the suffix preserved.

### Added — Mosaic / structured-coverage granules

Closes the mosaic-granules tier-2 item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). Read-side companion to the existing `c.CoverageStores.HarvestGranule` (which adds granules); the new surface lets callers list, inspect, and remove individual granules in image-mosaic / structured coverage stores.

- **`c.Coverages.InWorkspace(ws).InCoverageStore(cs).Granules(coverage)`** returns a new `*GranulesClient` scoped to the granule index of one published coverage.
- **`Schema(ctx)`** returns the granule attribute schema (`/index` endpoint), decoded from the `{"Schema":{"attributes":{"Attribute":[...]}}}` envelope.
- **`List(ctx, ListGranulesOptions{Filter, Offset, Limit})`** returns granules as `[]Granule` — typed wrapper around the GeoJSON FeatureCollection wire shape; geometry is preserved as `json.RawMessage` so callers can decode into the GeoJSON library of their choice.
- **`Get(ctx, granuleID)`** returns a single granule. Empty FeatureCollection (some 2.x versions' wire-quirk for "not found") surfaces as `(nil, nil)`; canonical 404 surfaces as `errors.Is(err, ErrNotFound)`.
- **`Delete(ctx, granuleID, DeleteGranuleOptions{Purge, UpdateBBox})`** removes a single granule, with the `purge` and `updateBBox` query params.
- **`DeleteByFilter(ctx, DeleteGranulesOptions{Filter, Purge, UpdateBBox})`** removes every granule matching the supplied CQL filter. Empty filter is rejected by the SDK to prevent accidental match-all wipes — pass `Filter:"INCLUDE"` to delete every granule deliberately.
- **Typed enums and structs** — `PurgeMode` (`PurgeNone` / `PurgeMetadata` / `PurgeAll`), `Granule`, `GranuleSchema`, `GranuleAttribute`, plus the three options structs.

## [2.0.0-beta.1] — 2026-05-03

First beta — the v2 public API surface is now considered **frozen for review**. Subsequent betas will tighten wire-format edge cases and absorb early-adopter feedback, but breaking changes to type names, method shapes, or constructor signatures will not land without a strong reason. v2 has been continuously verified against real GeoServer 2.27.4 LTS and 2.28.0 stable on every PR since alpha.1; this tag's surface is the candidate for `v2.0.0`.

beta.1 ships the alpha.4 surface plus the two tier-2 closures from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md) that landed since alpha.4 — ACL services / REST / catalog (security tier-2) and the Resource API (data-dir tier-2). The Importer extension and the dev/test docker image bake-in remain unchanged.

### Added — Resource API client

Closes the Resource API tier-2 item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). Generic byte-stream access to files in the GeoServer data directory — FreeMarker templates, SLD includes, external graphic icons, and arbitrary descendants of the data dir.

- **`v2/rest/resources/` package** — `c.Resources` exposes the `/rest/resource/{path}` endpoint:
  - **`Get(ctx, path) (io.ReadCloser, error)`** — stream file content.
  - **`Stat(ctx, path) (*Metadata, error)`** — bare metadata (no children listed).
  - **`List(ctx, path) (*Directory, error)`** — directory listing with children.
  - **`Exists(ctx, path) (bool, Type, error)`** — combined existence + type check.
  - **`Put(ctx, path, body, contentType) error`** — upload / overwrite a regular-file resource. Intermediate directories are created on the fly.
  - **`Move(ctx, srcPath, dstPath) error`** — relocate a resource via the upstream `?operation=move` form.
  - **`Copy(ctx, srcPath, dstPath) error`** — duplicate a resource via `?operation=copy`. Per upstream, copy is not supported on directories.
  - **`Delete(ctx, path) error`** — recursive delete (per upstream).
- **Typed enums and structs** — `Type` (`TypeResource` / `TypeDirectory` / `TypeUndefined`), `Metadata`, `Directory`, `Child`.
- **`coreAdapter.DoStream`** — previously unimplemented; now wired so any sub-client can stream a response body.
- **`coreAdapter.SynthesizeError`** — sub-clients can surface package-sentinel errors (e.g. `ErrNotFound`) for wire responses that are technically 2xx but semantically failures.

### Wire-format quirks (resources package)

Discovered via local integration testing against live GeoServer 2.28.0:

- **`operation=metadata` returns 200 with `type:"undefined"` for missing paths**, instead of 404. `Stat` translates that into an `ErrNotFound`-bearing error so callers can match with `errors.Is(err, geoserver.ErrNotFound)`.
- **`children.child` is a "may be array, single object, or empty string" field.** A directory with no children may serialize as `"child":""`. A directory with one child may collapse to `"child":{...}` (no array). `Children` field's custom `UnmarshalJSON` accepts all three shapes.

### Added — ACL services / REST / catalog

Closes the security tier-2 item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). The v2 ACL sub-client previously covered only `c.ACL.Layers()`; this round adds the three sibling surfaces under `/rest/security/acl/`.

- **`c.ACL.Services()`** — service ACL rules (`/rest/security/acl/services`). Rule key is the dotted pair `service.operation` (e.g. `wms.GetMap`, `*.*`). `List` / `Add` / `Update` / `Delete` mirror the layers shape.
- **`c.ACL.REST()`** — REST ACL rules (`/rest/security/acl/rest`). Rule key is `<URL Ant pattern>:<HTTP methods>` in body form, `<pattern>;<methods>` in URL-path form for DELETE. `List` / `Add` / `Update` / `Delete`.
- **`c.ACL.Catalog()`** — singleton catalog mode (`/rest/security/acl/catalog`). `Get` / `Update` (HIDE / MIXED / CHALLENGE) plus `Reload` (`/rest/security/acl/catalog/reload`) to reload the security configuration from disk.
- **`c.ACL.Layers().Update`** — additive PUT method to edit an existing layer rule's role list (matches the new shape on Services and REST). Previously only Add (POST) was wired.
- **Typed enums and rule structs** — `ServiceRule`, `RESTRule`, `CatalogMode` (with `CatalogModeHide` / `CatalogModeMixed` / `CatalogModeChallenge`); plus `Encode` / `Decode` helpers for round-tripping the wire format.

### Wire-format quirks (acl package)

- **Services validator forbids wildcard service + non-wildcard operation.** GeoServer rejects `{"*.GetMap":"..."}` with 422 "Invalid rule *.GetMap, when namespace is * then also layer must be *". Use either `*.*` (full wildcard) or a real `service.operation` pair (e.g. `wms.GetMap`).
- **REST rule body uses `:` separator, URL path uses `;`.** GeoServer documents the body example as `"/**:GET":"ADMIN"` but the path-segment form for DELETE as `/**;GET`. `RESTRule.Encode` emits the body form; `RESTRule.EncodePathSegment` emits the URL form. `DecodeRESTRule` accepts both.
- **REST DELETE is effectively non-functional on default GeoServer installs.** GeoServer's HTTP firewall (Spring Security) rejects URL paths containing `;` and `%2F` by default. `;` can be unblocked with `GEOSERVER_USE_STRICT_FIREWALL=false` (the dev/test docker stack now sets this); `%2F` requires Java-level Spring Security configuration that is not exposed via env vars or REST. `RESTClient.Delete` is wired for completeness but documented as requiring custom server configuration. `Add` / `List` / `Update` (which hit the list endpoint with no rule in URL) work against a default install. See `RESTClient` godoc for the full caveat.
- **Dev/test docker image now disables `StrictHttpFirewall`.** `docker/env/geoserver.env` adds `GEOSERVER_USE_STRICT_FIREWALL=false` so the integration suite can exercise REST ACL endpoints. Production deployments retain the default strict firewall unless they opt in.

## [2.0.0-alpha.4] — 2026-05-03

Fourth alpha. **Closes the planned "everyone needs it" REST API surface.** The narrower tier-2 backlog continues in [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md). Five focused PRs landed in sequence: layer–style associations, file-upload publishing on stores, per-service OWS settings (WMS/WFS/WCS/WMTS), GeoWebCache (layers + seed + diskquota), and the Importer extension. Dev/test docker image now bakes the importer plugin in. Public API may still refine before `v2.0.0` based on early-adopter feedback — no production guarantees yet.

### Added — Importer extension client

- **`v2/rest/imports/` package** — `c.Imports` exposes the GeoServer Importer extension at `/rest/imports` for batch ingest and migration workflows. Daily-driver session+tasks surface: `Create`, `List`, `Iter`, `Get`, `Delete`, `Execute`; `ListTasks`, `AddTask`, `GetTask`, `UpdateTask`, `DeleteTask`. Per-task Layer / Transforms / Data sub-resources and the `database`/`mosaic` data types are deferred to follow-ups.
- **Typed enums** — `State` (INIT, INIT_ERROR, PENDING, READY, RUNNING, COMPLETE, COMPLETE_ERROR), `DataType` (file, directory, remote).
- **`CreateOptions{Async, Execute}`** controls the async/exec query parameters on `POST /rest/imports`.
- **Wire-shape note** — TargetWorkspace and TargetStore are nested objects, not strings: `{"workspace":{"name":"<name>"}}` / `{"dataStore":{"name":"<name>"}}`. The `ImportRequest` API accepts flat string names and the package marshals them into the nested form.

### Docker image — Importer extension baked in

- **`docker/Dockerfile`** now downloads and installs the GeoServer Importer extension during the image build. CI's compose stack on both 2.27.4 LTS and 2.28.0 stable will exercise the full `v2/rest/imports` integration suite without skipping. The Dockerfile pre-extracts the WAR (Tomcat happily runs the unpacked form) and drops the importer JARs into `WEB-INF/lib/`.
- Without this change, `GET /rest/imports` returns 404 and the import-test suite skips silently. The integration test still has a `requireImporter` skip-gate to handle deployments where the extension is intentionally absent.

### Added — GeoWebCache REST client

- **`v2/rest/gwc/` package** — `c.GWC` exposes the GeoWebCache REST endpoints universal to any deployment serving map tiles. URL prefix is `/gwc/rest/` (outside the `/rest/` catalog tree).
- **`c.GWC.Layers()`** — `List(ctx)` returns layer names; `Get/Put(ctx, name, *LayerConfig)` use XML wire format (`<GeoServerLayer>` with `<mimeFormats>`, `<gridSubsets>`, `<metaWidthHeight>`, `<parameterFilters>`, etc.); `Delete(ctx, name)` removes the cache config.
- **`c.GWC.Seed()`** — `Submit(ctx, layer, *SeedRequest)` is asynchronous (POST returns immediately); `Status(ctx, layer)` and `StatusAll(ctx)` decode the `{"long-array-array":[[tilesProcessed, totalTiles, remainingSeconds, taskId, taskStatus], ...]}` wire shape into a flat `[]SeedTask`; `KillAll(ctx)` terminates running tasks. Typed `SeedOp` enum (`OpSeed` / `OpReseed` / `OpTruncate`) and `SeedTaskStatus` enum (`StatusAborted` / `StatusPending` / `StatusRunning` / `StatusDone`).
- **`c.GWC.DiskQuota()`** — `Get/Update(ctx, *DiskQuota)` for global LFU/LRU eviction policy and disk-usage cap.

### Wire-format quirks (gwc package)

Discovered via local integration testing against live GeoServer 2.28.0 — three quirks the docs don't surface:

- **DiskQuota PUT requires XML, not JSON**, and uses `<globalQuota><value>NUMBER</value><units>UNIT</units></globalQuota>` rather than the `<bytes>NUMBER</bytes>` form GET returns. The server-side parser (XStream's `QuotaXSTreamConverter`) is asymmetric between read and write paths. `DiskQuotaClient.Update` translates `Quota.Bytes` to the `value/units` XML form (always serializing as `B` bytes) and PUTs to `/gwc/rest/diskquota.xml`.
- **GWC returns 500 (not 404) for unknown layers** on `Layers.Get`. Integration test accepts either `ErrServerError` or `ErrNotFound`; unit test verifies the strict 404→`ErrNotFound` mapping.
- **`/gwc/rest/seed.json`** returns `{"long-array-array":[[...]]}` — a positional 5-element array per running task. `SeedStatus.UnmarshalJSON` decodes this into a typed `[]SeedTask` slice with named fields.

### Added — per-service OWS settings

- **`v2/rest/services/` package** — per-service OWS configuration for WMS / WFS / WCS / WMTS. The companion to `v2/rest/settings/` (which covers the global `/rest/settings` document). New entry-point `c.Services` exposes `.WMS()` / `.WFS()` / `.WCS()` / `.WMTS()`, each returning a typed client with `Get`/`Update` (global) and `.InWorkspace(ws)` returning a workspace-scoped client with `Get`/`Update`/`Delete` (DELETE removes the per-workspace override and falls back to the global config).
- **Per-service settings types** — `WMSSettings` (watermark, interpolation, max\* tunables, GFI/GetMap MIME-checking flags), `WFSSettings` (maxFeatures, serviceLevel BASIC/TRANSACTIONAL/COMPLETE, GML output config), `WCSSettings` (gmlPrefixing, latLon, max\*Memory in KB), `WMTSSettings` (no unique fields beyond ServiceInfo). All embed the common `ServiceInfo` block.

### Wire-format quirks (services package)

Discovered via local integration testing against live GeoServer 2.28.0 — three quirks the upstream OpenAPI YAML doesn't document:

- **`versions`** uses a class-name wrapper key (`org.geotools.util.Version`) and collapses single-element arrays to a scalar object. `Versions{List []string}` decodes both shapes into a flat slice; marshal emits the canonical array form.
- **`keywords.string`** collapses single-element arrays to a scalar string. `Keywords{Strings []string}` decodes both; marshal emits the array form.
- **`metadataLink`** is sent as `""` (empty string) when unset, not as `null` or absent. Custom UnmarshalJSON on `*MetadataLink` treats the empty-string form as "unset" — the resulting pointer is non-nil but with all-zero fields.

### Added — file-upload publishing on stores

- **`v2/rest/datastores.WorkspaceClient.UploadFile(ctx, name, body, opts)`** — `PUT /workspaces/{ws}/datastores/{name}/{file|url|external}[.<ext>]`. Publishes a file-backed datastore by uploading the file contents (`UploadMethodFile`, default), pointing at a remote URL the server fetches (`UploadMethodURL`), or referencing a server-local path with no transfer (`UploadMethodExternal`). Documented `Extension` values: `shp`, `properties`, `appschema`. Default `Content-Type` is `application/zip` for file uploads, `text/plain` for URL/external; override via `UploadOptions.ContentType`.
- **`v2/rest/coveragestores.WorkspaceClient.UploadFile(ctx, name, body, opts)`** — same shape for raster stores. Documented `Extension` values: `geotiff`, `worldimage`, `imagemosaic`. Override `ContentType` to `image/tiff` for a single GeoTIFF.
- **`v2/rest/coveragestores.WorkspaceClient.HarvestGranule(ctx, name, body, opts)`** — `POST` to the same endpoint. Appends a new granule to an existing image-mosaic store without reconfiguring the whole store. Use `UploadMethodExternal` with a server-local path to avoid transferring large rasters across HTTP.
- Both packages add `DoRaw` to their `Core` interface and route uploads through `transport.DoRaw` with `Accept: */*` (matches the workspace-scoped POST quirk handled by `styles.Client.Create` and the new `layers.AddStyle`).

### Added — layer–style associations

- **`v2/rest/layers.WorkspaceClient.ListStyles(ctx, layer)`** and **`AddStyle(ctx, layer, styleName, opts)`** — dedicated sub-resource for managing a layer's alternative-style list (`/workspaces/{ws}/layers/{l}/styles`). `AddStyleOptions{Default: true}` atomically promotes the new style to the layer's default in one wire round-trip, replacing the previous GET + mutate + PUT dance. Removing an alternative style is not exposed as a dedicated method (GeoServer doesn't document a DELETE on this sub-resource); use `Update()` with the unwanted reference removed from `Layer.Styles`.

### Added — OWS describe operations

- **`v2/ows/wfs.Client.DescribeFeatureType(ctx, opts)`** — fetches the XSD schema describing one or more published feature types and decodes it into a `*FeatureSchema`. The schema's `Attributes(typeName)` helper walks the typical `complexType > complexContent > extension > sequence > element*` shape and surfaces a flat `[]Attribute` list. Send multiple type names in `DescribeFeatureTypeOptions.TypeNames` (comma-joined for the WFS 2.0 `typeNames` query plus the WFS 1.1.0 `typeName` alias for cross-version compatibility). Companion free function `wfs.ParseFeatureSchema(io.Reader)` for out-of-band parsing.
- **`v2/ows/wcs.Client.DescribeCoverage(ctx, opts)`** — fetches detailed metadata for one or more coverages and decodes it into `*CoverageDescriptions` (CoverageId + BoundedBy envelope + DomainSet/RectifiedGrid + RangeType/DataRecord/Field). Useful for knowing the bands, units of measure, CRS, and pixel-space dimensions of a coverage before issuing a GetCoverage. Companion free function `wcs.ParseCoverageDescriptions(io.Reader)`.

### Added (tests)

- httptest unit tests for both Describe operations: parse fixtures with realistic GeoServer XML namespace declarations, query-parameter assertions (including the `typeNames` / `typeName` cross-version aliases for WFS), multi-id requests, empty-id rejection, and 404 → `ErrNotFound` sentinel.
- Integration tests for both — WFS DescribeFeatureType against `sf:archsites` (default-install feature type), and WCS DescribeCoverage against the first coverage discovered via `c.WCS.GetCapabilities`.
- Godoc `Example_*` for both new methods.

## [2.0.0-alpha.3] — 2026-05-03

Third alpha. Closes the OWS GetCapabilities trio with WFS + WCS, taking v2's surface beyond v1's. No breaking changes from `alpha.2`; existing callers can `go get @v2.0.0-alpha.3` and recompile. Public API may still refine before `v2.0.0` — no production guarantees yet.

### Added — OWS clients (2/3, 3/3)

- **`v2/ows/wfs/` package** — WFS GetCapabilities. Hand-written subset of WFS 1.1.0 / 2.0 schemas (ServiceIdentification, ServiceProvider, OperationsMetadata, FeatureTypeList with WGS84BoundingBox + DefaultSRS / OtherSRS / OutputFormats). New free function `wfs.ParseCapabilities(io.Reader)` and `c.WFS.GetCapabilities(ctx, opts)` — global by default; `c.WFS.InWorkspace(ws)` for a workspace-scoped capabilities view. Default version 2.0.0; 1.1.0 supported as well (same root element).
- **`v2/ows/wcs/` package** — WCS GetCapabilities for WCS 2.0.x. Hand-written subset (ServiceIdentification, ServiceProvider, OperationsMetadata, ServiceMetadata with formats + CRSes, Contents with CoverageSummary). New free function `wcs.ParseCapabilities(io.Reader)` and `c.WCS.GetCapabilities(ctx, opts)` with workspace scoping. Default version 2.0.1. WCS 1.0.0 / 1.1.1 use a different root element (`WCS_Capabilities`) and are not in scope here.

### Added (tests)

- `v2/ows/wfs/wfs_test.go` and `v2/ows/wcs/wcs_test.go` — fixture-based parse tests with realistic GeoServer namespace declarations, httptest tests for global + workspace scope, version override, and 404 → sentinel mapping.
- `v2/ows/wfs/wfs_integration_test.go` and `v2/ows/wcs/wcs_integration_test.go` — real GeoServer assertions (capabilities document non-empty for both global and a fresh empty workspace; WFS 1.1.0 version-override path).
- Godoc `Example_*` for both packages.

### Fixed (wire-format)

- **`v2/ows/wcs.GetCapabilities`** sends `service=WCS` (uppercase). GeoServer's WCS endpoint is case-sensitive on the `service` query parameter and returns 400 `"Error in service name, expected value: WCS"` on `service=wcs`. WMS and WFS accept lowercase, so the asymmetry only surfaces on WCS. Comment in `wcs.go` records the quirk.

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
