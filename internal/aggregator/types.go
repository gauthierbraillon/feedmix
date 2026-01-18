// Package aggregator combines feeds from multiple sources.
package aggregator

import "time"

type Source string

const SourceYouTube Source = "youtube"

type ItemType string

const (
	ItemTypeVideo ItemType = "video"
	ItemTypeLike  ItemType = "like"
)

type FeedItem struct {
	ID          string     `json:"id"`
	Source      Source     `json:"source"`
	Type        ItemType   `json:"type"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Author      string     `json:"author"`
	AuthorID    string     `json:"author_id"`
	URL         string     `json:"url"`
	Thumbnail   string     `json:"thumbnail,omitempty"`
	PublishedAt time.Time  `json:"published_at"`
	Engagement  Engagement `json:"engagement"`
}

type Engagement struct {
	Likes    int64 `json:"likes"`
	Comments int64 `json:"comments"`
	Views    int64 `json:"views,omitempty"`
}

type FeedOptions struct {
	Limit   int
	Since   time.Time
	Until   time.Time
	Sources []Source
	Types   []ItemType
}
