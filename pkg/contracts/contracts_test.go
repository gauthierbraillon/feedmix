package contracts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gauthierbraillon/feedmix/internal/youtube"
	"github.com/gauthierbraillon/feedmix/pkg/oauth"
)

var YouTubeSubscriptionContract = `{
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

var OAuthTokenContract = `{
	"access_token": "ya29.test",
	"token_type": "Bearer",
	"expires_in": 3600,
	"refresh_token": "1//test"
}`

func TestYouTubeClient_ParsesContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(YouTubeSubscriptionContract))
	}))
	defer server.Close()

	token := &oauth.Token{AccessToken: "test", TokenType: "Bearer"}
	client := youtube.NewClient(token, youtube.WithBaseURL(server.URL))

	subs, err := client.FetchSubscriptions(context.Background())
	if err != nil {
		t.Fatalf("should parse contract: %v", err)
	}
	if len(subs) != 1 || subs[0].ChannelID != "UC123abc" {
		t.Errorf("unexpected result: %+v", subs)
	}
}

func TestOAuthFlow_ParsesContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(OAuthTokenContract))
	}))
	defer server.Close()

	config := oauth.Config{
		ClientID: "id", ClientSecret: "secret",
		TokenURL: server.URL, RedirectURL: "http://localhost/callback",
		Scopes: []string{"read"},
	}

	token, err := oauth.NewFlow(config).ExchangeCode(context.Background(), "code")
	if err != nil {
		t.Fatalf("should parse contract: %v", err)
	}
	if token.AccessToken != "ya29.test" {
		t.Errorf("unexpected token: %+v", token)
	}
}

func TestContracts_ValidJSON(t *testing.T) {
	contracts := map[string]string{
		"YouTube": YouTubeSubscriptionContract,
		"OAuth":   OAuthTokenContract,
	}
	for name, c := range contracts {
		var v interface{}
		if err := json.Unmarshal([]byte(c), &v); err != nil {
			t.Errorf("%s invalid JSON: %v", name, err)
		}
	}
}
