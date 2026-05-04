# geoserver/v2

[![Go Reference](https://pkg.go.dev/badge/github.com/hishamkaram/geoserver/v2.svg)](https://pkg.go.dev/github.com/hishamkaram/geoserver/v2)
[![CI](https://github.com/hishamkaram/geoserver/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/hishamkaram/geoserver/actions/workflows/ci.yml)
[![Integration tests](https://github.com/hishamkaram/geoserver/actions/workflows/integration.yml/badge.svg?branch=master)](https://github.com/hishamkaram/geoserver/actions/workflows/integration.yml)
[![License: MIT](https://img.shields.io/github/license/hishamkaram/geoserver.svg)](../LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/hishamkaram/geoserver?include_prereleases&sort=semver)](https://github.com/hishamkaram/geoserver/releases)

> 🚀 **`v2.0.0` is published — first stable release.** v2 covers the gap-analysis plan's "everyone needs it" surface, every tier-2 item from [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md), plus four longer-tail surfaces (fonts, password rotation, GWC additions, monitoring). Public API is locked; no breaking changes will land in v2.x. Coverage:
>
> - **Catalog**: workspaces, datastores, feature types, coverage stores, coverages, layers (incl. add-style sub-resource), layer groups, styles, namespaces. **Fonts**: `c.Fonts.List` (JVM-exposed font families).
> - **Settings**: global `c.Settings` + per-service `c.Services.WMS()`/`WFS()`/`WCS()`/`WMTS()` (global + per-workspace overrides).
> - **System**: `c.System.Reload` and `ResetCache`. **About**: ping, version, manifests, system status. **Logging**: `c.Logging.Get`/`Update` for runtime log-level changes. **Monitoring** (`gs-monitor` ext baked into dev/test image): `c.Monitor.List`/`ListRaw`/`Get` for the request audit log.
> - **Security**: users, groups, roles, role-user assignment + ACL `Layers()`/`Services()`/`REST()`/`Catalog()` + auth providers / filters / filter chains + URL checks (SSRF allow-list) + `c.Security.MasterPassword` (keystore) + `c.Security.SelfPassword` (auth'd user rotation).
> - **Resources**: `c.Resources` Get / List / Stat / Exists / Put / Move / Copy / Delete against `/rest/resource/{path}` — read or write any file in the data dir. **Templates (FTL)**: `c.Templates` global + six fluent scopes.
> - **File-upload publishing**: `c.Datastores.UploadFile` (Shapefile / GeoPackage / external) and `c.CoverageStores.UploadFile` + `HarvestGranule` (GeoTIFF / ImageMosaic / mosaic granules).
> - **Mosaic granules**: `c.Coverages.InWorkspace(ws).InCoverageStore(cs).Granules(cov)` — Schema / List / Get / Delete / DeleteByFilter for image-mosaic / structured coverages.
> - **Cascaded stores + layers**: `c.WMSStores` / `c.WMSLayers` / `c.WMTSStores` / `c.WMTSLayers` for federation / proxy setups.
> - **GeoWebCache**: `c.GWC.Layers()` (cache config), `Seed()` (seed/reseed/truncate), `DiskQuota()`, `Global()` (singleton config), `Gridsets()` (named tile-matrix sets), `MassTruncate()` (bulk cache invalidation).
> - **Importer extension**: `c.Imports` (sessions + tasks). The dev/test docker image bakes the plugin in for CI integration coverage.
> - **OWS**: `c.WMS` / `c.WFS` / `c.WCS` GetCapabilities; WFS `DescribeFeatureType`; WCS `DescribeCoverage`. **WFS XSLT transforms**: `c.WFSTransforms` — surface preserved for legacy / custom builds; the `gs-xslt-wfs` extension was removed from upstream in 2.24 and is not shipped for 2.27 / 2.28.
>
> Surface is locked — no breaking changes will land in v2.x. v1.x is end-of-feature; security patches only on the `release/v1` branch (post-restructure). New integrations should target v2.

This module ships with its own `go.mod` at `/v2/`; v1 and v2 release independently (`v1.x.y` / `v2.x.y` tags).

## Contents

- [Install](#install)
- [Why v2 over v1?](#why-v2-over-v1)
- [Design tenets](#design-tenets)
- [Quick start](#quick-start)
- [Runnable examples](#runnable-examples)
- [Resource status](#resource-status)
- [Contributing to v2](#contributing-to-v2)

## Install

```bash
go get github.com/hishamkaram/geoserver/v2@v2.0.0
```

```go
import geoserver "github.com/hishamkaram/geoserver/v2"
```

> v2 is a separate Go module under `/v2/`; v1 and v2 release independently (`v1.x.y` / `v2.x.y` tags). v2 is the recommended line for new integrations.

Requirements: Go 1.25+. Tested against GeoServer 2.27.4 LTS and 2.28.0 stable on every PR.

## Why v2 over v1?

If you're starting a new integration today, v2 is the better foundation. It gives you:

- **Sub-clients per resource.** `c.Workspaces.Get(ctx, name)`, `c.Datastores.InWorkspace(ws).Create(ctx, ...)`, `c.FeatureTypes.InWorkspace(ws).InDatastore(ds).Discover(ctx, ...)` — instead of v1's monolithic ~90-method `*GeoServer`.
- **Immutable client; concurrency-safe by construction.** All fields private; configured via functional options at construction; no post-construction mutation. Auth is layered on as an `http.RoundTripper` so OpenTelemetry, Vault-rotated creds, and retry libs compose naturally.
- **Mandatory `context.Context` first arg on every method.** No `Background()` shims, no twin pairs.
- **`iter.Seq2` pagination** on every `List` endpoint. `for ws, err := range c.Workspaces.Iter(ctx, opts) { ... }`.
- **Surfaces v1 doesn't have** — per-service OWS settings (`c.Services.WMS()`/`WFS()`/`WCS()`/`WMTS()`), file-upload publishing on stores (`c.Datastores.UploadFile`, `c.CoverageStores.UploadFile`/`HarvestGranule`), GeoWebCache (`c.GWC.Layers()`/`Seed()`/`DiskQuota()`), the Importer extension (`c.Imports`), the Resource API (`c.Resources`), the full ACL surface (`c.ACL.Layers()`/`Services()`/`REST()`/`Catalog()`), WFS `DescribeFeatureType`, and WCS `DescribeCoverage`.

If you're already on v1 and don't need any of the above, there is no rush — v1.x is non-breaking and continues to receive security and bug-fix patches. See [`../docs/migration-v1-to-v2.md`](../docs/migration-v1-to-v2.md) for the per-resource migration mapping.

## Design tenets

v2 breaks v1's monolithic `*GeoServer` surface into a sub-client per resource, with a few lock-in decisions documented in `../ROADMAP.md`:

- **Immutable `*Client`.** All fields private; no post-construction mutation. Concurrent use is safe.
- **Mandatory `context.Context`** as first arg on every public method. No `Background` shims.
- **Sub-client pattern.** `c.Workspaces.List(ctx, opts)` instead of v1's `gs.GetWorkspacesContext(ctx)`.
- **Single error type** (`*APIError`) with package sentinels (`ErrNotFound`, `ErrConflict`, …). `errors.Is` and `errors.As` are the supported match styles.
- **Auth via `http.RoundTripper`.** Basic / bearer auth attaches at construction; per-call paths don't re-authenticate. Custom RoundTrippers (OpenTelemetry, Vault-rotated creds, retry libs) layer naturally.
- **Pagination via `iter.Seq2`.** `c.Workspaces.Iter(ctx, opts)` returns a `iter.Seq2[Workspace, error]`. Non-paginating endpoints fall back to single-page Seq2.
- **Zero runtime third-party deps.** stdlib `net/http`, `encoding/json`, `encoding/xml`, `log/slog`, `context`, `iter` only.

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

### Workspace-scoped resources

Resources nested under a workspace (datastores, feature types, coverages, …) are accessed via an `InWorkspace` scope:

```go
import "github.com/hishamkaram/geoserver/v2/rest/datastores"

ds := c.Datastores.InWorkspace("topp")

_ = ds.Create(ctx, datastores.PostGIS{
    Name: "states", Host: "db", Port: 5432, Database: "gis",
    User: "u", Password: "p",
})

stores, _ := ds.List(ctx, datastores.ListOptions{})
_ = ds.Delete(ctx, "states", datastores.DeleteOptions{Recurse: true})
```

### 2-level scoped resources (feature types, coverages)

Resources nested under both a workspace and a parent store drill in through two `In…` calls:

```go
import "github.com/hishamkaram/geoserver/v2/rest/featuretypes"

ft := c.FeatureTypes.InWorkspace("topp").InDatastore("states_pg")

// Discover tables in the datastore not yet configured.
names, _ := ft.Discover(ctx, featuretypes.DiscoverOptions{
    Kind: featuretypes.DiscoverAvailableWithGeometry,
})

// Publish one of them as a feature type.
_ = ft.Create(ctx, &featuretypes.FeatureType{
    Name: "states", NativeName: "states",
    SRS: "EPSG:4326", Enabled: true,
})
```

The same shape applies to coverages under a coverage store (raster side):

```go
import "github.com/hishamkaram/geoserver/v2/rest/coverages"

cov := c.Coverages.InWorkspace("ne").InCoverageStore("states_tiff")

// Publish a configured coverage from the underlying GeoTIFF.
_ = cov.Create(ctx, &coverages.Coverage{
    Name: "states_published", NativeCoverageName: "states.tif",
})
```

## Runnable examples

The [`examples/`](examples/) directory contains self-contained `main` packages demonstrating each idiom:

- [`workspaces/`](examples/workspaces/) — flat sub-client CRUD; `errors.Is` matching.
- [`publish-postgis/`](examples/publish-postgis/) — end-to-end workspace → datastore → feature type → layer flow with the hierarchical sub-clients.
- [`style-upload/`](examples/style-upload/) — two-step style publish via `Create` + `UploadSLD`.
- [`error-handling/`](examples/error-handling/) — full sentinel set + `*APIError` inspection via `errors.As`.

Run any with `go run ./v2/examples/<name>` against a `make compose-up` stack, or compile-check all with `make examples-v2`.

## Resource status

| Resource | v1 | v2 |
|---|---|---|
| Workspaces | full | **ported** (flat; `c.Workspaces`) |
| Datastores | full | **ported** (workspace-scoped; `c.Datastores.InWorkspace(ws)`) |
| Feature types | full | **ported** (2-level hierarchy; `c.FeatureTypes.InWorkspace(ws).InDatastore(ds)`) |
| Coverage stores | full | **ported** (workspace-scoped; `c.CoverageStores.InWorkspace(ws)`) |
| Coverages | full | **ported** (2-level hierarchy; `c.Coverages.InWorkspace(ws).InCoverageStore(cs)`) |
| Layers | full | **ported** + new add-style sub-resource (`c.Layers.InWorkspace(ws).AddStyle/ListStyles`) |
| Layer groups | full | **ported** (`c.LayerGroups.InWorkspace(ws)`) |
| Styles | full | **ported** (global + workspace scope; `UploadSLD` for body upload) |
| Namespaces | full | **ported** (`c.Namespaces`) |
| Global settings | full | **ported** (`c.Settings.Get` / `Update`) |
| Per-service OWS settings | (none) | **new** in v2 (`c.Services.WMS()` / `WFS()` / `WCS()` / `WMTS()` — global + per-workspace overrides) |
| System (reload + cache reset) | full | **ported** (`c.System.Reload`, `ResetCache`) |
| About | full | **ported** (`c.About.Ping`, `c.About.Version`) |
| Security (users, groups, roles) | full | **ported** (`c.Security.Users()`, `Groups()`, `Roles`) |
| ACL — layer rules | full | **ported** (`c.ACL.Layers()`) |
| ACL — service / REST / catalog rules | partial | **ported** in v2 (`c.ACL.Services()`, `c.ACL.REST()`, `c.ACL.Catalog()`) — REST DELETE has documented firewall caveat |
| Resource API (data-dir byte-stream access) | (none) | **new** in v2 (`c.Resources` Get/List/Stat/Exists/Put/Move/Copy/Delete) |
| File-upload publishing on stores | (none) | **new** in v2 (`c.Datastores.UploadFile`, `c.CoverageStores.UploadFile` / `HarvestGranule`) |
| GeoWebCache (cache config + seed + diskquota) | (none) | **new** in v2 (`c.GWC.Layers()`, `Seed()`, `DiskQuota()`, `Global()`, `Gridsets()`, `MassTruncate()`) |
| Importer extension (batch ingest) | (none) | **new** in v2 (`c.Imports`; dev/test docker image bakes the plugin in) |
| WMS GetCapabilities | full | **ported** (`c.WMS.GetCapabilities` + `InWorkspace`) |
| WFS GetCapabilities + DescribeFeatureType | (none — WMS only) | **new** in v2 (`c.WFS.GetCapabilities`, `DescribeFeatureType`) |
| WCS GetCapabilities + DescribeCoverage | (none — WMS only) | **new** in v2 (`c.WCS.GetCapabilities`, `DescribeCoverage`) |
| Mosaic / structured-coverage granules | (none) | **new** in v2 (`c.Coverages.InWorkspace(ws).InCoverageStore(cs).Granules(cov)` — Schema / List / Get / Delete / DeleteByFilter) |
| FTL templates | (none) | **new** in v2 (`c.Templates` + six fluent scopes; List / Get / Put / Delete) |
| Auth providers / filters / chains | (none) | **new** in v2 (`c.Security.AuthProviders`, `AuthFilters`, `FilterChains`) |
| URL checks (SSRF allow-list) | (none) | **new** in v2 (`c.URLChecks`) |
| Cascaded WMS / WMTS stores + layers | (none) | **new** in v2 (`c.WMSStores`, `c.WMSLayers`, `c.WMTSStores`, `c.WMTSLayers`) |
| WFS XSLT transforms | (none) | **new** in v2 (`c.WFSTransforms`) — surface preserved for legacy / custom builds; the `gs-xslt-wfs` extension was removed from upstream in 2.24 and is not shipped for 2.27 / 2.28 |
| About — manifests + system status | (none) | **new** in v2 (`c.About.Manifests`, `c.About.SystemStatus`) |
| Logging (runtime log-level config) | (none) | **new** in v2 (`c.Logging.Get` / `Update`) |
| Fonts list | (none) | **new** in v2 (`c.Fonts.List`) |
| Master password & self password | (none) | **new** in v2 (`c.Security.MasterPassword.Get`/`Update`, `c.Security.SelfPassword.Change`) |
| Monitoring (request audit log) | (none) | **new** in v2 (`c.Monitor.List`/`ListRaw`/`Get`; dev/test docker image bakes the `gs-monitor` plugin in) |

See [`../ROADMAP.md`](../ROADMAP.md) for the milestone checklist. The original tier-2 gap-analysis backlog is closed and the longer-tail surfaces above are also done; remaining longer-tail endpoints (`opensearch-eo` is community / SNAPSHOT-only on 2.27 / 2.28; revisit when stable) are tracked in [`../docs/v2-tier2-gaps.md`](../docs/v2-tier2-gaps.md).

## Contributing to v2

The "everyone needs it" surface and the tier-2 backlog are both closed; remaining work is the longer-tail list above plus wire-quirk fixes when adopters report them.

To add a new sub-client:

1. Pick a reference pattern that matches the shape:
   - **Flat CRUD**: `rest/workspaces/`, `rest/namespaces/`.
   - **Workspace-scoped**: `rest/datastores/`, `rest/coveragestores/`, `rest/layers/`.
   - **2-level hierarchy**: `rest/featuretypes/`, `rest/coverages/`.
   - **Generic-typed dispatch**: `rest/services/` (per-service WMS/WFS/WCS/WMTS).
   - **Out-of-`/rest/` URL prefix**: `rest/gwc/` (paths under `/gwc/rest/`).
   - **XML wire format**: `ows/wms/`, `ows/wfs/`, `ows/wcs/`.
2. Define `types.go` — wire-format request/response structs, public option types, custom `(Un)MarshalJSON` for any wire quirks.
3. Define `<resource>.go` — `type Client struct{ core Core }`, `func New(core Core)`, methods. Each subpackage's `Core` interface declares only the transport methods it actually uses (`Do` / `DoXML` / `DoRaw` / `DoStream`); add only what's needed.
4. Define `<resource>_test.go` (httptest unit tests) and `<resource>_integration_test.go` with the `//go:build integration` tag.
5. Wire into `*Client` in `../geoserver.go`.

**Run integration tests locally before push.** `make compose-up && cd v2 && go test -tags=integration ./rest/<resource>/`. CI's wire-format coverage runs on real GeoServer 2.27.4 LTS + 2.28.0 stable, but local-first catches quirks faster.

The `Core` interface in each subpackage is the abstraction over the parent `*Client`'s plumbing — it lets sub-clients issue requests without importing the root package (which would create an import cycle).

See [`../CONTRIBUTING.md`](../CONTRIBUTING.md) for the general PR workflow and [`../docs/migration-v1-to-v2.md`](../docs/migration-v1-to-v2.md) for the v1 → v2 migration guide.
