// Package oauth provides shared OAuth 2.0 utilities for feedmix.
//
// This package handles:
// - OAuth configuration validation
// - Authorization URL generation with CSRF protection
// - Token exchange and refresh
// - Secure token storage
package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ErrTokenNotFound is returned when a requested token doesn't exist.
var ErrTokenNotFound = errors.New("token not found")

// Config holds OAuth 2.0 configuration for a provider.
type Config struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	RedirectURL  string
	Scopes       []string
}

// Validate checks that all required configuration fields are present.
func (c Config) Validate() error {
	if c.ClientID == "" {
		return errors.New("client ID is required")
	}
	if c.ClientSecret == "" {
		return errors.New("client secret is required")
	}
	if c.RedirectURL == "" {
		return errors.New("redirect URL is required")
	}
	if len(c.Scopes) == 0 {
		return errors.New("at least one scope is required")
	}
	return nil
}

// Token represents an OAuth 2.0 token.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Flow handles the OAuth 2.0 authorization flow.
type Flow struct {
	config Config
}

// NewFlow creates a new OAuth flow with the given configuration.
func NewFlow(config Config) *Flow {
	return &Flow{config: config}
}

// GenerateAuthURL creates an authorization URL with CSRF state.
func (f *Flow) GenerateAuthURL() (authURL string, state string) {
	// Generate random state for CSRF protection
	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	state = hex.EncodeToString(stateBytes)

	// Build authorization URL
	params := url.Values{}
	params.Set("client_id", f.config.ClientID)
	params.Set("redirect_uri", f.config.RedirectURL)
	params.Set("scope", strings.Join(f.config.Scopes, " "))
	params.Set("state", state)
	params.Set("response_type", "code")

	authURL = fmt.Sprintf("%s?%s", f.config.AuthURL, params.Encode())
	return authURL, state
}

// ExchangeCode exchanges an authorization code for tokens.
func (f *Flow) ExchangeCode(code string) (*Token, error) {
	// TODO: Implement HTTP token exchange
	return nil, errors.New("not implemented")
}

// TokenStorage handles persistent storage of OAuth tokens.
type TokenStorage struct {
	dir string
}

// NewTokenStorage creates a token storage in the given directory.
func NewTokenStorage(dir string) *TokenStorage {
	return &TokenStorage{dir: dir}
}

// Save persists a token for the given provider.
func (s *TokenStorage) Save(provider string, token *Token) error {
	// Ensure directory exists
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	// Marshal token to JSON
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Write to file with restricted permissions
	tokenPath := filepath.Join(s.dir, provider+"_token.json")
	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// Load retrieves a token for the given provider.
func (s *TokenStorage) Load(provider string) (*Token, error) {
	tokenPath := filepath.Join(s.dir, provider+"_token.json")

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}
