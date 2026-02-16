// Package youtube provides a client for the YouTube Data API v3.
package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gauthierbraillon/feedmix/pkg/oauth"
)

const defaultBaseURL = "https://www.googleapis.com"

// HTTPClient interface for making HTTP requests (allows injection for testing).
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient HTTPClient) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithBaseURL sets a custom base URL (useful for testing).
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// Client is a YouTube Data API client.
type Client struct {
	token      *oauth.Token
	baseURL    string
	httpClient HTTPClient
}

// NewClient creates a new YouTube API client with the given OAuth token.
func NewClient(token *oauth.Token, opts ...ClientOption) *Client {
	c := &Client{
		token:      token,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// FetchSubscriptions retrieves the authenticated user's subscriptions.
func (c *Client) FetchSubscriptions(ctx context.Context) ([]Subscription, error) {
	url := fmt.Sprintf("%s/youtube/v3/subscriptions?part=snippet&mine=true&maxResults=50", c.baseURL)

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response subscriptionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse subscriptions response: %w", err)
	}

	subs := make([]Subscription, 0, len(response.Items))
	for _, item := range response.Items {
		publishedAt, _ := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
		thumbnail := ""
		if item.Snippet.Thumbnails.Default.URL != "" {
			thumbnail = item.Snippet.Thumbnails.Default.URL
		}

		subs = append(subs, Subscription{
			ChannelID:    item.Snippet.ResourceID.ChannelID,
			ChannelTitle: item.Snippet.Title,
			Description:  item.Snippet.Description,
			Thumbnail:    thumbnail,
			SubscribedAt: publishedAt,
		})
	}

	return subs, nil
}

// FetchRecentVideos retrieves recent videos from a channel.
func (c *Client) FetchRecentVideos(ctx context.Context, channelID string, limit int) ([]Video, error) {
	searchURL := fmt.Sprintf("%s/youtube/v3/search?part=snippet&channelId=%s&maxResults=%d&order=date&type=video",
		c.baseURL, channelID, limit)

	body, err := c.doRequest(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	var searchResp searchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	if len(searchResp.Items) == 0 {
		return []Video{}, nil
	}

	videoIDs := make([]string, 0, len(searchResp.Items))
	for _, item := range searchResp.Items {
		videoIDs = append(videoIDs, item.ID.VideoID)
	}

	videosURL := fmt.Sprintf("%s/youtube/v3/videos?part=statistics,contentDetails&id=%s",
		c.baseURL, joinIDs(videoIDs))

	body, err = c.doRequest(ctx, videosURL)
	if err != nil {
		return nil, err
	}

	var videosResp videosResponse
	if err := json.Unmarshal(body, &videosResp); err != nil {
		return nil, fmt.Errorf("failed to parse videos response: %w", err)
	}

	statsMap := make(map[string]videoStats)
	for _, item := range videosResp.Items {
		viewCount, _ := strconv.ParseInt(item.Statistics.ViewCount, 10, 64)
		likeCount, _ := strconv.ParseInt(item.Statistics.LikeCount, 10, 64)
		statsMap[item.ID] = videoStats{
			viewCount: viewCount,
			likeCount: likeCount,
			duration:  item.ContentDetails.Duration,
		}
	}

	videos := make([]Video, 0, len(searchResp.Items))
	for _, item := range searchResp.Items {
		publishedAt, _ := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
		thumbnail := ""
		if item.Snippet.Thumbnails.Default.URL != "" {
			thumbnail = item.Snippet.Thumbnails.Default.URL
		}

		stats := statsMap[item.ID.VideoID]
		videos = append(videos, Video{
			ID:           item.ID.VideoID,
			Title:        item.Snippet.Title,
			Description:  item.Snippet.Description,
			ChannelID:    item.Snippet.ChannelID,
			ChannelTitle: item.Snippet.ChannelTitle,
			Thumbnail:    thumbnail,
			PublishedAt:  publishedAt,
			ViewCount:    stats.viewCount,
			LikeCount:    stats.likeCount,
			Duration:     stats.duration,
			URL:          fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.ID.VideoID),
		})
	}

	return videos, nil
}

// FetchLikedVideos retrieves videos the authenticated user has liked.
func (c *Client) FetchLikedVideos(ctx context.Context, limit int) ([]LikedVideo, error) {
	url := fmt.Sprintf("%s/youtube/v3/playlistItems?part=snippet&playlistId=LL&maxResults=%d",
		c.baseURL, limit)

	body, err := c.doRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var response playlistItemsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse playlist items response: %w", err)
	}

	videos := make([]LikedVideo, 0, len(response.Items))
	for _, item := range response.Items {
		publishedAt, _ := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
		thumbnail := ""
		if item.Snippet.Thumbnails.Default.URL != "" {
			thumbnail = item.Snippet.Thumbnails.Default.URL
		}

		videos = append(videos, LikedVideo{
			Video: Video{
				ID:           item.Snippet.ResourceID.VideoID,
				Title:        item.Snippet.Title,
				Description:  item.Snippet.Description,
				ChannelID:    item.Snippet.ChannelID,
				ChannelTitle: item.Snippet.ChannelTitle,
				Thumbnail:    thumbnail,
				PublishedAt:  publishedAt,
				URL:          fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.Snippet.ResourceID.VideoID),
			},
			LikedAt: publishedAt,
		})
	}

	return videos, nil
}

func (c *Client) doRequest(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token.AccessToken))
	req.Header.Set("Accept", "application/json")

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
		return nil, c.handleAPIError(resp.StatusCode)
	}

	return body, nil
}

func joinIDs(ids []string) string {
	result := ""
	for i, id := range ids {
		if i > 0 {
			result += ","
		}
		result += id
	}
	return result
}

// API response types (private - implementation detail)

type subscriptionsResponse struct {
	Items []struct {
		Snippet struct {
			ResourceID struct {
				ChannelID string `json:"channelId"`
			} `json:"resourceId"`
			Title       string `json:"title"`
			Description string `json:"description"`
			PublishedAt string `json:"publishedAt"`
			Thumbnails  struct {
				Default struct {
					URL string `json:"url"`
				} `json:"default"`
			} `json:"thumbnails"`
		} `json:"snippet"`
	} `json:"items"`
}

type searchResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
		Snippet struct {
			Title        string `json:"title"`
			Description  string `json:"description"`
			ChannelID    string `json:"channelId"`
			ChannelTitle string `json:"channelTitle"`
			PublishedAt  string `json:"publishedAt"`
			Thumbnails   struct {
				Default struct {
					URL string `json:"url"`
				} `json:"default"`
			} `json:"thumbnails"`
		} `json:"snippet"`
	} `json:"items"`
}

type videosResponse struct {
	Items []struct {
		ID         string `json:"id"`
		Statistics struct {
			ViewCount string `json:"viewCount"`
			LikeCount string `json:"likeCount"`
		} `json:"statistics"`
		ContentDetails struct {
			Duration string `json:"duration"`
		} `json:"contentDetails"`
	} `json:"items"`
}

type playlistItemsResponse struct {
	Items []struct {
		Snippet struct {
			ResourceID struct {
				VideoID string `json:"videoId"`
			} `json:"resourceId"`
			Title        string `json:"title"`
			Description  string `json:"description"`
			ChannelID    string `json:"channelId"`
			ChannelTitle string `json:"channelTitle"`
			PublishedAt  string `json:"publishedAt"`
			Thumbnails   struct {
				Default struct {
					URL string `json:"url"`
				} `json:"default"`
			} `json:"thumbnails"`
		} `json:"snippet"`
	} `json:"items"`
}

type videoStats struct {
	viewCount int64
	likeCount int64
	duration  string
}

func (c *Client) handleAPIError(statusCode int) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("YouTube API authentication failed - please run 'feedmix auth' to re-authenticate")
	case http.StatusForbidden:
		return fmt.Errorf("YouTube API access denied - check your OAuth permissions")
	case http.StatusTooManyRequests:
		return fmt.Errorf("YouTube API rate limit exceeded - please try again later")
	case http.StatusServiceUnavailable:
		return fmt.Errorf("YouTube API temporarily unavailable - please try again in a few minutes")
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusGatewayTimeout:
		return fmt.Errorf("YouTube API server error - please try again later")
	default:
		return fmt.Errorf("YouTube API error (status %d) - please try again", statusCode)
	}
}
