package substack

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

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

// WithBaseURL overrides the base URL used when constructing feed URLs (useful for testing).
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// Client fetches RSS feeds from Substack publications.
type Client struct {
	httpClient HTTPClient
	baseURL    string
}

// NewClient creates a new Substack RSS client.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// FetchPosts fetches recent posts from a Substack publication RSS feed.
// publicationURL is the base URL (e.g. https://simonwillison.substack.com).
// /feed is appended internally. Results are limited to limit items.
func (c *Client) FetchPosts(ctx context.Context, publicationURL string, limit int) ([]Post, error) {
	feedURL := c.buildFeedURL(publicationURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("substack RSS feed returned HTTP %d for %s", resp.StatusCode, publicationURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read RSS feed: %w", err)
	}

	return parseRSS(body, limit)
}

func (c *Client) buildFeedURL(publicationURL string) string {
	if c.baseURL != "" {
		return strings.TrimRight(c.baseURL, "/") + "/feed"
	}
	return strings.TrimRight(resolveSubstackURL(publicationURL), "/") + "/feed"
}

// resolveSubstackURL converts https://substack.com/@username profile URLs to
// the subdomain form https://username.substack.com, which hosts the RSS feed.
// Traditional subdomain URLs are returned unchanged.
func resolveSubstackURL(publicationURL string) string {
	const profilePrefix = "https://substack.com/@"
	if strings.HasPrefix(publicationURL, profilePrefix) {
		username := strings.TrimPrefix(publicationURL, profilePrefix)
		return "https://" + username + ".substack.com"
	}
	return publicationURL
}

func parseRSS(data []byte, limit int) ([]Post, error) {
	var doc rssDoc
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	items := doc.Channel.Items
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	posts := make([]Post, 0, len(items))
	for _, item := range items {
		author := item.DCCreator
		if author == "" {
			author = item.Author
		}
		posts = append(posts, Post{
			ID:          item.GUID,
			Title:       item.Title,
			Description: item.Desc,
			Author:      author,
			URL:         item.Link,
			PublishedAt: parsePubDate(item.PubDate),
		})
	}
	return posts, nil
}

func parsePubDate(s string) time.Time {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// rssDoc and rssItem are private XML parsing structs.
type rssDoc struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title     string `xml:"title"`
	Link      string `xml:"link"`
	Author    string `xml:"author"`
	DCCreator string `xml:"creator"`
	PubDate   string `xml:"pubDate"`
	Desc      string `xml:"description"`
	GUID      string `xml:"guid"`
}
