// Package aggregator tests document the expected behavior of the feed aggregator.
//
// TDD Cycle: RED -> GREEN -> REFACTOR
//
// Test requirements (this file serves as documentation):
// - Aggregator merges items from multiple sources
// - Items are sorted chronologically (newest first)
// - Filtering by date range works correctly
// - Filtering by source works correctly
// - Filtering by item type works correctly
// - Limit option restricts number of results
package aggregator

import (
	"testing"
	"time"
)

// TestAggregator_MergeFeeds documents feed merging:
// - Items from different sources are combined
// - Result is sorted by PublishedAt (newest first)
func TestAggregator_MergeFeeds(t *testing.T) {
	now := time.Now()

	youtubeItems := []FeedItem{
		{
			ID:          "yt1",
			Source:      SourceYouTube,
			Type:        ItemTypeVideo,
			Title:       "YouTube Video 1",
			PublishedAt: now.Add(-1 * time.Hour),
		},
		{
			ID:          "yt2",
			Source:      SourceYouTube,
			Type:        ItemTypeVideo,
			Title:       "YouTube Video 2",
			PublishedAt: now.Add(-3 * time.Hour),
		},
	}

	linkedinItems := []FeedItem{
		{
			ID:          "li1",
			Source:      SourceLinkedIn,
			Type:        ItemTypePost,
			Title:       "LinkedIn Post 1",
			PublishedAt: now.Add(-2 * time.Hour),
		},
	}

	agg := New()
	agg.AddItems(youtubeItems)
	agg.AddItems(linkedinItems)

	result := agg.GetFeed(FeedOptions{})

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}

	// Should be sorted newest first
	expectedOrder := []string{"yt1", "li1", "yt2"}
	for i, expected := range expectedOrder {
		if result[i].ID != expected {
			t.Errorf("position %d: expected %s, got %s", i, expected, result[i].ID)
		}
	}
}

// TestAggregator_FilterByDate documents date filtering:
// - Since filters out items before the given time
// - Until filters out items after the given time
func TestAggregator_FilterByDate(t *testing.T) {
	now := time.Now()

	items := []FeedItem{
		{ID: "item1", PublishedAt: now.Add(-1 * time.Hour)},
		{ID: "item2", PublishedAt: now.Add(-2 * time.Hour)},
		{ID: "item3", PublishedAt: now.Add(-3 * time.Hour)},
		{ID: "item4", PublishedAt: now.Add(-4 * time.Hour)},
	}

	agg := New()
	agg.AddItems(items)

	// Filter: between 3.5 and 1.5 hours ago
	result := agg.GetFeed(FeedOptions{
		Since: now.Add(-3*time.Hour - 30*time.Minute),
		Until: now.Add(-1*time.Hour - 30*time.Minute),
	})

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}

	expectedIDs := map[string]bool{"item2": true, "item3": true}
	for _, item := range result {
		if !expectedIDs[item.ID] {
			t.Errorf("unexpected item: %s", item.ID)
		}
	}
}

// TestAggregator_FilterBySource documents source filtering:
// - Only items from specified sources are included
func TestAggregator_FilterBySource(t *testing.T) {
	now := time.Now()

	items := []FeedItem{
		{ID: "yt1", Source: SourceYouTube, PublishedAt: now},
		{ID: "li1", Source: SourceLinkedIn, PublishedAt: now},
		{ID: "yt2", Source: SourceYouTube, PublishedAt: now},
	}

	agg := New()
	agg.AddItems(items)

	result := agg.GetFeed(FeedOptions{
		Sources: []Source{SourceYouTube},
	})

	if len(result) != 2 {
		t.Fatalf("expected 2 YouTube items, got %d", len(result))
	}

	for _, item := range result {
		if item.Source != SourceYouTube {
			t.Errorf("expected YouTube source, got %s", item.Source)
		}
	}
}

// TestAggregator_FilterByType documents type filtering:
// - Only items of specified types are included
func TestAggregator_FilterByType(t *testing.T) {
	now := time.Now()

	items := []FeedItem{
		{ID: "video1", Type: ItemTypeVideo, PublishedAt: now},
		{ID: "post1", Type: ItemTypePost, PublishedAt: now},
		{ID: "like1", Type: ItemTypeLike, PublishedAt: now},
	}

	agg := New()
	agg.AddItems(items)

	result := agg.GetFeed(FeedOptions{
		Types: []ItemType{ItemTypeVideo, ItemTypePost},
	})

	if len(result) != 2 {
		t.Fatalf("expected 2 items (video + post), got %d", len(result))
	}

	for _, item := range result {
		if item.Type != ItemTypeVideo && item.Type != ItemTypePost {
			t.Errorf("unexpected type: %s", item.Type)
		}
	}
}

// TestAggregator_Limit documents result limiting:
// - No more than Limit items are returned
// - Most recent items are returned first
func TestAggregator_Limit(t *testing.T) {
	now := time.Now()

	items := []FeedItem{
		{ID: "item1", PublishedAt: now.Add(-1 * time.Hour)},
		{ID: "item2", PublishedAt: now.Add(-2 * time.Hour)},
		{ID: "item3", PublishedAt: now.Add(-3 * time.Hour)},
		{ID: "item4", PublishedAt: now.Add(-4 * time.Hour)},
		{ID: "item5", PublishedAt: now.Add(-5 * time.Hour)},
	}

	agg := New()
	agg.AddItems(items)

	result := agg.GetFeed(FeedOptions{Limit: 3})

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}

	// Should be the 3 newest
	expectedOrder := []string{"item1", "item2", "item3"}
	for i, expected := range expectedOrder {
		if result[i].ID != expected {
			t.Errorf("position %d: expected %s, got %s", i, expected, result[i].ID)
		}
	}
}

// TestAggregator_Empty documents empty feed handling:
// - Returns empty slice, not nil
func TestAggregator_Empty(t *testing.T) {
	agg := New()
	result := agg.GetFeed(FeedOptions{})

	if result == nil {
		t.Error("expected empty slice, got nil")
	}

	if len(result) != 0 {
		t.Errorf("expected 0 items, got %d", len(result))
	}
}

// TestAggregator_CombinedFilters documents multiple filters:
// - Multiple filters can be applied together
func TestAggregator_CombinedFilters(t *testing.T) {
	now := time.Now()

	items := []FeedItem{
		{ID: "yt-video-1", Source: SourceYouTube, Type: ItemTypeVideo, PublishedAt: now.Add(-1 * time.Hour)},
		{ID: "yt-like-1", Source: SourceYouTube, Type: ItemTypeLike, PublishedAt: now.Add(-2 * time.Hour)},
		{ID: "li-post-1", Source: SourceLinkedIn, Type: ItemTypePost, PublishedAt: now.Add(-3 * time.Hour)},
		{ID: "yt-video-2", Source: SourceYouTube, Type: ItemTypeVideo, PublishedAt: now.Add(-10 * time.Hour)},
	}

	agg := New()
	agg.AddItems(items)

	// YouTube videos from last 5 hours
	result := agg.GetFeed(FeedOptions{
		Sources: []Source{SourceYouTube},
		Types:   []ItemType{ItemTypeVideo},
		Since:   now.Add(-5 * time.Hour),
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}

	if result[0].ID != "yt-video-1" {
		t.Errorf("expected yt-video-1, got %s", result[0].ID)
	}
}
