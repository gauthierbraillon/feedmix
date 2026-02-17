# Architecture

## Overview

Feedmix is a CLI tool that authenticates with Google via OAuth 2.0, fetches subscription data from the YouTube Data API v3, aggregates it, and renders it in the terminal.

```
User
 │
 ▼
cmd/feedmix          ← CLI entry point (Cobra commands: auth, feed, config)
 │
 ├── pkg/oauth        ← OAuth 2.0 flow (browser launch + local callback server)
 │    └── pkg/browser ← Cross-platform browser launcher
 │
 ├── internal/youtube ← YouTube Data API v3 client (subscriptions, videos, search)
 │
 ├── internal/aggregator ← Combines and sorts feed items
 │
 └── internal/display ← Terminal output (relative timestamps, URL formatting)
```

## Data Flow

### `feedmix auth`

```
main → oauth.NewFlow() → launch browser → Google OAuth consent
                       → local HTTP server waits on :8080/callback
                       → exchange code for tokens
                       → save tokens to ~/.config/feedmix/youtube_token.json (0600)
```

### `feedmix feed`

```
main → load tokens from disk
     → youtube.NewClient(token)
     → client.FetchSubscriptions()  → YouTube API /subscriptions
     → for each channel:
         client.FetchRecentVideos() → YouTube API /search
     → aggregator.AddItems()
     → aggregator.GetFeed()         → sort by date, apply --limit
     → display.FormatFeed()         → print to stdout
```

## Package Responsibilities

| Package | Responsibility | Visibility |
|---------|---------------|------------|
| `cmd/feedmix` | CLI commands, flag parsing, wiring | binary |
| `pkg/oauth` | OAuth 2.0 token acquisition and storage | public |
| `pkg/browser` | Cross-platform browser launcher | public |
| `internal/youtube` | YouTube Data API v3 client | private |
| `internal/aggregator` | Feed aggregation and sorting | private |
| `internal/display` | Terminal rendering | private |
| `internal/ciconfig` | CI pipeline self-tests | private |
| `pkg/contracts` | YouTube API contract tests | public |

## Key Design Decisions

**Interfaces for external dependencies** — The YouTube HTTP client, OAuth client, and browser launcher are all behind interfaces. Tests inject fakes; production code injects real implementations.

**No background process** — Feedmix is a one-shot CLI tool. It runs, prints the feed, and exits. No daemon, no polling.

**Local token storage** — OAuth tokens are stored on disk at `~/.config/feedmix/` with mode 0600. No cloud storage, no keychain dependency — just a file.

**Single binary** — The entire application compiles to a single static binary with no runtime dependencies. Distributed via `go install` and GitHub Releases.

## Configuration

| Variable | Description |
|----------|-------------|
| `FEEDMIX_YOUTUBE_CLIENT_ID` | Google OAuth client ID |
| `FEEDMIX_YOUTUBE_CLIENT_SECRET` | Google OAuth client secret |
| `FEEDMIX_CONFIG_DIR` | Override token storage directory (default: `~/.config/feedmix`) |
| `FEEDMIX_API_URL` | Override YouTube API base URL (used in tests) |

## Testing Strategy

```
Unit tests (sociable)   → mock only external APIs and filesystem
Contract tests          → verify YouTube API response shapes
Integration tests       → real OAuth flow (-tags=integration)
E2E tests               → compiled binary black-box tests
```

All unit and contract tests run in under 2 seconds. Integration tests require network access and real credentials.
