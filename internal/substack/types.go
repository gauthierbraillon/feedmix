// Package substack provides a client for fetching Substack publication RSS feeds.
package substack

import "time"

// Post represents a Substack newsletter post.
type Post struct {
	ID          string
	Title       string
	Description string
	Author      string
	URL         string
	PublishedAt time.Time
}
