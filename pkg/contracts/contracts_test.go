// Package contracts provides contract testing for external API integrations.
//
// Contract tests verify that our client code correctly handles the expected
// API response format. These tests serve as executable documentation of the
// API contracts we depend on.
//
// To verify contracts against real APIs, run: go test -tags=integration ./pkg/contracts/...
package contracts

import (
	"encoding/json"
	"testing"
	"time"
)

// YouTubeSubscriptionResponse represents the expected YouTube API response.
// API Reference: https://developers.google.com/youtube/v3/docs/subscriptions/list
type YouTubeSubscriptionResponse struct {
	Kind     string `json:"kind"`
	Etag     string `json:"etag"`
	PageInfo struct {
		TotalResults   int `json:"totalResults"`
		ResultsPerPage int `json:"resultsPerPage"`
	} `json:"pageInfo"`
	Items []struct {
		Kind    string `json:"kind"`
		Etag    string `json:"etag"`
		ID      string `json:"id"`
		Snippet struct {
			PublishedAt time.Time `json:"publishedAt"`
			Title       string    `json:"title"`
			Description string    `json:"description"`
			ResourceID  struct {
				Kind      string `json:"kind"`
				ChannelID string `json:"channelId"`
			} `json:"resourceId"`
			ChannelID  string `json:"channelId"`
			Thumbnails struct {
				Default struct {
					URL string `json:"url"`
				} `json:"default"`
				Medium struct {
					URL string `json:"url"`
				} `json:"medium"`
				High struct {
					URL string `json:"url"`
				} `json:"high"`
			} `json:"thumbnails"`
		} `json:"snippet"`
	} `json:"items"`
}

// YouTubeSubscriptionContract is the expected API response format.
var YouTubeSubscriptionContract = `{
	"kind": "youtube#subscriptionListResponse",
	"etag": "test-etag",
	"pageInfo": {
		"totalResults": 1,
		"resultsPerPage": 25
	},
	"items": [
		{
			"kind": "youtube#subscription",
			"etag": "item-etag",
			"id": "sub-123",
			"snippet": {
				"publishedAt": "2024-01-15T10:00:00Z",
				"title": "Test Channel",
				"description": "A test channel description",
				"resourceId": {
					"kind": "youtube#channel",
					"channelId": "UC123abc"
				},
				"channelId": "owner-channel-id",
				"thumbnails": {
					"default": {"url": "https://example.com/thumb_default.jpg"},
					"medium": {"url": "https://example.com/thumb_medium.jpg"},
					"high": {"url": "https://example.com/thumb_high.jpg"}
				}
			}
		}
	]
}`

// TestYouTubeSubscriptionContract verifies our parsing handles the expected format.
func TestYouTubeSubscriptionContract(t *testing.T) {
	var response YouTubeSubscriptionResponse
	if err := json.Unmarshal([]byte(YouTubeSubscriptionContract), &response); err != nil {
		t.Fatalf("contract JSON should be valid: %v", err)
	}

	// Verify required fields are present
	if response.Kind != "youtube#subscriptionListResponse" {
		t.Errorf("expected kind 'youtube#subscriptionListResponse', got %q", response.Kind)
	}

	if len(response.Items) == 0 {
		t.Fatal("expected at least one item")
	}

	item := response.Items[0]
	if item.Snippet.ResourceID.ChannelID == "" {
		t.Error("channelId should be present in resourceId")
	}

	if item.Snippet.Title == "" {
		t.Error("title should be present in snippet")
	}
}

// LinkedInProfileResponse represents the expected LinkedIn API response.
// API Reference: https://learn.microsoft.com/en-us/linkedin/shared/integrations/people/profile-api
type LinkedInProfileResponse struct {
	ID        string `json:"id"`
	FirstName struct {
		Localized       map[string]string `json:"localized"`
		PreferredLocale struct {
			Country  string `json:"country"`
			Language string `json:"language"`
		} `json:"preferredLocale"`
	} `json:"firstName"`
	LastName struct {
		Localized       map[string]string `json:"localized"`
		PreferredLocale struct {
			Country  string `json:"country"`
			Language string `json:"language"`
		} `json:"preferredLocale"`
	} `json:"lastName"`
	Headline struct {
		Localized       map[string]string `json:"localized"`
		PreferredLocale struct {
			Country  string `json:"country"`
			Language string `json:"language"`
		} `json:"preferredLocale"`
	} `json:"headline"`
}

// LinkedInProfileContract is the expected API response format.
var LinkedInProfileContract = `{
	"id": "urn:li:person:abc123",
	"firstName": {
		"localized": {
			"en_US": "John"
		},
		"preferredLocale": {
			"country": "US",
			"language": "en"
		}
	},
	"lastName": {
		"localized": {
			"en_US": "Doe"
		},
		"preferredLocale": {
			"country": "US",
			"language": "en"
		}
	},
	"headline": {
		"localized": {
			"en_US": "Software Engineer"
		},
		"preferredLocale": {
			"country": "US",
			"language": "en"
		}
	}
}`

// TestLinkedInProfileContract verifies our parsing handles the expected format.
func TestLinkedInProfileContract(t *testing.T) {
	var response LinkedInProfileResponse
	if err := json.Unmarshal([]byte(LinkedInProfileContract), &response); err != nil {
		t.Fatalf("contract JSON should be valid: %v", err)
	}

	// Verify required fields are present
	if response.ID == "" {
		t.Error("id should be present")
	}

	if response.FirstName.Localized["en_US"] != "John" {
		t.Errorf("expected firstName 'John', got %q", response.FirstName.Localized["en_US"])
	}

	if response.LastName.Localized["en_US"] != "Doe" {
		t.Errorf("expected lastName 'Doe', got %q", response.LastName.Localized["en_US"])
	}
}

// OAuthTokenResponse represents the expected OAuth token response.
// Standard: https://datatracker.ietf.org/doc/html/rfc6749#section-5.1
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// OAuthTokenContract is the expected token response format.
var OAuthTokenContract = `{
	"access_token": "ya29.a0AfH6SMBx...",
	"token_type": "Bearer",
	"expires_in": 3600,
	"refresh_token": "1//0e...",
	"scope": "https://www.googleapis.com/auth/youtube.readonly"
}`

// TestOAuthTokenContract verifies our parsing handles the expected format.
func TestOAuthTokenContract(t *testing.T) {
	var response OAuthTokenResponse
	if err := json.Unmarshal([]byte(OAuthTokenContract), &response); err != nil {
		t.Fatalf("contract JSON should be valid: %v", err)
	}

	if response.AccessToken == "" {
		t.Error("access_token should be present")
	}

	if response.TokenType != "Bearer" {
		t.Errorf("expected token_type 'Bearer', got %q", response.TokenType)
	}

	if response.ExpiresIn <= 0 {
		t.Error("expires_in should be positive")
	}
}

// OAuthErrorResponse represents the expected OAuth error response.
// Standard: https://datatracker.ietf.org/doc/html/rfc6749#section-5.2
type OAuthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// OAuthErrorContract is the expected error response format.
var OAuthErrorContract = `{
	"error": "invalid_grant",
	"error_description": "Token has been expired or revoked."
}`

// TestOAuthErrorContract verifies our parsing handles error responses.
func TestOAuthErrorContract(t *testing.T) {
	var response OAuthErrorResponse
	if err := json.Unmarshal([]byte(OAuthErrorContract), &response); err != nil {
		t.Fatalf("contract JSON should be valid: %v", err)
	}

	if response.Error == "" {
		t.Error("error should be present")
	}

	if response.Error != "invalid_grant" {
		t.Errorf("expected error 'invalid_grant', got %q", response.Error)
	}
}
