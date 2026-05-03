# geoserver

A Go client library for the [GeoServer](https://geoserver.org/) REST API. Manage workspaces, datastores, feature types, layers, layer groups, styles, coverages, and namespaces from any Go application.

[![Go Reference](https://pkg.go.dev/badge/github.com/hishamkaram/geoserver.svg)](https://pkg.go.dev/github.com/hishamkaram/geoserver)
[![Go Report Card](https://goreportcard.com/badge/github.com/hishamkaram/geoserver)](https://goreportcard.com/report/github.com/hishamkaram/geoserver)
[![CI](https://github.com/hishamkaram/geoserver/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/hishamkaram/geoserver/actions/workflows/ci.yml)
[![Integration tests](https://github.com/hishamkaram/geoserver/actions/workflows/integration.yml/badge.svg?branch=master)](https://github.com/hishamkaram/geoserver/actions/workflows/integration.yml)
[![CodeQL](https://github.com/hishamkaram/geoserver/actions/workflows/codeql.yml/badge.svg?branch=master)](https://github.com/hishamkaram/geoserver/actions/workflows/codeql.yml)
[![License: MIT](https://img.shields.io/github/license/hishamkaram/geoserver.svg)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/hishamkaram/geoserver?include_prereleases&sort=semver)](https://github.com/hishamkaram/geoserver/releases)

---

## Highlights

- **Idiomatic 2026-era Go API** — `New(...)` with functional options, `context.Context` on every method, typed errors usable with `errors.Is` / `errors.As`.
- **Drop-in for v1.0 callers** — every legacy method shape (`GetCatalog`, `CreateWorkspace`, etc.) still works, just delegates to a context-aware sibling.
- **Structured logging via stdlib `log/slog`** — no third-party logger dependency; silent by default, fully pluggable through `WithLogger`.
- **Verified against real GeoServer** — every release tag triggers an integration matrix that spins up GeoServer 2.27 LTS and 2.28 stable in Docker and runs the full test suite.
- **Zero runtime third-party dependencies** — only stdlib `net/http`, `encoding/json`, `encoding/xml`, `log/slog`, `context`. (Test-only deps are testify and `gopkg.in/yaml.v3` for the YAML config helper.)

## Compatibility

| Component   | Supported                                       |
|-------------|-------------------------------------------------|
| Go          | **1.25+** (matches `go.mod`; toolchain auto-pulls go1.25.9 for the patched stdlib CVEs) |
| GeoServer   | **2.27 LTS** and **2.28** (current stable)      |
| GeoServer 3 | Tracked for v2 — see [Roadmap](#roadmap)        |

## Install

```bash
go get github.com/hishamkaram/geoserver@latest
```

```go
import "github.com/hishamkaram/geoserver"
```

> The legacy `gopkg.in/hishamkaram/geoserver.v1` import path still resolves but is deprecated. New code should use `github.com/hishamkaram/geoserver`.

## Quick start

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log/slog"
    "os"
    "time"

    "github.com/hishamkaram/geoserver"
)

func main() {
    gs := geoserver.New(
        "http://localhost:8080/geoserver/",
        "admin",
        "geoserver",
        geoserver.WithTimeout(15*time.Second),
        geoserver.WithUserAgent("my-service/1.0"),
        geoserver.WithLogger(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})),
    )

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Create a workspace, tolerating "already exists".
    if _, err := gs.CreateWorkspaceContext(ctx, "demo"); err != nil {
        if !errors.Is(err, geoserver.ErrConflict) {
            fmt.Println("create workspace:", err)
            return
        }
    }

    // List public layers.
    layers, err := gs.GetLayersContext(ctx, "")
    if err != nil {
        fmt.Println("list layers:", err)
        return
    }
    for _, l := range layers {
        fmt.Printf("%-30s %s\n", l.Name, l.Href)
    }
}
```

## Constructing a client

`New` is the v1.1+ entry point. It accepts functional options for everything you would normally tune on an HTTP client:

| Option                      | Default                              | Notes                                                                                                |
|-----------------------------|--------------------------------------|------------------------------------------------------------------------------------------------------|
| `WithHTTPClient(*http.Client)` | `&http.Client{Timeout: 30s}`     | Replace the entire client (e.g. instrumented transports, retries, custom auth).                      |
| `WithTimeout(d)`            | `30s`                                | Override the timeout on the underlying `http.Client`. Per-request deadlines should use `context`.    |
| `WithLogger(slog.Handler)`  | text handler at Info, stderr         | Pass `nil` to silence the library entirely.                                                          |
| `WithUserAgent(string)`     | Go's default                         | Wraps the transport with a `User-Agent`-setting `RoundTripper`; safe to layer over `WithHTTPClient`. |
| `WithBasicAuth(user, pass)` | from constructor args                | Chainable credential override.                                                                       |

The legacy `geoserver.GetCatalog(url, user, pass)` still works (it delegates to `New`) and is retained for v1.0 compatibility. It is marked `Deprecated` in godoc.

## Context propagation

Every public method has a `*Context` sibling that takes a `context.Context` as its first argument. The non-context name remains for v1.0 compatibility and is implemented as a one-line wrapper that calls the `*Context` version with `context.Background()`.

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

// Modern (recommended):
ws, err := gs.GetWorkspaceContext(ctx, "demo")

// Legacy (still works):
ws, err := gs.GetWorkspace("demo")
```

The `CatalogWithContext` interface bundles the context-aware service interfaces (`WorkspaceServiceWithContext`, `LayerServiceWithContext`, etc.) for callers that want to depend on an interface rather than `*GeoServer`.

## Typed errors

REST failures return a typed `*geoserver.Error` matchable against package sentinels:

```go
_, err := gs.GetWorkspaceContext(ctx, "missing")

if errors.Is(err, geoserver.ErrNotFound) {
    fmt.Println("workspace doesn't exist")
}

var apiErr *geoserver.Error
if errors.As(err, &apiErr) {
    fmt.Printf("status=%d url=%s body=%s\n",
        apiErr.StatusCode, apiErr.URL, apiErr.Body)
}
```

| Sentinel                  | Matches HTTP status |
|---------------------------|---------------------|
| `ErrBadRequest`           | 400                 |
| `ErrUnauthorized`         | 401                 |
| `ErrForbidden`            | 403                 |
| `ErrNotFound`             | 404                 |
| `ErrMethodNotAllowed`     | 405                 |
| `ErrConflict`             | 409                 |
| `ErrUnsupportedMediaType` | 415                 |
| `ErrRateLimited`          | 429                 |
| `ErrServerError`          | any 5xx             |

For v1.0 compatibility, `Error.Error()` preserves the historical `"abstract:%s\ndetails:%s\n"` text — any code that previously matched on error message strings continues to work.

## Logging

The library logs through stdlib [`log/slog`](https://pkg.go.dev/log/slog). It is silent by default unless you opt in:

```go
gs := geoserver.New(url, user, pass,
    geoserver.WithLogger(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    })),
)
```

| Level   | What you'll see                                       |
|---------|-------------------------------------------------------|
| Debug   | (reserved for future request-level traces)            |
| Info    | One line per outgoing request: URL + status           |
| Warn    | Non-fatal anomalies (e.g. ignored remote field shape) |
| Error   | Protocol failures, decode errors, transport errors    |

Pass `nil` to `WithLogger` to drop everything to a discard handler.

## API surface

The most common operations, grouped by resource. See [pkg.go.dev](https://pkg.go.dev/github.com/hishamkaram/geoserver) for the complete reference.

### Workspaces

```go
gs.GetWorkspacesContext(ctx)
gs.GetWorkspaceContext(ctx, "topp")
gs.WorkspaceExistsContext(ctx, "topp")
gs.CreateWorkspaceContext(ctx, "topp")
gs.DeleteWorkspaceContext(ctx, "topp", true /*recurse*/)
```

### Namespaces

```go
gs.GetNamespacesContext(ctx)
gs.CreateNamespaceContext(ctx, "topp", "http://www.openplans.org/topp")
gs.DeleteNamespaceContext(ctx, "topp")
```

### Datastores

```go
conn := geoserver.DatastoreConnection{
    Name:   "ny_postgis",
    Host:   "postgis", Port: 5432,
    DBName: "gis", DBUser: "golang", DBPass: "golang",
    Type:   "postgis",
}
gs.CreateDatastoreContext(ctx, conn, "topp")
gs.GetDatastoreDetailsContext(ctx, "topp", "ny_postgis")
gs.DeleteDatastoreContext(ctx, "topp", "ny_postgis", true)
```

### Feature types

```go
gs.GetFeatureTypesContext(ctx, "topp", "ny_postgis")
gs.GetFeatureTypeContext(ctx, "topp", "ny_postgis", "buildings")
gs.DeleteFeatureTypeContext(ctx, "topp", "ny_postgis", "buildings", true)
```

### Layers

```go
gs.GetLayersContext(ctx, "")                    // global, unscoped
gs.GetLayerContext(ctx, "topp", "states")
gs.UpdateLayerContext(ctx, "topp", "states", layer)
gs.DeleteLayerContext(ctx, "topp", "states", true)
gs.PublishPostgisLayerContext(ctx, "topp", "ny_postgis", "buildings", "buildings_table")
gs.UploadShapeFileContext(ctx, "/path/states.zip", "topp", "states")
```

### Layer groups

```go
gs.GetLayerGroupsContext(ctx, "")
gs.GetLayerGroupContext(ctx, "", "tiger-ny")
gs.CreateLayerGroupContext(ctx, "topp", &group)
gs.DeleteLayerGroupContext(ctx, "topp", "tiger-ny")
```

### Styles

```go
gs.GetStylesContext(ctx, "topp")
gs.GetStyleContext(ctx, "topp", "states_style")
gs.CreateStyleContext(ctx, "topp", "states_style")
gs.UploadStyleContext(ctx, sldReader, "topp", "states_style", false /*overwrite*/)
gs.DeleteStyleContext(ctx, "topp", "states_style", true /*purge*/)
```

### Coverages (raster layers)

```go
gs.GetCoveragesContext(ctx, "nurc")
gs.GetCoverageContext(ctx, "nurc", "Arc_Sample")
gs.PublishCoverageContext(ctx, "nurc", "arcGridSample", "Arc_Sample", "")
gs.UpdateCoverageContext(ctx, "nurc", &coverage)
```

### Settings & misc

```go
gs.GetGlobalSettingsContext(ctx)
gs.UpdateGlobalSettingsContext(ctx, settings)
gs.IsRunningContext(ctx)
gs.GetCapabilitiesContext(ctx, "")
```

## Concurrency

`*GeoServer` is safe for **concurrent reads** — calling its methods from multiple goroutines simultaneously is fine. Mutating its exported fields (`ServerURL`, `Username`, etc.) after construction is **not safe**. The recommended pattern is to construct once via `New(...)` and treat the returned value as read-only thereafter.

A v2 redesign with private fields, an immutable client, and `RoundTripper`-based auth is on the [Roadmap](#roadmap).

## Testing

The package ships two test layers:

### Unit tests — no Docker required

```bash
make test-unit
```

Runs `go test -race -short ./...`. Uses `httptest.NewServer` to mock GeoServer responses; covers happy paths plus 401/403/404/409/500 error mapping for the implemented services. Suitable for editor save-on-test or any CI pipeline.

### Integration tests — real GeoServer in Docker

```bash
make compose-up        # boots GeoServer 2.28 + PostGIS 16 with seeded data
make test-integration  # runs go test -tags=integration ./...
make compose-down
```

To target the LTS leg locally:

```bash
GEOSERVER_VERSION=2.27.4 make compose-up
```

In CI, the integration suite runs on every release tag (`v*.*.*`) and on a weekly schedule (Sun 03:17 UTC) against both **GeoServer 2.27.4 LTS** and **2.28.0 stable** in parallel. Manual dispatch is also available via the GitHub Actions UI.

## Project status

| Stream      | Status                                                                      |
|-------------|-----------------------------------------------------------------------------|
| Daily CI    | Lint, unit tests on Go 1.25, govulncheck, CodeQL — all green                 |
| Integration | GeoServer 2.27 LTS + 2.28 stable, run on every PR — all green                |
| Latest tag  | See [Releases](https://github.com/hishamkaram/geoserver/releases)           |

The library was dormant for ~3 years (Feb 2023 → May 2026) before being revived as **v1.1**. The revival kept the v1.0 public surface intact: every existing method shape still compiles and behaves the same way, just with bug fixes underneath and modern idiomatic siblings beside it. See [CHANGELOG.md](CHANGELOG.md) for the full breakdown.

## Roadmap

- **v1.1.x** — security fixes, integration test maintenance, additive Dependabot updates.
- **v2** — pre-alpha at `github.com/hishamkaram/geoserver/v2`. The full catalog (workspaces, datastores, feature types, coverage stores, coverages, layers, layer groups, styles, namespaces) plus security, ACL, settings, and about have been ported and unit-tested; only WMS GetCapabilities remains pending (deferred to a v2.x point release after the `ows/wms/` XML subpackage lands).

  Design themes (now realized in code):
  - Resource sub-clients (`c.Workspaces.Get(ctx, name)`, `c.Datastores.InWorkspace(ws).Get(...)`, `c.FeatureTypes.InWorkspace(ws).InDatastore(ds).Get(...)`).
  - Immutable `Client` with `RoundTripper`-based auth (concurrency-safe by construction).
  - Mandatory `context.Context`, no `Background` shims.
  - Range-over-func iterators (`iter.Seq2`) on every list endpoint.
  - Single error type (`*APIError`) with the same sentinel set as v1 (`ErrNotFound`, `ErrConflict`, …).
  - Zero runtime third-party deps — stdlib only.
  - Raw-body uploads (`UploadSLD`) with the workspace-scoped Accept-quirk handled automatically.

  Tagging is held until soak / API review; until then v2 is preview-quality. **For production code today, use v1.x.** See [v2/README.md](v2/README.md), [v2/examples/](v2/examples/), and [docs/migration-v1-to-v2.md](docs/migration-v1-to-v2.md).
- **GeoServer 3.0 support** — tracked as a v2.x point release once Tomcat 11 / Jakarta EE / ImageN settle. See [ROADMAP.md](ROADMAP.md).

## Documentation

- **API reference** — [pkg.go.dev/github.com/hishamkaram/geoserver](https://pkg.go.dev/github.com/hishamkaram/geoserver)
- **Architecture** — [docs/architecture.md](docs/architecture.md): package shape, the *Context twin pattern, error model, logging, concurrency, transport organization
- **GeoServer REST quirks** — [docs/geoserver-rest-quirks.md](docs/geoserver-rest-quirks.md): version-specific REST API quirks the client works around
- **Version compatibility** — [docs/version-compat.md](docs/version-compat.md): supported Go × GeoServer matrix with rationale
- **v1 → v2 migration** — [docs/migration-v1-to-v2.md](docs/migration-v1-to-v2.md): per-resource v1-method → v2-sub-client mapping tables, side-by-side workflow example. v2 lives at `github.com/hishamkaram/geoserver/v2`.
- **v2 README** — [v2/README.md](v2/README.md): design tenets, quick start for global / workspace-scoped / 2-level-scoped resources, resource status table.
- **v2 runnable examples** — [v2/examples/](v2/examples/): workspaces, publish-postgis, style-upload, error-handling. Run with `go run ./v2/examples/<name>` or compile-check via `make examples-v2`.
- **Roadmap** — [ROADMAP.md](ROADMAP.md): v1.x maintenance, v2.x design, GeoServer 3.0 timeline
- **Runnable examples** — [examples/](examples/): workspaces, publish-postgis, style-upload, error-handling. Run with `go run ./examples/<name>` against a `make compose-up` stack.
- **Reference flows in tests** — the integration suite (`*_test.go` under build tag `integration`) exercises the full surface end-to-end against GeoServer 2.27 LTS and 2.28 stable.
- **GeoServer REST itself** — [docs.geoserver.org/stable/en/user/rest](https://docs.geoserver.org/stable/en/user/rest/)

## Contributing

Pull requests welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) for the dev setup, [Conventional Commits](https://www.conventionalcommits.org/) convention, and the PR checklist.

For security issues, see [SECURITY.md](SECURITY.md) and use the private GitHub Security Advisory channel — please do not open a public issue.

By participating in this project you agree to abide by the [Code of Conduct](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE) © Hesham Karm and contributors.
