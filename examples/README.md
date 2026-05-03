# Runnable examples

Each subdirectory is a self-contained `main` package demonstrating one v1.1 idiom. Run any of them with:

```bash
go run ./examples/<name>
```

All examples expect a GeoServer + PostGIS stack reachable at the URL configured in code (default: `http://localhost:8080/geoserver/`, admin / geoserver). Boot the stack via `make compose-up` from the repo root.

| Example | Demonstrates |
|---|---|
| [`workspaces/`](workspaces/) | Functional-options constructor (`New`), `*Context` methods, list / create / delete a workspace. |
| [`publish-postgis/`](publish-postgis/) | End-to-end flow: workspace → PostGIS datastore (with `Options`) → `CreateFeatureType` → fetch the published layer. |
| [`style-upload/`](style-upload/) | Stream an SLD file from disk into GeoServer via `UploadStyle`. |
| [`error-handling/`](error-handling/) | Match GeoServer errors via `errors.Is(err, geoserver.ErrNotFound)` and inspect the typed `*geoserver.Error` via `errors.As`. |

These are reference flows, not test fixtures. The integration test suite (`make test-integration`) covers the same surface more rigorously; run those for behavior verification.

## Running against a different GeoServer

Each example reads the server URL and credentials from environment variables when set:

```bash
export GEOSERVER_URL="https://geoserver.example.com/geoserver/"
export GEOSERVER_USER="admin"
export GEOSERVER_PASS="hunter2"
go run ./examples/workspaces
```

Defaults match `make compose-up`: `http://localhost:8080/geoserver/`, `admin`, `geoserver`.
