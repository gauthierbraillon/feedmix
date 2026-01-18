package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{"valid", Config{"id", "secret", "", "", "http://localhost", []string{"read"}}, false},
		{"no client ID", Config{"", "secret", "", "", "http://localhost", []string{"read"}}, true},
		{"no secret", Config{"id", "", "", "", "http://localhost", []string{"read"}}, true},
		{"no redirect", Config{"id", "secret", "", "", "", []string{"read"}}, true},
		{"no scopes", Config{"id", "secret", "", "", "http://localhost", nil}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFlow_GenerateAuthURL(t *testing.T) {
	config := YouTubeOAuthConfig("client-id", "secret", "http://localhost/callback")
	authURL, state := NewFlow(config).GenerateAuthURL()

	parsed, _ := url.Parse(authURL)
	if parsed.Host != "accounts.google.com" {
		t.Errorf("wrong host: %s", parsed.Host)
	}
	if state == "" {
		t.Error("state should not be empty")
	}
	if !strings.Contains(authURL, "client-id") {
		t.Error("should contain client_id")
	}
}

func TestFlow_ExchangeCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "token", "token_type": "Bearer", "expires_in": 3600,
		})
	}))
	defer server.Close()

	config := Config{
		ClientID: "id", ClientSecret: "secret",
		TokenURL: server.URL, RedirectURL: "http://localhost",
		Scopes: []string{"read"},
	}

	token, err := NewFlow(config).ExchangeCode(context.Background(), "code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.AccessToken != "token" {
		t.Errorf("wrong token: %s", token.AccessToken)
	}
}

func TestCallbackServer(t *testing.T) {
	server := NewCallbackServer(18085)
	go func() {
		time.Sleep(50 * time.Millisecond)
		resp, _ := http.Get("http://localhost:18085/callback?code=abc&state=test-state")
		if resp != nil {
			resp.Body.Close()
		}
	}()

	code, err := server.WaitForCallback(context.Background(), "test-state", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "abc" {
		t.Errorf("wrong code: %s", code)
	}
}

func TestCallbackServer_InvalidState(t *testing.T) {
	server := NewCallbackServer(18086)
	go func() {
		time.Sleep(50 * time.Millisecond)
		resp, _ := http.Get("http://localhost:18086/callback?code=abc&state=wrong")
		if resp != nil {
			resp.Body.Close()
		}
	}()

	_, err := server.WaitForCallback(context.Background(), "correct", 2*time.Second)
	if err != ErrInvalidState {
		t.Errorf("expected ErrInvalidState, got %v", err)
	}
}

func TestTokenStorage(t *testing.T) {
	dir, _ := os.MkdirTemp("", "oauth-test")
	defer os.RemoveAll(dir)

	storage := NewTokenStorage(dir)
	token := &Token{AccessToken: "test", TokenType: "Bearer"}

	if err := storage.Save("youtube", token); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := storage.Load("youtube")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.AccessToken != "test" {
		t.Errorf("wrong token: %s", loaded.AccessToken)
	}
}

func TestTokenStorage_NotFound(t *testing.T) {
	dir, _ := os.MkdirTemp("", "oauth-test")
	defer os.RemoveAll(dir)

	_, err := NewTokenStorage(dir).Load("nonexistent")
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}
