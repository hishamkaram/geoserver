# Changelog — v2

All notable changes to `github.com/hishamkaram/geoserver/v2` are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). v2 ships independently of v1 (separate go.mod, separate `v2.x.y` tags).

## [Unreleased]

### Added

- Initial scaffold. `*Client` immutable constructor (`New`) with functional options (`WithHTTPClient`, `WithTransport`, `WithBasicAuth`, `WithBearerToken`, `WithLogger`, `WithUserAgent`, `WithTimeout`, `WithHeader`).
- Single error type `*APIError` with package sentinels: `ErrBadRequest`, `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrMethodNotAllowed`, `ErrConflict`, `ErrUnsupportedMediaType`, `ErrRateLimited`, `ErrServerError`, `ErrBadGateway`, `ErrServiceUnavailable`, `ErrGatewayTimeout`. Match via `errors.Is` and `errors.As`.
- `internal/transport/` package: `BuildURL` (PathEscape + RawPath preservation, ported from v1.1's bug-fixed algorithm), `DoJSON` (single chokepoint for REST calls), `AuthRoundTripper` and `HeaderRoundTripper` (auth and User-Agent attached via the transport stack rather than per-request).
- `rest/workspaces/` reference sub-client: `List`, `Iter` (`iter.Seq2`), `Get`, `Create`, `Update`, `Delete`. httptest unit tests cover 2xx happy paths and 401/404/409/500 sentinel mapping plus the URL-escaping regression guard.
- `rest/datastores/` sub-client (workspace-scoped): `c.Datastores.InWorkspace(ws)` returns a `*WorkspaceClient` exposing `List`, `Iter`, `Get`, `Create`, `Update`, `Delete`. Convenience connectors `PostGIS` and `JNDI` produce the wire-format payload; arbitrary drivers can be supplied via `Raw(Datastore)`. This is the reference for every other workspace-scoped resource (feature types, coverages, layers, …).

No release tag yet; the module's first published tag will be `v2.0.0-alpha.1` when the surface is wide enough to warrant a soak.
