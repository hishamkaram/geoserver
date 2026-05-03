# geoserver/v2

> ⚠️ **In development.** v2 is a clean redesign of `github.com/hishamkaram/geoserver` for 2026-era idiomatic Go. The public API is not yet stable; `Workspaces`, `Datastores`, and `FeatureTypes` are implemented today and the rest port in subsequent PRs. **For production use today, use the v1 line:**
>
> ```go
> import "github.com/hishamkaram/geoserver"          // v1 — stable, full surface
> import "github.com/hishamkaram/geoserver/v2"       // v2 — in development
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

### 2-level scoped resources (feature types)

Resources nested under both a workspace and a datastore (feature types now, coverages later) drill in through two `In…` calls:

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

## Resource status

| Resource | v1 | v2 |
|---|---|---|
| Workspaces | full | **ported** (flat reference resource) |
| Datastores | full | **ported** (workspace-scoped reference; `c.Datastores.InWorkspace(ws)`) |
| Feature types | full | **ported** (2-level hierarchy reference; `c.FeatureTypes.InWorkspace(ws).InDatastore(ds)`) |
| Coverages, coverage stores | full | not yet ported |
| Layers, layer groups | full | not yet ported |
| Styles | full | not yet ported |
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
