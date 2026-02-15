# OAuth Flow Patterns

## OAuth 2.0 Flow for Native Apps (RFC 8252)

Feedmix implements the OAuth 2.0 flow for native applications (CLI tools).

### Flow Overview

```
1. Start local callback server (localhost:8080)
2. Open browser to Google OAuth consent screen
3. User approves permissions
4. Google redirects to http://localhost:8080/callback?code=AUTH_CODE
5. Exchange AUTH_CODE for access token + refresh token
6. Save tokens to disk (~/.config/feedmix/tokens.json)
7. Shutdown callback server
```

### Implementation Details

**Step 1: Start Callback Server**
```go
// pkg/oauth/oauth.go
func (c *Client) StartCallbackServer() error {
    server := &http.Server{
        Addr: ":8080",
        Handler: http.HandlerFunc(c.handleCallback),
    }

    go server.ListenAndServe()
    return nil
}
```

**Step 2: Generate Authorization URL**
```go
func (c *Client) AuthURL() string {
    params := url.Values{
        "client_id":     {c.clientID},
        "redirect_uri":  {"http://localhost:8080/callback"},
        "response_type": {"code"},
        "scope":         {"https://www.googleapis.com/auth/youtube.readonly"},
        "access_type":   {"offline"}, // Get refresh token
        "prompt":        {"consent"},  // Force consent (ensures refresh token)
    }
    return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}
```

**Step 3: Open Browser**
```go
func (c *Client) Authenticate() error {
    c.StartCallbackServer()

    authURL := c.AuthURL()
    if err := browser.Open(authURL); err != nil {
        fmt.Printf("Could not open browser. Please visit:\n%s\n", authURL)
    }

    // Wait for callback...
    return c.WaitForCallback()
}
```

**Step 4: Handle Callback**
```go
func (c *Client) handleCallback(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    if code == "" {
        http.Error(w, "Missing code parameter", http.StatusBadRequest)
        return
    }

    c.authCodeChan <- code

    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "Authentication successful! You can close this window.")
}
```

**Step 5: Exchange Code for Tokens**
```go
func (c *Client) ExchangeCode(code string) (*Tokens, error) {
    data := url.Values{
        "client_id":     {c.clientID},
        "client_secret": {c.clientSecret},
        "code":          {code},
        "grant_type":    {"authorization_code"},
        "redirect_uri":  {"http://localhost:8080/callback"},
    }

    resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var tokens Tokens
    if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
        return nil, err
    }

    return &tokens, nil
}
```

**Step 6: Save Tokens**
```go
func SaveTokens(tokens *Tokens) error {
    configDir, err := os.UserConfigDir()
    if err != nil {
        return err
    }

    feedmixDir := filepath.Join(configDir, "feedmix")
    if err := os.MkdirAll(feedmixDir, 0700); err != nil {
        return err
    }

    tokenPath := filepath.Join(feedmixDir, "tokens.json")
    data, err := json.MarshalIndent(tokens, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(tokenPath, data, 0600) // User read/write only
}
```

## Token Refresh

Access tokens expire after 1 hour. Use refresh tokens to get new access tokens without re-authenticating.

### Refresh Flow

```go
func (c *Client) RefreshToken(refreshToken string) (*Tokens, error) {
    data := url.Values{
        "client_id":     {c.clientID},
        "client_secret": {c.clientSecret},
        "refresh_token": {refreshToken},
        "grant_type":    {"refresh_token"},
    }

    resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode == 400 {
        // Refresh token expired or revoked - require re-auth
        return nil, ErrRefreshTokenInvalid
    }

    var tokens Tokens
    if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
        return nil, err
    }

    // Refresh response doesn't include new refresh token - preserve old one
    tokens.RefreshToken = refreshToken

    return &tokens, nil
}
```

### Automatic Token Refresh

```go
func (c *Client) GetValidToken() (string, error) {
    tokens, err := LoadTokens()
    if err != nil {
        return "", ErrNotAuthenticated
    }

    // Check if token is expired (with 5-minute buffer)
    if time.Now().Add(5 * time.Minute).After(tokens.Expiry) {
        // Refresh token
        newTokens, err := c.RefreshToken(tokens.RefreshToken)
        if err == ErrRefreshTokenInvalid {
            return "", ErrNotAuthenticated // User must re-auth
        }
        if err != nil {
            return "", err
        }

        // Save new tokens
        if err := SaveTokens(newTokens); err != nil {
            return "", err
        }

        return newTokens.AccessToken, nil
    }

    return tokens.AccessToken, nil
}
```

## Security Considerations

### Token Storage
- Store tokens in user config directory (`~/.config/feedmix/` on Linux/macOS)
- Set file permissions to 0600 (user read/write only)
- Never log tokens or include in error messages
- Never commit tokens to version control

### Redirect URI Validation
- Only accept callbacks from `localhost:8080`
- Validate that `state` parameter matches (if using CSRF protection)
- Timeout callback server after 5 minutes

### Scopes
- Request minimum necessary scopes
- Current scope: `https://www.googleapis.com/auth/youtube.readonly`
- Never request write permissions unless absolutely necessary

### PKCE (Proof Key for Code Exchange)
- **TODO**: Implement PKCE for additional security
- Prevents authorization code interception attacks
- Recommended by RFC 8252 for native apps

## Common Issues

### Issue: "redirect_uri_mismatch"
**Cause**: Redirect URI in request doesn't match Google Cloud Console configuration
**Solution**: Ensure `http://localhost:8080/callback` is added as authorized redirect URI

### Issue: Refresh token not returned
**Cause**: User already granted consent previously
**Solution**: Add `prompt=consent` to force consent screen and guarantee refresh token

### Issue: "invalid_grant" on token refresh
**Cause**: Refresh token expired (after 6 months of inactivity) or revoked
**Solution**: Detect error and require user to re-authenticate

### Issue: Port 8080 already in use
**Cause**: Another process is using port 8080
**Solution**: Allow configurable port or find available port dynamically

## Testing OAuth Flow

### Unit Tests (Mock HTTP)
```go
func TestOAuthFlow_ExchangeCode(t *testing.T) {
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

    client := NewOAuthClient(mockHTTP)
    tokens, err := client.ExchangeCode("auth_code_123")

    assert.NoError(t, err)
    assert.Equal(t, "ya29.abc", tokens.AccessToken)
    assert.Equal(t, "1//xyz", tokens.RefreshToken)
}
```

### Integration Tests (Real Browser)
```go
// +build integration

func TestOAuthFlow_RealBrowser(t *testing.T) {
    client := NewOAuthClient(http.DefaultClient)

    // This will open a real browser
    err := client.Authenticate()
    assert.NoError(t, err)

    // Verify tokens saved
    tokens, err := LoadTokens()
    assert.NoError(t, err)
    assert.NotEmpty(t, tokens.AccessToken)
}
```

### Manual Testing
```bash
# Clear existing tokens
rm ~/.config/feedmix/tokens.json

# Run auth flow
feedmix auth

# Verify tokens saved
cat ~/.config/feedmix/tokens.json

# Test token refresh (wait 1 hour or manually edit expiry)
feedmix feed  # Should automatically refresh token
```

## References
- [RFC 8252: OAuth 2.0 for Native Apps](https://tools.ietf.org/html/rfc8252)
- [Google OAuth 2.0 for Mobile & Desktop Apps](https://developers.google.com/identity/protocols/oauth2/native-app)
- [OAuth 2.0 Security Best Practices](https://tools.ietf.org/html/draft-ietf-oauth-security-topics)
