# Security & Test Strategy Gap Analysis

## Test Strategy Comparison

### ADHD_Clock Template (Target)

1. **PRIMARY: Sociable Unit Tests** (< 2s)
   - Tests ARE acceptance criteria
   - Mock externals only
   - AC naming: "AC301: After last activity shows free time"

2. **SECONDARY: Integration Tests** (10-30s)
   - Real external systems (browser, database, filesystem)
   - Run in deployment pipeline BEFORE deploy

3. **TERTIARY: E2E Smoke Tests** (30-60s)
   - Run AFTER deployment on production environment
   - **ROLLBACK trigger** if fail

### Feedmix Current State

1. **PRIMARY: Sociable Unit Tests** ✅ (< 2s)
   - Mock externals only ✅
   - AC naming ❌ (uses descriptive names, not AC numbers)

2. **SECONDARY: Contract Tests** ⚠️ (should be lower priority)
   - Verifies YouTube API assumptions
   - Doesn't test real integration

3. **TERTIARY: Integration Tests** ✅ (5-15s)
   - Real OAuth flow, real API calls
   - Build tag: `integration`

4. **QUATERNARY: CLI E2E Tests** ⚠️ (not on production)
   - Tests built binaries locally
   - Doesn't verify production deployment

### Gaps

#### 1. Test Priority Misalignment
**Issue**: Contract tests are SECONDARY, but integration tests should be.

**Fix**: Reorder test types:
- PRIMARY: Sociable unit tests
- SECONDARY: Integration tests (real OAuth, real YouTube API)
- TERTIARY: Contract tests (documents API assumptions)
- QUATERNARY: E2E smoke tests (production verification)

#### 2. Missing Production E2E Smoke Tests
**Issue**: Current E2E tests run on locally built binaries, not deployed production.

**What's missing**:
```bash
# After GitHub Release is created
tests/e2e-smoke-production.sh
  - Download binary from GitHub Releases (not local build)
  - Test on clean environment (Docker container)
  - Verify version matches release
  - Test auth flow (if credentials available)
  - Test feed display
  - If any fail → ROLLBACK release
```

**Current**:
- GitHub Actions E2E runs on binaries built in CI
- No verification that released binaries work for end users
- No rollback if distributed binaries are broken

#### 3. Test Naming Convention
**Issue**: Tests don't follow AC naming convention.

**Current**: `TestOAuthFlow_SavesTokensAfterCallback`
**Should be**: `TestAC101_OAuthFlow_SavesTokensAfterCallback`

**Why it matters**:
- Tests ARE the requirements
- AC numbers link tests to acceptance criteria
- Makes traceability explicit

#### 4. No Test Organization Document
**Issue**: No clear documentation of what each test type covers.

**Needed**: Similar to ADHD_Clock's test organization:
```
tests/
├── unit/           # Sociable unit tests (AC-driven)
├── integration/    # Real system integration
├── contracts/      # API contract verification
└── e2e/           # Production smoke tests
```

## Security Gap Analysis

### ADHD_Clock Security (Template)

1. ✅ Authentication (email/password + hCaptcha)
2. ✅ Rate limiting (prevent brute force)
3. ✅ Audit logging (track auth events)
4. ✅ Input sanitization (prevent XSS/injection)
5. ✅ Error sanitization (no technical details to users)
6. ✅ RLS policies (row-level security)
7. ✅ Security scans (govulncheck, gosec)

### Feedmix Current State

1. ✅ Authentication (OAuth 2.0 with Google)
2. ✅ Token storage (0600 file permissions)
3. ❌ **CRITICAL: Command injection vulnerability** (browser launcher)
4. ❌ Rate limiting (YouTube API quota not tracked)
5. ❌ Audit logging (no security event logging)
6. ⚠️ Input validation (partial - needs improvement)
7. ❌ Error sanitization (not verified)
8. ⚠️ Security scans (gosec only in GitHub Actions, govulncheck skipped)

### Critical Security Issues (Blocking Deployment)

#### 1. **G204 (CWE-78): Command Injection** - HIGH SEVERITY
**Location**: `pkg/browser/browser.go:16,18,20`

**Vulnerability**:
```go
// VULNERABLE - url is not validated
exec.Command("xdg-open", url)
exec.Command("open", url)
exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
```

**Attack Vector**:
- Malicious OAuth provider returns redirect with shell metacharacters
- `url` could be: `http://evil.com; rm -rf /`
- Shell commands executed on user's machine

**Fix Required**:
```go
func Open(url string) error {
    // Validate URL before passing to shell
    parsedURL, err := neturl.Parse(url)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }

    // Whitelist allowed schemes
    if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
        return fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
    }

    // Now safe to pass to exec.Command
    // ... rest of code
}
```

#### 2. **G117 (CWE-499): Exposed Secret Fields** - FALSE POSITIVE
**Location**: `pkg/oauth/oauth.go:28,63,64`

**Issue**:
```go
type Token struct {
    AccessToken  string `json:"access_token"`   // gosec flags this
    RefreshToken string `json:"refresh_token"` // gosec flags this
}
```

**Not Actually Vulnerable**: These are struct fields for JSON marshaling, not secrets being logged.

**Fix Required**: Suppress false positive
```go
type Token struct {
    AccessToken  string `json:"access_token"`   // #nosec G117 - JSON field, not exposed secret
    RefreshToken string `json:"refresh_token"` // #nosec G117 - JSON field, not exposed secret
}
```

### High-Priority Security Improvements

#### 3. Input Validation (All User Inputs)
**Missing**:
- OAuth callback parameter validation (state, code)
- YouTube API response validation (prevent malformed data)
- File path validation (prevent path traversal)
- Environment variable validation (prevent injection)

**Add**:
```go
// pkg/oauth/validation.go
func ValidateCallbackParams(code, state string) error {
    if len(code) == 0 || len(code) > 512 {
        return ErrInvalidCode
    }
    if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(code) {
        return ErrInvalidCodeFormat
    }
    // ... validate state
}
```

#### 4. Rate Limiting / Quota Tracking
**Missing**: YouTube API quota monitoring

**Risk**:
- Exceed daily quota (10,000 units)
- User gets errors without warning
- Potential for abuse

**Add**:
```go
// internal/youtube/quota.go
type QuotaTracker struct {
    Used  int
    Limit int
    Reset time.Time
}

func (q *QuotaTracker) Track(cost int) error {
    if q.Used+cost > q.Limit {
        return ErrQuotaExceeded{ResetAt: q.Reset}
    }
    q.Used += cost
    return nil
}

func (q *QuotaTracker) Warn() bool {
    return float64(q.Used)/float64(q.Limit) > 0.8 // 80% threshold
}
```

#### 5. Security Scans in Local CI
**Current**:
- `govulncheck` skipped (user needs to install manually)
- `gosec` only runs in GitHub Actions

**Problem**:
- Developers don't see security issues until GitHub Actions
- Slow feedback loop (minutes vs seconds)

**Fix**: Make security scans mandatory in `scripts/ci.sh`
```bash
step "Security (govulncheck)..."
if ! command -v govulncheck &>/dev/null; then
    fail "govulncheck not installed. Install: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi
govulncheck ./... || fail "Vulnerabilities found"

step "Security (gosec)..."
if ! command -v gosec &>/dev/null; then
    fail "gosec not installed. Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"
fi
gosec -exclude=G117 ./... || fail "Security issues found"
```

#### 6. PKCE for OAuth (Recommended)
**Current**: OAuth 2.0 without PKCE

**Risk**:
- Authorization code interception attacks
- Less secure than RFC 8252 recommendation

**Add**:
```go
// pkg/oauth/pkce.go
func GenerateCodeVerifier() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(b), nil
}

func GenerateCodeChallenge(verifier string) string {
    h := sha256.Sum256([]byte(verifier))
    return base64.RawURLEncoding.EncodeToString(h[:])
}
```

### Lower-Priority Security Enhancements

#### 7. Audit Logging
**Purpose**: Track security events for forensics

**Events to log**:
- OAuth authentication attempts (success/failure)
- Token refresh (success/failure)
- API quota warnings
- Invalid input attempts

**Implementation**:
```go
// pkg/audit/logger.go
func LogAuthSuccess(userID string) {
    log.Printf("[AUDIT] AUTH_SUCCESS user=%s", userID)
}

func LogAuthFailure(reason string) {
    log.Printf("[AUDIT] AUTH_FAILURE reason=%s", reason)
}
```

#### 8. Error Sanitization
**Current**: Errors may leak technical details

**Risk**:
- Stack traces exposed to users
- Internal paths/config visible
- Aids attackers in reconnaissance

**Fix**:
```go
func SanitizeError(err error) string {
    // Don't expose internal errors to users
    if errors.Is(err, ErrTokenExpired) {
        return "Authentication expired. Please run: feedmix auth"
    }
    // Generic error for everything else
    return "An error occurred. Please try again or report this issue."
}
```

## Action Plan

### Immediate (Blocks Deployment)
1. ✅ Fix command injection in browser.go (Task #5)
2. ✅ Suppress G117 false positives (Task #6)
3. ✅ Add gosec and govulncheck to local CI (Task #8)

### High Priority (This Sprint)
4. ✅ Add comprehensive input validation (Task #7)
5. ✅ Align test strategy with template (Task #9)
6. Add production E2E smoke tests
7. Add YouTube API quota tracking

### Medium Priority (Next Sprint)
8. Implement PKCE for OAuth
9. Add audit logging
10. Add error sanitization
11. Update test naming to AC convention

### Low Priority (Future)
12. Add fuzzing tests
13. Add penetration testing
14. Implement rate limiting on CLI commands
15. Add security monitoring/alerts

## Conclusion

**Test Strategy**: 80% aligned, needs reordering and production E2E tests
**Security**: 60% aligned, **CRITICAL vulnerabilities block deployment**

**Next Steps**:
1. Fix blocking security issues (command injection)
2. Re-run deployment pipeline
3. Add missing security scans to local CI
4. Improve input validation
5. Add production E2E smoke tests
