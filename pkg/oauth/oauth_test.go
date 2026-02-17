package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAC100_OAuthFlow_ExchangesCodeForAccessToken(t *testing.T) {
	mockOAuthProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "ya29.access-token-abc",
			"refresh_token": "1//refresh-token-xyz",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer mockOAuthProvider.Close()

	config := Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenURL:     mockOAuthProvider.URL,
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/youtube.readonly"},
	}

	token, err := NewFlow(config).ExchangeCode(context.Background(), "authorization-code-from-google")

	if err != nil {
		t.Fatalf("user should get access token after approving in browser, got error: %v", err)
	}
	if token.AccessToken == "" {
		t.Fatal("user should receive access token to use YouTube API")
	}
	if token.RefreshToken == "" {
		t.Fatal("user should receive refresh token to avoid re-authentication")
	}
}

func TestAC101_CallbackServer_RejectsInvalidStateParameter(t *testing.T) {
	server := NewCallbackServer(18087)
	expectedState := "user-session-state-xyz"

	go func() {
		time.Sleep(50 * time.Millisecond)
		resp, _ := http.Get("http://localhost:18087/callback?code=attacker-code&state=attacker-state")
		if resp != nil {
			resp.Body.Close()
		}
	}()

	_, err := server.WaitForCallback(context.Background(), expectedState, 2*time.Second)

	if err != ErrInvalidState {
		t.Errorf("callback with wrong state should be rejected (CSRF protection), got: %v", err)
	}
}

func TestAC101_CallbackServer_AcceptsMatchingStateParameter(t *testing.T) {
	server := NewCallbackServer(18088)
	validState := "correct-state-abc"

	go func() {
		time.Sleep(50 * time.Millisecond)
		resp, _ := http.Get("http://localhost:18088/callback?code=auth-code&state=correct-state-abc")
		if resp != nil {
			resp.Body.Close()
		}
	}()

	code, err := server.WaitForCallback(context.Background(), validState, 2*time.Second)

	if err != nil {
		t.Fatalf("callback with correct state should be accepted, got: %v", err)
	}
	if code != "auth-code" {
		t.Errorf("should receive authorization code, got: %s", code)
	}
}

func TestAC102_TokenStorage_PersistsTokensBetweenSessions(t *testing.T) {
	configDir, _ := os.MkdirTemp("", "oauth-test")
	defer os.RemoveAll(configDir)

	storage := NewTokenStorage(configDir)
	userToken := &Token{
		AccessToken:  "ya29.user-access-token",
		RefreshToken: "1//user-refresh-token",
		TokenType:    "Bearer",
	}

	_ = storage.Save("youtube", userToken)

	loadedToken, err := storage.Load("youtube")

	if err != nil {
		t.Fatalf("user should be able to reuse saved tokens without re-authenticating, got: %v", err)
	}
	if loadedToken.AccessToken != userToken.AccessToken {
		t.Error("loaded token should match saved token for API access")
	}
	if loadedToken.RefreshToken != userToken.RefreshToken {
		t.Error("refresh token should be persisted for token renewal")
	}
}

func TestAC102_TokenStorage_ReturnsErrorWhenUserNotAuthenticated(t *testing.T) {
	configDir, _ := os.MkdirTemp("", "oauth-test")
	defer os.RemoveAll(configDir)

	_, err := NewTokenStorage(configDir).Load("youtube")

	if err != ErrTokenNotFound {
		t.Errorf("should indicate user needs to authenticate first, got: %v", err)
	}
}

func TestAC103_OAuthConfig_RejectsInvalidConfiguration(t *testing.T) {
	invalidConfigs := []struct {
		name   string
		config Config
	}{
		{"missing client ID", Config{ClientID: "", ClientSecret: "secret", RedirectURL: "http://localhost", Scopes: []string{"read"}}},
		{"missing client secret", Config{ClientID: "id", ClientSecret: "", RedirectURL: "http://localhost", Scopes: []string{"read"}}},
		{"missing redirect URL", Config{ClientID: "id", ClientSecret: "secret", RedirectURL: "", Scopes: []string{"read"}}},
		{"missing scopes", Config{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://localhost", Scopes: nil}},
	}

	for _, tc := range invalidConfigs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if err == nil {
				t.Errorf("%s: user should get error message to fix OAuth setup", tc.name)
			}
		})
	}
}

func TestCallbackServer_ErrorIncludesPort(t *testing.T) {
	// Bind the port first so starting the callback server fails
	port := 18099
	blocker, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Skipf("port %d unavailable for test setup: %v", port, err)
	}
	defer blocker.Close()

	server := NewCallbackServer(port)
	_, err = server.WaitForCallback(context.Background(), "state", time.Second)

	if err == nil {
		t.Fatal("expected error when port is already in use")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%d", port)) {
		t.Errorf("error message should include the port number (%d) to help diagnose the conflict, got: %v", port, err)
	}
}

func TestAC103_OAuthConfig_AcceptsValidConfiguration(t *testing.T) {
	validConfig := Config{
		ClientID:     "valid-client-id",
		ClientSecret: "valid-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/youtube.readonly"},
	}

	err := validConfig.Validate()

	if err != nil {
		t.Errorf("valid OAuth configuration should allow user to authenticate, got: %v", err)
	}
}
