# Feedmix

Aggregate your YouTube subscriptions into a CLI feed.

## Install

```bash
go install github.com/gauthierbraillon/feedmix/cmd/feedmix@latest
```

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
feedmix auth           # Authenticate with YouTube
feedmix feed           # View subscriptions
feedmix feed --limit 5 # Limit results
feedmix config         # Show config directory
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

**Automatic semantic releases** - no manual tags needed!

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
3. CI passes → **Release created automatically** 🚀
4. Users get update: `go install github.com/gauthierbraillon/feedmix/cmd/feedmix@latest`

**Manual release (if needed):**
```bash
git tag v1.0.0 && git push origin v1.0.0
```
