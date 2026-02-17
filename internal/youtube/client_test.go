// Package youtube tests document the expected behavior of the YouTube client.
//
// TDD Cycle: RED -> GREEN -> REFACTOR
//
// Test requirements (this file serves as documentation):
// - Client validates configuration on creation
// - Client fetches user subscriptions from YouTube API
// - Client fetches recent videos from subscribed channels
// - Client fetches user's liked videos
// - Client handles API errors gracefully
// - Client respects rate limits
package youtube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gauthierbraillon/feedmix/pkg/oauth"
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

// TestClient_FetchSubscriptions documents subscription fetching:
// - Returns list of subscribed channels
// - Each subscription has channel ID, title, and thumbnail
// - Handles pagination for users with many subscriptions
func TestClient_FetchSubscriptions(t *testing.T) {
	// Mock YouTube API response
	mockResponse := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"snippet": map[string]interface{}{
					"resourceId": map[string]interface{}{
						"channelId": "UC123",
					},
					"title":       "Test Channel",
					"description": "A test channel",
					"thumbnails": map[string]interface{}{
						"default": map[string]interface{}{
							"url": "https://example.com/thumb.jpg",
						},
					},
					"publishedAt": "2024-01-01T00:00:00Z",
				},
			},
		},
		"pageInfo": map[string]interface{}{
			"totalResults": 1,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-access-token" {
			t.Errorf("expected Bearer token in Authorization header, got %q", auth)
		}

		// Verify endpoint
		if r.URL.Path != "/youtube/v3/subscriptions" {
			t.Errorf("expected /youtube/v3/subscriptions, got %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	token := &oauth.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
	}

	client := NewClient(token, WithBaseURL(server.URL))

	ctx := context.Background()
	subs, err := client.FetchSubscriptions(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(subs) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(subs))
	}

	if subs[0].ChannelID != "UC123" {
		t.Errorf("expected channel ID UC123, got %q", subs[0].ChannelID)
	}

	if subs[0].ChannelTitle != "Test Channel" {
		t.Errorf("expected channel title 'Test Channel', got %q", subs[0].ChannelTitle)
	}
}

// TestClient_FetchRecentVideos documents recent video fetching:
// - Takes a channel ID and returns recent videos
// - Videos are sorted by publish date (newest first)
// - Includes video metadata (title, description, view count, etc.)
func TestClient_FetchRecentVideos(t *testing.T) {
	// Mock YouTube API search response
	searchResponse := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"id": map[string]interface{}{
					"videoId": "video123",
				},
				"snippet": map[string]interface{}{
					"title":       "Test Video",
					"description": "A test video",
					"channelId":   "UC123",
					"channelTitle": "Test Channel",
					"publishedAt": "2024-01-15T12:00:00Z",
					"thumbnails": map[string]interface{}{
						"default": map[string]interface{}{
							"url": "https://example.com/video-thumb.jpg",
						},
					},
				},
			},
		},
	}

	// Mock video details response
	videoResponse := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"id": "video123",
				"statistics": map[string]interface{}{
					"viewCount": "1000",
					"likeCount": "50",
				},
				"contentDetails": map[string]interface{}{
					"duration": "PT10M30S",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/youtube/v3/search" {
			_ = json.NewEncoder(w).Encode(searchResponse)
		} else if r.URL.Path == "/youtube/v3/videos" {
			_ = json.NewEncoder(w).Encode(videoResponse)
		}
	}))
	defer server.Close()

	token := &oauth.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
	}

	client := NewClient(token, WithBaseURL(server.URL))

	ctx := context.Background()
	videos, err := client.FetchRecentVideos(ctx, "UC123", 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(videos) != 1 {
		t.Fatalf("expected 1 video, got %d", len(videos))
	}

	if videos[0].ID != "video123" {
		t.Errorf("expected video ID video123, got %q", videos[0].ID)
	}

	if videos[0].Title != "Test Video" {
		t.Errorf("expected title 'Test Video', got %q", videos[0].Title)
	}

	if videos[0].ViewCount != 1000 {
		t.Errorf("expected view count 1000, got %d", videos[0].ViewCount)
	}
}

// TestClient_FetchLikedVideos documents liked video fetching:
// - Returns videos the authenticated user has liked
// - Includes the time the video was liked
// - Handles pagination for users with many liked videos
func TestClient_FetchLikedVideos(t *testing.T) {
	mockResponse := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"snippet": map[string]interface{}{
					"resourceId": map[string]interface{}{
						"videoId": "liked123",
					},
					"title":       "Liked Video",
					"description": "A liked video",
					"channelId":   "UC456",
					"channelTitle": "Another Channel",
					"publishedAt": "2024-01-10T08:00:00Z",
					"thumbnails": map[string]interface{}{
						"default": map[string]interface{}{
							"url": "https://example.com/liked-thumb.jpg",
						},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should call playlistItems with likes playlist
		if r.URL.Path != "/youtube/v3/playlistItems" {
			t.Errorf("expected /youtube/v3/playlistItems, got %q", r.URL.Path)
		}

		playlistID := r.URL.Query().Get("playlistId")
		if playlistID != "LL" {
			t.Errorf("expected playlistId=LL, got %q", playlistID)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	token := &oauth.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
	}

	client := NewClient(token, WithBaseURL(server.URL))

	ctx := context.Background()
	videos, err := client.FetchLikedVideos(ctx, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(videos) != 1 {
		t.Fatalf("expected 1 liked video, got %d", len(videos))
	}

	if videos[0].ID != "liked123" {
		t.Errorf("expected video ID liked123, got %q", videos[0].ID)
	}
}

// TestClient_APIError documents error handling:
// - Returns meaningful error on API failure
// - Includes HTTP status code in error
func TestClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    401,
				"message": "Invalid credentials",
			},
		})
	}))
	defer server.Close()

	token := &oauth.Token{
		AccessToken: "invalid-token",
		TokenType:   "Bearer",
	}

	client := NewClient(token, WithBaseURL(server.URL))

	ctx := context.Background()
	_, err := client.FetchSubscriptions(ctx)

	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}
}

// TestClient_Timeout documents timeout handling:
// - Respects context deadline
// - Returns context error on timeout
func TestClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
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

	_, err := client.FetchSubscriptions(ctx)

	if err == nil {
		t.Fatal("expected timeout error")
	}

	if ctx.Err() != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got %v", ctx.Err())
	}
}

func TestClient_FetchRecentVideos_URLEncodesChannelID(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": []interface{}{}})
	}))
	defer server.Close()

	token := &oauth.Token{AccessToken: "test-token", TokenType: "Bearer"}
	client := NewClient(token, WithBaseURL(server.URL))

	// Channel ID with characters that require URL encoding
	_, _ = client.FetchRecentVideos(context.Background(), "UC+special/id", 5)

	if strings.Contains(capturedURL, "UC+special/id") {
		t.Error("channel ID must be URL-encoded in the query string to prevent parameter injection")
	}
}
