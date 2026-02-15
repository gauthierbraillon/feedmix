# Feedmix Memory

This file contains persistent knowledge learned across development sessions. Keep it concise (<200 lines) as it's loaded into the system prompt.

## Project Context

- **Product**: CLI tool for aggregating YouTube subscriptions into a terminal feed
- **Language**: Go 1.24+
- **Distribution**: Single binary via `go install`, GitHub Releases for multiple platforms
- **Authentication**: OAuth 2.0 with Google, tokens stored locally

## Testing Strategy

- **Primary**: Sociable unit tests (fast, <2s, mock externals only)
- **Secondary**: Contract tests (verify YouTube API assumptions, catch breaking changes)
- **Tertiary**: Integration tests (real OAuth flow, real API calls, build tag: `integration`)
- **Quaternary**: CLI E2E tests (black box binary testing, post-build verification)

## Common Patterns

### OAuth Flow
- Start callback server → Open browser → User approves → Exchange code → Save tokens
- Tokens stored at `~/.config/feedmix/tokens.json` with 0600 permissions
- See [oauth.md](oauth.md) for detailed patterns and gotchas

### YouTube API Integration
- API: YouTube Data API v3
- Rate limits: 10,000 quota units/day
- See [youtube-api.md](youtube-api.md) for endpoint details and quirks

### Testing Approach
- Mock external dependencies (YouTube API, filesystem, browser) with interfaces
- Use real collaborators for internal packages (no mocks of our own code)
- See [testing.md](testing.md) for test patterns and examples

## Known Issues & Solutions

### Version Detection with `go install`
- **Issue**: Version embedded via `-ldflags` not available when installed via `go install`
- **Solution**: Auto-version detection from git tags (implemented in v1.2.0)
- **Details**: See [debugging.md](debugging.md#version-detection)

## Architecture Decisions

### Package Organization
- `cmd/feedmix/`: CLI entry point and command parsing
- `internal/`: Private packages (youtube client, aggregator, display)
- `pkg/`: Public packages (oauth, browser, contracts)
- **Rationale**: Follow Go conventions, prevent external imports of internal code

### Dependency Injection
- External dependencies use interfaces (mockable in tests)
- Production injects real implementations, tests inject mocks
- **Example**: `YouTubeClient` interface with `RealYouTubeClient` and `MockYouTubeClient`

## Workflow Reminders

### TDD Cycle (MANDATORY)
1. **RED**: Write failing test (verify it fails)
2. **GREEN**: Make test pass (minimum code)
3. **REFACTOR**: Clean up (run all tests)
4. **CI**: Run `./scripts/ci.sh` (all gates must pass)
5. **COMMIT**: `git add -A && git commit -m "type: description"`
6. **DEPLOY**: `git push` (triggers GitHub Actions)

### Before Every Commit
```bash
./scripts/ci.sh  # Vet + tests + race + integration + security + build + verify
```

### Release Process
```bash
git tag v1.2.3 && git push origin v1.2.3  # Triggers automated release
```

## References

- [debugging.md](debugging.md) - Debugging patterns and common bugs
- [testing.md](testing.md) - Test patterns and examples
- [oauth.md](oauth.md) - OAuth flow details and gotchas
- [youtube-api.md](youtube-api.md) - YouTube API integration patterns
