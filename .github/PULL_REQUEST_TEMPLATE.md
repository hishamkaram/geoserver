<!-- Thanks for opening a PR! Please complete the checklist below. -->

## What changes does this PR introduce?

<!-- A clear summary of the change and the motivation. -->

## Type of change

- [ ] Bug fix
- [ ] New feature (additive)
- [ ] Breaking change (targets `master` — v2 line)
- [ ] v1 security fix (must target `release/v1`, non-breaking)
- [ ] Documentation only
- [ ] Build / CI / chore

## Checklist

- [ ] My commits use [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `refactor:`, etc.)
- [ ] `make lint` passes locally
- [ ] `make test-unit` passes locally
- [ ] `make test-integration` passes locally (if my change affects the wire protocol)
- [ ] I added or updated unit tests for the change
- [ ] I updated the CHANGELOG entry under `## [Unreleased]` if user-visible
- [ ] I did not add new runtime dependencies (or I justified why in the description)
- [ ] If this PR targets `release/v1`, my change preserves backward compatibility (deprecate via `// Deprecated:` and add a sibling rather than changing signatures); breaking changes belong on `master` (v2)

## Related issues

<!-- e.g. Closes #123, Fixes #456 -->
