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

```bash
./scripts/ci.sh      # Run tests, lint, security
./scripts/deploy.sh  # Build release binaries
```

## Release

```bash
git tag v1.0.0 && git push origin v1.0.0
```

GitHub Actions builds and publishes the release automatically.
