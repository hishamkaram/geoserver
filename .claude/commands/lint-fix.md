---
description: Run golangci-lint with autofix where supported, then gofmt and goimports across the module. Reports the diff produced and any remaining unfixable findings.
allowed-tools: Bash(make fmt) Bash(make lint) Bash(golangci-lint:*) Bash(gofmt:*) Bash(goimports:*) Bash(git diff --stat) Bash(git diff)
---

Apply automatic lint and format fixes to the working tree.

Steps:

1. **Run autofix:** `golangci-lint run --fix ./...`. If `golangci-lint` isn't on `$PATH`, fall back to `make lint` — the Makefile installs `v2.1.6` on demand.
2. **Format:** `make fmt` (which runs `gofmt -s -w` then `goimports -w -local github.com/hishamkaram/geoserver`).
3. **Show what changed:** `git diff --stat` — concise summary of touched files.
4. **List remaining manual fixes.** Re-run `golangci-lint run ./...` (without `--fix`). For each remaining finding, format as `file:line — linter: message`. Group by linter.
5. Report verdict: **CLEAN** if no remaining findings, or **NEEDS WORK** with the manual-fix list.

Do NOT commit, push, or open a PR. The user reviews the diff and commits manually.
