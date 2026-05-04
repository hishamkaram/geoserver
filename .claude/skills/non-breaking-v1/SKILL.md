---
description: Verifies that a change preserves the v1.1.x non-breaking contract. Run before opening a PR to `release/v1` or before tagging a v1.1.x release. Walks through the exported-API surface, *Context twin pattern, deprecation discipline, and concurrency constraints. **For v2 PRs targeting `master`, this skill does not apply** — v2 is allowed to break.
disable-model-invocation: true
allowed-tools: Bash(git diff:*) Bash(git log:*) Bash(go doc:*) Bash(grep:*) Read Grep Glob
---

# Non-breaking v1.1.x checklist

Run this before opening a PR to `release/v1` or before tagging a v1.1.x release. The contract: **a caller pinned to v1.0 must be able to upgrade to v1.1.x by changing only their `go.mod` version and running `go mod tidy`. No source edits.**

`master` is v2 and does not honor this contract — do not run this skill against PRs targeting `master`.

This skill is manually-invoked (`/non-breaking-v1`) — Claude won't auto-trigger it because the workflow has timing implications (run it at PR / release time, not mid-edit).

## Steps

### 1. Compute exported-API diff against `release/v1`

```
git diff release/v1...HEAD -- '*.go' ':!*_test.go' | grep -E '^[-+](func|type|var|const)' || echo "no exported diffs"
```

If `gorelease` is available, prefer the authoritative tool:

```
gorelease -base=v1.0.0
```

### 2. Inspect every `-` line in the diff

For each removed or signature-changed exported symbol:

- **REMOVED?** That's BREAKING. Either restore the symbol (with a `// Deprecated:` annotation that delegates to its replacement) or escalate to v2.
- **SIGNATURE CHANGE?** That's BREAKING. Add a sibling with the new signature; keep the old one delegating.
- **STRUCT FIELD REMOVED OR RETYPED?** That's BREAKING. Restore.

### 3. Inspect every `+` exported symbol

For each new exported symbol:

- Is it a method on `*GeoServer`? Then it must come in a *Context twin pair — both `Foo(...)` and `FooContext(ctx, ...)`. Verify both exist:
  `grep -n 'func (g \*GeoServer) Foo[^a-zA-Z]' *.go` and the same for `FooContext`.
- Is it a method? Add it to the relevant `*Service` and `*ServiceWithContext` interface.
- Is it a struct field added to an existing exported struct? OK in principle, but if any caller uses **positional struct literals** (`Foo{a, b, c}` without field names), they break. Audit `_test.go` and known consumers; flag for the CHANGELOG.

### 4. Verify `// Deprecated:` discipline

```
grep -nB2 'Deprecated:' *.go
```

For every `// Deprecated:` marker, confirm:
- The replacement function / method exists.
- The deprecated form delegates to the replacement (doesn't reimplement).
- The CHANGELOG entry exists and explains the migration.

### 5. Concurrency check

`*GeoServer` exported fields must not be mutated post-construction in library code. Search for assignments to exported fields in non-test files:

```
grep -nE '^[^/]*g\.[A-Z][a-zA-Z]+\s*=' *.go | grep -v _test.go
```

If you see something like `g.HttpClient = ...` outside the constructor or option-application path, that's a smell (v1's compromise is: accept reads, document non-mutability — but don't mutate from inside the library).

### 6. CHANGELOG

Confirm `CHANGELOG.md` has a `## [1.1.x] — YYYY-MM-DD` section listing all additions, deprecations, and any non-breaking-but-noteworthy behavior changes (e.g., `ParseURL` now path-escapes per segment — documented as a silent behavior change for malformed inputs).

## Output

Report in this shape:

```
NON-BREAKING CHECK — <branch> vs master

BREAKING (n):
  <empty list ⇒ ✅ OK; non-empty ⇒ ❌ block release>

REVIEW (m):
  - <symbol> — <why this needs a human eyeball>

GOOD (k):
  - <new exports + correctly-paired *Context twins + correctly-marked deprecations>

VERDICT: [GO | NO-GO]
```
