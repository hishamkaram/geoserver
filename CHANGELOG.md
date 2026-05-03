# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] — 2026-05-03

This release modernizes the build, fixes long-standing bugs, and adds an idiomatic Go API surface alongside the existing one. Existing v1.0.x callers do not need source changes to upgrade.

### Added
- `New(serverURL, username, password string, opts ...Option) *GeoServer` — functional-options constructor.
- `Option` type and helpers: `WithHTTPClient`, `WithTimeout`, `WithLogger`, `WithUserAgent`, `WithBasicAuth`.
- Typed errors via the `*Error` type and sentinel values `ErrNotFound`, `ErrUnauthorized`, `ErrForbidden`, `ErrConflict`, `ErrServerError`, `ErrRateLimited`. Use with `errors.Is`/`errors.As`. Existing string-based error messages are preserved for backwards compatibility.
- `…Context(ctx context.Context, ...)` sibling methods for every existing exported method on `*GeoServer`. Old methods now delegate with `context.Background()`.
- `CatalogWithContext` interface bundling the new context-aware service interfaces.
- `CoverageService` interface — coverage operations are now exposed through `Catalog`.
- `SettingsService` is now embedded in `Catalog`.
- `wms.ParseCapabilitiesE([]byte) (*wms.Capabilities, error)` — non-fatal sibling of `ParseCapabilities`.
- httptest-based unit test layer (no Docker required): `make test-unit`.
- testcontainers-go integration test matrix against GeoServer 2.27 LTS and 2.28 stable: `make test-integration`.
- GitHub Actions CI (lint, unit, integration, vuln, codeql), Dependabot config, issue/PR templates.
- `Makefile`, `.golangci.yml`, `.editorconfig`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`.

### Changed
- **Go version requirement**: minimum Go 1.23 (was Go 1.15).
- **Logging**: switched from `github.com/sirupsen/logrus` to stdlib `log/slog`. Library logs at Debug for HTTP details, Warn for transport failures, Error for protocol violations. By default the logger is silent (`slog.DiscardHandler`); configure via `WithLogger(slog.Handler)`.
- HTTP client now has a default 30s timeout (was unlimited). Override via `WithHTTPClient` or `WithTimeout`.
- `ParseURL` now applies `url.PathEscape` per segment. Workspace/layer names with spaces, slashes, or non-ASCII characters now produce correct URLs (previously these produced malformed URLs).
- `statusErrorMapping` (`vars.go`) extended to cover 400, 409, 415, 429, 502, 503, 504 status codes.
- Updated dependencies: `stretchr/testify` v1.2.2 → v1.9.0, `gopkg.in/yaml.v2` v2.2.1 → `gopkg.in/yaml.v3` v3.0.1.
- Docker dev stack: Tomcat 10 + JDK 17, GeoServer 2.28.x (was Tomcat/JDK 8 + GeoServer 2.13). PostGIS 16-3.4 (was 10.0-2.4). New `docker-compose.test.yml` adds a 2.27 LTS leg.

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
- Docker base image upgraded from `tomcat:jdk8-adoptopenjdk-hotspot` (EOL) to `tomcat:10-jdk17-temurin`.
- GeoServer download in Dockerfile now verifies TLS certs (was `--no-check-certificate`).
- All transitive deps audited via `govulncheck` in CI.

## [1.0.1] — 2023-02-28

Pre-revival release. See git history for details.
