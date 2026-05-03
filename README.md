[![Go Reference](https://pkg.go.dev/badge/github.com/hishamkaram/geoserver.svg)](https://pkg.go.dev/github.com/hishamkaram/geoserver)
[![Go Report Card](https://goreportcard.com/badge/github.com/hishamkaram/geoserver)](https://goreportcard.com/report/github.com/hishamkaram/geoserver)
[![CI](https://github.com/hishamkaram/geoserver/actions/workflows/ci.yml/badge.svg)](https://github.com/hishamkaram/geoserver/actions/workflows/ci.yml)
[![GitHub License](https://img.shields.io/github/license/hishamkaram/geoserver.svg)](https://github.com/hishamkaram/geoserver/blob/master/LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/hishamkaram/geoserver.svg)](https://github.com/hishamkaram/geoserver/stargazers)

<p align="center">
  <img src="https://i.imgur.com/bVuV5v6.png" width="200"/>
</p>
<p align="center">
  <img src="https://i.imgur.com/31CL1xg.png" width="200"/>
</p>

# geoserver

`geoserver` is a Go client library for the [GeoServer](https://geoserver.org/) REST API. Manage workspaces, datastores, layers, styles, coverages, and more from your Go applications.

> **v1.1 revival (May 2026)** — this library was dormant for 3+ years and has been revived with modern Go tooling, an idiomatic `New()` constructor with functional options, full `context.Context` support, typed errors, structured logging via stdlib `log/slog`, and a httptest-based unit-test layer. See the [CHANGELOG](CHANGELOG.md) for details. v1.0 callers can upgrade by changing only their `go.mod` version.

---

## Install

```bash
go get github.com/hishamkaram/geoserver@latest
```

```go
import "github.com/hishamkaram/geoserver"
```

> **Note**: a legacy `gopkg.in/hishamkaram/geoserver.v1` import path also resolves but is deprecated. New code should use `github.com/hishamkaram/geoserver`.

## Requirements

| Component | Supported |
|---|---|
| Go | **1.23+** (CI tests against 1.23 and 1.25) |
| GeoServer | **2.27 LTS, 2.28** (current stable) |

GeoServer 3.0 support is tracked for v2.

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
    // v1.1 idiomatic constructor with functional options.
    gs := geoserver.New(
        "http://localhost:8080/geoserver/",
        "admin",
        "geoserver",
        geoserver.WithTimeout(15*time.Second),
        geoserver.WithUserAgent("my-service/1.0"),
        geoserver.WithLogger(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})),
    )

    // Every method has a *Context twin. Use them in production.
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    created, err := gs.CreateWorkspaceContext(ctx, "golang")
    if err != nil {
        if errors.Is(err, geoserver.ErrConflict) {
            fmt.Println("workspace already exists, continuing")
        } else {
            fmt.Printf("create error: %v\n", err)
            return
        }
    }
    fmt.Printf("created=%v\n", created)

    layers, err := gs.GetLayersContext(ctx, "")
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }
    for _, l := range layers {
        fmt.Printf("Name:%s  Href:%s\n", l.Name, l.Href)
    }
}
```

### Legacy non-context API

The original v1.0 method shapes still work. They internally call the `*Context` versions with `context.Background()`:

```go
gs := geoserver.GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
created, err := gs.CreateWorkspace("golang") // == gs.CreateWorkspaceContext(context.Background(), "golang")
```

`GetCatalog` is deprecated in favor of `New`.

### Typed errors

REST failures return a typed `*geoserver.Error` you can match against sentinel values:

```go
_, err := gs.GetWorkspaceContext(ctx, "missing")

if errors.Is(err, geoserver.ErrNotFound) {
    fmt.Println("workspace missing")
}

var apiErr *geoserver.Error
if errors.As(err, &apiErr) {
    fmt.Printf("status=%d body=%s\n", apiErr.StatusCode, apiErr.Body)
}
```

Available sentinels: `ErrNotFound`, `ErrUnauthorized`, `ErrForbidden`, `ErrConflict`, `ErrBadRequest`, `ErrMethodNotAllowed`, `ErrUnsupportedMediaType`, `ErrRateLimited`, `ErrServerError` (any 5xx).

The error's `Error()` string preserves the v1.0 `"abstract:%s\ndetails:%s\n"` format for callers that pattern-match on text.

## Concurrency

`*GeoServer` is safe for concurrent **reads** (calling methods from multiple goroutines is fine). Mutating exported fields after construction is **not safe**. Construct once via `New(...)` and treat the value as read-only thereafter. A v2 redesign with private fields and an immutable client is planned.

## Testing

This package ships two test layers:

### Unit tests (no Docker required)

```bash
make test-unit
```

Runs `go test -race -short ./...`. Uses `httptest.NewServer` to mock GeoServer responses. Covers happy paths plus 401/403/404/409/500 error mapping for the implemented services.

### Integration tests (real GeoServer)

```bash
make compose-up        # boots GeoServer 2.28 + PostGIS 16
make test-integration  # runs go test -tags=integration ./...
make compose-down
```

CI runs the integration suite against **GeoServer 2.27 LTS** and **2.28** in parallel. To target a specific version locally:

```bash
GEOSERVER_VERSION=2.27.4 make compose-up
```

## Documentation

- Full API: [pkg.go.dev/github.com/hishamkaram/geoserver](https://pkg.go.dev/github.com/hishamkaram/geoserver)
- Detailed examples: see the integration tests under `*_test.go`.
- GeoServer REST API itself: [docs.geoserver.org/stable/en/user/rest/](https://docs.geoserver.org/stable/en/user/rest/)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for dev setup, Conventional Commits convention, and PR checklist. Security issues should be reported privately per [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE)
