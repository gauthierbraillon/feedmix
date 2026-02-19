package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAC100_RefreshToken_ExchangesForAccessToken(t *testing.T) {
	mockTokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.FormValue("grant_type") != "refresh_token" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "ya29.fresh-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer mockTokenServer.Close()

	config := Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenURL:     mockTokenServer.URL,
	}

	token, err := NewFlow(config).RefreshAccessToken(context.Background(), "1//refresh-token")

	if err != nil {
		t.Fatalf("should exchange refresh token for access token without browser, got: %v", err)
	}
	if token.AccessToken == "" {
		t.Fatal("should receive fresh access token for YouTube API calls")
	}
}

func TestAC102_TokenStorage_PersistsTokensBetweenSessions(t *testing.T) {
	configDir, _ := os.MkdirTemp("", "oauth-test")
	defer func() { _ = os.RemoveAll(configDir) }()

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
	defer func() { _ = os.RemoveAll(configDir) }()

	_, err := NewTokenStorage(configDir).Load("youtube")

	if err != ErrTokenNotFound {
		t.Errorf("should indicate user needs to authenticate first, got: %v", err)
	}
}
