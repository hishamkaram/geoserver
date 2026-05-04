# Runnable v2 examples

Each subdirectory is a self-contained `main` package demonstrating one v2 idiom. Run any of them with:

```bash
go run ./v2/examples/<name>
```

(Or `cd v2 && go run ./examples/<name>`.)

All examples expect a GeoServer + PostGIS stack reachable at the URL configured in code (default: `http://localhost:8080/geoserver/`, admin / geoserver). Boot the stack via `make compose-up` from the repo root.

| Example | Demonstrates |
|---|---|
| [`workspaces/`](workspaces/) | The functional-options constructor (`New(url, opts...) (*Client, error)`); flat sub-client (`c.Workspaces`); list / create / get / delete; `errors.Is` for sentinel matching. |
| [`publish-postgis/`](publish-postgis/) | End-to-end flow: workspace → PostGIS datastore → feature type → fetch the published layer. Exercises the hierarchical sub-clients (`InWorkspace`, `InDatastore`). |
| [`style-upload/`](style-upload/) | Two-step style publish: register metadata via `Create`, then `UploadSLD` with a raw-XML body. |
| [`error-handling/`](error-handling/) | Match GeoServer errors via `errors.Is(err, geoserver.ErrNotFound)` and inspect the typed `*geoserver.APIError` via `errors.As`. |

These are reference flows, not test fixtures. The unit suite (`make test-v2-unit`) and the v2 integration ramp-up cover the same surface more rigorously.

## Running against a different GeoServer

Each example reads the server URL and credentials from environment variables when set:

```bash
export GEOSERVER_URL="https://geoserver.example.com/geoserver/"
export GEOSERVER_USER="admin"
export GEOSERVER_PASS="hunter2"
go run ./v2/examples/workspaces
```

Defaults match `make compose-up`: `http://localhost:8080/geoserver/`, `admin`, `geoserver`.

## v1 examples

The parent [`examples/`](../../examples/) directory contains the v1 reference flows. Compare them side-by-side with the v2 versions here to see the API surface differences laid out in [`docs/migration-v1-to-v2.md`](../../docs/migration-v1-to-v2.md).
