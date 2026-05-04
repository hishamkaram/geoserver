# Runnable examples

Each subdirectory is a self-contained `main` package demonstrating one client idiom. Run any of them from the repo root with:

```bash
go run ./examples/<name>
```

All examples expect a GeoServer + PostGIS stack reachable at the URL configured in code (default: `http://localhost:8080/geoserver/`, admin / geoserver). Boot the stack via `make compose-up` from the repo root.

| Example | Demonstrates |
|---|---|
| [`workspaces/`](workspaces/) | The functional-options constructor (`New(url, opts...) (*Client, error)`); flat sub-client (`c.Workspaces`); list / create / get / delete; `errors.Is` for sentinel matching. |
| [`publish-postgis/`](publish-postgis/) | End-to-end flow: workspace → PostGIS datastore → feature type → fetch the published layer. Exercises the hierarchical sub-clients (`InWorkspace`, `InDatastore`). |
| [`style-upload/`](style-upload/) | Two-step style publish: register metadata via `Create`, then `UploadSLD` with a raw-XML body. |
| [`error-handling/`](error-handling/) | Match GeoServer errors via `errors.Is(err, geoserver.ErrNotFound)` and inspect the typed `*geoserver.APIError` via `errors.As`. |

These are reference flows, not test fixtures. The unit suite (`make test-unit`) and the integration suite (`make test-integration`) cover the same surface more rigorously.

## Running against a different GeoServer

Each example reads the server URL and credentials from environment variables when set:

```bash
export GEOSERVER_URL="https://geoserver.example.com/geoserver/"
export GEOSERVER_USER="admin"
export GEOSERVER_PASS="hunter2"
go run ./examples/workspaces
```

Defaults match `make compose-up`: `http://localhost:8080/geoserver/`, `admin`, `geoserver`.

## v1 examples

v1's reference flows live on the `release/v1` branch under `examples/`. See [`docs/migration-v1-to-v2.md`](../docs/migration-v1-to-v2.md) for the API differences.
