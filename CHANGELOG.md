# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] — 2026-05-03

The v1.1 revival release. Modernizes the build, fixes long-standing bugs, adds a context-aware / typed-error / functional-options API surface alongside the existing one, ports two long-stalled community contributions (PRs #15 and #17), and lands the project's CI / governance scaffolding (Claude Code config, mandatory integration tests on PR, branch + squash-merge workflow). Existing v1.0.x callers can upgrade with only a `go.mod` bump — no source changes required.

### Added — core API surface
- `New(serverURL, username, password string, opts ...Option) *GeoServer` — functional-options constructor.
- `Option` type and helpers: `WithHTTPClient`, `WithTimeout`, `WithLogger`, `WithUserAgent`, `WithBasicAuth`.
- Typed errors via the `*Error` type and sentinel values `ErrNotFound`, `ErrUnauthorized`, `ErrForbidden`, `ErrConflict`, `ErrServerError`, `ErrRateLimited`. Use with `errors.Is` / `errors.As`. Existing string-based error messages are preserved for backwards compatibility.
- `…Context(ctx context.Context, ...)` sibling methods for every existing exported method on `*GeoServer`. Old methods now delegate with `context.Background()`.
- `CatalogWithContext` interface bundling the new context-aware service interfaces.
- `CoverageService` interface — coverage operations are now exposed through `Catalog`.
- `SettingsService` is now embedded in `Catalog`.
- `wms.ParseCapabilitiesE([]byte) (*wms.Capabilities, error)` — non-fatal sibling of `ParseCapabilities`.

### Added — security / ACL / JNDI / feature-type endpoints (port of PR #15 + PR #17)

Both contributions originally proposed by community contributors but never landed on `master`. Refactored to fit the v1.1 idiom (every method has a `*Context` sibling, errors map to typed sentinels via `errors.Is`, parallel `*ServiceWithContext` interfaces).

- **Security (users / groups / roles)** — new `security.go` and `SecurityService` / `SecurityServiceWithContext` interfaces:
  - `GetUsers`, `CreateUser`, `DeleteUser` (under a named user-group service; empty service resolves to `"default"`)
  - `GetGroups`, `CreateGroup`, `DeleteGroup`
  - `GetRoles`, `GetUserRoles`, `CreateRole`, `DeleteRole`
  - `AddUserRole`, `DeleteUserRole` (associate / disassociate role from user)
  - All have `…Context` siblings.
- **Layer ACL rules** — new `acl.go` and `ACLService` / `ACLServiceWithContext` interfaces:
  - `ACLRule` type with `ACLOpRead` / `ACLOpWrite` / `ACLOpAdmin` operation constants
  - `GetLayersACLRules`, `AddLayersACLRule`, `DeleteLayersACLRule` (+ `*Context` siblings)
  - `ACLRule.ToStrings` and `StringToACLRule` round-trip helpers
- **JNDI-backed datastores** — new `DatastoreJNDIConnection` struct + `CreateJNDIDatastore` / `CreateJNDIDatastoreContext`. Use when GeoServer is configured to look up its JDBC connection pool via JNDI (typically Tomcat-managed).
- **`DatastoreConnector` interface** — `*DatastoreConnection` and `DatastoreJNDIConnection` both satisfy it. New methods `CreateDatastoreFromConnector` / `CreateDatastoreFromConnectorContext` accept any connector. Useful for callers that want a single code path or for plugging in custom connector types.
- **`DatastoreConnection.Options []Entry`** — additional connection parameters (e.g., `"max connections"`, `"Expose primary keys"`). Field is appended at the end of the struct so v1.0 callers using positional struct literals continue to compile.
- **`CreateFeatureType` / `CreateFeatureTypeContext`** — register a feature type against a database-backed datastore (PostGIS, Oracle, etc.) without going through `UploadShapeFile`.
- **`GetFeatureTypeList` / `GetFeatureTypeListContext`** — calls `?list=` discovery endpoint at `/rest/workspaces/{ws}/datastores/{ds}/featuretypes`. Returns the flat list of feature-type names filtered by `FeatureTypeListKind`:
  - `FeatureTypeListConfigured` — only feature types already configured in GeoServer.
  - `FeatureTypeListAvailable` — tables in the underlying datastore not yet published.
  - `FeatureTypeListAvailableWithGeom` — like `available` but only tables carrying a geometry column.
  - `FeatureTypeListAll` (default for empty kind) — configured ∪ available.
- `Catalog` interface now embeds `SecurityService` and `ACLService`.

`GetRoles`, `GetUserRoles`, and `GetGroups` decode both the GeoServer 2.28+ response keys (`roles`, `groups`) and the older 2.x keys (`roleNames`, `groupNames`) so they work across the supported version matrix.

### Added — testing & CI infrastructure
- httptest-based unit test layer (no Docker required): `make test-unit`. Covers every service for 2xx + typical 4xx/5xx mapping to typed sentinels.
- Integration test matrix against GeoServer 2.27 LTS and 2.28 stable: `make test-integration`. Runs on every PR (mandatory gate).
- GitHub Actions CI: `Lint`, `Unit tests (Go 1.23)`, `Unit tests (Go 1.25)`, `govulncheck`, `Analyze (Go)` (CodeQL), `GeoServer 2.27.4`, `GeoServer 2.28.0` — all required for merge.
- Dependabot config, issue / PR templates, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`.
- Project `CLAUDE.md` (root) plus `.claude/agents`, `.claude/skills`, `.claude/commands` so Claude Code sessions in the repo auto-load the v1.1 conventions and have ready-made slash commands (`/integration-test`, `/lint-fix`, `/release-prep`, `/add-context-twin`, `/non-breaking-v1`).

### Changed
- **Go version requirement**: minimum Go 1.23 (was Go 1.15). For users running `govulncheck` locally, Go 1.25.9 or newer is recommended to clear advisories `GO-2026-4946` (`crypto/x509`) and `GO-2026-4870` (`crypto/tls`); CI uses the latest 1.25.x patch via `check-latest: true`.
- **Logging**: switched from `github.com/sirupsen/logrus` to stdlib `log/slog`. Library logs at Debug for HTTP details, Warn for transport failures, Error for protocol violations. By default the logger is silent (`slog.DiscardHandler`); configure via `WithLogger(slog.Handler)`.
- HTTP client now has a default 30s timeout (was unlimited). Override via `WithHTTPClient` or `WithTimeout`.
- `ParseURL` now applies `url.PathEscape` per segment. Workspace/layer names with spaces, slashes, or non-ASCII characters now produce correct URLs (previously these produced malformed URLs).
- `statusErrorMapping` (`vars.go`) extended to cover 400, 409, 415, 429, 502, 503, 504 status codes.
- Updated dependencies: `stretchr/testify` v1.2.2 → v1.11.x, `gopkg.in/yaml.v2` v2.2.1 → `gopkg.in/yaml.v3` v3.0.1.
- Docker dev stack: Tomcat 9 + JDK 17, GeoServer 2.28.x (was Tomcat/JDK 8 + GeoServer 2.13). PostGIS 16-3.4 (was 10.0-2.4). New `docker-compose.test.yml` adds a 2.27 LTS leg.

### Fixed
- Removed all `panic()` calls in library code (`utils.go`, previously panicked on unknown HTTP method, transport failure, URL parse failure).
- Removed `log.Fatal()` from `wms.ParseCapabilities` — it now logs and returns `nil` on error; new `ParseCapabilitiesE` returns the error explicitly.
- `SettingsService.UpdateGlobalSettings` (plural, matching the interface declaration) — previously the implementation was `UpdateGlobalSetting` (singular) which broke any code using the interface. The singular method is preserved and marked `Deprecated`.
- `SettingsService.GetGlobalSettings` return type aligned with the implementation.
- ~15 ignored errors fixed across `workspaces.go`, `styles.go`, `coverages.go`, `datastores.go`, `feature_types.go`, `layers.go`, `layergroups.go`, `coverage_stores.go`, `namespaces.go`, `settings.go`, `geoserver.go`.
- Type-assertion safety: `layergroups.go` and `feature_types.go` no longer panic on unexpected JSON shape.
- Replaced deprecated `io/ioutil` with `os` and `io` (`geoserver.go`, `utils.go`, `layers.go`).
- Bare `fmt.Sprintf` URL construction replaced with consistent `ParseURL` calls in styles, coverages, feature_types, layers, layergroups.
- Manual coverage store-name split (`coverages.go`) no longer panics on malformed input.
- `ParseURL` no longer double-encodes path segments containing characters that `url.PathEscape` percent-encodes (e.g., `"*"` → `"%2A"` → previously `"%252A"`). The encoded path is now preserved through `url.URL.String()` by setting `RawPath` alongside `Path`. GeoServer's StrictHttpFirewall rejected the previously-emitted double-encoded URLs as "potentially malicious"; the new ACL `DELETE` path (where rule strings carry literal `*` wildcards) was the trigger that surfaced the bug. `utils_unit_test.go` `TestParseURL_NoDoubleEncoding` guards the regression.

### Deprecated
- `GetCatalog(url, user, pass)` — prefer `New(url, user, pass, opts...)`.
- `GetLogger()` — configure logging via `WithLogger(slog.Handler)`.
- `UpdateGlobalSetting` (singular) — use `UpdateGlobalSettings`.
- `GetGeoserverRequest` — use `GetGeoserverRequestE` (returns error).
- `wms.ParseCapabilities` — use `wms.ParseCapabilitiesE`.

### Removed
- `.travis.yml` (Travis CI is shut down; replaced by GitHub Actions).
- `Gopkg.toml`, `Gopkg.lock` (leftover from incomplete `dep` → modules migration).

### Security
- Docker base image upgraded from `tomcat:jdk8-adoptopenjdk-hotspot` (EOL) to `tomcat:9-jdk17-temurin` (Tomcat 9 because GeoServer 2.x requires javax, not jakarta).
- GeoServer download in Dockerfile now verifies TLS certs (was `--no-check-certificate`).
- All transitive deps audited via `govulncheck` in CI. CI uses the latest Go 1.25.x patch (`check-latest: true`) which clears the `crypto/x509` and `crypto/tls` advisories that affected earlier 1.25.x.

### Acknowledgements
Thanks to **@archer-v** (Alexander Cherviakov / Mandalorian One, PR #15) for the original security / ACL / JNDI / `CreateFeatureType` work, and **@wichert** (Wichert Akkerman / Woven Planet, PR #17) for the feature-type discovery endpoint (`?list=available|configured|all`). Both contributions sat unmerged for years; they're in this release.

## [1.0.1] — 2023-02-28

Pre-revival release. See git history for details.
