# geoserver

A Go client for the GeoServer REST API.

[![Go Reference](https://pkg.go.dev/badge/github.com/hishamkaram/geoserver/v2.svg)](https://pkg.go.dev/github.com/hishamkaram/geoserver/v2)
[![CI](https://github.com/hishamkaram/geoserver/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/hishamkaram/geoserver/actions/workflows/ci.yml)
[![Integration tests](https://github.com/hishamkaram/geoserver/actions/workflows/integration.yml/badge.svg?branch=master)](https://github.com/hishamkaram/geoserver/actions/workflows/integration.yml)
[![License: MIT](https://img.shields.io/github/license/hishamkaram/geoserver.svg)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/hishamkaram/geoserver?sort=semver)](https://github.com/hishamkaram/geoserver/releases)

> Provision workspaces, register PostGIS / GeoTIFF / Shapefile data sources, publish layers, manage styles, configure security, drive GeoWebCache, and run the Importer extension — all from idiomatic Go, with mandatory `context.Context`, typed errors, and zero runtime third-party dependencies.

## Contents

- [What is GeoServer?](#what-is-geoserver)
- [What this client does](#what-this-client-does)
- [Install](#install)
- [Quick start](#quick-start)
- [Worked example: publish a PostGIS layer](#worked-example-publish-a-postgis-layer)
- [Errors](#errors)
- [Authentication & advanced configuration](#authentication--advanced-configuration)
- [Examples & further reading](#examples--further-reading)
- [Version compatibility](#version-compatibility)
- [Contributing](#contributing)

## What is GeoServer?

[GeoServer](https://geoserver.org/) is an open-source Java server that publishes geographic data via the OGC web standards: WMS (rendered map images), WFS (vector features), WCS (raster coverages), and WMTS (pre-rendered tiles). It's deployed by mapping companies, government agencies, and GIS teams to put PostGIS tables, GeoTIFFs, Shapefiles, and remote services behind a single web-services endpoint.

This package is for Go programs that need to **provision or operate** a GeoServer — typically pipelines that publish new data, infrastructure code that manages workspaces and security, or back-office tools that drive the Importer or seed GeoWebCache.

## What this client does

The client surface is broken into typed sub-clients on `*geoserver.Client`. Each bullet below names what you'd accomplish; the trailing fields are the entry points.

- **Catalog & publishing** — workspaces, datastores, feature types, coverage stores, coverages, layers, layer groups, styles, namespaces; file-upload publishing for Shapefile / GeoPackage / GeoTIFF / mosaic granules; layer–style associations.
  *Entry points:* `c.Workspaces`, `c.Datastores`, `c.FeatureTypes`, `c.CoverageStores`, `c.Coverages`, `c.Layers`, `c.LayerGroups`, `c.Styles`, `c.Namespaces`.
- **OGC services** — per-service WMS / WFS / WCS / WMTS configuration (global + per-workspace overrides), `GetCapabilities`, `DescribeFeatureType`, `DescribeCoverage`, cascaded WMS/WMTS stores + layers, WFS XSLT transforms.
  *Entry points:* `c.Services.WMS()` / `WFS()` / `WCS()` / `WMTS()`, `c.WMS`, `c.WFS`, `c.WCS`, `c.WMSStores`, `c.WMSLayers`, `c.WMTSStores`, `c.WMTSLayers`, `c.WFSTransforms`.
- **Tile caching** — GeoWebCache layer config, seed / reseed / truncate, disk quota, gridsets, mass-truncate, global GWC settings.
  *Entry point:* `c.GWC.Layers()` / `Seed()` / `DiskQuota()` / `Global()` / `Gridsets()` / `MassTruncate()`.
- **Security** — users, groups, roles, full ACL surface (layers, services, REST, catalog), auth providers / filters / chains, URL checks (SSRF allow-list), master & self password rotation.
  *Entry points:* `c.Security`, `c.ACL.Layers()` / `Services()` / `REST()` / `Catalog()`, `c.URLChecks`.
- **Operations** — system reload, cache reset, runtime logging, monitoring (`gs-monitor`), manifests, system status, fonts, global settings.
  *Entry points:* `c.System`, `c.Settings`, `c.About`, `c.Logging`, `c.Monitor`, `c.Fonts`.
- **Data plane & extras** — Resource API (read/write any file under the data dir), FTL templates, Importer extension (batch ingest).
  *Entry points:* `c.Resources`, `c.Templates`, `c.Imports`.

The full method-level reference lives at [pkg.go.dev](https://pkg.go.dev/github.com/hishamkaram/geoserver/v2).

## Install

```bash
go get github.com/hishamkaram/geoserver/v2@latest
```

```go
import geoserver "github.com/hishamkaram/geoserver/v2"
```

Requirements: Go 1.25+. Tested against GeoServer 2.27.4 LTS and 2.28.0 stable on every PR.

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

    geoserver "github.com/hishamkaram/geoserver/v2"
    "github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

func main() {
    c, err := geoserver.New("http://localhost:8080/geoserver/",
        geoserver.WithBasicAuth("admin", "geoserver"),
        geoserver.WithTimeout(10*time.Second),
        geoserver.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, nil))),
    )
    if err != nil {
        panic(err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    list, err := c.Workspaces.List(ctx, workspaces.ListOptions{})
    if err != nil {
        panic(err)
    }
    for _, ws := range list {
        fmt.Println(ws.Name)
    }

    _, err = c.Workspaces.Get(ctx, "doesnotexist")
    if errors.Is(err, geoserver.ErrNotFound) {
        fmt.Println("workspace doesn't exist")
    }
}
```

`*Client` is immutable after construction and safe for concurrent use across goroutines. Sub-clients hold no shared mutable state of their own.

## Worked example: publish a PostGIS layer

The canonical end-to-end flow — create a workspace, register a PostGIS datastore, publish a feature type, fetch the auto-created layer back. Compile-checked from [`examples/publish-postgis/main.go`](examples/publish-postgis/main.go); run it with `go run ./examples/publish-postgis` against a `make compose-up` stack.

```go
import (
    geoserver "github.com/hishamkaram/geoserver/v2"
    "github.com/hishamkaram/geoserver/v2/rest/datastores"
    "github.com/hishamkaram/geoserver/v2/rest/featuretypes"
    "github.com/hishamkaram/geoserver/v2/rest/workspaces"
)

// 1. Workspace.
_ = c.Workspaces.Create(ctx, &workspaces.Workspace{Name: "demo"})

// 2. PostGIS datastore (workspace-scoped via InWorkspace).
ds := c.Datastores.InWorkspace("demo")
_ = ds.Create(ctx, datastores.PostGIS{
    Name:     "lbldyt_pg",
    Host:     "postgis",
    Port:     5432,
    Database: "geoserver",
    User:     "postgres",
    Password: "postgres",
})

// 3. 2-level scope: feature types live under (workspace, datastore).
ft := c.FeatureTypes.InWorkspace("demo").InDatastore("lbldyt_pg")

// 4. Discover available tables not yet configured.
available, _ := ft.Discover(ctx, featuretypes.DiscoverOptions{
    Kind: featuretypes.DiscoverAvailableWithGeometry,
})
fmt.Println("available:", available)

// 5. Publish one as a feature type — NativeName must match the DB table.
_ = ft.Create(ctx, &featuretypes.FeatureType{
    Name: "lbldyt", NativeName: "lbldyt",
    SRS: "EPSG:4326", Enabled: true,
})

// 6. Fetch the auto-created layer.
layer, _ := c.Layers.InWorkspace("demo").Get(ctx, "lbldyt")
fmt.Printf("layer: %s (queryable=%t)\n", layer.Name, layer.Queryable)
```

The same `InWorkspace(...).In<Parent>(...)` pattern applies to coverages under a coverage store (see [`rest/coverages/`](rest/coverages/)).

## Errors

Every non-2xx GeoServer response surfaces as `*geoserver.APIError` and wraps one of twelve package sentinels — `ErrBadRequest`, `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrMethodNotAllowed`, `ErrConflict`, `ErrUnsupportedMediaType`, `ErrRateLimited`, `ErrServerError`, `ErrBadGateway`, `ErrServiceUnavailable`, `ErrGatewayTimeout`. Match with `errors.Is`; inspect status code, op, method, URL, and the (8 KiB-capped) response body via `errors.As`.

```go
var apiErr *geoserver.APIError
switch {
case errors.Is(err, geoserver.ErrNotFound):
    // 404 — workspace, layer, etc. not present.
case errors.Is(err, geoserver.ErrConflict):
    // 409 — resource already exists.
case errors.As(err, &apiErr):
    log.Printf("op=%s status=%d body=%s", apiErr.Op, apiErr.StatusCode, apiErr.Body)
}
```

Never compare error strings — only `errors.Is` / `errors.As` are supported. See [`examples/error-handling/main.go`](examples/error-handling/main.go) for a runnable demo of all twelve sentinels.

## Authentication & advanced configuration

All transport-level concerns are configured at construction via functional options. The constructor returns immediately; nothing in `*Client` is mutable after `New` returns.

```go
c, _ := geoserver.New(serverURL,
    // Auth — pick one (or compose your own RoundTripper):
    geoserver.WithBasicAuth("admin", "geoserver"),
    geoserver.WithBearerToken(os.Getenv("GS_TOKEN")),

    // Transport / observability:
    geoserver.WithHTTPClient(myClient),                 // bring your own *http.Client
    geoserver.WithTransport(otelhttp.NewTransport(...)), // custom http.RoundTripper
    geoserver.WithTimeout(30 * time.Second),
    geoserver.WithUserAgent("my-pipeline/1.4"),
    geoserver.WithHeader("X-Trace-Id", traceID),

    // Logging — defaults to slog.DiscardHandler.
    geoserver.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, nil))),
)
```

`WithTransport` is the natural integration point for OpenTelemetry, Vault-rotated credentials, and retry libraries — anything that satisfies `http.RoundTripper` composes without re-authenticating per call.

**Pagination.** Every `List` endpoint has a sibling `Iter` returning `iter.Seq2[T, error]` for range-over-func streaming:

```go
for ws, err := range c.Workspaces.Iter(ctx, workspaces.ListOptions{}) {
    if err != nil {
        return err
    }
    fmt.Println(ws.Name)
}
```

## Examples & further reading

Self-contained `main` packages under [`examples/`](examples/):

- [`workspaces/`](examples/workspaces/) — flat sub-client CRUD; `errors.Is` matching; `Iter` range-over-func.
- [`publish-postgis/`](examples/publish-postgis/) — end-to-end workspace → datastore → feature type → layer flow.
- [`style-upload/`](examples/style-upload/) — two-step style publish (`Create` metadata + `UploadSLD` body).
- [`error-handling/`](examples/error-handling/) — every sentinel + `*APIError` inspection via `errors.As`.

Run any with `go run ./examples/<name>` against a `make compose-up` stack, or compile-check all with `make examples`.

Reference docs:

- [pkg.go.dev godoc](https://pkg.go.dev/github.com/hishamkaram/geoserver/v2) — full method-level API reference.
- [`docs/architecture.md`](docs/architecture.md) — sub-client pattern, transport layer, error model.
- [`docs/geoserver-rest-quirks.md`](docs/geoserver-rest-quirks.md) — public catalog of GeoServer 2.x REST API edge cases this client works around.
- [`docs/version-compat.md`](docs/version-compat.md) — Go and GeoServer version matrix.
- [`ROADMAP.md`](ROADMAP.md) and [`CHANGELOG.md`](CHANGELOG.md).

## Version compatibility

- **Go**: 1.25+.
- **GeoServer**: 2.27 LTS and 2.28 stable. Both legs run on every PR via the integration matrix in [`.github/workflows/integration.yml`](.github/workflows/integration.yml).
- **Module path**: `github.com/hishamkaram/geoserver/v2`. The `/v2` suffix is required by Go's semantic import versioning rule for v2+ modules.
- **API stability**: v2's public API is locked — no breaking changes will land in v2.x.
- **v1**: end-of-feature on the [`release/v1` branch](https://github.com/hishamkaram/geoserver/tree/release/v1) (security patches only). Existing v1 users: see [`docs/migration-v1-to-v2.md`](docs/migration-v1-to-v2.md) for the per-resource upgrade mapping.

## Contributing

To add a new sub-client, pick the reference shape that matches your resource and follow the existing layout:

- **Flat CRUD**: `rest/workspaces/`, `rest/namespaces/`.
- **Workspace-scoped**: `rest/datastores/`, `rest/coveragestores/`, `rest/layers/`.
- **2-level hierarchy**: `rest/featuretypes/`, `rest/coverages/`.
- **Generic-typed dispatch**: `rest/services/` (per-service WMS/WFS/WCS/WMTS).
- **Out-of-`/rest/` URL prefix**: `rest/gwc/` (paths under `/gwc/rest/`).
- **XML wire format**: `ows/wms/`, `ows/wfs/`, `ows/wcs/`.

Each sub-client is structured the same way:

1. `types.go` — wire-format request/response structs, public option types, custom `(Un)MarshalJSON` for any wire quirks.
2. `<resource>.go` — `type Client struct{ core Core }`, `func New(core Core)`, methods. Each subpackage's `Core` interface declares only the transport methods it actually uses (`Do` / `DoXML` / `DoRaw` / `DoStream`).
3. `<resource>_test.go` (httptest unit tests) and `<resource>_integration_test.go` with the `//go:build integration` tag.
4. Wire into `*Client` in `geoserver.go`.

Run integration tests locally before pushing: `make compose-up && go test -tags=integration ./rest/<resource>/`. CI runs the full matrix on real GeoServer 2.27.4 LTS + 2.28.0 stable, but local-first catches wire-format quirks faster.

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the general PR workflow and [`docs/geoserver-rest-quirks.md`](docs/geoserver-rest-quirks.md) for the catalog of known REST API edge cases.
