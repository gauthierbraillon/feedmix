// Package linkedin tests document the expected behavior of the LinkedIn client.
//
// TDD Cycle: RED -> GREEN -> REFACTOR
//
// Test requirements (this file serves as documentation):
// - Client validates configuration on creation
// - Client fetches authenticated user's profile
// - Client fetches feed posts (requires Marketing Developer Platform)
// - Client handles API errors gracefully
// - Client respects rate limits
//
// NOTE: LinkedIn API is restrictive. Many features require special access.
// Tests are designed for the ideal API behavior.
package linkedin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"feedmix/pkg/oauth"
)

// TestNewClient documents client creation requirements:
// - Valid OAuth token is required
// - Returns configured client ready to make API calls
func TestNewClient(t *testing.T) {
	token := &oauth.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
	}

	client := NewClient(token)

	if client == nil {
		t.Fatal("client should not be nil")
	}
}

// TestClient_FetchProfile documents profile fetching:
// - Returns authenticated user's profile
// - Profile includes name, headline, and picture
func TestClient_FetchProfile(t *testing.T) {
	mockResponse := map[string]interface{}{
		"id":        "urn:li:person:abc123",
		"firstName": map[string]interface{}{
			"localized": map[string]interface{}{
				"en_US": "John",
			},
			"preferredLocale": map[string]interface{}{
				"country":  "US",
				"language": "en",
			},
		},
		"lastName": map[string]interface{}{
			"localized": map[string]interface{}{
				"en_US": "Doe",
			},
			"preferredLocale": map[string]interface{}{
				"country":  "US",
				"language": "en",
			},
		},
		"headline": map[string]interface{}{
			"localized": map[string]interface{}{
				"en_US": "Software Engineer",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-access-token" {
			t.Errorf("expected Bearer token in Authorization header, got %q", auth)
		}

		// Verify endpoint
		if r.URL.Path != "/v2/me" {
			t.Errorf("expected /v2/me, got %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	token := &oauth.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
	}

	client := NewClient(token, WithBaseURL(server.URL))

	ctx := context.Background()
	profile, err := client.FetchProfile(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if profile.FirstName != "John" {
		t.Errorf("expected first name John, got %q", profile.FirstName)
	}

	if profile.LastName != "Doe" {
		t.Errorf("expected last name Doe, got %q", profile.LastName)
	}
}

// TestClient_FetchFeed documents feed fetching:
// - Returns posts from user's feed
// - Posts include author info, text, engagement stats
// NOTE: Requires Marketing Developer Platform access
func TestClient_FetchFeed(t *testing.T) {
	mockResponse := map[string]interface{}{
		"elements": []map[string]interface{}{
			{
				"id":     "urn:li:share:123",
				"author": "urn:li:person:author1",
				"text": map[string]interface{}{
					"text": "This is a test post",
				},
				"created": map[string]interface{}{
					"time": 1704067200000, // 2024-01-01 00:00:00 UTC
				},
				"socialDetail": map[string]interface{}{
					"totalSocialActivityCounts": map[string]interface{}{
						"numLikes":    10,
						"numComments": 5,
						"numShares":   2,
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/feed" {
			t.Errorf("expected /v2/feed, got %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	token := &oauth.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
	}

	client := NewClient(token, WithBaseURL(server.URL))

	ctx := context.Background()
	posts, err := client.FetchFeed(ctx, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	if posts[0].Text != "This is a test post" {
		t.Errorf("expected text 'This is a test post', got %q", posts[0].Text)
	}

	if posts[0].LikeCount != 10 {
		t.Errorf("expected 10 likes, got %d", posts[0].LikeCount)
	}
}

// TestClient_FetchReactions documents reaction fetching:
// - Returns user's reactions (likes, etc.)
// - Includes reaction type and timestamp
func TestClient_FetchReactions(t *testing.T) {
	mockResponse := map[string]interface{}{
		"elements": []map[string]interface{}{
			{
				"actor":        "urn:li:person:user1",
				"object":       "urn:li:share:post123",
				"reactionType": "LIKE",
				"created":      1704153600000, // 2024-01-02 00:00:00 UTC
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/reactions" {
			t.Errorf("expected /v2/reactions, got %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	token := &oauth.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
	}

	client := NewClient(token, WithBaseURL(server.URL))

	ctx := context.Background()
	reactions, err := client.FetchReactions(ctx, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reactions) != 1 {
		t.Fatalf("expected 1 reaction, got %d", len(reactions))
	}

	if reactions[0].ReactionType != "LIKE" {
		t.Errorf("expected LIKE reaction, got %q", reactions[0].ReactionType)
	}

	if reactions[0].PostID != "urn:li:share:post123" {
		t.Errorf("expected post ID urn:li:share:post123, got %q", reactions[0].PostID)
	}
}

// TestClient_APIError documents error handling:
// - Returns meaningful error on API failure
// - Includes HTTP status code in error
func TestClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Invalid access token",
		})
	}))
	defer server.Close()

	token := &oauth.Token{
		AccessToken: "invalid-token",
		TokenType:   "Bearer",
	}

	client := NewClient(token, WithBaseURL(server.URL))

	ctx := context.Background()
	_, err := client.FetchProfile(ctx)

	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}
}

// TestClient_Timeout documents timeout handling:
// - Respects context deadline
// - Returns context error on timeout
func TestClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	token := &oauth.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	client := NewClient(token, WithBaseURL(server.URL))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.FetchProfile(ctx)

	if err == nil {
		t.Fatal("expected timeout error")
	}

	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", ctx.Err())
	}
}
