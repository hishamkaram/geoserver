---
description: Pre-release verification for a v1.1.x tag on the `release/v1` branch (the v1 maintenance line). Runs the local-runnable subset of CI gates — vet, lint, govulncheck, unit tests on the active Go toolchain — plus a dry-run of the breaking-change-checker subagent against `release/v1`. Does NOT run the full CI matrix (Go 1.23 + 1.25 + integration). Does NOT tag or push. Reports a go/no-go per gate. **For v2 releases on `master`, this command does not apply** — v2 ships via a separate process.
allowed-tools: Bash(make vet) Bash(make lint) Bash(make vuln) Bash(make test-unit) Bash(go version) Bash(go env:*) Read Grep
---

Verify the current branch is ready to tag as the next v1.1.x release. **Pre-flight:** confirm the current branch is `release/v1` or descends from it — `git merge-base --is-ancestor release/v1 HEAD`. If not, abort with NO-GO and tell the user this command is for the v1 maintenance line; v2 releases on `master` ship via a different process.

## Gates (each runs even if a previous one failed; aggregate go/no-go at the end)

1. **`make vet`** — `go vet ./...`. Must be clean.
2. **`make lint`** — `golangci-lint run`. Must be clean.
3. **`make vuln`** — `govulncheck ./...`. Must be clean, OR document any accepted advisories in the report.
4. **`make test-unit`** — unit tests with `-race`. Must be green. Note: this runs against the **active Go toolchain only** — the full CI matrix (1.23 + 1.25) requires GitHub Actions.
5. **Breaking-change check** — invoke the `breaking-change-checker` subagent against `release/v1`. Must produce zero **BREAKING** findings.
6. **CHANGELOG sanity** — `grep -nE '^## \[1\.1\.' CHANGELOG.md` should show a stanza for the next version (or the most recent unreleased one). If the stanza is missing or empty, flag NO-GO.

## Report format

```
RELEASE PREP — <branch> @ <short-sha>

  vet              [PASS / FAIL]   <one-line note>
  lint             [PASS / FAIL]   <one-line note>
  vuln             [PASS / FAIL]   <one-line note>
  test-unit        [PASS / FAIL]   <test count, duration>
  non-breaking     [PASS / FAIL]   <breaking-change-checker verdict>
  changelog        [PASS / FAIL]   <found stanza for v1.1.x>

VERDICT: [GO | NO-GO]
NOTES:
  - <anything the user should see before tagging>
  - REMINDER: full CI matrix (Go 1.23 + 1.25, GeoServer 2.27 + 2.28 integration) only runs after the tag is pushed; consider opening a PR first.
```

**Do not** run `git tag`, `git push`, or `gh release create`. The user tags manually after seeing the GO verdict.
