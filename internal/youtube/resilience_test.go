package youtube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gauthierbraillon/feedmix/pkg/oauth"
)

func TestAC400_YouTubeAPI_IgnoresUnexpectedFields(t *testing.T) {
	mockResponse := map[string]interface{}{
		"kind": "youtube#subscriptionListResponse",
		"items": []map[string]interface{}{
			{
				"snippet": map[string]interface{}{
					"resourceId":        map[string]interface{}{"channelId": "UC123"},
					"title":             "Test Channel",
					"newFieldFromGoogle": "surprise feature!",
					"anotherNewField":   []string{"we", "added", "this"},
					"thumbnails":        map[string]interface{}{"default": map[string]interface{}{"url": "https://example.com/thumb.jpg"}},
					"publishedAt":       "2024-01-01T00:00:00Z",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	token := &oauth.Token{AccessToken: "test", TokenType: "Bearer"}
	client := NewClient(token, WithBaseURL(server.URL))

	subs, err := client.FetchSubscriptions(context.Background())

	if err != nil {
		t.Fatalf("user should see subscriptions even when YouTube adds new fields, got error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatal("user should see their subscription")
	}
	if subs[0].ChannelID != "UC123" {
		t.Error("user should see correct channel even with unexpected fields present")
	}
	if subs[0].ChannelTitle != "Test Channel" {
		t.Error("user should see channel title even with unexpected fields present")
	}
}

func TestAC401_YouTubeAPI_HandlesEmptyResponse(t *testing.T) {
	mockResponse := map[string]interface{}{
		"kind":  "youtube#subscriptionListResponse",
		"items": []map[string]interface{}{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	token := &oauth.Token{AccessToken: "test", TokenType: "Bearer"}
	client := NewClient(token, WithBaseURL(server.URL))

	subs, err := client.FetchSubscriptions(context.Background())

	if err != nil {
		t.Fatalf("user with no subscriptions should see empty feed, not error: %v", err)
	}
	if subs == nil {
		t.Fatal("should return empty slice, not nil")
	}
	if len(subs) != 0 {
		t.Errorf("user with no subscriptions should see 0 items, got %d", len(subs))
	}
}

func TestAC402_YouTubeAPI_HandlesMissingOptionalFields(t *testing.T) {
	mockResponse := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"snippet": map[string]interface{}{
					"resourceId":  map[string]interface{}{"channelId": "UC123"},
					"title":       "Minimal Channel",
					"publishedAt": "2024-01-01T00:00:00Z",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	token := &oauth.Token{AccessToken: "test", TokenType: "Bearer"}
	client := NewClient(token, WithBaseURL(server.URL))

	subs, err := client.FetchSubscriptions(context.Background())

	if err != nil {
		t.Fatalf("user should see subscription even without optional fields like thumbnail, got error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatal("user should see subscription with minimal data")
	}
	if subs[0].ChannelTitle != "Minimal Channel" {
		t.Error("user should see channel title from minimal response")
	}
}

func TestAC403_YouTubeAPI_ReturnsUserFriendlyErrorOnServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Service temporarily unavailable"))
	}))
	defer server.Close()

	token := &oauth.Token{AccessToken: "test", TokenType: "Bearer"}
	client := NewClient(token, WithBaseURL(server.URL))

	_, err := client.FetchSubscriptions(context.Background())

	if err == nil {
		t.Fatal("user should see error message when YouTube API is down")
	}
	errMsg := strings.ToLower(err.Error())
	if !strings.Contains(errMsg, "youtube") && !strings.Contains(errMsg, "api") {
		t.Errorf("error should mention YouTube API for user clarity, got: %v", err)
	}
}

func TestAC404_YouTubeAPI_ReturnsUserFriendlyErrorOnAuthFailure(t *testing.T) {
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

	token := &oauth.Token{AccessToken: "expired-token", TokenType: "Bearer"}
	client := NewClient(token, WithBaseURL(server.URL))

	_, err := client.FetchSubscriptions(context.Background())

	if err == nil {
		t.Fatal("user should see error when authentication fails")
	}
	errMsg := strings.ToLower(err.Error())
	hasAuthHint := strings.Contains(errMsg, "auth") ||
		strings.Contains(errMsg, "credential") ||
		strings.Contains(errMsg, "token") ||
		strings.Contains(errMsg, "login")
	if !hasAuthHint {
		t.Errorf("error should indicate authentication issue for user to re-authenticate, got: %v", err)
	}
}

func TestAC405_YouTubeAPI_HandlesRateLimitGracefully(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Header().Set("Retry-After", "60")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    429,
				"message": "Quota exceeded",
			},
		})
	}))
	defer server.Close()

	token := &oauth.Token{AccessToken: "test", TokenType: "Bearer"}
	client := NewClient(token, WithBaseURL(server.URL))

	_, err := client.FetchSubscriptions(context.Background())

	if err == nil {
		t.Fatal("user should see error when rate limit exceeded")
	}
	errMsg := strings.ToLower(err.Error())
	hasRateLimitHint := strings.Contains(errMsg, "quota") ||
		strings.Contains(errMsg, "rate") ||
		strings.Contains(errMsg, "limit") ||
		strings.Contains(errMsg, "many requests")
	if !hasRateLimitHint {
		t.Errorf("error should indicate rate limiting for user understanding, got: %v", err)
	}
}

func TestAC406_YouTubeAPI_HandlesMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"invalid": json}`))
	}))
	defer server.Close()

	token := &oauth.Token{AccessToken: "test", TokenType: "Bearer"}
	client := NewClient(token, WithBaseURL(server.URL))

	_, err := client.FetchSubscriptions(context.Background())

	if err == nil {
		t.Fatal("user should see error when YouTube returns malformed response")
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "panic") || strings.Contains(errMsg, "runtime error") {
		t.Error("error should be handled gracefully, not panic or crash")
	}
}

func TestAC407_YouTubeAPI_HandlesNullFields(t *testing.T) {
	mockResponse := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"snippet": map[string]interface{}{
					"resourceId":  map[string]interface{}{"channelId": "UC123"},
					"title":       "Test Channel",
					"description": nil,
					"thumbnails":  nil,
					"publishedAt": "2024-01-01T00:00:00Z",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	token := &oauth.Token{AccessToken: "test", TokenType: "Bearer"}
	client := NewClient(token, WithBaseURL(server.URL))

	subs, err := client.FetchSubscriptions(context.Background())

	if err != nil {
		t.Fatalf("user should see subscription even with null optional fields, got error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatal("user should see subscription despite null fields")
	}
	if subs[0].ChannelTitle != "Test Channel" {
		t.Error("user should see channel title when other fields are null")
	}
}

func TestAC408_YouTubeAPI_HandlesPartialResponseDuringNetworkIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items": [{"snippet": {"resourceId": {"channelId": "UC123"}, "title": "Test`))
	}))
	defer server.Close()

	token := &oauth.Token{AccessToken: "test", TokenType: "Bearer"}
	client := NewClient(token, WithBaseURL(server.URL))

	_, err := client.FetchSubscriptions(context.Background())

	if err == nil {
		t.Fatal("user should see error when response is incomplete due to network issue")
	}
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "panic") {
		t.Error("partial response should be handled gracefully without panic")
	}
}
