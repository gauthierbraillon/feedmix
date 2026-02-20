// Package substack tests document the expected behavior of the Substack RSS client.
//
// TDD Cycle: RED -> GREEN -> REFACTOR
//
// Test requirements (this file serves as documentation):
// - Client fetches and parses RSS feed from a Substack publication URL
// - Client limits results to the requested count
// - Client appends /feed to the publication URL
// - Client returns errors on HTTP failures
// - Client returns errors on malformed XML
package substack

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const validRSSXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <channel>
    <title>Test Publication</title>
    <item>
      <title>Hello World</title>
      <link>https://example.substack.com/p/hello-world</link>
      <dc:creator>Jane Doe</dc:creator>
      <pubDate>Mon, 01 Jan 2024 12:00:00 +0000</pubDate>
      <description>A great article about things.</description>
      <guid>https://example.substack.com/p/hello-world</guid>
    </item>
    <item>
      <title>Second Post</title>
      <link>https://example.substack.com/p/second-post</link>
      <dc:creator>Jane Doe</dc:creator>
      <pubDate>Tue, 02 Jan 2024 12:00:00 +0000</pubDate>
      <description>Another article.</description>
      <guid>https://example.substack.com/p/second-post</guid>
    </item>
  </channel>
</rss>`

// TestClient_FetchPosts_ReturnsParsedPosts documents RSS parsing:
// - Parses title, author (dc:creator), URL (link), pubDate, description, and guid as ID
func TestClient_FetchPosts_ReturnsParsedPosts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprint(w, validRSSXML)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	posts, err := client.FetchPosts(context.Background(), server.URL, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}

	post := posts[0]
	if post.Title != "Hello World" {
		t.Errorf("expected title 'Hello World', got %q", post.Title)
	}
	if post.Author != "Jane Doe" {
		t.Errorf("expected author 'Jane Doe', got %q", post.Author)
	}
	if post.URL != "https://example.substack.com/p/hello-world" {
		t.Errorf("expected URL 'https://example.substack.com/p/hello-world', got %q", post.URL)
	}
	if post.Description != "A great article about things." {
		t.Errorf("expected description 'A great article about things.', got %q", post.Description)
	}
	if post.ID != "https://example.substack.com/p/hello-world" {
		t.Errorf("expected ID (guid) 'https://example.substack.com/p/hello-world', got %q", post.ID)
	}
	if post.PublishedAt.IsZero() {
		t.Error("expected non-zero PublishedAt")
	}
}

// TestClient_FetchPosts_RespectsLimit documents limit behavior:
// - RSS feed has more items than limit → only limit items returned
func TestClient_FetchPosts_RespectsLimit(t *testing.T) {
	const tenItemsRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <item><title>Post 1</title><link>http://x.com/1</link><guid>1</guid></item>
    <item><title>Post 2</title><link>http://x.com/2</link><guid>2</guid></item>
    <item><title>Post 3</title><link>http://x.com/3</link><guid>3</guid></item>
    <item><title>Post 4</title><link>http://x.com/4</link><guid>4</guid></item>
    <item><title>Post 5</title><link>http://x.com/5</link><guid>5</guid></item>
    <item><title>Post 6</title><link>http://x.com/6</link><guid>6</guid></item>
    <item><title>Post 7</title><link>http://x.com/7</link><guid>7</guid></item>
    <item><title>Post 8</title><link>http://x.com/8</link><guid>8</guid></item>
    <item><title>Post 9</title><link>http://x.com/9</link><guid>9</guid></item>
    <item><title>Post 10</title><link>http://x.com/10</link><guid>10</guid></item>
  </channel>
</rss>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, tenItemsRSS)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	posts, err := client.FetchPosts(context.Background(), server.URL, 3)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 3 {
		t.Errorf("expected 3 posts (limit), got %d", len(posts))
	}
}

// TestClient_FetchPosts_ReturnsErrorOnHTTPError documents HTTP error handling:
// - 404 or other non-200 status → descriptive error returned
func TestClient_FetchPosts_ReturnsErrorOnHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	_, err := client.FetchPosts(context.Background(), server.URL, 10)

	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
}

// TestClient_FetchPosts_ReturnsErrorOnInvalidXML documents XML parse error handling:
// - Garbage response body → parse error returned
func TestClient_FetchPosts_ReturnsErrorOnInvalidXML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "this is not xml <<garbage>>")
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL))
	_, err := client.FetchPosts(context.Background(), server.URL, 10)

	if err == nil {
		t.Fatal("expected error for invalid XML, got nil")
	}
}

// TestClient_FetchPosts_AppendsRSSPathToPublicationURL documents URL construction:
// - Client appends /feed to the publication URL before requesting
func TestClient_FetchPosts_AppendsRSSPathToPublicationURL(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		fmt.Fprint(w, validRSSXML)
	}))
	defer server.Close()

	client := NewClient()
	publicationURL := server.URL
	_, _ = client.FetchPosts(context.Background(), publicationURL, 10)

	if !strings.HasSuffix(capturedPath, "/feed") {
		t.Errorf("expected request path to end with /feed, got %q", capturedPath)
	}
}
