// Package oauth provides shared OAuth 2.0 utilities for feedmix.
//
// These tests serve as documentation for the OAuth package behavior:
// - Config validation ensures required fields are present
// - Auth URL generation creates properly formatted authorization URLs
// - Token exchange handles the OAuth callback flow
// - Token storage persists tokens securely to disk
//
// TDD Cycle: RED -> GREEN -> REFACTOR
// This file represents the RED phase - tests that define expected behavior.
package oauth

import (
	"os"
	"path/filepath"
	"testing"
)

// TestConfig_Validate documents config validation requirements:
// - ClientID must not be empty
// - ClientSecret must not be empty
// - RedirectURL must not be empty
// - Scopes must have at least one entry
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config passes validation",
			config: Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "http://localhost:8080/callback",
				Scopes:       []string{"read", "write"},
			},
			wantErr: false,
		},
		{
			name: "empty client ID fails validation",
			config: Config{
				ClientID:     "",
				ClientSecret: "test-client-secret",
				RedirectURL:  "http://localhost:8080/callback",
				Scopes:       []string{"read"},
			},
			wantErr: true,
			errMsg:  "client ID is required",
		},
		{
			name: "empty client secret fails validation",
			config: Config{
				ClientID:     "test-client-id",
				ClientSecret: "",
				RedirectURL:  "http://localhost:8080/callback",
				Scopes:       []string{"read"},
			},
			wantErr: true,
			errMsg:  "client secret is required",
		},
		{
			name: "empty redirect URL fails validation",
			config: Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "",
				Scopes:       []string{"read"},
			},
			wantErr: true,
			errMsg:  "redirect URL is required",
		},
		{
			name: "empty scopes fails validation",
			config: Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				RedirectURL:  "http://localhost:8080/callback",
				Scopes:       []string{},
			},
			wantErr: true,
			errMsg:  "at least one scope is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestFlow_GenerateAuthURL documents authorization URL generation:
// - URL contains the correct authorization endpoint
// - URL includes client_id parameter
// - URL includes redirect_uri parameter
// - URL includes scope parameter
// - URL includes state parameter for CSRF protection
func TestFlow_GenerateAuthURL(t *testing.T) {
	config := Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		AuthURL:      "https://example.com/oauth/authorize",
		TokenURL:     "https://example.com/oauth/token",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"read", "write"},
	}

	flow := NewFlow(config)
	url, state := flow.GenerateAuthURL()

	// URL should contain required OAuth parameters
	if url == "" {
		t.Error("auth URL should not be empty")
	}

	// State should be non-empty for CSRF protection
	if state == "" {
		t.Error("state should not be empty")
	}

	// URL should contain required components
	tests := []struct {
		name     string
		contains string
	}{
		{"auth endpoint", "https://example.com/oauth/authorize"},
		{"client_id", "client_id=test-client-id"},
		{"redirect_uri", "redirect_uri="},
		{"scope", "scope="},
		{"state", "state=" + state},
		{"response_type", "response_type=code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !containsString(url, tt.contains) {
				t.Errorf("URL should contain %q, got %q", tt.contains, url)
			}
		})
	}
}

// TestTokenStorage_SaveAndLoad documents token persistence:
// - Tokens can be saved to a file
// - Saved tokens can be loaded back
// - Loading non-existent token returns specific error
// - Token file has restricted permissions (0600)
func TestTokenStorage_SaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "feedmix-oauth-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewTokenStorage(tmpDir)
	provider := "youtube"

	// Test saving token
	token := &Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}

	err = storage.Save(provider, token)
	if err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Verify file permissions
	tokenPath := filepath.Join(tmpDir, provider+"_token.json")
	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Errorf("token file should exist: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("token file should have 0600 permissions, got %o", info.Mode().Perm())
	}

	// Test loading token
	loaded, err := storage.Load(provider)
	if err != nil {
		t.Errorf("failed to load token: %v", err)
	}

	if loaded.AccessToken != token.AccessToken {
		t.Errorf("access token mismatch: got %q, want %q", loaded.AccessToken, token.AccessToken)
	}
	if loaded.RefreshToken != token.RefreshToken {
		t.Errorf("refresh token mismatch: got %q, want %q", loaded.RefreshToken, token.RefreshToken)
	}
}

// TestTokenStorage_LoadNonExistent documents error handling:
// - Loading a token that doesn't exist returns ErrTokenNotFound
func TestTokenStorage_LoadNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "feedmix-oauth-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := NewTokenStorage(tmpDir)

	_, err = storage.Load("nonexistent")
	if err != ErrTokenNotFound {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
