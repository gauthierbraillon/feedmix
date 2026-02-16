package aggregator

import (
	"testing"
	"time"
)

func TestAC200_Feed_ShowsNewestItemsFirst(t *testing.T) {
	now := time.Now()
	items := []FeedItem{
		{ID: "oldest", PublishedAt: now.Add(-3 * time.Hour)},
		{ID: "newest", PublishedAt: now.Add(-1 * time.Hour)},
		{ID: "middle", PublishedAt: now.Add(-2 * time.Hour)},
	}

	agg := New()
	agg.AddItems(items)
	feed := agg.GetFeed(FeedOptions{})

	if len(feed) != 3 {
		t.Fatalf("user should see all 3 items, got %d", len(feed))
	}
	if feed[0].ID != "newest" {
		t.Errorf("user should see newest item first, got: %s", feed[0].ID)
	}
	if feed[1].ID != "middle" {
		t.Errorf("user should see middle item second, got: %s", feed[1].ID)
	}
	if feed[2].ID != "oldest" {
		t.Errorf("user should see oldest item last, got: %s", feed[2].ID)
	}
}

func TestAC200_Feed_SortsAcrossMultipleSources(t *testing.T) {
	now := time.Now()
	items := []FeedItem{
		{ID: "yt-old", Source: SourceYouTube, Type: ItemTypeVideo, PublishedAt: now.Add(-3 * time.Hour)},
		{ID: "yt-new", Source: SourceYouTube, Type: ItemTypeVideo, PublishedAt: now.Add(-1 * time.Hour)},
		{ID: "yt-like", Source: SourceYouTube, Type: ItemTypeLike, PublishedAt: now.Add(-2 * time.Hour)},
	}

	agg := New()
	agg.AddItems(items)
	feed := agg.GetFeed(FeedOptions{})

	expectedOrder := []string{"yt-new", "yt-like", "yt-old"}
	for i, expectedID := range expectedOrder {
		if feed[i].ID != expectedID {
			t.Errorf("position %d: user should see %s, got %s", i+1, expectedID, feed[i].ID)
		}
	}
}

func TestAC201_Feed_ShowsOnlyItemsWithinDateRange(t *testing.T) {
	now := time.Now()
	items := []FeedItem{
		{ID: "recent", PublishedAt: now.Add(-1 * time.Hour)},
		{ID: "yesterday", PublishedAt: now.Add(-25 * time.Hour)},
		{ID: "last-week", PublishedAt: now.Add(-7 * 24 * time.Hour)},
	}

	agg := New()
	agg.AddItems(items)
	feed := agg.GetFeed(FeedOptions{
		Since: now.Add(-26 * time.Hour),
		Until: now.Add(-23 * time.Hour),
	})

	if len(feed) != 1 {
		t.Fatalf("user should see 1 item in date range, got %d", len(feed))
	}
	if feed[0].ID != "yesterday" {
		t.Errorf("user should see 'yesterday' item, got: %s", feed[0].ID)
	}
}

func TestAC202_Feed_ShowsOnlyItemsFromSelectedSource(t *testing.T) {
	now := time.Now()
	items := []FeedItem{
		{ID: "yt1", Source: SourceYouTube, PublishedAt: now},
		{ID: "yt2", Source: SourceYouTube, PublishedAt: now},
		{ID: "other", Source: Source("reddit"), PublishedAt: now},
	}

	agg := New()
	agg.AddItems(items)
	feed := agg.GetFeed(FeedOptions{Sources: []Source{SourceYouTube}})

	if len(feed) != 2 {
		t.Fatalf("user filtering by YouTube should see 2 items, got %d", len(feed))
	}
	for _, item := range feed {
		if item.Source != SourceYouTube {
			t.Errorf("user should only see YouTube items, got source: %s", item.Source)
		}
	}
}

func TestAC203_Feed_ShowsOnlySelectedContentTypes(t *testing.T) {
	now := time.Now()
	items := []FeedItem{
		{ID: "video1", Type: ItemTypeVideo, PublishedAt: now},
		{ID: "video2", Type: ItemTypeVideo, PublishedAt: now},
		{ID: "like1", Type: ItemTypeLike, PublishedAt: now},
	}

	agg := New()
	agg.AddItems(items)
	feed := agg.GetFeed(FeedOptions{Types: []ItemType{ItemTypeVideo}})

	if len(feed) != 2 {
		t.Fatalf("user filtering by videos should see 2 items, got %d", len(feed))
	}
	for _, item := range feed {
		if item.Type != ItemTypeVideo {
			t.Errorf("user should only see videos, got type: %s", item.Type)
		}
	}
}

func TestAC204_Feed_RespectsUserRequestedLimit(t *testing.T) {
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
	feed := agg.GetFeed(FeedOptions{Limit: 2})

	if len(feed) != 2 {
		t.Fatalf("user requesting limit 2 should see 2 items, got %d", len(feed))
	}
	if feed[0].ID != "item1" || feed[1].ID != "item2" {
		t.Error("user should see newest 2 items when limit applied")
	}
}

func TestAC204_Feed_ShowsAllItemsWhenNoLimitSpecified(t *testing.T) {
	now := time.Now()
	items := []FeedItem{
		{ID: "item1", PublishedAt: now},
		{ID: "item2", PublishedAt: now},
		{ID: "item3", PublishedAt: now},
	}

	agg := New()
	agg.AddItems(items)
	feed := agg.GetFeed(FeedOptions{})

	if len(feed) != 3 {
		t.Errorf("user without limit should see all 3 items, got %d", len(feed))
	}
}

func TestAC205_Feed_HandlesEmptyFeedGracefully(t *testing.T) {
	agg := New()
	feed := agg.GetFeed(FeedOptions{})

	if feed == nil {
		t.Fatal("feed should return empty slice, not nil")
	}
	if len(feed) != 0 {
		t.Errorf("user with no subscriptions should see empty feed, got %d items", len(feed))
	}
}
