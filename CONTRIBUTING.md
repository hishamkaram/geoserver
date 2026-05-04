# Contributing to geoserver

Thanks for your interest in contributing! This document describes how to get a development environment running and what we expect from pull requests.

## Development environment

Requirements:

- Go **1.25+** (matches the `go.mod` directive; CI runs unit, integration, and govulncheck against 1.25.x)
- Docker + Docker Compose v2 (for integration tests)
- `make`
- `golangci-lint` v1.59+ (`go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`)
- `govulncheck` (`go install golang.org/x/vuln/cmd/govulncheck@latest`)

Clone and bootstrap:

```bash
git clone https://github.com/hishamkaram/geoserver
cd geoserver
make tidy
make lint
make test-unit
```

To run the full integration suite:

```bash
make compose-up           # boots GeoServer + PostGIS
make test-integration     # runs tests with -tags=integration
make compose-down
```

See [`docker/README.md`](docker/README.md) for what's in the image (Importer extension baked in, supported GeoServer versions, env file, PostGIS seed).

## Make targets

| Target | What it does |
|---|---|
| `make test` | Runs `make test-unit` then `make test-integration` |
| `make test-unit` | Unit tests, no Docker required (`go test -race -short ./...`) |
| `make test-integration` | Integration tests against compose-managed GeoServer + PostGIS |
| `make lint` | `golangci-lint run` |
| `make fmt` | `gofmt -s -w` + `goimports` |
| `make tidy` | `go mod tidy && go mod verify` |
| `make vuln` | `govulncheck ./...` |
| `make cover` | Runs unit tests with coverage profile |
| `make compose-up` / `make compose-down` | Start/stop the dev stack |

## Pull requests

1. **Pick the right base branch.** Most PRs branch from `master` (the v2 main line). Security patches for v1 branch from `release/v1` and must remain non-breaking — see point 6. Use a short descriptive name (`fix/url-escape`, `feat/coverage-iter`). Never commit directly to either branch.
2. **One concern per PR.** Smaller PRs review faster.
3. **Conventional Commits.** Commit messages follow [Conventional Commits 1.0](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`, `build:`, `ci:`). The CHANGELOG is generated from these.
4. **Tests are mandatory.** New code needs unit tests (`*_unit_test.go`, `httptest`-based). Behavioral changes also need an integration test entry. **Integration tests run on every PR** against both GeoServer 2.27.4 LTS and 2.28.0 stable — both legs must pass.
5. **Lint clean.** `make lint` must pass with zero warnings.
6. **Backward compatibility depends on the target branch.** PRs against `master` (v2) MAY introduce breaking changes — v2 is the active line and a major-version bump is the natural home for them; flag them clearly in the description. PRs against `release/v1` MUST be non-breaking: deprecate via `// Deprecated:` and add a sibling rather than changing signatures.
7. **No new runtime dependencies** without prior discussion. Test-only deps are fine.
8. **All CI checks must pass before merge.** The required gates on every PR: `Lint`, `Unit tests (Go 1.25)`, `govulncheck`, `Analyze (Go)` (CodeQL), `GeoServer 2.27.4`, `GeoServer 2.28.0`. Don't merge with any check red, pending, or skipped.
9. **Squash merge.** PRs are squash-merged into the target branch so each merge produces exactly one commit on the trunk and the CHANGELOG stays clean. Don't use rebase- or merge-commit strategies.

## Project layout

The `master` branch is v2 (`github.com/hishamkaram/geoserver/v2`). The v1 line lives on `release/v1` and is end-of-feature (security fixes only); its layout is documented in CONTRIBUTING.md on that branch.

```
.
├── *.go                       # Root package: *Client, options, *APIError, sentinels (geoserver.go, errors.go, options.go, doc.go)
├── rest/                      # One subpackage per REST resource (workspaces, datastores, featuretypes,
│                              #   coveragestores, coverages, layers, layergroups, styles, namespaces,
│                              #   settings, about, security, acl, system, imports, gwc, services,
│                              #   resources, templates, urlchecks, wmsstores, wmslayers, wmtsstores,
│                              #   wmtslayers, wfstransforms, logging, fonts, monitor — 28 in total)
├── ows/                       # OWS read-only clients: wms, wfs, wcs (GetCapabilities + descriptors)
├── internal/transport/        # HTTP dispatch (DoJSON, DoXML, DoRaw, DoStream) and URL building (BuildURL)
├── internal/wire/             # Helpers for delicate wire-format quirks (mixed-shape arrays, etc.)
├── examples/                  # Runnable reference flows (workspaces, publish-postgis, style-upload, error-handling)
├── docs/                      # Architecture, REST quirks, version compat, v1→v2 migration, tier-2 gap backlog
├── docker/                    # Dockerfile for the dev/test GeoServer (Importer + Monitor extensions baked in)
├── docker-compose.yml         # Default dev stack (GeoServer 2.28 + PostGIS 16)
├── docker-compose.test.yml    # Integration test stack with 2.27 LTS leg
├── testdata/                  # Test fixtures (SLDs, shapefiles, JSON responses)
├── scripts/                   # Test helper scripts
└── .github/                   # Issue / PR templates, CODEOWNERS, Dependabot, workflows
```

## Reporting bugs / asking questions

- **Bugs:** open a GitHub issue with the bug-report template.
- **Security issues:** see [SECURITY.md](SECURITY.md). Do not open a public issue.
- **Questions:** GitHub Discussions if available, otherwise an issue.

## Releasing

Releases are cut by maintainers via tags. v2 releases (`v2.x.y`) tag from `master`; v1 security-patch releases (`v1.1.x`) tag from `release/v1`. The `release.yml` workflow assembles release notes from Conventional Commit messages.

`CHANGELOG.md` on `master` is the v2 changelog; the v1 changelog lives at `CHANGELOG.md` on the `release/v1` branch. Both follow [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The `[Unreleased]` block at the top of each file accumulates entries between tags; cutting a release promotes the block contents into a dated stanza and leaves a fresh `[Unreleased]` placeholder.

## Working with Claude Code in this repo

The repo ships project-scoped Claude Code config so any contributor using Claude Code (CLI, IDE extensions, web app) gets the same conventions and shortcuts. Auto-loaded from `CLAUDE.md` (root) and `.claude/`.

Available helpers:

| Type | Name | Purpose |
|---|---|---|
| Slash command | `/integration-test [version]` | Boot the docker-compose stack and run the integration suite (default 2.28; `2.27` for the LTS leg). |
| Slash command | `/lint-fix` | Run `golangci-lint --fix` + `gofmt` + `goimports`, report the diff and any remaining manual fixes. |
| Slash command | `/release-prep` | Local-runnable subset of CI gates for a v1.1.x tag on `release/v1` (v1-only — v2 ships via a separate process). |
| Skill | `/add-context-twin <method> <file>` | Apply the `*Context` twin pattern to a new method on `*GeoServer`. **`release/v1` only** — v2 is context-first natively, no twins. |
| Skill | `/non-breaking-v1` | Pre-PR checklist for the v1.1.x non-breaking contract. **`release/v1` only.** |
| Skill | `/geoserver-rest-quirks` | Reference for GeoServer 2.x REST quirks the client works around (auto-loads when relevant). |
| Subagent | `go-reviewer` | Go-idiom review of changed files. Adapts its rule set per target branch — adds non-breaking-v1 + twin checks when reviewing PRs against `release/v1`. |
| Subagent | `integration-runner` | Boots the stack, runs integration tests, diagnoses failures from container logs. |
| Subagent | `breaking-change-checker` | Computes the exported-API diff against `release/v1` / `v1.0.0` and classifies each change. **`release/v1` only.** |

Personal per-machine settings live in `.claude/settings.local.json` (gitignored). The team baseline (permissions, attribution, hooks) is intentionally **not** committed; revisit if the project grows enough contributors to warrant it.
