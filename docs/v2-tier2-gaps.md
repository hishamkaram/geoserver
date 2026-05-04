# v2 — tier-2 gap-analysis backlog

The v2 client closes every "everyone needs it" REST API surface plus two of the original tier-2 items (security ACL services / REST / catalog rules and the Resource API) in the published `v2.0.0-beta.1` (see [`../v2/CHANGELOG.md`](../v2/CHANGELOG.md)). What's documented here is the remaining tier-2 backlog — endpoint groups that real deployments do reach for, but that help a narrower audience than the surface already shipped.

Each entry below is independently tractable as its own follow-up PR. None block `v2.0.0`. Each is grounded in the official GeoServer REST docs (`https://docs.geoserver.org/latest/en/user/rest/`) and reuses the existing v2 plumbing (`internal/transport.BuildURL`, `transport.DoJSON` / `DoXML` / `DoRaw`, the per-resource `Core` interface, the `*Client → InWorkspace(ws) → *WorkspaceClient` scoping pattern). PRs welcome — open an issue first if the work touches a new wire-format quirk so the design conversation can happen in public.

## Backlog (priority order)

| # | Gap | Audience | Rough scope |
|---|-----|----------|-------------|
| 1 | **URL checks** — `/urlchecks` | SSRF-conscious deployments | Allow / deny lists for external URL references in styles, mosaics, and remote stores. |
| 2 | **Cascaded WMS / WMTS stores** — `/wmsstores`, `/wmtsstores` | Federation / proxy setups | Re-publish a remote WMS / WMTS through your own server. |
| 3 | **XSLT transforms** — `/services/wfs/transforms` | WFS-T payload customizers | Upload XSLT for custom GetFeature output formats. |
| 4 | **Manifests + system status** — `/about/manifest`, `/about/system-status` | Ops / capacity planning | Inspect installed plugins; live JVM and OS stats. |
| 5 | **Logging** — `/logging` | Production debugging | Adjust log levels and configurations at runtime without bouncing the server. |

## Already shipped

- **ACL services / REST / catalog rules** — `c.ACL.Services()` / `c.ACL.REST()` / `c.ACL.Catalog()`. Closed in beta.1.
- **Resource API** — `c.Resources` Get / List / Stat / Exists / Put / Move / Copy / Delete against `/rest/resource/{path}`. Closed in beta.1.
- **Mosaic / structured-coverage granules** — `c.Coverages.InWorkspace(ws).InCoverageStore(cs).Granules(cov)` Schema / List / Get / Delete / DeleteByFilter. Closed post-beta.1.
- **Templates (FTL)** — `c.Templates` (global) plus six fluent scopes (`InWorkspace`/`InDatastore`/`InFeatureType`/`InCoverageStore`/`InCoverage`); List / Get / Put / PutString / Delete. Closed post-beta.1.
- **Auth providers + filter chains** — `c.Security.AuthProviders` / `c.Security.AuthFilters` / `c.Security.FilterChains` (each List / Get / Create / Update / Delete; AuthProviders + FilterChains also have SetOrder). Closed post-beta.1.

Beyond these: CRS list, fonts list, monitoring, master password, self-admin password, usergroup-service registration, individual filter-chain editing, and the OWS `oseo` (OpenSearch for Earth Observation) service settings. Each is a single endpoint or two and can ride along with whichever neighbor lands first.

## How to contribute

1. Pick an item, file an issue summarizing the surface you intend to add (URL paths, request shapes, return shapes — verify against the official docs and the upstream OpenAPI YAML at `geoserver/geoserver/doc/en/api/1.0.0/`).
2. Match the existing v2 patterns — see [`../v2/README.md#contributing-to-v2`](../v2/README.md#contributing-to-v2) for the per-pattern reference subpackage to copy from.
3. Run integration locally before push — `make compose-up && cd v2 && go test -tags=integration ./rest/<resource>/`.
4. CI's wire-format coverage runs on real GeoServer 2.27.4 LTS + 2.28.0 stable; both legs must pass.

## Out of scope this round

- **OGC API endpoints** (Tiles / Features / Maps / Styles / DGGS) — data-delivery endpoints, not config. v2 today is a config / admin client; whether to also be a *consumer* of OGC API services is a separate scoping conversation.
- **GeoServer 3.0 support** — once Jakarta EE / Tomcat 11 / ImageN settle. Tracked in [`../ROADMAP.md`](../ROADMAP.md).
- **WFS-T / GetMap / GetCoverage operations** — high-volume request-path operations, not admin operations. Different perf and streaming requirements.
- **GeoWebCache endpoints not documented at `/latest/`** — masstruncate, blobstores, gridsets, statistics, global. Land once their source-of-truth docs URL is established.

See also [`../ROADMAP.md`](../ROADMAP.md) for v1.x maintenance, v2.x milestones, and GeoServer 3.0 timeline.
