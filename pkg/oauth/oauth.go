// Package oauth provides OAuth 2.0 utilities for feedmix.
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

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrInvalidState  = errors.New("invalid state parameter")
)

type Config struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	RedirectURL  string
	Scopes       []string
}

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

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
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

func (f *Flow) GenerateAuthURL() (authURL string, state string) {
	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	state = hex.EncodeToString(stateBytes)

	params := url.Values{}
	params.Set("client_id", f.config.ClientID)
	params.Set("redirect_uri", f.config.RedirectURL)
	params.Set("scope", strings.Join(f.config.Scopes, " "))
	params.Set("state", state)
	params.Set("response_type", "code")
	params.Set("access_type", "offline")

	return fmt.Sprintf("%s?%s", f.config.AuthURL, params.Encode()), state
}

func (f *Flow) ExchangeCode(ctx context.Context, code string) (*Token, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", f.config.ClientID)
	data.Set("client_secret", f.config.ClientSecret)
	data.Set("redirect_uri", f.config.RedirectURL)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: status %d", resp.StatusCode)
	}

	var token Token
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &token, nil
}

type CallbackServer struct {
	port int
}

func NewCallbackServer(port int) *CallbackServer {
	return &CallbackServer{port: port}
}

func (s *CallbackServer) WaitForCallback(ctx context.Context, expectedState string, timeout time.Duration) (string, error) {
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != expectedState {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Invalid state"))
			errChan <- ErrInvalidState
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Missing code"))
			errChan <- errors.New("missing authorization code")
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Success! Close this window."))
		codeChan <- code
	})

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return "", fmt.Errorf("failed to start server: %w", err)
	}

	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
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

	return os.WriteFile(filepath.Join(s.dir, provider+"_token.json"), data, 0600)
}

func (s *TokenStorage) Load(provider string) (*Token, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, provider+"_token.json"))
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
