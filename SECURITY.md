# Security Policy

## Supported versions

| Version | Supported |
|---|---|
| 2.x     | ✅ Active development and security fixes — `github.com/hishamkaram/geoserver/v2` on `master` (latest `v2.0.0`) |
| 1.1.x   | ⚠️ Security fixes only — `github.com/hishamkaram/geoserver` on the `release/v1` branch (latest `v1.1.2`; end-of-feature) |
| ≤ 1.0   | ❌ No support |

## Reporting a vulnerability

**Please do not open public GitHub issues for security vulnerabilities.**

Use one of these private channels:

1. **GitHub Security Advisories** (preferred) — go to the repository's [Security tab](https://github.com/hishamkaram/geoserver/security) → "Report a vulnerability".
2. **Email** — open a private advisory request via GitHub if you cannot use the Security tab.

Please include:

- A description of the vulnerability and its impact.
- A minimal reproduction (code snippet, configuration, or steps).
- The version of `github.com/hishamkaram/geoserver` and Go where the issue was observed.

## What to expect

- Acknowledgement within 5 business days.
- A triage response with severity assessment within 10 business days.
- A fix, mitigation, or workaround as quickly as severity allows. Critical issues are prioritized.
- A coordinated disclosure: we will agree on a public disclosure timeline with the reporter before publishing.

## Scope

This policy covers the Go client library code in this repository. Vulnerabilities in:

- Upstream GeoServer (the server) — report to the [GeoServer project](https://geoserver.org/) directly.
- Dependencies — please also report upstream so the broader ecosystem benefits.
