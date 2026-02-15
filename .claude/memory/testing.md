# Testing Patterns

## Test Organization

### File Naming Conventions
- Unit tests: `*_test.go` (same package)
- Integration tests: `*_integration_test.go` (build tag: `// +build integration`)
- Contract tests: `contracts_test.go` (in `pkg/contracts/`)

### Test Naming Conventions
- Pattern: `Test<Function>_<Scenario>_<ExpectedBehavior>`
- Examples:
  - `TestOAuthFlow_ValidCode_SavesTokensToFile`
  - `TestOAuthFlow_InvalidCode_ReturnsError`
  - `TestYouTubeClient_FetchSubscriptions_ReturnsChannelList`
  - `TestYouTubeClient_APIError_PropagatesError`

## Unit Test Patterns

### Sociable Unit Tests (Mock Externals Only)

**DO**: Mock external dependencies
```go
func TestOAuthFlow_SavesTokens(t *testing.T) {
    // Mock external HTTP client
    mockHTTP := &MockHTTPClient{
        Response: &http.Response{
            StatusCode: 200,
            Body: io.NopCloser(strings.NewReader(`{"access_token":"abc"}`)),
        },
    }

    // Mock external filesystem
    mockFS := &MockFileSystem{
        files: make(map[string][]byte),
    }

    // Use real OAuth implementation (our code)
    oauth := NewOAuthClient(mockHTTP, mockFS)

    // Test the flow
    token, err := oauth.ExchangeCode("code123")
    assert.NoError(t, err)
    assert.Equal(t, "abc", token.AccessToken)
}
```

**DON'T**: Mock our own code
```go
// BAD - Don't do this
func TestFeedAggregator(t *testing.T) {
    mockYouTubeClient := &MockYouTubeClient{} // Our own code!
    aggregator := NewAggregator(mockYouTubeClient)
    // Testing what? Just the mock?
}

// GOOD - Use real collaborators
func TestFeedAggregator(t *testing.T) {
    // Mock external HTTP (YouTube API)
    mockHTTP := &MockHTTPClient{...}

    // Use real YouTube client (our code)
    youtubeClient := NewYouTubeClient(mockHTTP)

    // Use real aggregator (our code)
    aggregator := NewAggregator(youtubeClient)

    // Test the full flow through our code
    feed, err := aggregator.Aggregate()
    assert.NoError(t, err)
}
```

### Table-Driven Tests

Use for testing multiple scenarios:
```go
func TestParseTimestamp(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected time.Time
        wantErr  bool
    }{
        {
            name:     "valid ISO8601",
            input:    "2024-01-15T10:30:00Z",
            expected: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
            wantErr:  false,
        },
        {
            name:     "invalid format",
            input:    "not-a-date",
            expected: time.Time{},
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ParseTimestamp(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

## Integration Test Patterns

### Build Tags

Always use build tags for integration tests:
```go
// +build integration

package oauth_test

import "testing"

func TestOAuthFlow_RealBrowser(t *testing.T) {
    // This test opens a real browser
    // Only runs with: go test -tags=integration
}
```

### Skipping When Dependencies Missing

```go
func TestYouTubeAPI_Integration(t *testing.T) {
    clientID := os.Getenv("FEEDMIX_YOUTUBE_CLIENT_ID")
    if clientID == "" {
        t.Skip("Skipping integration test: FEEDMIX_YOUTUBE_CLIENT_ID not set")
    }

    // Test with real API...
}
```

## Contract Test Patterns

### API Contract Verification

Contract tests document assumptions about external APIs:
```go
// pkg/contracts/contracts_test.go
func TestYouTubeAPI_SubscriptionsEndpoint_ReturnsExpectedFormat(t *testing.T) {
    // This test verifies our understanding of the YouTube API
    // If this fails, YouTube changed their API (breaking change)

    client := setupRealYouTubeClient(t)

    resp, err := client.FetchSubscriptions()
    require.NoError(t, err)

    // Verify contract assumptions
    assert.NotNil(t, resp.Items)
    assert.Greater(t, len(resp.Items), 0)

    item := resp.Items[0]
    assert.NotEmpty(t, item.Snippet.Title)
    assert.NotEmpty(t, item.Snippet.ChannelID)
    assert.NotEmpty(t, item.Snippet.Thumbnails.Default.URL)
}
```

### When to Write Contract Tests

Write contract tests when:
- You depend on an external API (YouTube, Google OAuth)
- The API could change without notice
- You want to catch breaking changes early

Don't write contract tests for:
- Standard libraries (maintained by language team)
- Internal APIs you control
- Well-established, stable APIs with versioning guarantees

## CLI E2E Test Patterns

### Black Box Binary Testing

```go
// cmd/feedmix/version_integration_test.go
// +build integration

func TestCLI_Version_ShowsCorrectFormat(t *testing.T) {
    // Build the binary
    cmd := exec.Command("go", "build", "-o", "feedmix-test", "./cmd/feedmix")
    require.NoError(t, cmd.Run())
    defer os.Remove("feedmix-test")

    // Run the binary
    output, err := exec.Command("./feedmix-test", "--version").CombinedOutput()
    require.NoError(t, err)

    // Verify output
    assert.Contains(t, string(output), "feedmix version")
}
```

### Testing Error Cases

```go
func TestCLI_MissingCredentials_ShowsHelpfulError(t *testing.T) {
    // Clear environment
    os.Unsetenv("FEEDMIX_YOUTUBE_CLIENT_ID")

    cmd := exec.Command("./feedmix", "auth")
    output, err := cmd.CombinedOutput()

    // Should fail
    assert.Error(t, err)

    // Should show helpful error
    assert.Contains(t, string(output), "FEEDMIX_YOUTUBE_CLIENT_ID")
    assert.Contains(t, string(output), ".env.example")
}
```

## Test Helpers

### Setup and Teardown

```go
func TestOAuthFlow(t *testing.T) {
    // Setup
    tempDir := t.TempDir() // Automatically cleaned up
    oldHome := os.Getenv("HOME")
    os.Setenv("HOME", tempDir)
    defer os.Setenv("HOME", oldHome)

    // Test...
}
```

### Custom Assertions

```go
func assertTokenValid(t *testing.T, token *Token) {
    t.Helper() // Marks this as a helper (better error messages)

    assert.NotEmpty(t, token.AccessToken)
    assert.NotEmpty(t, token.RefreshToken)
    assert.True(t, token.Expiry.After(time.Now()))
}
```

## Common Test Antipatterns

### ❌ Testing Implementation Details

```go
// BAD - Tests internal structure
func TestAggregator_HasYouTubeClient(t *testing.T) {
    agg := NewAggregator()
    assert.NotNil(t, agg.youtubeClient) // Implementation detail!
}
```

### ❌ Fragile Tests (Hardcoded Times)

```go
// BAD - Will break when test runs at different time
func TestTokenExpiry(t *testing.T) {
    token := &Token{
        Expiry: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
    }
    assert.True(t, token.IsExpired()) // Depends on current time!
}

// GOOD - Mock time or use relative times
func TestTokenExpiry(t *testing.T) {
    now := time.Now()
    token := &Token{
        Expiry: now.Add(-1 * time.Hour), // Expired 1 hour ago
    }
    assert.True(t, token.IsExpired())
}
```

### ❌ Tests That Don't Test Anything

```go
// BAD - Just tests the mock
func TestFeed(t *testing.T) {
    mock := &MockClient{
        Videos: []Video{{Title: "test"}},
    }
    result := mock.GetVideos()
    assert.Equal(t, "test", result[0].Title) // Of course it matches!
}
```

## Coverage Guidelines

### What to Test
- All public functions and methods
- Error handling paths
- Edge cases (empty input, nil values, boundary conditions)
- Integration points (API calls, file I/O, external commands)

### What NOT to Test
- Trivial getters/setters
- Third-party library code
- Generated code
- Obvious pass-through functions

### Coverage Targets
- **Unit tests**: 80%+ coverage of your own code
- **Contract tests**: 100% coverage of external API assumptions
- **Integration tests**: Critical paths (OAuth flow, feed aggregation)
- **E2E tests**: User-facing commands and error messages

### Check Coverage
```bash
go test -cover ./...                    # Quick coverage summary
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out        # Visual coverage report
```
