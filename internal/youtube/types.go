// Package youtube provides a client for the YouTube Data API v3.
//
// This package enables feedmix to:
// - Authenticate with YouTube via OAuth 2.0
// - Fetch user's subscriptions
// - Get recent videos from subscribed channels
// - Retrieve liked videos
package youtube

import "time"

// Subscription represents a YouTube channel subscription.
type Subscription struct {
	ChannelID    string    `json:"channel_id"`
	ChannelTitle string    `json:"channel_title"`
	Description  string    `json:"description"`
	Thumbnail    string    `json:"thumbnail"`
	SubscribedAt time.Time `json:"subscribed_at"`
}

// Video represents a YouTube video.
type Video struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	ChannelID    string    `json:"channel_id"`
	ChannelTitle string    `json:"channel_title"`
	Thumbnail    string    `json:"thumbnail"`
	PublishedAt  time.Time `json:"published_at"`
	ViewCount    int64     `json:"view_count"`
	LikeCount    int64     `json:"like_count"`
	Duration     string    `json:"duration"`
	URL          string    `json:"url"`
}

// LikedVideo represents a video the user has liked.
type LikedVideo struct {
	Video
	LikedAt time.Time `json:"liked_at"`
}
