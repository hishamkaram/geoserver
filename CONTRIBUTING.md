# Contributing to geoserver

Thanks for your interest in contributing! This document describes how to get a development environment running and what we expect from pull requests.

## Development environment

Requirements:

- Go **1.23+** (CI tests against 1.23 and 1.25)
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

1. **Branch from `master`.** Use a short descriptive name (`fix/url-escape`, `feat/context-methods`). Never commit directly to `master`.
2. **One concern per PR.** Smaller PRs review faster.
3. **Conventional Commits.** Commit messages follow [Conventional Commits 1.0](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`, `build:`, `ci:`). The CHANGELOG is generated from these.
4. **Tests are mandatory.** New code needs unit tests (`*_unit_test.go`, `httptest`-based). Behavioral changes also need an integration test entry. **Integration tests run on every PR** against both GeoServer 2.27.4 LTS and 2.28.0 stable — both legs must pass.
5. **Lint clean.** `make lint` must pass with zero warnings.
6. **Backward compatibility.** v1.x is non-breaking. If your change requires breaking a public symbol, open a discussion first — it likely belongs in v2.
7. **No new runtime dependencies** without prior discussion. Test-only deps are fine.
8. **All CI checks must pass before merge.** The required gates on every PR: `Lint`, `Unit tests (Go 1.23)`, `Unit tests (Go 1.25)`, `govulncheck`, `Analyze (Go)` (CodeQL), `GeoServer 2.27.4`, `GeoServer 2.28.0`. Don't merge with any check red, pending, or skipped.
9. **Squash merge.** PRs are squash-merged into `master` so each merge produces exactly one commit on the trunk and the CHANGELOG stays clean. Don't use rebase- or merge-commit strategies.

## Project layout

```
.
├── *.go                       # v1 public API (one file per resource)
├── wms/                       # WMS XML types (parser)
├── docker/                    # Dockerfile for the dev GeoServer
├── docker-compose.yml         # Default dev stack (GeoServer 2.28 + PostGIS 16)
├── docker-compose.test.yml    # Integration test stack with 2.27 LTS leg
├── testdata/                  # Test fixtures (SLDs, shapefiles, JSON responses)
└── scripts/                   # Test helper scripts
```

## Reporting bugs / asking questions

- **Bugs:** open a GitHub issue with the bug-report template.
- **Security issues:** see [SECURITY.md](SECURITY.md). Do not open a public issue.
- **Questions:** GitHub Discussions if available, otherwise an issue.

## Releasing

Releases are cut by maintainers via tags (`v1.x.y`). The `release.yml` workflow assembles release notes from Conventional Commit messages.

## Working with Claude Code in this repo

The repo ships project-scoped Claude Code config so any contributor using Claude Code (CLI, IDE extensions, web app) gets the same conventions and shortcuts. Auto-loaded from `CLAUDE.md` (root) and `.claude/`.

Available helpers:

| Type | Name | Purpose |
|---|---|---|
| Slash command | `/integration-test [version]` | Boot the docker-compose stack and run the integration suite (default 2.28; `2.27` for the LTS leg). |
| Slash command | `/lint-fix` | Run `golangci-lint --fix` + `gofmt` + `goimports`, report the diff and any remaining manual fixes. |
| Slash command | `/release-prep` | Local-runnable subset of CI gates for a v1.1.x tag. |
| Skill | `/add-context-twin <method> <file>` | Apply the *Context twin pattern to a new method on `*GeoServer`. |
| Skill | `/non-breaking-v1` | Pre-PR checklist for the v1.1.x non-breaking contract. |
| Skill | `/geoserver-rest-quirks` | Reference for GeoServer 2.x REST quirks the client works around (auto-loads when relevant). |
| Subagent | `go-reviewer` | Idiom + non-breaking review of changed Go files. |
| Subagent | `integration-runner` | Boots the stack, runs integration tests, diagnoses failures from container logs. |
| Subagent | `breaking-change-checker` | Computes the exported-API diff against `master` / `v1.0.0` and classifies each change. |

Personal per-machine settings live in `.claude/settings.local.json` (gitignored). The team baseline (permissions, attribution, hooks) is intentionally **not** committed yet — see `~/.claude/plans/tranquil-hatching-rabin.md` for the design if we decide to add it later.
