// Package linkedin provides a client for the LinkedIn API.
//
// IMPORTANT: LinkedIn API has significant restrictions.
// Most feed-related endpoints require Marketing Developer Platform access.
// This package provides the interface for what would be ideal functionality.
//
// This package enables feedmix to:
// - Authenticate with LinkedIn via OAuth 2.0
// - Fetch user's profile information
// - Get posts from connections (limited access)
// - Retrieve user's reactions/likes
package linkedin

import "time"

// Profile represents a LinkedIn user profile.
type Profile struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Headline  string `json:"headline"`
	Picture   string `json:"picture"`
}

// Post represents a LinkedIn post/update.
type Post struct {
	ID           string    `json:"id"`
	AuthorID     string    `json:"author_id"`
	AuthorName   string    `json:"author_name"`
	AuthorPicture string   `json:"author_picture"`
	Text         string    `json:"text"`
	MediaURL     string    `json:"media_url,omitempty"`
	MediaType    string    `json:"media_type,omitempty"`
	LikeCount    int       `json:"like_count"`
	CommentCount int       `json:"comment_count"`
	ShareCount   int       `json:"share_count"`
	PublishedAt  time.Time `json:"published_at"`
	URL          string    `json:"url"`
}

// Reaction represents a user's reaction (like) on a post.
type Reaction struct {
	PostID       string    `json:"post_id"`
	ReactionType string    `json:"reaction_type"` // LIKE, CELEBRATE, SUPPORT, etc.
	ReactedAt    time.Time `json:"reacted_at"`
}

// Connection represents a LinkedIn connection.
type Connection struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Headline  string `json:"headline"`
	Picture   string `json:"picture"`
}
