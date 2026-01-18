# Claude Code Instructions

## Workflow: TDD + CD

Every code change must follow this workflow:

### 1. RED: Write Failing Test
```bash
# Write test first, run to confirm it fails
go test ./path/to/package -run TestName -v
```

### 2. GREEN: Minimal Implementation
```bash
# Write minimal code to pass the test
go test ./path/to/package -run TestName -v
```

### 3. REFACTOR: Improve Code
```bash
# Refactor while keeping tests green
go test ./...
```

### 4. CI: Run Full Pipeline
```bash
./scripts/ci.sh
```

### 5. COMMIT: Only After CI Passes
```bash
git add -A
git commit -m "feat/fix/refactor: description

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

## Rules

1. **Never skip steps** - RED before GREEN, CI before COMMIT
2. **Tests document behavior** - Not implementation
3. **Mock only externals** - HTTP APIs, file system, time
4. **Black box CLI tests** - Build and run actual binary
5. **Contract tests** - Verify API schema compatibility
6. **Run CI locally** - Before every commit

## Commands

| Action | Command |
|--------|---------|
| Run all tests | `go test ./...` |
| Run with race detector | `go test -race ./...` |
| Run specific test | `go test ./pkg/oauth -run TestFlow -v` |
| Run CI pipeline | `./scripts/ci.sh` |
| Build binary | `go build -o feedmix ./cmd/feedmix` |
| Build release | `./scripts/deploy.sh` |

## Test Patterns

```go
// Mock HTTP API
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(response)
}))
client := NewClient(token, WithBaseURL(server.URL))

// Functional options for DI
func WithBaseURL(url string) Option {
    return func(c *Client) { c.baseURL = url }
}
```

## CI Pipeline Gates

All must pass before merge:
- Unit tests with race detector
- Contract tests
- golangci-lint
- govulncheck (security)
- gosec (security)
- Build verification
