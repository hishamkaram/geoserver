# Architecture

This document describes how `github.com/hishamkaram/geoserver` is laid out, the design tenets, and the trade-offs that drive the v1.x shape. It is reference material; release notes live in [`CHANGELOG.md`](../CHANGELOG.md), and the forward-looking direction lives in [`../ROADMAP.md`](../ROADMAP.md).

## Package shape

The library is a single `package geoserver` at the module root, with one subpackage:

| Path | Purpose |
|---|---|
| `github.com/hishamkaram/geoserver` | Public API — `*GeoServer` client, all resource methods, options, errors |
| `github.com/hishamkaram/geoserver/wms` | XML types and parser for WMS GetCapabilities responses |
| `github.com/hishamkaram/geoserver/internal/transport` (v1.1.x+) | HTTP request building / dispatch and URL construction. Implementation detail — not importable by external code |

A separate module exists at `github.com/hishamkaram/geoserver/v2` (latest preview `v2.0.0-alpha.4`) with a different shape — sub-clients per resource (`c.Workspaces`, `c.Datastores.InWorkspace(ws)`, `c.Services.WMS()`, `c.GWC.Seed()`, `c.Imports`, etc.) and surfaces v1 never had. v1 and v2 ship independently. See [`migration-v1-to-v2.md`](./migration-v1-to-v2.md) for the side-by-side mapping.

## Public API entry points

There is **one** type users construct: `*GeoServer`. There are **two** ways to construct it:

1. **`New(serverURL, username, password string, opts ...Option) *GeoServer`** — recommended (v1.1+). Functional options for HTTP client, timeout, logger, user agent, basic auth.
2. **`GetCatalog(serverURL, username, password string) *GeoServer`** — legacy v1.0 entry point. Marked `// Deprecated: prefer New(...)`. Internally a one-liner around `New`.

`*GeoServer` exposes ~90 methods covering the GeoServer REST resource set: workspaces, datastores, coverage stores, coverages, feature types, layers, layer groups, styles, namespaces, settings, security (users / groups / roles), ACL, about, capabilities, configuration. See `pkg.go.dev/github.com/hishamkaram/geoserver` for the full list.

## *Context twin pattern (mandatory for new exports)

Every exported method on `*GeoServer` comes in a pair:

```go
// Non-context wrapper — delegates with context.Background()
func (g *GeoServer) GetWorkspaces() ([]*Resource, error) {
    return g.GetWorkspacesContext(context.Background())
}

// Context-aware variant — does the actual work
func (g *GeoServer) GetWorkspacesContext(ctx context.Context) ([]*Resource, error) {
    // ...
}
```

Reference: `workspaces.go:16-38,57-79`. The non-context form exists only for source-compatibility with v1.0 callers; **new code should always use the `*Context` form** so it can honor cancellation and deadlines.

When a contributor adds a new method on `*GeoServer`, both the non-context wrapper and the `*Context` sibling must land together, plus the corresponding entry in the parallel `*ServiceWithContext` interface.

## Errors

Every HTTP error is a `*Error` (`errors.go`):

```go
type Error struct {
    Op         string  // operation name, e.g. "GetWorkspaces"
    URL        string  // request URL
    StatusCode int     // HTTP status
    Body       []byte  // response body, truncated to 8 KiB
    Err        error   // wrapped sentinel (one of ErrNotFound, ErrConflict, ...)
}
```

Status codes map to package sentinel errors. The mapping (`errors.go:13-32`) covers `ErrBadRequest`, `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrMethodNotAllowed`, `ErrConflict`, `ErrUnsupportedMediaType`, `ErrRateLimited`, `ErrServerError`. Match via `errors.Is`:

```go
_, err := gs.GetWorkspaceContext(ctx, "doesnotexist")
if errors.Is(err, geoserver.ErrNotFound) {
    // handle
}
```

The `*Error.Error()` string preserves v1.0's `"abstract:%s\ndetails:%s\n"` format so existing string-matching callers don't break.

## Logging

`g.logger` is a `*Logger` wrapper (`logging.go:11-71`) over stdlib `*slog.Logger`. The wrapper exists for v1.0 source-compat (preserves `Errorf`/`Warnf`/`Infof`/`Debugf` and `Error`/`Warn`/`Info`/`Debug` shapes from the original logrus-based API).

Configure via `New(url, u, p, WithLogger(handler))` where `handler` is any `slog.Handler`. Default is `slog.DiscardHandler` (silent).

The library logs at:

- **Debug** — HTTP details (URL, status, body length).
- **Warn** — transport-level retry exhaustion, unexpected response shapes.
- **Error** — protocol violations, deserialization failures, type-assertion mismatches.

There is no `Info` chatter.

## HTTP transport

All REST calls funnel through `(g *GeoServer).DoRequestContext` (`utils.go`). Exported, but not the recommended call surface — resource methods (e.g., `GetWorkspacesContext`) are. Internal organization in v1.1.x splits the algorithm:

- `*GeoServer.DoRequestContext` — the public method, kept for v1.0 source-compat.
- `internal/transport/transport.go` — the actual algorithm.

`*GeoServer.DoRequest` is the `context.Background()` shim around `DoRequestContext` (same twin pattern as resource methods).

## URL building

`g.ParseURL(parts ...string)` (`utils.go`) builds REST URLs from segments with two non-trivial bits of correctness:

1. **`url.PathEscape` per segment** — workspace and layer names with spaces, slashes, or non-ASCII characters produce correctly-escaped URLs. v1.0 used `fmt.Sprintf` and produced malformed URLs for these inputs.
2. **`RawPath` preservation** — the encoded path survives `(*url.URL).String()` instead of being re-encoded. Without this, a segment escaped to `%2A` would be re-encoded to `%252A` by `String()`, which GeoServer's `StrictHttpFirewall` rejects as potentially malicious. The bug surfaced when ACL `DELETE` paths started carrying literal `*` wildcards. Regression-guarded by `utils_unit_test.go` `TestParseURL_NoDoubleEncoding`.

## Concurrency

`*GeoServer` exported fields are **not safe for concurrent mutation**. Construct once with `New(...)` and treat the returned value as immutable. Concurrent reads are safe (the `*http.Client` underneath is concurrency-safe by stdlib contract; the `*Logger` is `*slog.Logger`-backed, also safe).

Concurrent reads include concurrent calls to different resource methods on the same `*GeoServer`. These are routine; the test suite exercises them via `-race`.

The "exported fields, accept reads, document non-mutability" model is a v1.0 carryover. v2 fixes this with private fields; see [`migration-v1-to-v2.md`](migration-v1-to-v2.md).

## Test split

Two test layers, distinguished by file naming and build tag:

| Layer | Naming | Build tag | What it does |
|---|---|---|---|
| Unit | `*_unit_test.go` | none | `httptest.Server` mocks; covers each method's request shape, response decode, status-code → sentinel mapping. `make test-unit` runs them in <5s. No Docker required. |
| Integration | `*_test.go` (no `_unit_` suffix) | `//go:build integration` | Real GeoServer + PostGIS via `docker compose`. Exercises end-to-end flows. `make test-integration` boots the stack first. |

This split is non-standard for Go (idiomatic Go puts unit tests in `*_test.go` next to the code, period), but the deliberate naming makes the split greppable. Both layers are mandatory on every PR; CI runs unit on `Lint` and `Unit tests (Go 1.25)`, integration on `GeoServer 2.27.4` and `GeoServer 2.28.0`.

## GeoServer REST quirks

GeoServer's REST API has version-specific quirks that this client works around. See [`geoserver-rest-quirks.md`](geoserver-rest-quirks.md) for the catalog.

## Cross-references

- [`../README.md`](../README.md) — quickstart, install, examples
- [`../ROADMAP.md`](../ROADMAP.md) — v1.x maintenance, v2.x design, GeoServer 3 timeline
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — dev setup, PR workflow, required CI checks
- [`geoserver-rest-quirks.md`](geoserver-rest-quirks.md) — GeoServer 2.x REST quirks the client handles
- [`version-compat.md`](version-compat.md) — Go × GeoServer version matrix
- [`migration-v1-to-v2.md`](migration-v1-to-v2.md) — v1 → v2 migration (in progress)
