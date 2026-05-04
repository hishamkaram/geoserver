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

## *Context twin pattern (mandatory for new exports)

Every exported method on `*GeoServer` has a `…Context(ctx context.Context, …)` sibling. The non-context form delegates with `context.Background()`. New methods MUST follow this. Canonical shape: `workspaces.go:16-38,57-79`.

```go
func (g *GeoServer) GetFoo(args...) (...)        { return g.GetFooContext(context.Background(), args...) }
func (g *GeoServer) GetFooContext(ctx context.Context, args...) (...) { /* real impl uses ctx */ }
```

Each service interface has a parallel `…WithContext` interface (e.g., `WorkspaceServiceWithContext`). The legacy interface stays alongside.

## Typed errors

- All HTTP errors return `*Error` (defined in `errors.go`).
- Status codes map to sentinels via `errors.Is` (`ErrNotFound`, `ErrUnauthorized`, `ErrForbidden`, `ErrConflict`, `ErrBadRequest`, `ErrMethodNotAllowed`, `ErrUnsupportedMediaType`, `ErrRateLimited`, `ErrServerError`).
- **Never compare error strings.** `errors.Is(err, ErrNotFound)` is the only correct test.
- The `*Error.Error()` string preserves v1.0's `"abstract:%s\ndetails:%s\n"` format so existing string-matchers don't break.

## Logging

- `g.logger` is a `*Logger` wrapper (defined in `logging.go`) over `*slog.Logger`.
- Exposes printf-style: `Errorf`, `Warnf`, `Infof`, `Debugf`. Plus Sprint-style: `Error`, `Warn`, `Info`, `Debug`. This shape exists for v1.0-source compatibility.
- Configure via the `WithLogger(slog.Handler)` option. **Never mutate fields** to swap the logger.
- Library logs Debug for HTTP details, Warn for retry-exhausted, Error for protocol violations. No Info chatter.

## Concurrency

- `*GeoServer` exported fields are **not safe for concurrent mutation.** Construct once with `New(...)` and treat as immutable.
- Concurrent reads on the same instance are safe.
- Don't add concurrency hardening (locks, atomic swaps) to v1 — that is a v2 redesign concern.

## GeoServer version matrix

- **Supported: 2.27 LTS + 2.28 stable.** Integration tests run against both via the CI matrix (`integration.yml`).
- Don't add code paths gated on other GeoServer versions without adding a corresponding CI matrix entry.
- GeoServer 3.0 (April 2026 GA, Jakarta EE / Tomcat 11 / ImageN) is parked for a v2.x point release after the migration settles.

## Common GeoServer REST quirks (cross-reference)

- Workspace-scoped `POST /workspaces/{ws}/styles` requires `Accept: */*`, not `application/json` — see `styles.go:178-186`.
- Empty styles collection comes back as `{"styles":""}` (bare string, not object) — see `styles.go:93-104`.
- `LayerGroup.styles.style` is a mixed `[string|object]` array — custom `UnmarshalJSON` in `layergroups.go`.
- Full quirk reference: `/skill geoserver-rest-quirks` or read `.claude/skills/geoserver-rest-quirks/SKILL.md`.

## Build & lint surfaces

- `make` is the canonical entry point. CI workflow names match Make targets exactly. Don't bypass `make` with raw `go test`/`golangci-lint` invocations in scripts.
- `.golangci.yml` enables: errcheck, govet, staticcheck, ineffassign, unused, bodyclose, errorlint, noctx, copyloopvar, revive, gocritic, misspell, unconvert, prealloc, gosec.
- Don't add `//nolint:` comments outside the existing exemptions in `.golangci.yml` (which cover v1.x compat-frozen field names like `HttpClient`/`Id`/`XmlPostRequestLogBufferSize` and historic error string capitalization in `vars.go`).

## HTTP & URL hygiene

- Every request goes through `g.DoRequestContext` (in `utils.go`). Don't call `http.Client.Do` directly outside `utils.go`.
- URL building uses `g.ParseURL(parts...)` which path-escapes each segment. Never `fmt.Sprintf("%srest/%s...", ...)` — that pattern was the source of multiple v1.0 bugs.

## Conventions and don'ts

- **Never commit directly to `master`.** Always create a feature branch, push it, open a PR, wait for CI to go green, then squash-merge.
- **Never add Claude (or any AI assistant) as a git co-author.** Do not append `Co-Authored-By: Claude ...` trailers. Commit messages are authored by the user only.
- **Never commit planning markdowns** — design docs, revival plans, research notes belong in `~/.claude/plans/`, not in this repo.
- **No panics in library code.** The v1.0 audit removed five (`utils.go:49,60,119,134`; `wms/wms.go:213`); don't reintroduce. Tests may use `t.Fatalf` freely.
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
