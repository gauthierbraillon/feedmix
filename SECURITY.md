# Security Policy

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Open a [GitHub Security Advisory](https://github.com/gauthierbraillon/feedmix/security/advisories/new) (private disclosure).

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (optional)

You will receive a response within 72 hours.

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest  | Yes       |
| Older   | No        |

Always install the latest version: `go install github.com/gauthierbraillon/feedmix/cmd/feedmix@latest`

## Security Design

### OAuth Token Storage

- Tokens stored at `~/.config/feedmix/youtube_token.json`
- File permissions: `0600` (owner read/write only)
- Tokens never logged or printed to stdout

### OAuth Flow

- Local callback server binds only to `localhost:8080`
- Authorization code exchanged immediately and discarded
- Redirect URI validated against expected value

### No Telemetry

Feedmix makes no network requests other than:
1. Google OAuth endpoints (during `feedmix auth`)
2. YouTube Data API v3 (during `feedmix feed`)

No analytics, no crash reporting, no usage tracking.

### Dependency Scanning

Every CI run executes:
- `govulncheck` — Go vulnerability database checks
- `gosec` — static security analysis
- `gitleaks` — secret scanning across git history
- `go test -race` — data race detection

### Supply Chain

All GitHub Actions are pinned to immutable commit SHAs. A Dependabot configuration keeps them updated weekly via automated PRs.
