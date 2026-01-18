// Package linkedin provides a client for the LinkedIn API.
//
// NOTE: LinkedIn API has significant restrictions.
// Most feed-related endpoints require Marketing Developer Platform access.
package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"feedmix/pkg/oauth"
)

// Client is a LinkedIn API client.
type Client struct {
	token      *oauth.Token
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new LinkedIn API client with the given OAuth token.
func NewClient(token *oauth.Token) *Client {
	return &Client{
		token:      token,
		baseURL:    "https://api.linkedin.com",
		httpClient: &http.Client{},
	}
}

// SetBaseURL sets the base URL for API requests (used for testing).
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// FetchProfile retrieves the authenticated user's profile.
func (c *Client) FetchProfile(ctx context.Context) (*Profile, error) {
	url := fmt.Sprintf("%s/v2/me", c.baseURL)

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response profileResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse profile response: %w", err)
	}

	profile := &Profile{
		ID:        response.ID,
		FirstName: getLocalizedValue(response.FirstName),
		LastName:  getLocalizedValue(response.LastName),
		Headline:  getLocalizedValue(response.Headline),
	}

	return profile, nil
}

// FetchFeed retrieves posts from the user's feed.
// NOTE: Requires Marketing Developer Platform access.
func (c *Client) FetchFeed(ctx context.Context, limit int) ([]Post, error) {
	url := fmt.Sprintf("%s/v2/feed?count=%d", c.baseURL, limit)

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response feedResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse feed response: %w", err)
	}

	posts := make([]Post, 0, len(response.Elements))
	for _, elem := range response.Elements {
		publishedAt := time.UnixMilli(elem.Created.Time)

		post := Post{
			ID:           elem.ID,
			AuthorID:     elem.Author,
			Text:         elem.Text.Text,
			LikeCount:    elem.SocialDetail.TotalSocialActivityCounts.NumLikes,
			CommentCount: elem.SocialDetail.TotalSocialActivityCounts.NumComments,
			ShareCount:   elem.SocialDetail.TotalSocialActivityCounts.NumShares,
			PublishedAt:  publishedAt,
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// FetchReactions retrieves the user's reactions (likes, etc.).
func (c *Client) FetchReactions(ctx context.Context, limit int) ([]Reaction, error) {
	url := fmt.Sprintf("%s/v2/reactions?count=%d", c.baseURL, limit)

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response reactionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse reactions response: %w", err)
	}

	reactions := make([]Reaction, 0, len(response.Elements))
	for _, elem := range response.Elements {
		reactedAt := time.UnixMilli(elem.Created)

		reaction := Reaction{
			PostID:       elem.Object,
			ReactionType: elem.ReactionType,
			ReactedAt:    reactedAt,
		}
		reactions = append(reactions, reaction)
	}

	return reactions, nil
}

// doRequest performs an authenticated HTTP request.
func (c *Client) doRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token.AccessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	return body, nil
}

// getLocalizedValue extracts the localized value from LinkedIn's nested structure.
func getLocalizedValue(lv localizedValue) string {
	if lv.Localized == nil {
		return ""
	}
	// Try common locales
	for _, locale := range []string{"en_US", "en_GB", "en"} {
		if v, ok := lv.Localized[locale]; ok {
			return v
		}
	}
	// Return first available
	for _, v := range lv.Localized {
		return v
	}
	return ""
}

// API response types

type localizedValue struct {
	Localized map[string]string `json:"localized"`
}

type profileResponse struct {
	ID        string         `json:"id"`
	FirstName localizedValue `json:"firstName"`
	LastName  localizedValue `json:"lastName"`
	Headline  localizedValue `json:"headline"`
}

type feedResponse struct {
	Elements []struct {
		ID     string `json:"id"`
		Author string `json:"author"`
		Text   struct {
			Text string `json:"text"`
		} `json:"text"`
		Created struct {
			Time int64 `json:"time"`
		} `json:"created"`
		SocialDetail struct {
			TotalSocialActivityCounts struct {
				NumLikes    int `json:"numLikes"`
				NumComments int `json:"numComments"`
				NumShares   int `json:"numShares"`
			} `json:"totalSocialActivityCounts"`
		} `json:"socialDetail"`
	} `json:"elements"`
}

type reactionsResponse struct {
	Elements []struct {
		Actor        string `json:"actor"`
		Object       string `json:"object"`
		ReactionType string `json:"reactionType"`
		Created      int64  `json:"created"`
	} `json:"elements"`
}
