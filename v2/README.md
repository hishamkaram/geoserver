# geoserver/v2

> 🧪 **`v2.0.0-alpha.4` is published.** v2 closes the gap-analysis plan's "everyone needs it" surface. Coverage at `master`:
>
> - **Catalog**: workspaces, datastores, feature types, coverage stores, coverages, layers (incl. add-style sub-resource), layer groups, styles, namespaces.
> - **Settings**: global `c.Settings` + per-service `c.Services.WMS()`/`WFS()`/`WCS()`/`WMTS()` (global + per-workspace overrides).
> - **System**: `c.System.Reload` and `ResetCache`. **About**: ping + version.
> - **Security**: users, groups, roles, role-user assignment + layer ACL rules.
> - **File-upload publishing**: `c.Datastores.UploadFile` (Shapefile / GeoPackage / external) and `c.CoverageStores.UploadFile` + `HarvestGranule` (GeoTIFF / ImageMosaic / mosaic granules).
> - **GeoWebCache**: `c.GWC.Layers()` (cache config), `Seed()` (seed/reseed/truncate), `DiskQuota()`.
> - **Importer extension**: `c.Imports` (sessions + tasks). The dev/test docker image bakes the plugin in for CI integration coverage.
> - **OWS**: `c.WMS` / `c.WFS` / `c.WCS` GetCapabilities; WFS `DescribeFeatureType`; WCS `DescribeCoverage`.
>
> Public API may still change before `v2.0.0` based on early-adopter feedback — no production guarantees yet. **For stable production use, stay on the v1 line:**
>
> ```go
> import "github.com/hishamkaram/geoserver"          // v1.1.x — stable, full surface
> import "github.com/hishamkaram/geoserver/v2"       // v2.0.0-alpha.4 — preview
> ```
>
> Install:
> ```sh
> go get github.com/hishamkaram/geoserver/v2@v2.0.0-alpha.4
> ```

This module ships with its own `go.mod` at `/v2/`; v1 and v2 release independently (`v1.x.y` / `v2.x.y` tags).

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
| Workspaces | full | **ported** (flat reference resource) |
| Datastores | full | **ported** (workspace-scoped reference; `c.Datastores.InWorkspace(ws)`) |
| Feature types | full | **ported** (2-level hierarchy reference; `c.FeatureTypes.InWorkspace(ws).InDatastore(ds)`) |
| Coverage stores | full | **ported** (workspace-scoped; `c.CoverageStores.InWorkspace(ws)`) |
| Coverages | full | **ported** (2-level hierarchy; `c.Coverages.InWorkspace(ws).InCoverageStore(cs)`) |
| Layers | full | **ported** (workspace-scoped; `c.Layers.InWorkspace(ws)`) |
| Layer groups | full | **ported** (workspace-scoped; `c.LayerGroups.InWorkspace(ws)`) |
| Styles | full | **ported** (global by default; `c.Styles.InWorkspace(ws)` for workspace scope; `UploadSLD` for body upload) |
| Namespaces | full | **ported** (flat global; `c.Namespaces`) |
| Settings | full | **ported** (singleton; `c.Settings.Get` / `Update`) |
| About | full | **ported** (`c.About.Ping`, `c.About.Version`) |
| Security (users, groups, roles) | full | **ported** (`c.Security.Users()`, `c.Security.Groups()`, `c.Security.Roles`) |
| ACL (layer rules) | full | **ported** (`c.ACL.Layers()`) |
| WMS GetCapabilities | partial (XML) | not yet ported (deferred to v2.x; needs `ows/wms/`) |
| Namespaces | full | not yet ported |
| Settings | full | not yet ported |
| Security (users, groups, roles) | full | not yet ported |
| ACL | full | not yet ported |
| About, capabilities | full | not yet ported |
| WMS / WFS / WCS (OWS) | partial (WMS XML) | not yet ported |

See [`../ROADMAP.md`](../ROADMAP.md) for the milestone checklist.

## Contributing to v2

Each resource port is its own PR. Use `rest/workspaces/` as the reference pattern:

1. Define `types.go` — wire-format request/response structs, public option types.
2. Define `<resource>.go` — `type Client struct{ core Core }`, `func New(core Core)`, methods (`List`, `Get`, `Create`, `Update`, `Delete`, optional `Iter`).
3. Define `<resource>_test.go` — `httptest.Server`-based unit tests for 2xx + each relevant 4xx/5xx mapping.
4. Wire into `*Client` in `../geoserver.go`.

The `Core` interface in each subpackage is the abstraction over the parent `*Client`'s plumbing — it lets sub-clients issue requests without importing the root package (which would create an import cycle).

See [`../CONTRIBUTING.md`](../CONTRIBUTING.md) for the general PR workflow.
