package contracts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gauthierbraillon/feedmix/internal/youtube"
	"github.com/gauthierbraillon/feedmix/pkg/oauth"
)

// TestYouTubeSubscriptionResponse_MatchesGoogleContract validates that our
// mock response matches the actual YouTube Data API v3 schema from Google's
// Discovery Service.
func TestYouTubeSubscriptionResponse_MatchesGoogleContract(t *testing.T) {
	// Load the real Google Discovery document
	discoveryPath := filepath.Join("youtube-discovery.json")
	discoveryData, err := os.ReadFile(discoveryPath)
	if err != nil {
		t.Fatalf("failed to read discovery document: %v", err)
	}

	var discovery map[string]interface{}
	if err := json.Unmarshal(discoveryData, &discovery); err != nil {
		t.Fatalf("failed to parse discovery document: %v", err)
	}

	// Extract the SubscriptionListResponse schema
	schemas := discovery["schemas"].(map[string]interface{})
	listResponseSchema := schemas["SubscriptionListResponse"].(map[string]interface{})
	subscriptionSchema := schemas["Subscription"].(map[string]interface{})
	snippetSchema := schemas["SubscriptionSnippet"].(map[string]interface{})

	// Verify schema structure exists
	if listResponseSchema["properties"] == nil {
		t.Fatal("SubscriptionListResponse schema missing properties")
	}
	if subscriptionSchema["properties"] == nil {
		t.Fatal("Subscription schema missing properties")
	}
	if snippetSchema["properties"] == nil {
		t.Fatal("SubscriptionSnippet schema missing properties")
	}

	// Our mock response
	mockResponse := map[string]interface{}{
		"kind": "youtube#subscriptionListResponse",
		"items": []map[string]interface{}{
			{
				"snippet": map[string]interface{}{
					"publishedAt": "2024-01-15T10:00:00Z",
					"title":       "Test Channel",
					"description": "Test description",
					"resourceId":  map[string]interface{}{"channelId": "UC123abc"},
					"thumbnails":  map[string]interface{}{"default": map[string]interface{}{"url": "https://example.com/thumb.jpg"}},
				},
			},
		},
	}

	// Validate kind field matches schema
	listProps := listResponseSchema["properties"].(map[string]interface{})
	kindSchema := listProps["kind"].(map[string]interface{})
	expectedKind := kindSchema["default"].(string)
	if mockResponse["kind"] != expectedKind {
		t.Errorf("kind mismatch: got %q, schema expects %q", mockResponse["kind"], expectedKind)
	}

	// Validate snippet fields exist in schema
	snippetProps := snippetSchema["properties"].(map[string]interface{})
	requiredFields := []string{"publishedAt", "title", "description", "resourceId", "thumbnails"}
	for _, field := range requiredFields {
		if _, exists := snippetProps[field]; !exists {
			t.Errorf("mock uses field %q but it's not in Google's schema", field)
		}
	}

	// Validate publishedAt format
	publishedAtSchema := snippetProps["publishedAt"].(map[string]interface{})
	if format, ok := publishedAtSchema["format"]; ok && format != "date-time" {
		t.Errorf("publishedAt format should be date-time, got %v", format)
	}
}

// TestOAuthTokenResponse_MatchesRFC6749 validates that our OAuth token
// response follows RFC 6749 (OAuth 2.0) specification.
func TestOAuthTokenResponse_MatchesRFC6749(t *testing.T) {
	mockToken := map[string]interface{}{
		"access_token":  "ya29.test",
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": "1//test",
	}

	// RFC 6749 Section 5.1 requires these fields
	requiredFields := []string{"access_token", "token_type"}
	for _, field := range requiredFields {
		if _, exists := mockToken[field]; !exists {
			t.Errorf("OAuth token missing required field: %s (RFC 6749)", field)
		}
	}

	// token_type must be "Bearer" for Google OAuth
	if mockToken["token_type"] != "Bearer" {
		t.Errorf("token_type should be Bearer, got %v", mockToken["token_type"])
	}

	// expires_in should be numeric
	if _, ok := mockToken["expires_in"].(int); !ok {
		t.Errorf("expires_in should be numeric")
	}
}

// Integration test: Verify client can parse YouTube API response
func TestYouTubeClient_ParsesAPIResponse(t *testing.T) {
	mockResponse := `{
		"kind": "youtube#subscriptionListResponse",
		"items": [{
			"snippet": {
				"publishedAt": "2024-01-15T10:00:00Z",
				"title": "Test Channel",
				"description": "Test description",
				"resourceId": {"channelId": "UC123abc"},
				"thumbnails": {"default": {"url": "https://example.com/thumb.jpg"}}
			}
		}]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	token := &oauth.Token{AccessToken: "test", TokenType: "Bearer"}
	client := youtube.NewClient(token, youtube.WithBaseURL(server.URL))

	subs, err := client.FetchSubscriptions(context.Background())
	if err != nil {
		t.Fatalf("should parse response: %v", err)
	}
	if len(subs) != 1 || subs[0].ChannelID != "UC123abc" {
		t.Errorf("unexpected result: %+v", subs)
	}
}

// Integration test: Verify OAuth flow can parse token response
func TestOAuthFlow_ParsesTokenResponse(t *testing.T) {
	mockResponse := `{
		"access_token": "ya29.test",
		"token_type": "Bearer",
		"expires_in": 3600,
		"refresh_token": "1//test"
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	config := oauth.Config{
		ClientID: "id", ClientSecret: "secret",
		TokenURL: server.URL,
	}

	token, err := oauth.NewFlow(config).RefreshAccessToken(context.Background(), "1//test-refresh")
	if err != nil {
		t.Fatalf("should parse response: %v", err)
	}
	if token.AccessToken != "ya29.test" {
		t.Errorf("unexpected token: %+v", token)
	}
}

// Validation test: Ensure contracts are valid JSON
func TestContracts_ValidJSON(t *testing.T) {
	discoveryPath := filepath.Join("youtube-discovery.json")
	data, err := os.ReadFile(discoveryPath)
	if err != nil {
		t.Fatalf("failed to read discovery document: %v", err)
	}

	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Errorf("discovery document is invalid JSON: %v", err)
	}
}
