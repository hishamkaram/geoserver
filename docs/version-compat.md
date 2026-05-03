# Version compatibility

This document is the reference for what versions of Go and GeoServer this library supports, and why.

## Go

| Go version | Status | Notes |
|---|---|---|
| **1.25.x** | Supported (CI) | `go-version: "1.25"` with `check-latest: true` so CI auto-pulls the latest patch (currently 1.25.9). |
| 1.26.x | Untested | Will likely work; CI doesn't run against it yet. |
| ≤ 1.24 | Unsupported | `go.mod` declares `go 1.25`; older toolchains will refuse to build. |

**Why 1.25 minimum:** The library uses `slog.DiscardHandler` (Go 1.24+ stdlib) and is built / linted with golangci-lint v2.12.x which is itself built with Go 1.25. Pinning to 1.25 also clears two crypto/x509 + crypto/tls advisories that affect 1.25.8 and earlier (`GO-2026-4946`, `GO-2026-4870`) — fixed in 1.25.9.

CI uses `setup-go@v6` with `check-latest: true`, so the unit, vuln, and integration jobs always run against the latest 1.25.x patch.

## GeoServer

| GeoServer | Status | CI matrix | Notes |
|---|---|---|---|
| **2.27.x LTS** | Supported (CI) | `GeoServer 2.27.4` | Long-term support release. |
| **2.28.x stable** | Supported (CI) | `GeoServer 2.28.0` | Current stable. |
| 2.18 – 2.26 | Best-effort | not in CI | Many endpoints work; security and feature-type-discovery responses may differ in shape. |
| ≤ 2.17 | Unsupported | not in CI | Pre-modern security API; major drift in JSON response shapes. |
| 3.0.x | Tracked for v2.x | not in CI | Jakarta EE / Tomcat 11 / ImageN raster engine. Validates only after the migration settles in production deploys. See [`../ROADMAP.md`](../ROADMAP.md). |

**Integration coverage:** every PR runs the full integration suite against both 2.27.4 LTS and 2.28.0 stable. Both legs must pass before merge. Cross-version differences in REST response shapes are decoded transparently — `security.go` handles both `roles` and `roleNames` keys, both `groups` and `groupNames`, etc. See [`geoserver-rest-quirks.md`](geoserver-rest-quirks.md) for the full quirks catalog.

## Tomcat / Java

The dev / test Docker stack uses **Tomcat 9 + JDK 17 (Temurin)**. Why Tomcat 9 specifically: GeoServer 2.x (including the supported 2.27 / 2.28) is built against the `javax.*` servlet namespace. Tomcat 10+ moved to `jakarta.*` and breaks GeoServer 2.x at WAR-deploy time. GeoServer 3.0 will unblock Tomcat 11 — see roadmap.

This affects only the test stack. Production users build their own GeoServer container or use the official one; this library doesn't dictate runtime. See [`../docker/README.md`](../docker/README.md) for the full dev/test image contents (Importer extension bake-in, env vars, PostGIS seed).

## PostGIS

The dev / test Docker stack uses **`postgis/postgis:16-3.4`**. Older PostGIS versions (≥ 2.5) work for the integration tests but aren't gated in CI.

## Module path

| Path | Status |
|---|---|
| `github.com/hishamkaram/geoserver` | v1.x — supported |
| `github.com/hishamkaram/geoserver/v2` | v2.x — beta (latest `v2.0.0-beta.1`); public API frozen for review until `v2.0.0` final |
| `gopkg.in/hishamkaram/geoserver.v1` | Legacy alias — deprecated; resolves to the same source. New code should import the canonical path. |

## When this matrix changes

- **Go 1.26 release** → add to the matrix as untested, then move to supported once CI validates.
- **GeoServer 2.29 release** → swap 2.28 for 2.29 in the integration matrix; keep 2.27 LTS as the LTS leg.
- **GeoServer 2.27 LTS retires** → drop from the matrix when GeoServer's own LTS support ends.
- **GeoServer 3.0 stabilizes** → add as a third matrix entry once Jakarta EE / Tomcat 11 / ImageN settle.

Roadmap context: [`../ROADMAP.md`](../ROADMAP.md).
