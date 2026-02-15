# Feedmix - Project Guide

## Product Vision

Feedmix is a CLI tool for aggregating YouTube subscriptions into a terminal feed.

**Core Features:**
- **Authentication**: OAuth 2.0 flow with Google (opens browser, saves tokens)
- **Feed Display**: Terminal-based view of recent subscription videos
- **Configuration**: Environment variables or .env file for credentials
- **Installation**: Single binary via `go install github.com/gauthierbraillon/feedmix/cmd/feedmix@latest`

**Target Users:**
- Developers who prefer terminal workflows
- Users who want lightweight YouTube subscription monitoring
- Privacy-conscious users (OAuth tokens stored locally)

**Distribution:**
- Source: GitHub repository
- Install: Go toolchain (`go install`)
- Releases: GitHub Releases with auto-built binaries (Linux, macOS, Windows)

## Testing Philosophy

### ATDD (Acceptance Test Driven Development)

This project uses **ATDD, not pure TDD**. The key differences:

**ATDD (What We Do):**
- Tests ARE the acceptance criteria and testable requirements
- Test names describe behavior (e.g., "TestOAuthFlow_ReturnsTokensAfterCallback")
- Write test as specification first, then implement to pass the test
- No separate requirements document - tests document requirements
- Tests describe observable behavior, not internal design

**Pure TDD (What We Don't Do):**
- Uses tests primarily to drive internal design decisions
- Tests often focus on class structure and object interactions
- May create tests that drive unnecessary abstractions
- Tests can be coupled to implementation details

**Our Approach:**
1. Product Manager defines requirements as acceptance criteria
2. Developer writes test as executable specification (RED)
3. Developer implements minimum code to pass test (GREEN)
4. Developer refactors without changing behavior (REFACTOR)
5. QA Engineer verifies test coverage is complete (CI)
6. SRE deploys and monitors (DEPLOY)

### Test Types (Development Focus)

**PRIMARY: Sociable Unit Tests (Fast Feedback During Development)**
- **Tests ARE the acceptance criteria and testable requirements** (no separate requirements doc)
- Test full flow through OUR code/modules
- Mock external dependencies only (YouTube API, filesystem, browser)
- No mocks of our own code - use real collaborators
- Fast execution (< 2 seconds for all unit tests)
- Run constantly during RED → GREEN → REFACTOR cycle
- Test names describe requirements
- Examples: OAuth flow with token storage, feed aggregation with YouTube client, display formatting

**SECONDARY: Contract Tests (API Behavior Verification)**
- Test assumptions about external APIs (YouTube Data API v3)
- Verify our understanding of external API contracts
- Located in `pkg/contracts/contracts_test.go`
- Run with `go test ./pkg/contracts/...`
- Catch breaking changes in external APIs early
- Examples: YouTube API response structure, OAuth token format

**TERTIARY: Integration Tests (Real System Behavior)**
- Test integration with real external systems
- Build tags: `// +build integration`
- Run with `go test -tags=integration ./...`
- Slower execution (5-15 seconds)
- Run in CI pipeline before build
- Examples: Real OAuth flow (opens browser), real YouTube API calls, filesystem operations

**QUATERNARY: CLI E2E Tests (Binary Behavior)**
- Black box testing of compiled binary
- Build binary, run commands, verify output
- Test actual user experience
- Located in `cmd/feedmix/*_integration_test.go`
- Run in CI after build step
- Examples: `feedmix --version`, `feedmix auth`, `feedmix feed`

### Test Organization

```
cmd/feedmix/
├── main.go
├── *_test.go                    # Unit tests for CLI logic
└── *_integration_test.go        # Black box binary tests

internal/
├── youtube/
│   ├── client.go
│   └── client_test.go           # Sociable unit tests
├── aggregator/
│   ├── aggregator.go
│   └── aggregator_test.go       # Sociable unit tests
└── display/
    ├── terminal.go
    └── terminal_test.go         # Sociable unit tests

pkg/
├── oauth/
│   ├── oauth.go
│   ├── oauth_test.go            # Sociable unit tests
│   └── oauth_integration_test.go # Real OAuth flow (build tag)
└── contracts/
    └── contracts_test.go        # API contract verification
```

### What to Mock

- ✅ External APIs (YouTube Data API, OAuth endpoints)
- ✅ Filesystem operations (token storage)
- ✅ Browser launcher (OAuth flow)
- ✅ HTTP clients (use httptest for external calls)
- ❌ Our own code (internal packages)
- ❌ Simple utilities (time formatting, string manipulation)

### Test Pyramid Rules

**Test at lowest level possible. No duplicate coverage.**
- Unit test? → Don't test again at integration level
- Contract test? → Documents API assumptions only
- Integration test? → Tests real system integration only
- E2E test? → Tests binary behavior only
- Fast feedback: unit (<2s) > contract (~5s) > integration (~15s) > E2E (~30s)

**Go test execution includes contracts by default.**
- `go test ./...` runs unit tests AND contract tests
- Contract tests are just regular tests in pkg/contracts/
- No need to run contracts separately locally (fast enough)
- Integration tests require `-tags=integration` flag (slower, skip during dev)

## Working with Specialized Agents

**IMPORTANT**: This project uses specialized AI agents that **automatically activate** based on task context. You don't need to manually invoke them - they switch seamlessly as work progresses.

### When Agents Activate

- **Product Manager** → Defining requirements, breaking down features, prioritizing work
- **Developer** → Writing code, following TDD cycle, implementing features
- **QA Engineer** → Writing/reviewing tests, checking coverage, testing edge cases
- **Architect** → Making technology choices, designing system architecture
- **SRE** → Handling deployments, pipeline issues, production monitoring
- **Security Engineer** → Reviewing auth/authorization, OAuth security, input validation

### Multi-Agent Collaboration

Complex tasks automatically involve multiple agents in sequence:
1. Product Manager defines requirements as acceptance criteria
2. Architect (if needed) designs system approach
3. Developer implements with TDD (RED → GREEN → REFACTOR)
4. QA Engineer verifies test coverage
5. Security Engineer reviews security implications
6. SRE deploys and monitors

**Agent switching is automatic and transparent** throughout task execution.

## Using Claude Plugins & MCP Servers

**IMPORTANT**: When available, use Claude plugins (MCP servers) to enhance development workflow and code quality.

### When to Use Plugins

Consider using plugins for:
- **Go Tools**: Go documentation lookup, package search, godoc generation
- **Code Quality**: Advanced linters beyond golangci-lint, security scanners
- **Testing**: Test generators, coverage visualizers, fuzzing tools
- **API Testing**: HTTP clients, API explorers, OAuth debugging tools
- **Performance**: Profilers, benchmarking tools, memory analyzers
- **Documentation**: Architecture diagrams, README generators

### Plugin Discovery

Before implementing a task, check if relevant plugins are available:
- Search for plugins that could help with the current task
- Evaluate if plugin usage would save time or improve quality
- Use plugins when they add value, skip when manual approach is simpler

**Balance**: Use plugins to enhance workflow, but don't over-complicate simple tasks. If the manual approach is clear and quick, prefer it.

## Development Workflow

### Continuous Delivery Principles

1. **Small batches**: Deploy after each feature/fix
2. **Fast feedback**: All tests run in < 2 minutes
3. **Green to deploy**: If tests pass, code is deployable
4. **No manual gates**: Automation decides quality

### Deployment Pipeline

**Simplified Continuous Delivery:**
```bash
# 1. During development (RED → GREEN → REFACTOR)
./scripts/ci.sh      # Fast local feedback (< 2 min)

# 2. After commit (DEPLOY)
./scripts/deploy.sh  # Full validation + automatic push (< 5 min)
```

**What deploy.sh does automatically:**
- ✅ Runs full CI (vet, tests, race, integration, security)
- ✅ Builds binaries for all platforms
- ✅ Runs E2E smoke tests
- ✅ **Pushes to GitHub** (no manual gate!)
- ✅ Triggers GitHub Actions deployment

**Gates (must all pass):**

**Local CI (scripts/ci.sh):**
1. **Vet**: `go vet ./...` (static analysis)
2. **Tests**: `go test -race -cover ./...` (unit + contract tests with race detector)
3. **Integration**: `go test -tags=integration ./pkg/oauth/... ./cmd/feedmix/...` (real system tests)
4. **Security**: `govulncheck ./...` (vulnerability scanning)
5. **Build**: `go build -o feedmix ./cmd/feedmix` (compilation verification)
6. **Verify**: `./feedmix --version` (binary smoke test)

**GitHub Actions (merge gates + deploy):**
1. **All local CI gates** (vet, tests, integration, security, build)
2. **Lint**: `golangci-lint run` (comprehensive linting)
3. **Security**: `gosec ./...` (additional security scanning)
4. **Release**: Auto-versioning from git tags
5. **Deploy**: Build binaries for Linux/macOS/Windows → GitHub Releases
6. **Post-deploy**: E2E smoke tests on released binaries

**Pre-push gates (1-6):** If any fail, push aborts (code never reaches GitHub)
**Post-deploy smoke tests (6):** If fail, ROLLBACK deployment immediately (delete release, revert tag)

### Writing Code

**🚨 CRITICAL: ALWAYS Follow the TDD Cycle (RED → GREEN → REFACTOR → CI → COMMIT → DEPLOY) 🚨**

**This workflow is MANDATORY for ALL code changes - no exceptions:**
- ✅ Bug fixes → Follow workflow
- ✅ New features → Follow workflow
- ✅ Refactoring → Follow workflow
- ✅ Configuration changes → Follow workflow
- ✅ Documentation updates → May skip if purely textual, but consider validation tests

**If you skip the workflow, you're doing it wrong.** The workflow ensures quality, prevents bugs, and maintains continuous delivery discipline.

**TDD Cycle (RED → GREEN → REFACTOR → COMMIT → DEPLOY):**

Agents automatically activate at each phase:

1. **RED phase** - Write tests that ARE the acceptance criteria (Developer + QA Engineer)
   - **Step 1: Understand the problem** - What observable behavior needs to change?
     - For simple bugs: understand the incorrect behavior
     - For complex changes: break down into smaller testable behaviors (Product Manager may activate)
     - Think in terms of "Given-When-Then" scenarios
   - **Step 2: Write sociable unit tests** - Tests ARE the acceptance criteria and requirements
     - Test names describe the requirement (e.g., "TestOAuthFlow_SavesTokensAfterCallback")
     - Test full flow through our code with mocked externals
     - Mock external systems only (YouTube API, filesystem, browser)
     - Keep tests fast (< 2 seconds for all unit tests)
     - Tests ARE the specification - no separate requirements document needed
   - **Step 3: Run test to verify RED** - Confirm test fails for the right reason
     ```bash
     go test ./pkg/oauth -run TestOAuthFlow -v  # Must fail
     ```

2. **GREEN phase** - Make test pass (fast feedback) (Developer)
   - Implement minimum code to pass the test
   - Run ONLY the failing test (fast feedback loop)
   - Iterate quickly until test passes
   - Don't refactor yet - just make it work
   ```bash
   go test ./pkg/oauth -run TestOAuthFlow -v  # Must pass
   ```

3. **REFACTOR phase** - Clean up code AND tests (Developer + QA Engineer)
   - **CRITICAL: ACTUALLY REFACTOR THE CODE** - Don't just run tests and move on
   - **Look for code smells**:
     - Duplication (repeated logic, magic strings, duplicate conditions)
     - Poor naming (unclear variable/function names, inconsistent terminology)
     - Long functions (extract helper functions for clarity)
     - Hidden dependencies (calculate values once, not repeatedly)
     - Magic numbers/strings (extract to named constants)
   - **Refactor the implementation**: Extract functions, remove duplication, improve naming
   - **Refactor tests if needed**: Improve test readability, reduce test duplication
   - **Run ALL tests** to ensure refactoring didn't break anything
   - Tests stay sociable - no new mocks of our own code
   ```bash
   go test ./...  # All unit + contract tests must pass
   ```

4. **COMMIT phase** - Commit changes with conventional commit message (Developer)
   - **Commit changes** with descriptive conventional commit message
   ```bash
   git add -A && git commit -m "feat: add OAuth token refresh logic"
   git add -A && git commit -m "fix: prevent race condition in token storage"
   git add -A && git commit -m "refactor: extract YouTube client interface"
   git add -A && git commit -m "test: add contract tests for YouTube API"
   git add -A && git commit -m "docs: update OAuth setup instructions"
   ```
   - **Conventional commit types**: feat, fix, refactor, test, docs, chore, ci, perf
   - **Message format**: `type: description` (lowercase, no period, imperative mood)

5. **DEPLOY phase** - Automatic CD pipeline (SRE)
   - **Run deploy script** - Validates everything and auto-deploys
   ```bash
   ./scripts/deploy.sh
   ```
   - **What happens automatically**:
     1. Checks for uncommitted changes (fails if found - commit first!)
     2. Runs full CI pipeline (vet, tests, race detector, integration, security, build)
     3. Builds release binaries for all platforms
     4. Runs E2E smoke tests on built binaries
     5. **Automatically pushes to GitHub** (if all validations pass)
     6. GitHub Actions deploys (full validation, release creation, post-deploy smoke tests)
     7. If smoke tests fail → ROLLBACK immediately
   - **No manual gates**: If tests pass, code deploys. Period.
   - **Fast feedback**: Know within minutes if deployment succeeds
   - **Update .claude/memory/MEMORY.md** with lessons learned (Developer)

**Why This Workflow is NON-NEGOTIABLE:**
- **Prevents bugs**: Tests catch issues before they reach production
- **Documents behavior**: Tests serve as executable specifications
- **Enables refactoring**: Safe to improve code when tests verify behavior
- **Maintains quality**: Every commit is tested and deployable
- **Fast feedback**: Catch issues in seconds, not hours or days
- **Continuous delivery**: Small, safe, frequent deployments

**Code Style:**
- Follow [Effective Go](https://go.dev/doc/effective_go) conventions
- Use `gofmt` (enforced by golangci-lint)
- No comments except for package documentation and complex algorithms
- Prefer small, focused functions over large monolithic functions
- Use interfaces for external dependencies (mockable in tests)
- Avoid unnecessary abstractions (YAGNI - You Ain't Gonna Need It)

## Architecture

### File Structure

```
cmd/feedmix/
├── main.go                      # CLI entry point, command parsing
├── *_test.go                    # Unit tests for CLI logic
└── *_integration_test.go        # Black box binary tests

internal/
├── youtube/
│   ├── types.go                 # YouTube-specific types
│   ├── client.go                # YouTube Data API v3 client
│   └── client_test.go           # Sociable unit tests
├── aggregator/
│   ├── types.go                 # Feed aggregation types
│   ├── aggregator.go            # Feed aggregation logic
│   └── aggregator_test.go       # Sociable unit tests
└── display/
    ├── terminal.go              # Terminal output formatting
    └── terminal_test.go         # Sociable unit tests

pkg/
├── oauth/
│   ├── oauth.go                 # OAuth 2.0 flow (browser + callback server)
│   ├── oauth_test.go            # Sociable unit tests
│   └── oauth_integration_test.go # Real OAuth flow tests
├── browser/
│   └── browser.go               # Browser launcher (cross-platform)
└── contracts/
    └── contracts_test.go        # YouTube API contract tests

scripts/
├── ci.sh                        # Local CI pipeline
└── deploy.sh                    # Release script (tag + GitHub Actions)

.github/workflows/
└── release.yml                  # GitHub Actions workflow
```

### Key Concepts

**Package Organization**
- `cmd/`: Entry points (binaries)
- `internal/`: Private packages (not importable by other projects)
- `pkg/`: Public packages (importable by other projects)
- `scripts/`: Build and deployment automation

**Dependency Injection**
- External dependencies use interfaces (YouTube client, OAuth client, browser launcher)
- Tests inject mocks of external dependencies
- Production code injects real implementations

**Configuration Management**
- Environment variables: `FEEDMIX_YOUTUBE_CLIENT_ID`, `FEEDMIX_YOUTUBE_CLIENT_SECRET`
- `.env` file support (via godotenv)
- Never commit secrets (`.env` in `.gitignore`)

**Token Storage**
- OAuth tokens stored in user config directory
- Location: `~/.config/feedmix/tokens.json` (Linux/macOS)
- Permissions: 0600 (user read/write only)
- Format: JSON with access token, refresh token, expiry

**OAuth Flow**
1. Start local callback server on `localhost:8080`
2. Open browser to Google OAuth consent screen
3. User approves, Google redirects to `http://localhost:8080/callback?code=...`
4. Exchange code for tokens
5. Save tokens to disk
6. Shutdown callback server

**YouTube API Integration**
- API: YouTube Data API v3
- Endpoints: `/subscriptions`, `/videos`, `/channels`
- Rate limits: 10,000 quota units/day (monitored but not enforced in code)
- Authentication: OAuth 2.0 access token in `Authorization: Bearer` header

## Common Tasks

**Add new CLI command:** (Developer + QA Engineer)
1. Define acceptance criteria (Product Manager)
2. Write failing test in `cmd/feedmix/*_test.go` (Developer + QA)
3. Implement command logic in `cmd/feedmix/main.go` (Developer)
4. Add integration test in `cmd/feedmix/*_integration_test.go` (QA)
5. Update README.md usage section (Developer)
6. Run `./scripts/ci.sh` (Developer)
7. Commit and push (Developer)

**Add new YouTube API endpoint:** (Developer + QA Engineer + Security Engineer)
1. Add contract test to `pkg/contracts/contracts_test.go` (QA)
2. Write failing unit test in `internal/youtube/client_test.go` (Developer)
3. Implement endpoint in `internal/youtube/client.go` (Developer)
4. Verify with integration test (real API call) (QA)
5. Review security implications (rate limiting, input validation) (Security Engineer)
6. Run `./scripts/ci.sh` (Developer)
7. Commit and push (Developer)

**Fix production bug:** (Product Manager + Developer + QA Engineer + SRE)
1. **Reproduce** - Verify bug exists in production (QA Engineer)
2. **Test** - Write test that catches bug (should fail) (Developer)
   ```bash
   go test ./pkg/oauth -run TestTokenRefresh -v  # Must fail
   ```
3. **Fix** - Minimum code to pass test (Developer)
   ```bash
   go test ./pkg/oauth -run TestTokenRefresh -v  # Must pass
   ```
4. **Refactor** - Clean up code (Developer)
   ```bash
   go test ./...  # All tests pass
   ```
5. **CI** - Run local CI (Developer)
   ```bash
   ./scripts/ci.sh
   ```
6. **Deploy** - Commit, push, monitor (SRE)
   ```bash
   git add -A && git commit -m "fix: handle expired refresh tokens"
   git push
   ```
7. **Learn** - Update `.claude/memory/MEMORY.md` with lesson (Developer)

**Design new feature:** (Product Manager + Architect + Developer + Security Engineer + SRE)
1. **Define requirements** - Break down into acceptance criteria (Product Manager)
2. **Design system** - Choose architecture approach (Architect)
3. **Implement** - Follow TDD workflow (Developer)
4. **Review security** - Check for vulnerabilities (Security Engineer)
5. **Deploy** - Run full pipeline (SRE)

## Deployment

### Local Development

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Run with race detector
go test -race ./...

# Run integration tests
go test -tags=integration ./...

# Run local CI
./scripts/ci.sh

# Build
go build -o feedmix ./cmd/feedmix

# Test binary
./feedmix --version
./feedmix auth
./feedmix feed
```

### Release Process

**Automated via GitHub Actions:**

```bash
# 1. Ensure CI passes locally
./scripts/ci.sh

# 2. Create and push tag (triggers GitHub Actions)
git tag v1.2.3 && git push origin v1.2.3

# 3. GitHub Actions automatically:
#    - Runs all CI gates
#    - Builds binaries (Linux, macOS, Windows)
#    - Creates GitHub Release
#    - Attaches binaries to release
#    - Runs E2E smoke tests on release binaries
#    - If smoke tests fail → ROLLBACK (delete release, revert tag)
```

**Manual Rollback (if needed):**

```bash
# Delete broken release
gh release delete v1.2.3 --yes

# Delete tag locally and remotely
git tag -d v1.2.3
git push origin :refs/tags/v1.2.3

# Users automatically fall back to previous version via `go install`
```

## Security Reminders

**Never commit:**
- Secrets (OAuth client ID/secret, access tokens, refresh tokens)
- `.env` files (use `.env.example` as template)
- User configuration (`~/.config/feedmix/`)

**Always:**
- Use environment variables for secrets
- Validate all user inputs (file paths, URLs, API responses)
- Set restrictive file permissions on token storage (0600)
- Keep dependencies updated (`go get -u ./... && go mod tidy`)
- Run security scanners (`govulncheck`, `gosec`)
- Follow principle of least privilege (only request necessary OAuth scopes)

**OAuth Security:**
- Use PKCE (Proof Key for Code Exchange) if supported by provider
- Validate redirect URI matches expected value
- Store tokens securely (file permissions, encryption at rest)
- Rotate tokens regularly (implement refresh token logic)
- Invalidate tokens on logout

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Testing](https://go.dev/doc/tutorial/add-a-test)
- [Minimum Continuous Delivery](https://minimumcd.org/)
- [YouTube Data API v3](https://developers.google.com/youtube/v3)
- [OAuth 2.0 for Native Apps (RFC 8252)](https://tools.ietf.org/html/rfc8252)
