// Package oauth provides OAuth 2.0 utilities for feedmix.
package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var ErrTokenNotFound = errors.New("token not found")

type Config struct {
	ClientID     string
	ClientSecret string // #nosec G117 - JSON field for OAuth config, not an exposed secret
	TokenURL     string
}

func YouTubeOAuthConfig(clientID, clientSecret string) Config {
	return Config{ // #nosec G101 -- OAuth URLs are public API endpoints, not hardcoded credentials
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     "https://oauth2.googleapis.com/token",
	}
}

type Token struct {
	AccessToken  string `json:"access_token"`  // #nosec G117 - JSON field for OAuth token, not an exposed secret
	RefreshToken string `json:"refresh_token"` // #nosec G117 - JSON field for OAuth token, not an exposed secret
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Flow struct {
	config     Config
	httpClient HTTPClient
}

type FlowOption func(*Flow)

func WithHTTPClient(client HTTPClient) FlowOption {
	return func(f *Flow) { f.httpClient = client }
}

func NewFlow(config Config, opts ...FlowOption) *Flow {
	f := &Flow{config: config, httpClient: http.DefaultClient}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *Flow) RefreshAccessToken(ctx context.Context, refreshToken string) (*Token, error) {
	data := url.Values{}
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", f.config.ClientID)
	data.Set("client_secret", f.config.ClientSecret)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: status %d", resp.StatusCode)
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &token, nil
}

type TokenStorage struct {
	dir string
}

func NewTokenStorage(dir string) *TokenStorage {
	return &TokenStorage{dir: dir}
}

func (s *TokenStorage) Save(provider string, token *Token) error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	cleanProvider := filepath.Base(provider)
	return os.WriteFile(filepath.Join(s.dir, cleanProvider+"_token.json"), data, 0600)
}

func (s *TokenStorage) Load(provider string) (*Token, error) {
	cleanProvider := filepath.Base(provider)
	data, err := os.ReadFile(filepath.Join(s.dir, cleanProvider+"_token.json")) // #nosec G304 -- provider is sanitized
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to read token: %w", err)
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}
