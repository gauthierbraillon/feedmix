# Debugging Patterns

## Version Detection

### Issue: Version Shows "dev" After `go install`

**Problem**: When users install via `go install github.com/gauthierbraillon/feedmix/cmd/feedmix@latest`, the version shows "dev" instead of the actual version.

**Root Cause**: The `-ldflags "-X main.version=$VERSION"` flag is only applied during local builds, not when Go downloads and builds from source.

**Solution**: Auto-version detection from git tags (implemented in commit 7ad084c)
```go
// cmd/feedmix/main.go
var version = "dev"

func init() {
    if version == "dev" {
        // Try to detect version from git tag
        if v := detectVersion(); v != "" {
            version = v
        }
    }
}
```

**Lesson**: Don't rely on build-time flags for distributed binaries. Implement runtime version detection.

## OAuth Flow

### Issue: Browser Doesn't Open on Headless Systems

**Problem**: `feedmix auth` fails on servers without a display (SSH sessions, Docker containers, CI environments).

**Root Cause**: Browser launcher requires a display (`$DISPLAY` environment variable).

**Solution**: Provide manual URL fallback
```go
// pkg/browser/browser.go
func Open(url string) error {
    if err := openBrowser(url); err != nil {
        fmt.Printf("Could not open browser automatically.\n")
        fmt.Printf("Please visit this URL manually:\n%s\n", url)
        return nil
    }
    return nil
}
```

**Lesson**: Always provide manual fallbacks for automated workflows.

### Issue: Race Condition in Token Storage

**Problem**: Concurrent `feedmix feed` commands corrupt `tokens.json`.

**Root Cause**: Multiple processes writing to the same file simultaneously.

**Solution**: File locking (to be implemented)
```go
// pkg/oauth/storage.go
import "syscall"

func SaveTokens(tokens *Tokens) error {
    file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
    if err != nil {
        return err
    }
    defer file.Close()

    // Acquire exclusive lock
    if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
        return err
    }
    defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

    // Write tokens...
}
```

**Status**: Not yet implemented (low priority, rare edge case)

## YouTube API

### Issue: API Quota Exceeded

**Problem**: `feedmix feed` fails with "quota exceeded" error after ~100 calls.

**Root Cause**: YouTube API has daily quota limit (10,000 units/day). Each subscription fetch costs ~100 units.

**Solution**: Rate limiting and caching (to be implemented)
- Track quota usage in local cache
- Warn user when approaching limit (80% threshold)
- Cache subscription data for 1 hour

**Status**: Not yet implemented (monitoring needed first)

## Testing

### Issue: Integration Tests Fail in CI

**Problem**: `go test -tags=integration ./...` fails in GitHub Actions with "missing credentials".

**Root Cause**: Integration tests require real OAuth credentials (client ID/secret), which aren't available in CI.

**Solution**: Skip integration tests when credentials aren't available
```go
// pkg/oauth/oauth_integration_test.go
func TestOAuthFlow_Integration(t *testing.T) {
    if os.Getenv("FEEDMIX_YOUTUBE_CLIENT_ID") == "" {
        t.Skip("Skipping integration test: missing credentials")
    }
    // Test real OAuth flow...
}
```

**Lesson**: Integration tests should gracefully skip when dependencies aren't available.

## Build Issues

### Issue: Cross-Compilation Fails on macOS ARM64

**Problem**: Building macOS ARM64 binary on Linux fails with "invalid flag in pkg-config --libs".

**Root Cause**: CGO is enabled by default, causing cross-compilation issues.

**Solution**: Disable CGO for pure Go builds
```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o feedmix-darwin-arm64 ./cmd/feedmix
```

**Status**: Implemented in `scripts/deploy.sh` (implicit, Go uses CGO_ENABLED=0 for cross-compilation)

## Common Go Gotchas

### Defer in Loops

**Problem**: Deferring inside a loop can cause resource leaks (defers run at function end, not loop iteration end).

**Bad**:
```go
for _, file := range files {
    f, _ := os.Open(file)
    defer f.Close()  // Defers accumulate, files stay open!
}
```

**Good**:
```go
for _, file := range files {
    func() {
        f, _ := os.Open(file)
        defer f.Close()  // Runs at end of anonymous function
        // Process file...
    }()
}
```

### Nil Interface Values

**Problem**: An interface containing a nil pointer is not nil.

```go
var client *YouTubeClient = nil
var iface YouTubeClientInterface = client
if iface == nil {  // FALSE! Interface is not nil, it contains a typed nil
    // This won't run
}
```

**Solution**: Check both the interface and its value:
```go
if iface == nil || reflect.ValueOf(iface).IsNil() {
    // Correctly detects nil
}
```
