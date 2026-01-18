# Feedmix

Aggregate feeds from YouTube and LinkedIn into a unified CLI view.

## Install

```bash
# From source
go install github.com/gauthierbraillon/feedmix/cmd/feedmix@latest

# Or download binary from releases
curl -L https://github.com/gauthierbraillon/feedmix/releases/latest/download/feedmix-linux-amd64 -o feedmix
chmod +x feedmix
```

## Setup

1. **YouTube**: Create OAuth credentials at [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. **LinkedIn**: Create app at [LinkedIn Developers](https://developer.linkedin.com/)

```bash
export FEEDMIX_YOUTUBE_CLIENT_ID="your-client-id"
export FEEDMIX_YOUTUBE_CLIENT_SECRET="your-client-secret"
export FEEDMIX_LINKEDIN_CLIENT_ID="your-client-id"
export FEEDMIX_LINKEDIN_CLIENT_SECRET="your-client-secret"
```

## Usage

```bash
# Authenticate
feedmix auth youtube
feedmix auth linkedin

# View feed
feedmix feed
feedmix feed --source youtube --limit 10

# Check config
feedmix config
```

## Development

```bash
# Run tests
go test ./...

# Run full CI (tests, lint, security)
./scripts/ci.sh

# Build release binaries
./scripts/deploy.sh
```

## Release

```bash
git tag v1.0.0
git push origin v1.0.0
# GitHub Actions builds and publishes release
```

## CI Pipeline

| Check | Tool |
|-------|------|
| Tests | `go test -race` |
| Coverage | codecov |
| Lint | golangci-lint |
| Security | govulncheck, gosec |
| Contracts | API schema validation |

## License

MIT
