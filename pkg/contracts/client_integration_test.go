// Package contracts integration tests verify that actual clients
// correctly parse API responses matching the defined contracts.
package contracts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"feedmix/internal/linkedin"
	"feedmix/internal/youtube"
	"feedmix/pkg/oauth"
)

// TestYouTubeClient_ParsesContract verifies the YouTube client
// correctly parses responses matching the contract schema.
func TestYouTubeClient_ParsesContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(YouTubeSubscriptionContract))
	}))
	defer server.Close()

	token := &oauth.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	client := youtube.NewClient(token, youtube.WithBaseURL(server.URL))
	subs, err := client.FetchSubscriptions(context.Background())

	if err != nil {
		t.Fatalf("client should parse contract response: %v", err)
	}

	if len(subs) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(subs))
	}

	sub := subs[0]
	if sub.ChannelID != "UC123abc" {
		t.Errorf("expected channelId 'UC123abc', got %q", sub.ChannelID)
	}

	if sub.ChannelTitle != "Test Channel" {
		t.Errorf("expected title 'Test Channel', got %q", sub.ChannelTitle)
	}

	if sub.Description != "A test channel description" {
		t.Errorf("expected description 'A test channel description', got %q", sub.Description)
	}
}

// TestLinkedInClient_ParsesContract verifies the LinkedIn client
// correctly parses responses matching the contract schema.
func TestLinkedInClient_ParsesContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(LinkedInProfileContract))
	}))
	defer server.Close()

	token := &oauth.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	client := linkedin.NewClient(token, linkedin.WithBaseURL(server.URL))
	profile, err := client.FetchProfile(context.Background())

	if err != nil {
		t.Fatalf("client should parse contract response: %v", err)
	}

	if profile.FirstName != "John" {
		t.Errorf("expected firstName 'John', got %q", profile.FirstName)
	}

	if profile.LastName != "Doe" {
		t.Errorf("expected lastName 'Doe', got %q", profile.LastName)
	}

	if profile.ID != "urn:li:person:abc123" {
		t.Errorf("expected id 'urn:li:person:abc123', got %q", profile.ID)
	}
}

// TestOAuthFlow_ParsesContract verifies the OAuth flow
// correctly parses token responses matching the contract schema.
func TestOAuthFlow_ParsesContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(OAuthTokenContract))
	}))
	defer server.Close()

	config := oauth.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenURL:     server.URL,
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"read"},
	}

	flow := oauth.NewFlow(config)
	token, err := flow.ExchangeCode(context.Background(), "test-code")

	if err != nil {
		t.Fatalf("flow should parse contract response: %v", err)
	}

	if token.AccessToken != "ya29.a0AfH6SMBx..." {
		t.Errorf("expected access_token 'ya29.a0AfH6SMBx...', got %q", token.AccessToken)
	}

	if token.TokenType != "Bearer" {
		t.Errorf("expected token_type 'Bearer', got %q", token.TokenType)
	}

	if token.RefreshToken != "1//0e..." {
		t.Errorf("expected refresh_token '1//0e...', got %q", token.RefreshToken)
	}
}

// TestContractSchemas_AreValidJSON ensures all contract strings are valid JSON.
func TestContractSchemas_AreValidJSON(t *testing.T) {
	contracts := map[string]string{
		"YouTubeSubscription": YouTubeSubscriptionContract,
		"LinkedInProfile":     LinkedInProfileContract,
		"OAuthToken":          OAuthTokenContract,
		"OAuthError":          OAuthErrorContract,
	}

	for name, contract := range contracts {
		var v interface{}
		if err := json.Unmarshal([]byte(contract), &v); err != nil {
			t.Errorf("%s contract is not valid JSON: %v", name, err)
		}
	}
}
