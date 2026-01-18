package aggregator

import (
	"testing"
	"time"
)

func TestAggregator_MergeFeeds(t *testing.T) {
	now := time.Now()
	items := []FeedItem{
		{ID: "yt1", Source: SourceYouTube, Type: ItemTypeVideo, PublishedAt: now.Add(-1 * time.Hour)},
		{ID: "yt2", Source: SourceYouTube, Type: ItemTypeVideo, PublishedAt: now.Add(-3 * time.Hour)},
		{ID: "yt3", Source: SourceYouTube, Type: ItemTypeLike, PublishedAt: now.Add(-2 * time.Hour)},
	}

	agg := New()
	agg.AddItems(items)
	result := agg.GetFeed(FeedOptions{})

	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}

	expectedOrder := []string{"yt1", "yt3", "yt2"}
	for i, exp := range expectedOrder {
		if result[i].ID != exp {
			t.Errorf("pos %d: expected %s, got %s", i, exp, result[i].ID)
		}
	}
}

func TestAggregator_FilterByDate(t *testing.T) {
	now := time.Now()
	items := []FeedItem{
		{ID: "item1", PublishedAt: now.Add(-1 * time.Hour)},
		{ID: "item2", PublishedAt: now.Add(-2 * time.Hour)},
		{ID: "item3", PublishedAt: now.Add(-3 * time.Hour)},
	}

	agg := New()
	agg.AddItems(items)
	result := agg.GetFeed(FeedOptions{
		Since: now.Add(-2*time.Hour - 30*time.Minute),
		Until: now.Add(-1*time.Hour - 30*time.Minute),
	})

	if len(result) != 1 || result[0].ID != "item2" {
		t.Errorf("expected item2, got %v", result)
	}
}

func TestAggregator_FilterBySource(t *testing.T) {
	now := time.Now()
	items := []FeedItem{
		{ID: "yt1", Source: SourceYouTube, PublishedAt: now},
		{ID: "yt2", Source: SourceYouTube, PublishedAt: now},
	}

	agg := New()
	agg.AddItems(items)
	result := agg.GetFeed(FeedOptions{Sources: []Source{SourceYouTube}})

	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
}

func TestAggregator_FilterByType(t *testing.T) {
	now := time.Now()
	items := []FeedItem{
		{ID: "video1", Type: ItemTypeVideo, PublishedAt: now},
		{ID: "like1", Type: ItemTypeLike, PublishedAt: now},
	}

	agg := New()
	agg.AddItems(items)
	result := agg.GetFeed(FeedOptions{Types: []ItemType{ItemTypeVideo}})

	if len(result) != 1 || result[0].ID != "video1" {
		t.Errorf("expected video1, got %v", result)
	}
}

func TestAggregator_Limit(t *testing.T) {
	now := time.Now()
	items := []FeedItem{
		{ID: "item1", PublishedAt: now.Add(-1 * time.Hour)},
		{ID: "item2", PublishedAt: now.Add(-2 * time.Hour)},
		{ID: "item3", PublishedAt: now.Add(-3 * time.Hour)},
	}

	agg := New()
	agg.AddItems(items)
	result := agg.GetFeed(FeedOptions{Limit: 2})

	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
	if result[0].ID != "item1" || result[1].ID != "item2" {
		t.Error("wrong items returned")
	}
}

func TestAggregator_Empty(t *testing.T) {
	result := New().GetFeed(FeedOptions{})
	if result == nil || len(result) != 0 {
		t.Error("expected empty slice")
	}
}
