# Docker dev/test stack

This directory builds the GeoServer container the project's integration tests run against. It is a **dev/test stack only** — do not use this image in production. The admin password is the literal string `geoserver`, the container ships without TLS, and JVM/resource limits are tuned for a fast `compose up` rather than for a hardened deploy.

## What's in the image

| Layer | Choice | Why |
|---|---|---|
| Base | `tomcat:9-jdk17-temurin` | GeoServer 2.x uses the `javax.*` servlet namespace and does not run on Tomcat 10/11 (which moved to `jakarta.*`). GeoServer 3.0 will be the Tomcat 11 / Jakarta EE target — see [`../ROADMAP.md`](../ROADMAP.md). |
| GeoServer | 2.28.0 by default; 2.27.4 LTS for the test leg | The supported matrix. CI runs both legs on every PR. Override with `--build-arg GEOSERVER_VERSION=...`. |
| Layout | WAR pre-extracted into `webapps/geoserver/` | Lets the Dockerfile drop extension JARs into `WEB-INF/lib/`. Tomcat happily runs the unpacked form. |
| Importer extension | Baked in | Required for the v2 SDK's `c.Imports` integration tests. Without it `GET /rest/imports` returns 404 and the suite skips silently. With the bake-in, CI exercises the full `v2/rest/imports` surface against a real server on both 2.27.4 and 2.28.0. |
| Healthcheck | `curl -fsS http://localhost:8080/geoserver/web/` every 30s after a 120s start period | Compose `depends_on` waits on this before starting the test runner. |

The Importer extension is the only non-vanilla piece. Everything else is the upstream GeoServer WAR + standard Tomcat config.

## Boot the stack

From the repo root:

```bash
make compose-up        # default — GeoServer 2.28.0 + PostGIS 16
make test-integration  # runs go test -tags=integration ./...
make compose-down
```

For the LTS leg (used by CI's matrix integration job):

```bash
GEOSERVER_VERSION=2.27.4 make compose-up
```

PostGIS exposes port `5436` on the host (mapped from the container's `5432`) so it doesn't collide with a local Postgres install. Default credentials: `golang` / `golang`, database `gis`.

## Files in this directory

| File | Role |
|---|---|
| `Dockerfile` | Assembles the image. Pulls the GeoServer WAR + Importer plugin from `downloads.sourceforge.net` over TLS, unpacks the WAR, drops the importer JARs into `WEB-INF/lib/`. |
| `env/geoserver.env` | Environment variables consumed by `docker-compose.yml` (and `docker-compose.test.yml`) at container start: `GEOSERVER_ADMIN_PASSWORD`, JVM `INITIAL_MEMORY` / `MAXIMUM_MEMORY`, plus a few feature toggles (`ENABLE_JSONP`, `MAX_FILTER_RULES`, `OPTIMIZE_LINE_WIDTH`, `XFRAME_OPTIONS`). |
| `postgis/init/01-lbldyt.sql` | Bootstraps the `public.lbldyt` table the `TestPublishPostgisLayer` integration test publishes against. The PostgreSQL image runs `*.sql` files from `/docker-entrypoint-initdb.d/` alphabetically on first boot of an empty data volume. To re-run after a schema change, recreate the volume: `docker compose down -v && docker compose up -d --wait`. The table name is a historical fixture from the v1.0 era; see [`../docs/geoserver-rest-quirks.md`](../docs/geoserver-rest-quirks.md) quirk #5 for why the seed table needs columns at all. |

## Building the image manually

The compose files build automatically, but you can also build directly:

```bash
docker build \
  --build-arg GEOSERVER_VERSION=2.28.0 \
  -t geoserver-dev:2.28.0 \
  -f docker/Dockerfile \
  .
```

(The build context is the repo root because nothing in this Dockerfile actually needs files from outside `docker/`, but compose files invoke it that way and we keep the call sites consistent.)

## Production caveat

Do not deploy this image as-is. To use GeoServer in production, build your own image (or use the official one from the GeoServer project) with at minimum: a real admin password, TLS in front, JVM limits tuned for your traffic, persistent volumes for the data dir, and CORS / CSRF configured for your tenancy model.

## See also

- [`../docker-compose.yml`](../docker-compose.yml) — default dev stack (2.28).
- [`../docker-compose.test.yml`](../docker-compose.test.yml) — LTS-leg integration stack (2.27.4).
- [`../docs/version-compat.md`](../docs/version-compat.md) — supported Go × GeoServer matrix and the Tomcat 9 / JDK 17 rationale.
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — how to run lint, unit, and integration tests locally.
