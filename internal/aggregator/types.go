// Package aggregator combines feeds from multiple sources into a unified view.
//
// This package enables feedmix to:
// - Merge content from YouTube and LinkedIn chronologically
// - Filter content by date range
// - Provide a unified FeedItem interface for display
package aggregator

import "time"

// Source identifies the origin of a feed item.
type Source string

const (
	SourceYouTube  Source = "youtube"
	SourceLinkedIn Source = "linkedin"
)

// ItemType identifies the type of content.
type ItemType string

const (
	ItemTypeVideo    ItemType = "video"
	ItemTypePost     ItemType = "post"
	ItemTypeLike     ItemType = "like"
	ItemTypeReaction ItemType = "reaction"
)

// FeedItem represents a unified item from any source.
type FeedItem struct {
	ID          string    `json:"id"`
	Source      Source    `json:"source"`
	Type        ItemType  `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	AuthorID    string    `json:"author_id"`
	URL         string    `json:"url"`
	Thumbnail   string    `json:"thumbnail,omitempty"`
	PublishedAt time.Time `json:"published_at"`
	Engagement  Engagement `json:"engagement"`
}

// Engagement holds engagement metrics for a feed item.
type Engagement struct {
	Likes    int64 `json:"likes"`
	Comments int64 `json:"comments"`
	Shares   int64 `json:"shares"`
	Views    int64 `json:"views,omitempty"`
}

// FeedOptions configures feed retrieval.
type FeedOptions struct {
	Limit     int
	Since     time.Time
	Until     time.Time
	Sources   []Source
	Types     []ItemType
}
