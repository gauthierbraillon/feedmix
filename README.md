# Feedmix

[![CI](https://github.com/gauthierbraillon/feedmix/actions/workflows/ci.yml/badge.svg)](https://github.com/gauthierbraillon/feedmix/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/gauthierbraillon/feedmix)](https://github.com/gauthierbraillon/feedmix/releases/latest)
[![Go version](https://img.shields.io/badge/go-1.24+-blue)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Aggregate your YouTube subscriptions and Substack newsletters into a single lightweight terminal feed — a privacy-focused RSS alternative for developers who prefer the command line.

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

Copy the example config and fill in your credentials:

```bash
cp .env.example .env
```

Then follow the sections below for each source you want to enable.

---

### YouTube setup

**Step 1 — Create OAuth credentials**

1. Go to [Google Cloud Console → APIs & Services → Credentials](https://console.cloud.google.com/apis/credentials)
2. Click **Create credentials → OAuth client ID**
3. Choose **Desktop app**, give it any name, click **Create**
4. Copy your **Client ID** and **Client Secret** into `.env`:
   ```
   FEEDMIX_YOUTUBE_CLIENT_ID=...
   FEEDMIX_YOUTUBE_CLIENT_SECRET=...
   ```

**Step 2 — Enable the YouTube Data API**

1. Go to [APIs & Services → Library](https://console.cloud.google.com/apis/library)
2. Search for **YouTube Data API v3** and click **Enable**

**Step 3 — Get a refresh token**

1. Open [OAuth 2.0 Playground](https://developers.google.com/oauthplayground)
2. Click the gear icon (⚙) in the top-right → check **Use your own OAuth credentials**
3. Enter your Client ID and Client Secret
4. In the scope list on the left, find **YouTube Data API v3** and select `https://www.googleapis.com/auth/youtube.readonly`, then click **Authorize APIs**
5. Sign in with the Google account whose subscriptions you want to follow
6. Click **Exchange authorization code for tokens**
7. Copy the **Refresh token** value into `.env`:
   ```
   FEEDMIX_YOUTUBE_REFRESH_TOKEN=...
   ```

---

### Substack setup

No API key needed — Substack publishes a public RSS feed for every publication.

Find the base URL of each newsletter you follow (e.g. `https://simonwillison.substack.com`) and add them as a comma-separated list in `.env`:

```
FEEDMIX_SUBSTACK_URLS=https://simonwillison.substack.com,https://stratechery.com
```

Substack is optional — omitting `FEEDMIX_SUBSTACK_URLS` shows only YouTube items.

---

## Usage

```bash
feedmix feed             # Unified feed from all configured sources
feedmix feed --limit 10  # Show at most 10 items
```

Example output:

```
[Simon Willison]  Everything I built with Claude Sonnet     3h ago
                  https://simonwillison.substack.com/p/...

[Tech Channel]    New video title here                      5h ago
                  https://www.youtube.com/watch?v=dQw4w9WgXcQ

[Dev Talks]       Another great talk on Go performance      1d ago
                  https://www.youtube.com/watch?v=abc123xyz
```

## Development

### Quick iteration (during development)
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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). All contributions start with a failing test.

## License

MIT — see [LICENSE](LICENSE).
