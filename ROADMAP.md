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
- [ ] **v2.0.0-alpha.1** — scaffold tag, signals public API direction.
- [ ] **Resource port (1/3)** — `datastores`, `coverage_stores`, `feature_types`.
- [ ] **Resource port (2/3)** — `layers`, `layergroups`, `coverages`, `styles`.
- [ ] **Resource port (3/3)** — `namespaces`, `settings`, `security`, `acl`, `about`, `capabilities`.
- [ ] **v2.0.0-beta.1** — REST surface complete, integration matrix on 2.27 + 2.28, public API frozen for review.
- [ ] **OWS clients** — `ows/wms/`, `ows/wfs/`, `ows/wcs/` subpackages. (May slip to v2.x point release.)
- [ ] **Migration guide** — `docs/migration-v1-to-v2.md` filled out with concrete mappings.
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
