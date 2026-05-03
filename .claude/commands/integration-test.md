---
description: Boot the GeoServer + PostGIS docker-compose stack and run the integration test suite. Tears down on success, leaves the stack up on failure for inspection.
argument-hint: [version]
allowed-tools: Bash(make compose-up) Bash(make compose-test-up) Bash(make compose-down) Bash(make compose-test-down) Bash(make test-integration) Bash(docker compose:*)
---

Run the integration suite against GeoServer `$ARGUMENTS` (default: `2.28.0`; pass `2.27` for the LTS leg).

Steps:

1. **Boot the stack.**
   - `2.28` (default) → `make compose-up` (uses `docker-compose.yml`).
   - `2.27` → `make compose-test-up` (uses `docker-compose.test.yml`).
2. **Wait for healthcheck.** Poll `docker compose ps` until the geoserver service is `healthy`. First boot takes ~60s while Tomcat unpacks the WAR.
3. **Run the suite:** `make test-integration`. Capture full output.
4. **On success:**
   - `2.28` → `make compose-down`.
   - `2.27` → `make compose-test-down`.
   - Report green: test count + duration.
5. **On failure:**
   - Print the failed test names (grep for `--- FAIL:` blocks).
   - Dump `docker compose logs --tail=200 geoserver`.
   - Dump `docker compose logs --tail=100 postgis` if any failure mentions PostGIS / featuretypes / shapefile.
   - **Do NOT tear the stack down** — leave it running so the user can `curl` against it.
   - Stop. Don't propose code edits.
