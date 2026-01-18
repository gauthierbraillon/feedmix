// Package oauth flow tests document the OAuth browser authentication flow.
//
// TDD Cycle: RED -> GREEN -> REFACTOR
//
// Test requirements (this file serves as documentation):
// - StartCallbackServer starts HTTP server on specified port
// - Server handles OAuth callback with code and state
// - Server validates state matches expected value (CSRF protection)
// - ExchangeCode exchanges authorization code for tokens
// - Flow integrates: generate URL -> callback -> exchange -> save token
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// TestCallbackServer_ReceivesCode documents callback server behavior:
// - Starts HTTP server and waits for callback
// - Extracts authorization code from query params
// - Validates state parameter matches expected value
func TestCallbackServer_ReceivesCode(t *testing.T) {
	expectedState := "test-state-123"

	server := NewCallbackServer(8085)

	// Start server in background
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)
	go func() {
		code, err := server.WaitForCallback(context.Background(), expectedState, 5*time.Second)
		if err != nil {
			errChan <- err
			return
		}
		codeChan <- code
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Simulate OAuth provider callback
	resp, err := http.Get(fmt.Sprintf("http://localhost:8085/callback?code=auth-code-xyz&state=%s", expectedState))
	if err != nil {
		t.Fatalf("failed to make callback request: %v", err)
	}
	resp.Body.Close()

	select {
	case code := <-codeChan:
		if code != "auth-code-xyz" {
			t.Errorf("expected code 'auth-code-xyz', got %q", code)
		}
	case err := <-errChan:
		t.Fatalf("callback server error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for callback")
	}
}

// TestCallbackServer_RejectsInvalidState documents CSRF protection:
// - Rejects callbacks with mismatched state parameter
func TestCallbackServer_RejectsInvalidState(t *testing.T) {
	expectedState := "correct-state"

	server := NewCallbackServer(8086)

	errChan := make(chan error, 1)
	go func() {
		_, err := server.WaitForCallback(context.Background(), expectedState, 5*time.Second)
		errChan <- err
	}()

	time.Sleep(50 * time.Millisecond)

	// Send callback with wrong state
	resp, err := http.Get("http://localhost:8086/callback?code=some-code&state=wrong-state")
	if err != nil {
		t.Fatalf("failed to make callback request: %v", err)
	}
	resp.Body.Close()

	// Should return error about invalid state
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid state, got %d", resp.StatusCode)
	}
}

// TestCallbackServer_Timeout documents timeout behavior:
// - Returns error if no callback received within timeout
func TestCallbackServer_Timeout(t *testing.T) {
	server := NewCallbackServer(8087)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := server.WaitForCallback(ctx, "some-state", 100*time.Millisecond)

	if err == nil {
		t.Error("expected timeout error")
	}
}

// TestFlow_ExchangeCode_Success documents token exchange:
// - Sends POST request to token endpoint
// - Includes code, client_id, client_secret, redirect_uri
// - Parses token response
func TestFlow_ExchangeCode_Success(t *testing.T) {
	// Mock token endpoint
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse form: %v", err)
		}

		// Verify required parameters
		if r.FormValue("code") != "test-auth-code" {
			t.Errorf("expected code 'test-auth-code', got %q", r.FormValue("code"))
		}
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("expected grant_type 'authorization_code', got %q", r.FormValue("grant_type"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "new-access-token",
			"refresh_token": "new-refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	config := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenURL:     tokenServer.URL,
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"read"},
	}

	flow := NewFlow(config)
	token, err := flow.ExchangeCode(context.Background(), "test-auth-code")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token.AccessToken != "new-access-token" {
		t.Errorf("expected access token 'new-access-token', got %q", token.AccessToken)
	}

	if token.RefreshToken != "new-refresh-token" {
		t.Errorf("expected refresh token 'new-refresh-token', got %q", token.RefreshToken)
	}
}

// TestFlow_ExchangeCode_Error documents error handling:
// - Returns error when token endpoint returns error
func TestFlow_ExchangeCode_Error(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":             "invalid_grant",
			"error_description": "Code expired",
		})
	}))
	defer tokenServer.Close()

	config := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenURL:     tokenServer.URL,
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"read"},
	}

	flow := NewFlow(config)
	_, err := flow.ExchangeCode(context.Background(), "expired-code")

	if err == nil {
		t.Error("expected error for invalid grant")
	}
}

// TestGetAuthURL_YouTube documents YouTube OAuth URL:
// - Uses correct YouTube OAuth endpoints
// - Includes required scopes
func TestGetAuthURL_YouTube(t *testing.T) {
	config := YouTubeOAuthConfig("client-id", "client-secret", "http://localhost:8080/callback")

	flow := NewFlow(config)
	authURL, state := flow.GenerateAuthURL()

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("invalid URL: %v", err)
	}

	if parsed.Host != "accounts.google.com" {
		t.Errorf("expected accounts.google.com, got %s", parsed.Host)
	}

	if state == "" {
		t.Error("state should not be empty")
	}
}

// TestGetAuthURL_LinkedIn documents LinkedIn OAuth URL:
// - Uses correct LinkedIn OAuth endpoints
// - Includes required scopes
func TestGetAuthURL_LinkedIn(t *testing.T) {
	config := LinkedInOAuthConfig("client-id", "client-secret", "http://localhost:8080/callback")

	flow := NewFlow(config)
	authURL, state := flow.GenerateAuthURL()

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("invalid URL: %v", err)
	}

	if parsed.Host != "www.linkedin.com" {
		t.Errorf("expected www.linkedin.com, got %s", parsed.Host)
	}

	if state == "" {
		t.Error("state should not be empty")
	}
}
