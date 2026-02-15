# Feedmix Code Explanation

## Table of Contents
1. [Architecture Overview](#architecture-overview)
2. [Entry Point](#entry-point)
3. [OAuth Flow](#oauth-flow)
4. [YouTube API Client](#youtube-api-client)
5. [Feed Aggregation](#feed-aggregation)
6. [Terminal Display](#terminal-display)
7. [Key Flows](#key-flows)
8. [Testing Strategy](#testing-strategy)
9. [CI/CD Pipeline](#cicd-pipeline)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                   User Types                        │
│              feedmix auth / feed                    │
└──────────────────┬──────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────┐
│           cmd/feedmix/main.go                       │
│  • Cobra CLI (auth, feed, config commands)          │
│  • Version resolution (ldflags or build info)       │
│  • Environment variable loading (.env)              │
└──────────┬──────────┬───────────┬───────────────────┘
           │          │           │
           ▼          ▼           ▼
    ┌──────────┐ ┌─────────┐ ┌──────────┐
    │  OAuth   │ │ YouTube │ │Aggregator│
    │pkg/oauth/│ │internal/│ │internal/ │
    └──────────┘ │youtube/ │ │aggregat..│
                 └─────────┘ └──────────┘
                      │            │
                      ▼            ▼
                 ┌─────────────────────┐
                 │   Display           │
                 │   internal/display/ │
                 └─────────────────────┘
                           │
                           ▼
                    Terminal Output
```

**Design Principles:**
- **`cmd/`** - Entry points (binaries)
- **`internal/`** - Private business logic (can't be imported by other projects)
- **`pkg/`** - Public libraries (reusable by other projects)

---

## Entry Point: `cmd/feedmix/main.go`

### Version Resolution
```go
var version = "dev"  // Default fallback

func init() {
    buildInfo, _ := debug.ReadBuildInfo()
    version = resolveVersion(version, buildInfo)
}
```

**How it works:**
1. **Build-time injection**: `go build -ldflags="-X main.version=v1.0.0"`
2. **Build info**: `go install` automatically embeds version from git tag
3. **Fallback**: "dev" if neither is available

**Why this matters:**
- Users see correct version: `feedmix --version`
- Works with both local builds AND `go install`

### CLI Structure (Cobra)
```go
func newRootCmd() *cobra.Command {
    rootCmd := &cobra.Command{
        Use:   "feedmix",
        Short: "Aggregate feeds from YouTube",
    }

    rootCmd.AddCommand(newAuthCmd())    // feedmix auth
    rootCmd.AddCommand(newFeedCmd())    // feedmix feed
    rootCmd.AddCommand(newConfigCmd())  // feedmix config

    return rootCmd
}
```

**Three commands:**
1. **`feedmix auth`** - Initiates OAuth flow, saves tokens
2. **`feedmix feed`** - Fetches subscriptions, displays feed
3. **`feedmix config`** - Shows where tokens are stored

---

## OAuth Flow: `pkg/oauth/`

### The OAuth 2.0 Dance

**Step 1: Generate Authorization URL**
```go
func (f *Flow) GenerateAuthURL() (authURL string, state string) {
    stateBytes := make([]byte, 16)
    rand.Read(stateBytes)
    state = hex.EncodeToString(stateBytes)  // CSRF protection

    params := url.Values{}
    params.Set("client_id", f.config.ClientID)
    params.Set("redirect_uri", "http://localhost:8080/callback")
    params.Set("scope", "https://www.googleapis.com/auth/youtube.readonly")
    params.Set("response_type", "code")
    params.Set("state", state)  // MUST match on callback

    return authURL, state
}
```

**Why state parameter?**
- CSRF protection
- Attacker can't inject malicious callback
- We verify state matches on return

**Step 2: Start Callback Server**
```go
type CallbackServer struct {
    port       int
    server     *http.Server
    codeChan   chan string
    errorChan  chan error
}

func (s *CallbackServer) WaitForCallback(ctx context.Context, expectedState string, timeout time.Duration) (code string, err error) {
    // Start HTTP server on localhost:8080
    // Wait for GET /callback?code=...&state=...
    // Verify state matches
    // Return authorization code
}
```

**Step 3: Exchange Code for Tokens**
```go
func (f *Flow) ExchangeCode(ctx context.Context, code string) (*Token, error) {
    data := url.Values{}
    data.Set("client_id", f.config.ClientID)
    data.Set("client_secret", f.config.ClientSecret)
    data.Set("code", code)
    data.Set("grant_type", "authorization_code")
    data.Set("redirect_uri", f.config.RedirectURL)

    // POST to https://oauth2.googleapis.com/token
    // Returns: access_token, refresh_token, expires_in
}
```

**Step 4: Save Tokens**
```go
type TokenStorage struct {
    configDir string  // ~/.config/feedmix/
}

func (s *TokenStorage) Save(service string, token *Token) error {
    path := filepath.Join(s.configDir, service+"_token.json")

    // Create directory with 0700 (user only)
    os.MkdirAll(s.configDir, 0700)

    // Save token with 0600 (user read/write only)
    json.Marshal(token)
    os.WriteFile(path, data, 0600)
}
```

**Security:**
- File permissions: 0600 (only you can read)
- Directory permissions: 0700 (only you can access)
- No tokens in logs or error messages

### OAuth Flow Diagram
```
┌──────┐                         ┌────────┐                    ┌────────┐
│ User │                         │Feedmix │                    │ Google │
└──┬───┘                         └───┬────┘                    └───┬────┘
   │                                 │                             │
   │ feedmix auth                    │                             │
   ├────────────────────────────────>│                             │
   │                                 │ 1. Generate auth URL        │
   │                                 │    with state=random        │
   │                                 │ 2. Open browser             │
   │                                 ├────────────────────────────>│
   │                                 │                             │
   │                                 │ 3. User approves            │
   │<─────────────────────────────────────────────────────────────┤
   │                                 │                             │
   │ 4. Redirect to localhost:8080/callback?code=...&state=...   │
   ├────────────────────────────────>│                             │
   │                                 │ 5. Verify state             │
   │                                 │ 6. Exchange code for tokens │
   │                                 ├────────────────────────────>│
   │                                 │<────────────────────────────┤
   │                                 │ {access_token, refresh_token}
   │                                 │ 7. Save to ~/.config/feedmix/
   │<────────────────────────────────┤    youtube_token.json (0600)
   │ "Successfully authenticated!"   │                             │
```

---

## YouTube API Client: `internal/youtube/`

### Client Structure
```go
type Client struct {
    token      *oauth.Token
    httpClient oauth.HTTPClient
    baseURL    string  // https://www.googleapis.com/youtube/v3
}

func NewClient(token *oauth.Token, opts ...ClientOption) *Client {
    c := &Client{
        token:      token,
        httpClient: http.DefaultClient,
        baseURL:    "https://www.googleapis.com/youtube/v3",
    }
    return c
}
```

**Why interfaces?**
```go
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}
```
- **Testability**: Mock HTTP in tests
- **Flexibility**: Can swap implementations

### Fetching Subscriptions
```go
func (c *Client) FetchSubscriptions(ctx context.Context) ([]Subscription, error) {
    var allSubs []Subscription
    pageToken := ""

    for {
        // Build URL with pagination
        url := fmt.Sprintf("%s/subscriptions?part=snippet&mine=true&maxResults=50&pageToken=%s",
            c.baseURL, pageToken)

        req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
        req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)

        resp, err := c.httpClient.Do(req)
        // ... parse JSON response

        allSubs = append(allSubs, response.Items...)

        // Check if more pages
        if response.NextPageToken == "" {
            break
        }
        pageToken = response.NextPageToken
    }

    return allSubs, nil
}
```

**Key points:**
- **Pagination**: YouTube returns max 50 items per request
- **Authorization header**: `Bearer {access_token}`
- **Context**: Respects timeouts and cancellation

### Fetching Recent Videos
```go
func (c *Client) FetchRecentVideos(ctx context.Context, channelID string, maxResults int) ([]Video, error) {
    // 1. Get channel's "uploads" playlist ID
    channel := c.fetchChannelDetails(ctx, channelID)
    uploadsPlaylistID := channel.ContentDetails.RelatedPlaylists.Uploads

    // 2. Fetch videos from uploads playlist
    url := fmt.Sprintf("%s/playlistItems?part=snippet,contentDetails&playlistId=%s&maxResults=%d",
        c.baseURL, uploadsPlaylistID, maxResults)

    // 3. Parse and return videos
    return videos, nil
}
```

**Why this approach?**
- YouTube API doesn't have "recent videos by channel" endpoint
- Must go through uploads playlist
- More API calls, but gets us the data we need

---

## Feed Aggregation: `internal/aggregator/`

### Data Model
```go
type FeedItem struct {
    ID          string
    Source      Source      // YouTube, Reddit, etc.
    Type        ItemType    // Video, Post, etc.
    Title       string
    Description string
    Author      string
    AuthorID    string
    URL         string
    Thumbnail   string
    PublishedAt time.Time
    Engagement  Engagement
}

type Engagement struct {
    Views int64
    Likes int64
}
```

**Design:** Generic enough for multiple sources (YouTube, Reddit, etc.)

### Aggregator Logic
```go
type Aggregator struct {
    items []FeedItem
}

func (a *Aggregator) AddItems(items []FeedItem) {
    a.items = append(a.items, items...)
}

func (a *Aggregator) GetFeed(opts FeedOptions) []FeedItem {
    // 1. Sort by publish date (newest first)
    sort.Slice(a.items, func(i, j int) bool {
        return a.items[i].PublishedAt.After(a.items[j].PublishedAt)
    })

    // 2. Apply limit
    if opts.Limit > 0 && len(a.items) > opts.Limit {
        return a.items[:opts.Limit]
    }

    return a.items
}
```

**Simple but effective:**
- Collect all items from all sources
- Sort by date (newest first)
- Apply limit
- Ready to extend with filters, ranking, etc.

---

## Terminal Display: `internal/display/`

### Terminal Formatter
```go
type TerminalFormatter struct{}

func (f *TerminalFormatter) FormatFeed(items []aggregator.FeedItem) string {
    var buf strings.Builder

    for i, item := range items {
        // Format each item
        fmt.Fprintf(&buf, "%d. %s\n", i+1, item.Title)
        fmt.Fprintf(&buf, "   Author: %s\n", item.Author)
        fmt.Fprintf(&buf, "   URL: %s\n", item.URL)
        fmt.Fprintf(&buf, "   Published: %s\n", item.PublishedAt.Format("2006-01-02 15:04"))

        if item.Engagement.Views > 0 {
            fmt.Fprintf(&buf, "   Views: %s\n", formatNumber(item.Engagement.Views))
        }

        fmt.Fprintf(&buf, "\n")
    }

    return buf.String()
}
```

**Simple text output:**
- Numbered list
- Essential metadata
- URLs for opening in browser

---

## Key Flows

### Flow 1: Authentication (`feedmix auth`)

```
1. User runs: feedmix auth
2. Load credentials from .env or environment
3. Create OAuth config (client ID, secret, scopes)
4. Generate authorization URL with random state
5. Open browser to Google consent screen
6. Start local HTTP server on localhost:8080
7. User approves in browser
8. Google redirects to http://localhost:8080/callback?code=AUTH_CODE&state=STATE
9. Verify state matches (CSRF protection)
10. Exchange authorization code for tokens (access + refresh)
11. Save tokens to ~/.config/feedmix/youtube_token.json (0600 permissions)
12. Print "Successfully authenticated!"
```

### Flow 2: Viewing Feed (`feedmix feed`)

```
1. User runs: feedmix feed --limit 20
2. Load saved token from ~/.config/feedmix/youtube_token.json
3. Create YouTube client with token
4. Fetch user's subscriptions from YouTube API
   - Paginate through all results (50 per page)
5. For each subscription:
   - Fetch recent videos (5 per channel)
   - Convert to FeedItem format
   - Add to aggregator
6. Sort all items by publish date (newest first)
7. Apply limit (20 items)
8. Format as terminal output
9. Print to stdout
```

---

## Testing Strategy

### Unit Tests (Sociable, Mock Externals Only)

**Example: OAuth Flow**
```go
func TestFlow_ExchangeCode(t *testing.T) {
    // Mock HTTP client (external dependency)
    mockHTTP := &MockHTTPClient{
        Response: &http.Response{
            StatusCode: 200,
            Body: io.NopCloser(strings.NewReader(`{
                "access_token": "ya29.abc",
                "refresh_token": "1//xyz",
                "expires_in": 3600
            }`)),
        },
    }

    // Real OAuth flow (our code)
    config := YouTubeOAuthConfig("client_id", "client_secret", "http://localhost")
    flow := NewFlow(config, WithHTTPClient(mockHTTP))

    // Test real flow with mocked HTTP
    token, err := flow.ExchangeCode(context.Background(), "auth_code_123")

    assert.NoError(t, err)
    assert.Equal(t, "ya29.abc", token.AccessToken)
}
```

**What we mock:**
- ✅ External HTTP calls (YouTube API, OAuth endpoints)
- ✅ Filesystem (in some tests)
- ✅ Browser launcher

**What we DON'T mock:**
- ❌ Our own code (oauth.Flow, youtube.Client, aggregator.Aggregator)
- ❌ Standard library (json, url, time)

### Integration Tests (Real Systems)

**Example: Real OAuth Flow**
```go
// +build integration

func TestOAuthFlow_RealBrowser(t *testing.T) {
    if os.Getenv("FEEDMIX_YOUTUBE_CLIENT_ID") == "" {
        t.Skip("Skipping integration test: missing credentials")
    }

    // Real OAuth flow with real browser
    config := YouTubeOAuthConfig(
        os.Getenv("FEEDMIX_YOUTUBE_CLIENT_ID"),
        os.Getenv("FEEDMIX_YOUTUBE_CLIENT_SECRET"),
        "http://localhost:8080/callback",
    )

    flow := NewFlow(config)
    authURL, state := flow.GenerateAuthURL()

    // Opens REAL browser
    browser.Open(authURL)

    // Waits for REAL callback
    server := NewCallbackServer(8080)
    code, err := server.WaitForCallback(context.Background(), state, 5*time.Minute)

    assert.NoError(t, err)
    assert.NotEmpty(t, code)
}
```

**Run with:** `go test -tags=integration ./...`

### Contract Tests (API Assumptions)

**Example: YouTube API Contract**
```go
func TestYouTubeAPI_SubscriptionsEndpoint_ReturnsExpectedFormat(t *testing.T) {
    // This test verifies our assumptions about YouTube API
    // If this fails, YouTube changed their API!

    client := setupRealYouTubeClient(t)

    resp, err := client.FetchSubscriptions(context.Background())
    require.NoError(t, err)

    // Verify contract
    assert.NotEmpty(t, resp)

    item := resp[0]
    assert.NotEmpty(t, item.ChannelID)
    assert.NotEmpty(t, item.ChannelTitle)
    assert.NotEmpty(t, item.Thumbnail)
}
```

**Purpose:** Catch breaking changes in external APIs early

---

## CI/CD Pipeline

### Local CI (`./scripts/ci.sh`)

**Fast feedback during development:**
```bash
#!/bin/bash
# 1. Go vet (static analysis)
go vet ./...

# 2. Unit + contract tests with race detector
go test -race -cover ./...

# 3. Integration tests
go test -tags=integration ./pkg/oauth/... ./cmd/feedmix/...

# 4. Security (govulncheck - optional locally)
govulncheck ./... || echo "Skipped"

# 5. Build
go build -o feedmix ./cmd/feedmix

# 6. Verify binary works
./feedmix --version
```

**Runtime:** ~2 minutes (fast!)

### Deployment (`./scripts/deploy.sh`)

**Full validation + automatic push:**
```bash
#!/bin/bash
# 1. Check for uncommitted changes (fail if found)
git status --porcelain

# 2. Run full CI
./scripts/ci.sh

# 3. Build release binaries (5 platforms)
for platform in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
    GOOS=$GOOS GOARCH=$GOARCH go build -o dist/feedmix-$VERSION-$GOOS-$GOARCH
done

# 4. E2E smoke tests on built binaries
./tests/e2e-smoke.sh

# 5. Auto-push to GitHub (no manual gate!)
git push origin main
```

**Runtime:** ~5 minutes

### GitHub Actions CI

**Triggered on every push to main:**

1. **Test** - Run all unit tests with race detector
2. **Contract Tests** - Verify API assumptions
3. **Integration Tests** - Real OAuth flow (skipped if no creds)
4. **Lint** - golangci-lint
5. **Security** - gitleaks (PII), gosec (code), govulncheck (vulnerabilities)
6. **Build** - Compile and verify binary
7. **Semantic Release** (if feat/fix commit):
   - Determine version bump (feat=minor, fix=patch)
   - Create git tag
   - Build binaries for all platforms
   - Create GitHub Release with binaries

### Semantic Release (Automatic)

**Conventional commits trigger releases:**
```bash
feat: add feature    → v0.4.0 (minor bump)
fix: bug fix         → v0.4.1 (patch bump)
docs: update README  → (no release)
```

**What happens automatically:**
1. Parse commit message
2. Calculate next version
3. Create git tag (v0.4.1)
4. Build binaries (linux, macOS, Windows)
5. Create GitHub Release
6. Upload binaries as release assets
7. Users get it via: `go install github.com/gauthierbraillon/feedmix/cmd/feedmix@latest`

---

## Security Features

### 1. URL Validation (Browser Launcher)
```go
func Open(urlString string) error {
    // Parse and validate URL
    parsedURL, err := url.Parse(urlString)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }

    // Whitelist allowed schemes
    if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
        return fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
    }

    // Now safe to pass to shell
    exec.Command("xdg-open", urlString)  // #nosec G204 - URL validated above
}
```

**Prevents:** Command injection attacks

### 2. PII Scanning (gitleaks)

**Automatically scans for:**
- Email addresses
- API keys, tokens, secrets
- Private keys
- Home directory paths with usernames
- IP addresses

**Runs on:** Every commit in GitHub Actions

### 3. Code Security (gosec)

**Checks for:**
- SQL injection vulnerabilities
- Path traversal
- Unsafe use of crypto
- Command injection
- Exposed secrets

### 4. Vulnerability Scanning (govulncheck)

**Checks for:**
- Known CVEs in dependencies
- Vulnerable Go standard library versions

---

## Summary

**What makes this codebase good:**

1. ✅ **Clear separation of concerns**
   - CLI (cmd) separate from business logic (internal)
   - Reusable packages (pkg) for OAuth, browser

2. ✅ **Testable architecture**
   - Interfaces for external dependencies
   - Mock externals, use real collaborators
   - Fast unit tests, slower integration tests

3. ✅ **Security-first**
   - Input validation (URLs, state parameters)
   - Secure token storage (0600 permissions)
   - Automated security scanning

4. ✅ **Modern Go practices**
   - Context for cancellation
   - Functional options pattern
   - Cobra for CLI

5. ✅ **Continuous Deployment**
   - Semantic versioning from commits
   - Automated releases
   - E2E smoke tests with rollback

**This is production-ready code that showcases modern Go development practices!**
