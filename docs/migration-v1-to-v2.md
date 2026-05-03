# Migration from v1.x to v2.x

> **Status: in progress.** v2 is being scaffolded at `github.com/hishamkaram/geoserver/v2`. This document fills in as v2 stabilizes. Until v2 ships an alpha/beta tag, **prefer v1.x** for production usage.

This guide will walk through the concrete API differences between v1.x and v2.x once v2 is closer to release. It exists now as a placeholder so contributors and watchers can see where it's headed.

## Module path

```diff
- import "github.com/hishamkaram/geoserver"
+ import "github.com/hishamkaram/geoserver/v2"
```

v2 lives at the `/v2/` subdirectory of the same repository, with its own `go.mod`. v1 and v2 ship independent tags (v1.x.y and v2.x.y).

## Design tenets that drive the breakage

Each of the following is a deliberate departure from v1; details live in [`../ROADMAP.md`](../ROADMAP.md):

- **Immutable `*Client`** — all fields private, configured via functional options at construction time.
- **Mandatory `context.Context`** as first arg on every public method. No `Background` shims, no twin pairs.
- **Sub-client pattern** — `c.Workspaces.List(ctx, opts)` instead of `gs.GetWorkspacesContext(ctx)`.
- **Single error type** (`*APIError`) with sentinels via `errors.Is`. No string matching.
- **`Create` returns just `error`** instead of `(bool, error)`. `err == nil` is the success signal.
- **Auth via `http.RoundTripper`** wrapping the configured transport. No `request.SetBasicAuth` per call.

## Mapping table (will be filled in as v2 ports each resource)

| v1 | v2 | Status |
|---|---|---|
| `geoserver.GetCatalog(url, u, p)` | `geoserver.New(url, geoserver.WithBasicAuth(u, p))` | TBD |
| `geoserver.New(url, u, p, opts...)` | `geoserver.New(url, append(opts, geoserver.WithBasicAuth(u, p))...)` | TBD |
| `(*GeoServer).GetWorkspaces` / `GetWorkspacesContext` | `(*Client).Workspaces.List(ctx, ListOptions{})` | scaffolded |
| `(*GeoServer).CreateWorkspace(name)` | `(*Client).Workspaces.Create(ctx, &Workspace{Name: name})` | scaffolded |
| `(*GeoServer).DeleteWorkspace(name, recurse)` | `(*Client).Workspaces.Delete(ctx, name, DeleteOptions{Recurse: recurse})` | scaffolded |
| `errors.Is(err, geoserver.ErrNotFound)` | `errors.Is(err, geoserver.ErrNotFound)` | unchanged (sentinel names preserved) |
| `var e *geoserver.Error; errors.As(err, &e)` | `var e *geoserver.APIError; errors.As(err, &e)` | type rename |
| (more rows added as resources port) | | |

## When to upgrade

When v2 hits `v2.0.0` (final, not alpha/beta), this document will document a recommended migration window. Until then, v1.x is the supported path.

## Contributing to v2

v2 development happens in the same repository under `/v2/`. To contribute:

1. Open / claim an issue scoping the resource port (e.g., "Port datastores to v2").
2. Follow the `workspaces` reference resource (`v2/rest/workspaces/`) as the pattern.
3. Open a PR. CI runs `Unit tests v2 (Go 1.25)` against `/v2/`.

See [`../CONTRIBUTING.md`](../CONTRIBUTING.md) for the general PR workflow.
