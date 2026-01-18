// Package oauth provides shared OAuth 2.0 utilities for feedmix.
//
// This package handles:
// - OAuth configuration validation
// - Authorization URL generation with CSRF protection
// - Token exchange and refresh
// - Secure token storage
// - Browser-based OAuth callback handling
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrTokenNotFound is returned when a requested token doesn't exist.
var ErrTokenNotFound = errors.New("token not found")

// ErrInvalidState is returned when OAuth state doesn't match.
var ErrInvalidState = errors.New("invalid state parameter")

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

// YouTubeOAuthConfig returns OAuth config for YouTube API.
func YouTubeOAuthConfig(clientID, clientSecret, redirectURL string) Config {
	return Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		RedirectURL:  redirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/youtube.readonly"},
	}
}

// LinkedInOAuthConfig returns OAuth config for LinkedIn API.
func LinkedInOAuthConfig(clientID, clientSecret, redirectURL string) Config {
	return Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://www.linkedin.com/oauth/v2/authorization",
		TokenURL:     "https://www.linkedin.com/oauth/v2/accessToken",
		RedirectURL:  redirectURL,
		Scopes:       []string{"r_liteprofile", "r_emailaddress"},
	}
}

// Token represents an OAuth 2.0 token.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// HTTPClient interface for making HTTP requests (allows testing).
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Flow handles the OAuth 2.0 authorization flow.
type Flow struct {
	config     Config
	httpClient HTTPClient
}

// FlowOption configures the Flow.
type FlowOption func(*Flow)

// WithHTTPClient sets a custom HTTP client for the flow.
func WithHTTPClient(client HTTPClient) FlowOption {
	return func(f *Flow) {
		f.httpClient = client
	}
}

// NewFlow creates a new OAuth flow with the given configuration.
func NewFlow(config Config, opts ...FlowOption) *Flow {
	f := &Flow{
		config:     config,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// GenerateAuthURL creates an authorization URL with CSRF state.
func (f *Flow) GenerateAuthURL() (authURL string, state string) {
	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	state = hex.EncodeToString(stateBytes)

	params := url.Values{}
	params.Set("client_id", f.config.ClientID)
	params.Set("redirect_uri", f.config.RedirectURL)
	params.Set("scope", strings.Join(f.config.Scopes, " "))
	params.Set("state", state)
	params.Set("response_type", "code")
	params.Set("access_type", "offline")

	authURL = fmt.Sprintf("%s?%s", f.config.AuthURL, params.Encode())
	return authURL, state
}

// ExchangeCode exchanges an authorization code for tokens.
func (f *Flow) ExchangeCode(ctx context.Context, code string) (*Token, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", f.config.ClientID)
	data.Set("client_secret", f.config.ClientSecret)
	data.Set("redirect_uri", f.config.RedirectURL)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: status %d", resp.StatusCode)
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &token, nil
}

// CallbackServer handles OAuth callback requests.
type CallbackServer struct {
	port int
}

// NewCallbackServer creates a callback server on the specified port.
func NewCallbackServer(port int) *CallbackServer {
	return &CallbackServer{port: port}
}

// WaitForCallback starts the server and waits for OAuth callback.
func (s *CallbackServer) WaitForCallback(ctx context.Context, expectedState string, timeout time.Duration) (string, error) {
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if state != expectedState {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "Invalid state parameter")
			errChan <- ErrInvalidState
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "Missing authorization code")
			errChan <- errors.New("missing authorization code")
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Authorization successful! You can close this window.")
		codeChan <- code
	})

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return "", fmt.Errorf("failed to start callback server: %w", err)
	}

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case code := <-codeChan:
		return code, nil
	case err := <-errChan:
		return "", err
	case <-timeoutCtx.Done():
		return "", timeoutCtx.Err()
	}
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
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

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
