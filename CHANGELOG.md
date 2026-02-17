# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security
- Pin all GitHub Actions to immutable commit SHAs to prevent supply chain attacks
- Add `llms.txt`, `ARCHITECTURE.md`, `SECURITY.md`, `CONTRIBUTING.md` for project transparency

## [v0.5.0] - 2026-02-15

### Added
- Automated PII and secrets scanning with gitleaks in CI pipeline

## [v0.4.1] - 2026-02-15

### Fixed
- Semantic release now builds and publishes binaries directly from the release workflow

## [v0.4.0] - 2026-02-15

### Added
- Automatic semantic releases triggered by conventional commit messages (`feat:`, `fix:`, `feat!:`)
- Automatic CD pipeline following Bryan Finster's Minimum CD principles
- ATDD philosophy with E2E smoke tests and rollback on failure

### Fixed
- Critical security vulnerabilities (G204, G117) flagged by gosec
- Add `#nosec` annotations for gosec false positives with explanations

## [v0.3.1] - 2026-01-21

### Fixed
- Version detection for `go install` installations (falls back to binary metadata)

## [v0.3.0] - 2026-01-20

### Added
- Auto-versioning from git tags injected at build time via `ldflags`

## [v0.2.0] - 2026-01-19

### Added
- Display clickable video URLs (e.g. `https://www.youtube.com/watch?v=...`) instead of channel URLs
- Real contract tests against YouTube Data API v3 discovery document
- Integration tests for OAuth flow and CLI binary behaviour

## [v0.1.0] - 2026-01-19

### Added
- OAuth 2.0 browser-based authentication flow with local callback server
- YouTube Data API v3 subscription feed aggregation
- Terminal display with relative timestamps and truncated titles
- `feedmix auth` — authenticate with Google/YouTube
- `feedmix feed` — view recent videos from subscriptions
- `feedmix feed --limit N` — limit number of results
- `feedmix config` — show configuration directory
- Single binary distribution via `go install`
- Multi-platform release binaries (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
- Tokens stored locally in `~/.config/feedmix/` with 0600 permissions

[Unreleased]: https://github.com/gauthierbraillon/feedmix/compare/v0.5.0...HEAD
[v0.5.0]: https://github.com/gauthierbraillon/feedmix/compare/v0.4.1...v0.5.0
[v0.4.1]: https://github.com/gauthierbraillon/feedmix/compare/v0.4.0...v0.4.1
[v0.4.0]: https://github.com/gauthierbraillon/feedmix/compare/v0.3.1...v0.4.0
[v0.3.1]: https://github.com/gauthierbraillon/feedmix/compare/v0.3.0...v0.3.1
[v0.3.0]: https://github.com/gauthierbraillon/feedmix/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/gauthierbraillon/feedmix/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/gauthierbraillon/feedmix/releases/tag/v0.1.0
