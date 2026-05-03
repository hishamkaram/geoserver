# Roadmap

This document describes the project's direction. It is intentionally conservative — items here are commitments to *consider* in a release window, not a guarantee of inclusion. Concrete release content is tracked in [`CHANGELOG.md`](CHANGELOG.md).

## v1.x — maintenance line

Stable, non-breaking. Existing v1.0 callers can upgrade to any v1.x release with only a `go.mod` bump.

- **Scope:** bug fixes, security patches, additive features that fit on the existing `*GeoServer` surface, dependency updates, CI/Docker modernization.
- **Cadence:** as needed. No fixed schedule.
- **GeoServer matrix:** 2.27 LTS + 2.28 stable. Not extended to GeoServer 3.0 (defers to v2).
- **Go matrix:** 1.25.x. Floor moves with the supported Go version.
- **Out of v1.x:** any change that breaks a v1.0 caller. Examples that won't land here: removing exported symbols, changing signatures, restructuring the root package into resource subdirectories, replacing the `*Logger` wrapper with raw `*slog.Logger`.

## v2.x — clean redesign

Lives at module path `github.com/hishamkaram/geoserver/v2` (subdir form). Idiomatic 2026-era Go SDK.

**Design tenets (locked in):**

- Immutable `*Client` — all fields private, configured via functional options at construction.
- Mandatory `context.Context` first argument on every public method. No `Background` shims.
- Sub-client pattern — `c.Workspaces`, `c.Datastores.InWorkspace(ws)`, etc. Each resource is its own subpackage in `rest/`.
- Single error type — `*APIError` with sentinel errors via `errors.Is`. No string matching.
- Auth via `http.RoundTripper` wrapping the configured transport. Allows OpenTelemetry / Vault / custom retry libs to layer on.
- Streaming uploads — no `io.ReadAll` of large files into memory.
- Pagination via `iter.Seq2` (Go 1.23+ range-over-func).
- Zero runtime third-party dependencies — stdlib only.

**Milestones (checklist; tags applied as each is reached):**

- [x] **Scaffold** — `/v2/` skeleton + `Workspaces` reference resource + transport layer + `internal/transport/` algorithms.
- [x] **v2.0.0-alpha.1** — scaffold tag, signals public API direction. *Tagged 2026-05-03.*
- [x] **Resource port (1/3)** — `datastores`, `feature_types`, `coverage_stores`, `coverages`.
- [x] **Resource port (2/3)** — `layers`, `layergroups`, `styles`.
- [x] **Resource port (3/3)** — `namespaces`, `settings`, `security`, `acl`, `about`.
- [x] **System endpoints** — `c.System.Reload` / `c.System.ResetCache` (port of v1's `configuration.go`).
- [x] **OWS clients (1/3, 2/3, 3/3)** — `ows/wms/`, `ows/wfs/`, `ows/wcs/` (GetCapabilities + workspace scope on each). All three follow the same shape: free-function `ParseCapabilities(io.Reader)` plus a sub-client method routed through `transport.DoXML`.
- [x] **Migration guide** — `docs/migration-v1-to-v2.md` populated with concrete v1 → v2 mappings for every resource.
- [x] **v1 parity at `master`** — every v1 surface (REST + OWS) has a v2 equivalent.
- [x] **v2.0.0-alpha.2** — bundles the post-`alpha.1` work (godoc Examples, README refresh, OWS WMS port, system reload/reset). *Tagged 2026-05-03.*
- [x] **v2.0.0-alpha.3** — adds WFS + WCS GetCapabilities (closes the OWS GetCapabilities trio) plus WFS `DescribeFeatureType` and WCS `DescribeCoverage`. *Tagged 2026-05-03.*
- [x] **Top-5 gap-analysis surface** — closes the planned "everyone needs it" gaps: layer–style associations (`c.Layers.AddStyle`/`ListStyles`), file-upload publishing on stores (`c.Datastores.UploadFile` + `c.CoverageStores.UploadFile`/`HarvestGranule`), per-service OWS settings (`c.Services.WMS()`/`WFS()`/`WCS()`/`WMTS()`), GeoWebCache (`c.GWC.Layers()`/`Seed()`/`DiskQuota()`), and Importer extension (`c.Imports`). Dev/test docker image bakes in the importer plugin so CI exercises it without skipping. Tier-2 backlog (narrower-audience endpoints) is tracked in [`docs/v2-tier2-gaps.md`](docs/v2-tier2-gaps.md).
- [x] **v2.0.0-alpha.4** — bundles the top-5 gap closures + docs refresh. *Tagged 2026-05-03.*
- [ ] **Tier-2 also-rans** — narrower-audience endpoints from the gap-analysis tier-2 list: mosaic granules, Resource API, FTL templates, auth providers + filter chains, ACL services/REST/catalog, URL checks, cascaded WMS/WMTS stores, XSLT transforms, manifests, runtime logging config. Each is tractable as its own follow-up PR.
- [ ] **v2.0.0-beta.1** — public API frozen for review. Surface freeze comes after enough early-adopter feedback on `alpha.4` to lock in shapes.
- [ ] **v2.0.0** — final tag.

## GeoServer 3.0 support

Tracked as a v2.x point release. GeoServer 3.0 (April 2026 GA) brings Jakarta EE migration, Tomcat 11, and the new ImageN raster engine. We will validate against 3.0 in CI once:

1. The 3.0 REST API surface stabilizes against the 2.x reference.
2. Tomcat 11 + Jakarta EE is broadly adopted in production GeoServer deploys.
3. ImageN's coverage-store wire shape is documented.

Until then, this library targets 2.27 LTS + 2.28 stable.

## Out-of-roadmap

Things this project will **not** do:

- Replace the v1 module path. `github.com/hishamkaram/geoserver` continues to ship v1.x patch releases as long as users depend on it.
- Adopt third-party HTTP, JSON, or logging libraries in v2. Stdlib only.
- Wrap GeoServer's WMS/WFS/WCS request building in a generated-from-XSD style. The `ows/` clients use hand-written types because the GeoServer XML response shapes are inconsistent (see [`docs/geoserver-rest-quirks.md`](docs/geoserver-rest-quirks.md)).
- Provide a CLI binary. This is a library; tooling can be built on top in a separate repository.

## How to influence the roadmap

- File a feature request as a GitHub issue with the `feature_request.yml` template.
- Open a discussion if the work would touch core design decisions (immutable Client, sub-client layout, etc.).
- For drive-by improvements within v1.x non-breaking constraints, open a PR directly.
