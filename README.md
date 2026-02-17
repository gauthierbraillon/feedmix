# Feedmix

[![CI](https://github.com/gauthierbraillon/feedmix/actions/workflows/ci.yml/badge.svg)](https://github.com/gauthierbraillon/feedmix/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/gauthierbraillon/feedmix)](https://github.com/gauthierbraillon/feedmix/releases/latest)
[![Go version](https://img.shields.io/badge/go-1.22+-blue)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Aggregate your YouTube subscriptions into a lightweight terminal feed — a privacy-focused RSS alternative for developers who prefer the command line.

## Why Feedmix?

YouTube's subscription feed buries new videos under recommendations and ads. RSS readers require third-party bridges that may break or log your activity. Feedmix cuts through both:

- **Terminal-native** — runs anywhere, renders in your shell, no browser required
- **Lightweight** — single static binary, zero runtime dependencies
- **Privacy-focused** — OAuth tokens stored locally at `~/.config/feedmix/` (0600), nothing sent to third parties
- **No third-party account** — uses your own Google OAuth credentials; your data stays yours

## Install

```bash
go install github.com/gauthierbraillon/feedmix/cmd/feedmix@latest
```

Or download a pre-built binary from [GitHub Releases](https://github.com/gauthierbraillon/feedmix/releases/latest) (Linux, macOS, Windows — amd64 & arm64).

## Setup

Create OAuth credentials at [Google Cloud Console](https://console.cloud.google.com/apis/credentials):
1. Create OAuth 2.0 Client ID (Desktop app)
2. Add `http://localhost:8080/callback` as authorized redirect URI

**Option 1: Using .env file (recommended)**
```bash
cp .env.example .env
# Edit .env and add your credentials
```

**Option 2: Using environment variables**
```bash
export FEEDMIX_YOUTUBE_CLIENT_ID="your-client-id"
export FEEDMIX_YOUTUBE_CLIENT_SECRET="your-client-secret"
```

## Usage

```bash
feedmix auth           # Authenticate with YouTube (opens browser)
feedmix feed           # View recent videos from your subscriptions
feedmix feed --limit 5 # Limit to 5 results
feedmix config         # Show config directory
```

Example output:

```
[Tech Channel]  New video title here                     2h ago
                https://www.youtube.com/watch?v=dQw4w9WgXcQ

[Dev Talks]     Another great talk on Go performance      1d ago
                https://www.youtube.com/watch?v=abc123xyz
```

## Development

### Quick Iteration (during development)
```bash
./scripts/ci.sh      # Fast feedback: tests, lint, security (<2 min)
```

### Deploy (when ready to ship)
```bash
git add -A && git commit -m "feat: add feature"
./scripts/deploy.sh  # Full validation + auto-push (<5 min)
```

## Releases

**Automatic semantic releases** — no manual tags needed!

Releases are created automatically based on conventional commit messages:

| Commit Type | Version Bump | Example |
|-------------|--------------|---------|
| `feat:` | Minor (v1.1.0) | `feat: add search feature` |
| `fix:` | Patch (v1.0.1) | `fix: handle expired tokens` |
| `feat!:` or `BREAKING CHANGE:` | Major (v2.0.0) | `feat!: change API` |
| `docs:`, `chore:`, `refactor:` | No release | (CI only) |

**Workflow:**
1. Commit with conventional message: `git commit -m "feat: add feature"`
2. Push: `git push` (or run `./scripts/deploy.sh`)
3. CI passes → **Release created automatically**
4. Users get update: `go install github.com/gauthierbraillon/feedmix/cmd/feedmix@latest`

**Manual release (if needed):**
```bash
git tag v1.0.0 && git push origin v1.0.0
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). All contributions start with a failing test.

## License

MIT — see [LICENSE](LICENSE).
