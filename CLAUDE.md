# Project memory for `github.com/hishamkaram/geoserver`

This file is auto-loaded by Claude Code at the start of every session in this repository. Keep it concise (~150 lines, hard cap 200) and focused on **what cannot be derived from the code itself**.

## Project identity

- **Module**: `github.com/hishamkaram/geoserver`
- **Purpose**: Go client library for the GeoServer REST API (workspaces, datastores, feature types, layers, layer groups, styles, coverages, namespaces, settings).
- **Maintainer / display name**: **Hesham Karm** (note: no trailing 'a'; first name is "Hesham" not "Hisham"). Use this exact spelling in LICENSE, README authorship, AUTHORS files, commit signatures, and any user-facing credits. The legacy GitHub handle `hishamkaram` and the email `hishamwaleedkaram@gmail.com` are historical identity handles — do not "correct" them.

## Versioning regime

- **v2 (the main line) lives at the repo root on `master`** with module path `github.com/hishamkaram/geoserver/v2`. The `/v2` suffix is required by Go's semantic import versioning rule for v2+ modules; it stays even though v2 is at the root. Latest published tag is `v2.0.0` — public API is stable; no breaking changes will land in v2.x. v2 has full v1 feature parity plus surfaces v1 never had: per-service OWS settings, file-upload publishing, layer–style associations, GeoWebCache (full surface), the Importer extension, the full ACL surface, the Resource API, the OWS read-only trio (GetCapabilities + DescribeFeatureType + DescribeCoverage), templates (FTL), auth providers / filters / chains, URL checks, cascaded WMS/WMTS, manifests + system status, runtime logging, fonts, password rotation, and monitoring (gs-monitor). Run `make test-unit` and `make test-integration`.
- **v1 is end-of-feature** on the `release/v1` branch (security patches only). The latest v1 tag is `v1.1.2` (deprecation marker via `// Deprecated:` in `go.mod`; no code change from `v1.1.1`). v1's wounds that need breaking changes (exported mutable fields, four-positional-arg `PublishPostgisLayer`, etc.) are healed in v2. v1.x is **still non-breaking** for the patches that do land on `release/v1` — no signature changes; deprecate via `// Deprecated:` and add a sibling. **Why:** v1.0 callers must be able to upgrade across the whole v1 line with only a `go.mod` bump.
- **Don't auto-tag releases.** Tagging is an explicit user action — never run `git tag` or push tags from a Claude session without explicit user authorization.

## Test split

- `*_unit_test.go` (no build tag) — **fast, hermetic, httptest-based.** Run by default.
- `*_test.go` with `//go:build integration` — **integration tests against a real GeoServer + PostGIS stack.** Never invoke without `--tags=integration` AND a live compose stack.
- The `make test-unit` and `make test-integration` targets are the canonical entry points; CI mirrors them exactly.
- **Both unit and integration tests are mandatory on every PR.** Integration runs against GeoServer 2.27.4 LTS and 2.28.0 stable; both legs must pass for the PR to merge.

## Context handling (mandatory for new exports)

v2 is **context-first**: every exported method on every sub-client takes `ctx context.Context` as its first argument. No `*Context` twins, no `context.Background()` delegators — that pattern was a v1.x compat shim and does not exist in v2. Canonical shapes: `rest/workspaces/workspaces.go:38,73`, `rest/about/about.go:84,99`.

```go
func (c *Client) GetFoo(ctx context.Context, args...) (...) { /* real impl uses ctx */ }
```

If you need a no-context entry point in caller code, use `context.Background()` at the call site — the library never papers over it.

## Typed errors

- All non-2xx GeoServer responses surface as `*APIError` (defined in `errors.go`). Renamed from v1's `*Error` as part of the v2 clean break.
- Status codes map to 12 sentinels via `errors.Is`: `ErrBadRequest`, `ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrMethodNotAllowed`, `ErrConflict`, `ErrUnsupportedMediaType`, `ErrRateLimited`, `ErrServerError`, `ErrBadGateway`, `ErrServiceUnavailable`, `ErrGatewayTimeout`.
- **Never compare error strings.** `errors.Is(err, ErrNotFound)` is the only correct test.
- The `*APIError.Error()` string format is `"geoserver: <Op> <Method> <URL>: <status> <statusText>: <body-preview>"` — body capped at 8 KiB internally and previewed at ~120 bytes. v1's `"abstract:%s\ndetails:%s\n"` format was deliberately dropped at the v2 boundary.

## Logging

- v2 uses `*slog.Logger` directly. The `*Logger` wrapper and `logging.go` from v1 do not exist — that wrapper was a v1.0-source-compat shim and was deliberately dropped.
- Configure via `WithLogger(l *slog.Logger)`. Pass `slog.New(slog.DiscardHandler)` to silence; the default is the discard logger.
- Internal call sites use structured logging — `logger.Debug(msg, args...)` with key/value pairs, not printf-style.
- Library logs Debug for HTTP details, Warn for retry-exhausted, Error for protocol violations. No Info chatter.

## Concurrency

- `*Client` is **immutable after `New(...)` returns.** All struct fields are private or pointers to sub-clients set once at construction and never reassigned. Concurrent use across goroutines is safe by design — no caller-side locking required.
- Same posture for every sub-client (`workspaces.Client`, `datastores.Client`, etc.): they expose methods only, holding a single private `core Core` interface reference.
- Don't introduce shared mutable state to `clientCore` or any sub-client (caches, counters, request-scoped buffers held as struct fields). If you need per-call state, allocate it inside the method. The race-proof guarantee is verified by `TestClient_ConcurrentRequests` running under `go test -race` in CI.
- User-supplied transports (`WithHTTPClient`, `WithTransport`) are the caller's responsibility — if their `RoundTripper` mutates shared state, the race lives in their code, not ours.

## GeoServer version matrix

- **Supported: 2.27 LTS + 2.28 stable.** Integration tests run against both via the CI matrix (`integration.yml`).
- Don't add code paths gated on other GeoServer versions without adding a corresponding CI matrix entry.
- GeoServer 3.0 (April 2026 GA, Jakarta EE / Tomcat 11 / ImageN) is parked for a v2.x point release after the migration settles.

## Common GeoServer REST quirks (cross-reference)

- Workspace-scoped `POST /workspaces/{ws}/styles` requires `Accept: */*`, not `application/json` — see `rest/styles/styles.go`.
- Empty styles collection comes back as `{"styles":""}` (bare string, not object) — `json.RawMessage` decode at `rest/styles/styles.go:85` tolerates both shapes.
- `LayerGroup.styles.style` is a mixed `[string|object]` array — custom `UnmarshalJSON` in `rest/layergroups/types.go:97-108`. The `Published` type has the same shape — `rest/layergroups/types.go:57-60`.
- Full quirk reference: `/skill geoserver-rest-quirks` or read `.claude/skills/geoserver-rest-quirks/SKILL.md`.

## Build & lint surfaces

- `make` is the canonical entry point. CI workflow names match Make targets exactly. Don't bypass `make` with raw `go test`/`golangci-lint` invocations in scripts.
- `.golangci.yml` enables: errcheck, govet, staticcheck, ineffassign, unused, bodyclose, errorlint, noctx, copyloopvar, revive, gocritic, misspell, unconvert, prealloc, gosec.
- Don't add `//nolint:` comments outside the existing exemptions in `.golangci.yml`. The remaining live exemption for v2 covers `internal/transport/transport.go` G704 gosec (SSRF false-positive on URLs built from path-escaped configured segments).

## HTTP & URL hygiene

- Every sub-client call goes through `coreAdapter.Do(ctx, op, method, url, body, query, out)` (geoserver.go:429), which delegates to `transport.DoJSON / DoXML / DoRaw / DoStream` in `internal/transport/transport.go`. Don't call `http.Client.Do` directly outside `internal/transport/`.
- URL building uses `coreAdapter.URL(parts...)` (geoserver.go:422) which delegates to `transport.BuildURL` (`internal/transport/url.go`); each segment is path-escaped. Never `fmt.Sprintf("%srest/%s...", ...)` — that pattern was the source of multiple v1.0 bugs.

## Conventions and don'ts

- **Never commit directly to `master`.** Always create a feature branch, push it, open a PR, wait for CI to go green, then squash-merge.
- **Never add Claude (or any AI assistant) as a git co-author.** Do not append `Co-Authored-By: Claude ...` trailers. Commit messages are authored by the user only.
- **Never commit planning markdowns** — design docs, revival plans, research notes belong in `~/.claude/plans/`, not in this repo.
- **No panics in library code.** v2 library code (root + `rest/` + `ows/` + `internal/`) currently contains zero `panic(` calls; don't reintroduce. Tests may use `t.Fatalf` freely.
- **No new runtime dependencies** without prior discussion — keep `go.mod` minimal (currently only stdlib + `testify` test-only + `yaml.v3` for `LoadConfig`).
- **Don't auto-tag releases** and don't merge a PR with red or pending CI — both are explicit user actions.

## Index of project Claude config

Subagents (delegated workers, own context window):

- `go-reviewer` — surfaces idiom violations and v1.1.x non-breaking constraint breaches
- `integration-runner` — boots compose, runs integration suite, dumps logs on failure
- `breaking-change-checker` — verifies non-breaking-v1 contract before tagging

Skills (loadable knowledge / procedures):

- `/geoserver-rest-quirks` — GeoServer 2.x REST API quirks reference (auto-loads when relevant)
- `/non-breaking-v1` — pre-PR checklist for v1.1.x non-breaking contract (manual)
- `/add-context-twin <method-name> <file>` — adds a *Context sibling method following the canonical pattern

Slash commands (callable recipes):

- `/integration-test [version]` — boot compose stack and run integration suite
- `/lint-fix` — golangci-lint with autofix + gofmt + goimports
- `/release-prep` — local-runnable subset of CI gates for a v1.1.x tag
