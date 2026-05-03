# Migration from v1.x to v2.x

> **Status: pre-alpha.** v2 has feature parity with v1 (catalog, security, ACL, settings, namespaces, about); only WMS GetCapabilities is deferred to a later v2.x point release. The public API may shift before `v2.0.0-alpha.1`. Until a stable v2 tag ships, **prefer v1.x for production usage**.

This guide walks through the concrete API differences between v1.x (`github.com/hishamkaram/geoserver`) and v2.x (`github.com/hishamkaram/geoserver/v2`). Each section pairs a v1 snippet with the v2 equivalent.

## Module path

```diff
- import "github.com/hishamkaram/geoserver"
+ import "github.com/hishamkaram/geoserver/v2"
```

v2 lives at the `/v2/` subdirectory of the same repository, with its own `go.mod`. v1 and v2 ship independent tags (v1.x.y and v2.x.y) and can coexist in the same `go.mod` during incremental migration.

## Design tenets that drive the breakage

Each of the following is a deliberate departure from v1:

- **Immutable `*Client`** — all fields private, configured via functional options at construction time.
- **Mandatory `context.Context`** as first arg on every public method. No `Background` shims, no twin pairs (`Foo` / `FooContext`).
- **Sub-client pattern** — `c.Workspaces.List(ctx, opts)` instead of `gs.GetWorkspacesContext(ctx)`.
- **Single error type** (`*APIError`) with sentinels via `errors.Is`. No string matching.
- **`Create` returns just `error`** instead of `(bool, error)`. `err == nil` is the success signal.
- **Auth via `http.RoundTripper`** wrapping the configured transport. No per-call `request.SetBasicAuth`; layers naturally with OpenTelemetry, Vault rotation, retry libs.
- **Pagination via `iter.Seq2`** on every list endpoint that may grow.
- **Zero runtime third-party deps** — stdlib only.

## Constructor

```go
// v1
gs := geoserver.GetCatalog("http://localhost:8080/geoserver/", "admin", "geoserver")
// or with options:
gs, _ := geoserver.New("http://...", "admin", "geoserver",
    geoserver.WithTimeout(30*time.Second))
```

```go
// v2
c, err := geoserver.New("http://localhost:8080/geoserver/",
    geoserver.WithBasicAuth("admin", "geoserver"),
    geoserver.WithTimeout(30*time.Second),
)
if err != nil { /* serverURL parse / option validation */ }
```

Notes:
- v2 returns `error` from the constructor — bad URL or option misconfiguration surfaces immediately rather than at first call.
- Credentials move from positional args to options (`WithBasicAuth`, `WithBearerToken`).
- `geoserver.GetCatalog` is gone.

## Errors

```go
// v1
_, err := gs.GetWorkspaceContext(ctx, "missing")
if errors.Is(err, geoserver.ErrNotFound) { /* not found */ }
var e *geoserver.Error
if errors.As(err, &e) { fmt.Println(e.StatusCode) }
```

```go
// v2
_, err := c.Workspaces.Get(ctx, "missing")
if errors.Is(err, geoserver.ErrNotFound) { /* not found */ }
var e *geoserver.APIError
if errors.As(err, &e) { fmt.Println(e.StatusCode) }
```

The sentinel set is unchanged — `ErrNotFound`, `ErrConflict`, `ErrUnauthorized`, `ErrForbidden`, `ErrBadRequest`, `ErrServerError`, etc. The wrapper type renames from `*Error` to `*APIError` for clarity.

## Mapping table

### Workspaces

| v1 | v2 |
|---|---|
| `gs.GetWorkspacesContext(ctx)` | `c.Workspaces.List(ctx, workspaces.ListOptions{})` |
| `gs.GetWorkspaceContext(ctx, name)` | `c.Workspaces.Get(ctx, name)` |
| `gs.WorkspaceExistsContext(ctx, name)` | call `Get`, check `errors.Is(err, geoserver.ErrNotFound)` |
| `gs.CreateWorkspaceContext(ctx, name)` | `c.Workspaces.Create(ctx, &workspaces.Workspace{Name: name})` |
| `gs.DeleteWorkspaceContext(ctx, name, recurse)` | `c.Workspaces.Delete(ctx, name, workspaces.DeleteOptions{Recurse: recurse})` |
| (none) | `c.Workspaces.Iter(ctx, opts)` — `iter.Seq2[Workspace, error]` |
| (none) | `c.Workspaces.Update(ctx, name, &workspaces.WorkspacePatch{Isolated: &b})` |

### Datastores

| v1 | v2 |
|---|---|
| `gs.GetDatastoresContext(ctx, ws)` | `c.Datastores.InWorkspace(ws).List(ctx, datastores.ListOptions{})` |
| `gs.GetDatastoreDetailsContext(ctx, ws, name)` | `c.Datastores.InWorkspace(ws).Get(ctx, name)` |
| `gs.CreateDatastoreContext(ctx, conn, ws)` | `c.Datastores.InWorkspace(ws).Create(ctx, datastores.PostGIS{...})` |
| `gs.CreateJNDIDatastoreContext(ctx, conn, ws)` | `c.Datastores.InWorkspace(ws).Create(ctx, datastores.JNDI{...})` |
| `gs.CreateDatastoreFromConnectorContext(ctx, c, ws)` | `c.Datastores.InWorkspace(ws).Create(ctx, datastores.Raw(d))` |
| `gs.DeleteDatastoreContext(ctx, ws, name, recurse)` | `c.Datastores.InWorkspace(ws).Delete(ctx, name, datastores.DeleteOptions{Recurse: recurse})` |

The hierarchical `InWorkspace(ws)` returns a workspace-scoped client; per-method validation surfaces empty-workspace errors so the fluent path stays single-error.

### Feature types

| v1 | v2 |
|---|---|
| `gs.GetFeatureTypesContext(ctx, ws, ds)` | `c.FeatureTypes.InWorkspace(ws).InDatastore(ds).List(ctx, featuretypes.ListOptions{})` |
| `gs.GetFeatureTypeContext(ctx, ws, ds, name)` | `c.FeatureTypes.InWorkspace(ws).InDatastore(ds).Get(ctx, name)` |
| `gs.GetFeatureTypeListContext(ctx, ws, ds, kind)` | `c.FeatureTypes.InWorkspace(ws).InDatastore(ds).Discover(ctx, featuretypes.DiscoverOptions{Kind: featuretypes.DiscoverAvailable})` |
| `gs.CreateFeatureTypeContext(ctx, ws, ds, ft)` | `c.FeatureTypes.InWorkspace(ws).InDatastore(ds).Create(ctx, ft)` |
| `gs.DeleteFeatureTypeContext(ctx, ws, ds, name, recurse)` | `c.FeatureTypes.InWorkspace(ws).InDatastore(ds).Delete(ctx, name, featuretypes.DeleteOptions{Recurse: recurse})` |

The 2-level hierarchy reflects GeoServer's REST topology. `Discover` separates "tables in the datastore not yet published" from `List` (configured feature types) since the wire shapes differ — `[]string` for available, `[]FeatureType` for configured.

### Coverage stores

| v1 | v2 |
|---|---|
| `gs.GetCoverageStoresContext(ctx, ws)` | `c.CoverageStores.InWorkspace(ws).List(ctx, coveragestores.ListOptions{})` |
| `gs.GetCoverageStoreContext(ctx, ws, name)` | `c.CoverageStores.InWorkspace(ws).Get(ctx, name)` |
| `gs.CreateCoverageStoreContext(ctx, ws, store)` | `c.CoverageStores.InWorkspace(ws).Create(ctx, &coveragestores.CoverageStore{...})` |
| `gs.UpdateCoverageStoreContext(ctx, ws, store)` | `c.CoverageStores.InWorkspace(ws).Update(ctx, name, &coveragestores.Patch{...})` |
| `gs.DeleteCoverageStoreContext(ctx, ws, name, recurse)` | `c.CoverageStores.InWorkspace(ws).Delete(ctx, name, coveragestores.DeleteOptions{Recurse: recurse})` |

### Coverages

| v1 | v2 |
|---|---|
| `gs.GetCoveragesContext(ctx, ws)` | (use `InCoverageStore`; cross-store list isn't exposed) |
| `gs.GetStoreCoveragesContext(ctx, ws, cs)` | `c.Coverages.InWorkspace(ws).InCoverageStore(cs).Discover(ctx, coverages.DiscoverOptions{})` |
| `gs.GetCoverageContext(ctx, ws, name)` | `c.Coverages.InWorkspace(ws).InCoverageStore(cs).Get(ctx, name)` |
| `gs.PublishCoverageContext(ctx, ws, cs, name, publishName)` | `c.Coverages.InWorkspace(ws).InCoverageStore(cs).Create(ctx, &coverages.Coverage{Name: publishName, NativeCoverageName: name})` |
| `gs.UpdateCoverageContext(ctx, ws, cov)` | `c.Coverages.InWorkspace(ws).InCoverageStore(cs).Update(ctx, name, cov)` |
| `gs.DeleteCoverageContext(ctx, ws, name, recurse)` | `c.Coverages.InWorkspace(ws).InCoverageStore(cs).Delete(ctx, name, coverages.DeleteOptions{Recurse: recurse})` |

v2's coverages are 2-level scoped (workspace + coverage_store), which fixes v1's awkward `coverage.Store.Name` "workspace:store" parsing in Update.

### Layers and layer groups

| v1 | v2 |
|---|---|
| `gs.GetLayersContext(ctx, ws)` | `c.Layers.InWorkspace(ws).List(ctx, layers.ListOptions{})` |
| `gs.GetLayerContext(ctx, ws, name)` | `c.Layers.InWorkspace(ws).Get(ctx, name)` |
| `gs.UpdateLayerContext(ctx, ws, name, layer)` | `c.Layers.InWorkspace(ws).Update(ctx, name, &layer)` |
| `gs.DeleteLayerContext(ctx, ws, name, recurse)` | `c.Layers.InWorkspace(ws).Delete(ctx, name, layers.DeleteOptions{Recurse: recurse})` |
| `gs.GetLayerGroupsContext(ctx, ws)` | `c.LayerGroups.InWorkspace(ws).List(ctx, layergroups.ListOptions{})` |
| `gs.GetLayerGroupContext(ctx, ws, name)` | `c.LayerGroups.InWorkspace(ws).Get(ctx, name)` |
| `gs.CreateLayerGroupContext(ctx, ws, group)` | `c.LayerGroups.InWorkspace(ws).Create(ctx, group)` |
| `gs.DeleteLayerGroupContext(ctx, ws, name)` | `c.LayerGroups.InWorkspace(ws).Delete(ctx, name)` |

There is no `Create` on layers — they are auto-created when a feature type or coverage is published. Layer-group `Delete` does not accept a recurse query (GeoServer ignores it).

### Styles

| v1 | v2 |
|---|---|
| `gs.GetStylesContext(ctx, "")` | `c.Styles.List(ctx, styles.ListOptions{})` (global) |
| `gs.GetStylesContext(ctx, ws)` | `c.Styles.InWorkspace(ws).List(ctx, styles.ListOptions{})` |
| `gs.GetStyleContext(ctx, ws, name)` | `c.Styles.InWorkspace(ws).Get(ctx, name)` |
| `gs.CreateStyleContext(ctx, ws, name)` | `c.Styles.InWorkspace(ws).Create(ctx, &styles.Style{Name: name})` |
| `gs.UploadStyleContext(ctx, body, ws, name, overwrite)` | `c.Styles.InWorkspace(ws).Create(...)` then `UploadSLD(ctx, name, body, styles.UploadOptions{})` |
| `gs.DeleteStyleContext(ctx, ws, name, purge)` | `c.Styles.InWorkspace(ws).Delete(ctx, name, styles.DeleteOptions{Purge: purge})` |

The workspace-scoped `Accept: */*` quirk is automatic in v2's `Create` — no caller workaround needed. `UploadSLD` is split from `Create` for cleaner two-step publishing.

### Namespaces

| v1 | v2 |
|---|---|
| `gs.GetNamespacesContext(ctx)` | `c.Namespaces.List(ctx, namespaces.ListOptions{})` |
| `gs.GetNamespaceContext(ctx, prefix)` | `c.Namespaces.Get(ctx, prefix)` |
| `gs.CreateNamespaceContext(ctx, prefix, uri)` | `c.Namespaces.Create(ctx, &namespaces.Namespace{Prefix: prefix, URI: uri})` |
| `gs.DeleteNamespaceContext(ctx, prefix)` | `c.Namespaces.Delete(ctx, prefix)` |

### Settings

| v1 | v2 |
|---|---|
| `gs.GetGlobalSettingsContext(ctx)` | `c.Settings.Get(ctx)` |
| `gs.UpdateGlobalSettingsContext(ctx, s)` | `c.Settings.Update(ctx, s)` |

The `interface{}` Contact / Jaiext fields in v1 become typed `*Contact` / `*JAIExt` with custom `UnmarshalJSON` to handle GeoServer's empty-string wire form (`"contact":""`).

### Security (users / groups / roles)

| v1 | v2 |
|---|---|
| `gs.GetUsersContext(ctx, svc)` | `c.Security.UsersInService(svc).List(ctx, security.ListOptions{})` (or `c.Security.Users()` for default) |
| `gs.CreateUserContext(ctx, name, pw, svc)` | `c.Security.Users().Create(ctx, &security.User{Name: name, Password: pw, Enabled: true})` |
| `gs.DeleteUserContext(ctx, name, svc)` | `c.Security.UsersInService(svc).Delete(ctx, name)` |
| `gs.GetGroupsContext(ctx, svc)` | `c.Security.GroupsInService(svc).List(ctx, security.ListOptions{})` |
| `gs.CreateGroupContext(ctx, name, svc)` | `c.Security.GroupsInService(svc).Create(ctx, name)` |
| `gs.DeleteGroupContext(ctx, name, svc)` | `c.Security.GroupsInService(svc).Delete(ctx, name)` |
| `gs.GetRolesContext(ctx)` | `c.Security.Roles.List(ctx, security.ListOptions{})` |
| `gs.GetUserRolesContext(ctx, name)` | `c.Security.Roles.ForUser(ctx, name)` |
| `gs.CreateRoleContext(ctx, name)` | `c.Security.Roles.Create(ctx, name)` |
| `gs.DeleteRoleContext(ctx, name)` | `c.Security.Roles.Delete(ctx, name)` |
| `gs.AddUserRoleContext(ctx, role, user)` | `c.Security.Roles.AssignToUser(ctx, role, user)` |
| `gs.DeleteUserRoleContext(ctx, role, user)` | `c.Security.Roles.UnassignFromUser(ctx, role, user)` |

Default user/group service is `"default"`; pass `""` to `UsersInService` / `GroupsInService` to use it, or call `Users()` / `Groups()` for the same effect.

### ACL

| v1 | v2 |
|---|---|
| `gs.GetLayersACLRulesContext(ctx)` | `c.ACL.Layers().List(ctx, acl.ListOptions{})` |
| `gs.AddLayersACLRuleContext(ctx, rule)` | `c.ACL.Layers().Add(ctx, rule)` |
| `gs.DeleteLayersACLRuleContext(ctx, rule)` | `c.ACL.Layers().Delete(ctx, rule)` |
| `geoserver.ACLRule` | `acl.Rule` |
| `geoserver.ACLOpRead/Write/Admin` | `acl.OpRead` / `acl.OpWrite` / `acl.OpAdmin` |
| `(rule).ToStrings()` | `(rule).Encode()` |
| `geoserver.StringToACLRule(...)` | `acl.DecodeRule(...)` |

### About / health

| v1 | v2 |
|---|---|
| `gs.IsRunningContext(ctx)` | `c.About.Ping(ctx)` (returns `error`, not `(bool, error)`) |
| (none) | `c.About.Version(ctx)` — full component version document |

### WMS GetCapabilities

Deferred. v2 will introduce an `ows/wms/` subpackage in a follow-up release; until then continue using v1's `gs.GetCapabilitiesContext(ctx, ws)` for capability XML parsing.

## Side-by-side: a typical workflow

### v1.x

```go
gs, _ := geoserver.New("http://localhost:8080/geoserver/", "admin", "geoserver")

if _, err := gs.CreateWorkspaceContext(ctx, "demo"); err != nil { return err }

if _, err := gs.CreateDatastoreContext(ctx, geoserver.DatastoreConnection{
    Name: "states_pg", Host: "db", Port: 5432, DBName: "gis",
    DBUser: "u", DBPass: "p", Type: "postgis",
}, "demo"); err != nil { return err }

if _, err := gs.CreateFeatureTypeContext(ctx, "demo", "states_pg",
    &geoserver.FeatureType{Name: "states", NativeName: "states", Srs: "EPSG:4326", Enabled: true},
); err != nil { return err }
```

### v2.x

```go
c, err := geoserver.New("http://localhost:8080/geoserver/",
    geoserver.WithBasicAuth("admin", "geoserver"))
if err != nil { return err }

if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: "demo"}); err != nil { return err }

if err := c.Datastores.InWorkspace("demo").Create(ctx, datastores.PostGIS{
    Name: "states_pg", Host: "db", Port: 5432, Database: "gis",
    User: "u", Password: "p",
}); err != nil { return err }

if err := c.FeatureTypes.InWorkspace("demo").InDatastore("states_pg").
    Create(ctx, &featuretypes.FeatureType{
        Name: "states", NativeName: "states", SRS: "EPSG:4326", Enabled: true,
    }); err != nil { return err }
```

## When to upgrade

When v2 hits `v2.0.0` (final, not alpha/beta), this section will document a recommended migration window. Until then, v1.x is the supported path; v2 is preview-quality and the surface may shift.

## Contributing to v2

v2 development happens in the same repository under `/v2/`. To contribute:

1. Open / claim an issue scoping the resource port (e.g., "Port WMS GetCapabilities to v2" once the `ows/wms/` subpackage lands).
2. Follow the `workspaces` reference resource (`v2/rest/workspaces/`) as the pattern.
3. Open a PR. CI runs `Unit tests v2 (Go 1.25)` against `/v2/`.

See [`../CONTRIBUTING.md`](../CONTRIBUTING.md) for the general PR workflow.
