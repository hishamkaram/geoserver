# Architecture

This document describes how `github.com/hishamkaram/geoserver/v2` is laid out, the design tenets, and the trade-offs behind v2's shape. It is reference material; release notes live in [`../CHANGELOG.md`](../CHANGELOG.md), and the forward-looking direction lives in [`../ROADMAP.md`](../ROADMAP.md).

## Package shape

The library is rooted at `package geoserver` (the public client surface) plus one resource-client subpackage per REST resource and one OWS subpackage per service. The HTTP transport is hidden in `internal/`.

| Path | Purpose |
|---|---|
| `github.com/hishamkaram/geoserver/v2` | Public surface — `*Client`, options, `*APIError`, sentinel errors. The constructor lives here; the resource methods live in their per-resource subpackages, surfaced via exported fields on `*Client`. |
| `github.com/hishamkaram/geoserver/v2/rest/<resource>` | One subpackage per REST resource: `workspaces`, `datastores`, `featuretypes`, `coveragestores`, `coverages`, `layers`, `layergroups`, `styles`, `namespaces`, `settings`, `about`, `security`, `acl`, `system`, `imports`, `gwc`, `services`, `resources`, `templates`, `urlchecks`, `wmsstores`, `wmslayers`, `wmtsstores`, `wmtslayers`, `wfstransforms`, `logging`, `fonts`, `monitor`. Each exposes a `*Client` (and where applicable scoped `*WorkspaceClient`, `*DatastoreClient`, etc.). |
| `github.com/hishamkaram/geoserver/v2/ows/{wms,wfs,wcs}` | OWS read-only clients: `GetCapabilities` + `DescribeFeatureType` (WFS) / `DescribeCoverage` (WCS). Separate from `rest/services` because OWS endpoints are XML-over-HTTP and live at different URL roots. |
| `github.com/hishamkaram/geoserver/v2/internal/transport` | HTTP request building, URL construction, JSON/XML/raw/stream dispatch. Implementation detail — not importable by external code. |
| `github.com/hishamkaram/geoserver/v2/internal/wire` | Internal helpers for the more delicate wire-format quirks (mixed-shape arrays, empty-collection string-vs-object payloads). Not importable. |

v1 is a separate, end-of-feature release line on the `release/v1` branch (security fixes only; latest tag `v1.1.2`). See [`migration-v1-to-v2.md`](./migration-v1-to-v2.md) for the side-by-side mapping.

## Public API entry points

There is **one** type users construct: `*Client`. The constructor is functional-options-only:

```go
func New(serverURL string, opts ...Option) (*Client, error)
```

Options live in `options.go`: `WithHTTPClient`, `WithTransport`, `WithTimeout`, `WithLogger`, `WithUserAgent`, `WithBasicAuth`, `WithBearerToken`, `WithHeader`. Credentials are passed through options, not positional args — that's the v2 break with v1's `New(url, user, pass, opts...)` shape.

`*Client` exposes 31 sub-clients as exported pointer fields (`c.Workspaces`, `c.Datastores`, `c.FeatureTypes`, `c.Styles`, …). Sub-client methods do the actual REST work; the parent `*Client` only constructs and owns them.

Hierarchical resources scope through fluent `In*` methods that return a new lightweight scoped client:

```go
// Datastore scoped to a workspace
c.Datastores.InWorkspace("topp").Create(ctx, datastores.PostGIS{...})

// Feature type scoped to a workspace + datastore
c.FeatureTypes.InWorkspace("topp").InDatastore("nyc").Create(ctx, ft)

// Coverage scoped to a workspace + coverage store
c.Coverages.InWorkspace("nurc").InCoverageStore("dem").Get(ctx, "elev")
```

Scoped clients are immutable value-shaped wrappers around the parent's `Core` interface — cheap to allocate, safe to discard.

## Context-first methods

Every exported method on every sub-client takes `ctx context.Context` as its first argument. There are **no** `*Context` twin methods, and no `context.Background()` shims; v1's twin pattern was a v1.0 source-compat affordance and was deliberately dropped at the v2 boundary.

```go
func (c *Client) GetWorkspaces(ctx context.Context) ([]*workspaces.Workspace, error)
```

If a caller has no context, they pass `context.Background()` at the call site. The library itself never does.

## Errors

Every non-2xx GeoServer response surfaces as `*APIError` (`errors.go`):

```go
type APIError struct {
    Op         string  // operation name, e.g. "Workspaces.Create"
    Method     string  // HTTP method
    URL        string  // request URL
    StatusCode int     // HTTP status
    Body       []byte  // response body, capped at 8 KiB internally
}
```

`*APIError.Is(target)` matches a fixed set of 12 sentinels: `ErrBadRequest`, `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrMethodNotAllowed`, `ErrConflict`, `ErrUnsupportedMediaType`, `ErrRateLimited`, `ErrServerError`, `ErrBadGateway`, `ErrServiceUnavailable`, `ErrGatewayTimeout`. Match via `errors.Is`:

```go
_, err := c.Workspaces.Get(ctx, "doesnotexist")
if errors.Is(err, geoserver.ErrNotFound) {
    // handle
}
```

The `Error()` string is a stable, parseable form:

```
geoserver: <Op> <Method> <URL>: <statusCode> <statusText>: <body-preview>
```

Body preview is truncated to ~120 bytes. Don't parse error strings for control flow — use `errors.Is` (sentinel) or `errors.As` (inspect fields).

The v2 type rename (`*Error` → `*APIError`) and the format break (v1's `"abstract:%s\ndetails:%s\n"` is gone) are deliberate v2-boundary changes; v1.x callers cannot port unmodified.

## Logging

The library uses `*slog.Logger` directly. There is no `*Logger` wrapper — that abstraction was a v1.0 source-compat shim and was dropped in v2.

Configure via `WithLogger(l *slog.Logger)`. Default is `slog.New(slog.DiscardHandler)` (silent). Internal call sites use structured logging — `logger.Debug(msg, args...)` with key/value pairs, not printf-style.

The library logs at:

- **Debug** — HTTP details (URL, status, body length).
- **Warn** — transport-level retry exhaustion, unexpected response shapes.
- **Error** — protocol violations, deserialization failures.

There is no `Info` chatter.

## HTTP transport

Every sub-client call funnels through `coreAdapter.Do(ctx, op, method, url, body, query, out)` (`geoserver.go:429`), which delegates to `transport.DoJSON / DoXML / DoRaw / DoStream` in `internal/transport/transport.go`. Sub-clients never touch `*http.Client.Do` directly.

The `coreAdapter` is the bridge between resource-client subpackages and the private `clientCore` (configured `*http.Client`, base URL, headers, logger). Sub-clients consume only the `Core` interface, not `*Client` itself, so they can be composed and unit-tested in isolation.

URL building goes through `coreAdapter.URL(parts ...string)` (`geoserver.go:422`), which delegates to `transport.BuildURL` (`internal/transport/url.go`). Each segment is path-escaped and `RawPath` is preserved, so workspace/layer names with spaces, slashes, or non-ASCII characters produce correctly-escaped URLs that survive `(*url.URL).String()` without double-encoding. Regression-guarded by `internal/transport/url_test.go`.

## Concurrency

`*Client` is **immutable after `New(...)` returns.** All struct fields are private or pointers to sub-clients set once at construction and never reassigned. Concurrent use across goroutines is safe by design — no caller-side locking required.

The same posture holds for every sub-client (`workspaces.Client`, `datastores.Client`, …): they expose methods only, holding a single private `Core` interface reference. Scoped clients (`*WorkspaceClient`, `*DatastoreClient`, …) are value-shaped and similarly stateless.

The race-safety guarantee is verified by `TestClient_ConcurrentRequests` (`geoserver_concurrent_test.go:17`) running under `go test -race` in CI.

User-supplied transports passed via `WithHTTPClient` / `WithTransport` are the caller's responsibility — if their `RoundTripper` mutates shared state, the race lives in their code.

## Test split

Two test layers, distinguished by file naming and build tag:

| Layer | Naming | Build tag | What it does |
|---|---|---|---|
| Unit | `*_unit_test.go` | none | `httptest.Server` fakes; covers each method's request shape, response decode, status-code → sentinel mapping. `make test-unit` runs them in <5s, no Docker required. |
| Integration | `*_integration_test.go` | `//go:build integration` | Real GeoServer + PostGIS via `docker compose`. Exercises end-to-end flows. `make test-integration` boots the stack first. |

Both layers are mandatory on every PR. CI runs unit on **Go 1.23 + 1.25** and integration on **GeoServer 2.27.4 LTS + 2.28.0 stable** — all four legs must go green.

## GeoServer REST quirks

GeoServer's REST API has version-specific quirks the client works around (mixed-shape JSON arrays, empty-collection string-vs-object payloads, workspace-scoped style-endpoint Accept-header dispatch, etc.). See [`geoserver-rest-quirks.md`](geoserver-rest-quirks.md) for the catalog with code locations.

## Cross-references

- [`../README.md`](../README.md) — quickstart, install, capability surface, worked example
- [`../ROADMAP.md`](../ROADMAP.md) — v1 maintenance window, v2 evolution, GeoServer 3 timeline
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — dev setup, PR workflow, required CI checks
- [`migration-v1-to-v2.md`](migration-v1-to-v2.md) — v1 → v2 migration mapping
- [`version-compat.md`](version-compat.md) — Go × GeoServer support matrix
- [`geoserver-rest-quirks.md`](geoserver-rest-quirks.md) — wire-format quirks the client handles
