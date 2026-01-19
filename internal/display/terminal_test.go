package display

import (
	"strings"
	"testing"
	"time"

	"github.com/gauthierbraillon/feedmix/internal/aggregator"
)

func TestFormatFeedItem(t *testing.T) {
	item := aggregator.FeedItem{
		ID:          "test123",
		Source:      aggregator.SourceYouTube,
		Type:        aggregator.ItemTypeVideo,
		Title:       "Test Video Title",
		Author:      "Test Author",
		URL:         "https://youtube.com/watch?v=test123",
		PublishedAt: time.Now().Add(-2 * time.Hour),
		Engagement:  aggregator.Engagement{Views: 1000, Likes: 50},
	}

	output := NewTerminalFormatter().FormatItem(item)

	if !strings.Contains(output, "Test Video Title") {
		t.Error("should contain title")
	}
	if !strings.Contains(strings.ToLower(output), "youtube") {
		t.Error("should indicate source")
	}
	if !strings.Contains(output, "Test Author") {
		t.Error("should contain author")
	}
}

func TestFormatTimestamp(t *testing.T) {
	f := NewTerminalFormatter()
	tests := []struct {
		name, contains string
		time           time.Time
	}{
		{"minutes ago", "min", time.Now().Add(-30 * time.Minute)},
		{"hours ago", "hour", time.Now().Add(-3 * time.Hour)},
		{"days ago", "day", time.Now().Add(-48 * time.Hour)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(strings.ToLower(f.FormatTimestamp(tt.time)), tt.contains) {
				t.Errorf("should contain %q", tt.contains)
			}
		})
	}
}

func TestTruncateText(t *testing.T) {
	f := NewTerminalFormatter()

	short := f.TruncateText("Hello", 10)
	if short != "Hello" {
		t.Error("short text should be unchanged")
	}

	long := f.TruncateText("This is a very long text", 10)
	if len(long) > 10 || !strings.HasSuffix(long, "...") {
		t.Error("long text should be truncated with ...")
	}
}

func TestFormatFeed(t *testing.T) {
	items := []aggregator.FeedItem{
		{ID: "1", Source: aggregator.SourceYouTube, Title: "First", Author: "A", PublishedAt: time.Now()},
		{ID: "2", Source: aggregator.SourceYouTube, Title: "Second", Author: "B", PublishedAt: time.Now()},
	}

	output := NewTerminalFormatter().FormatFeed(items)
	if !strings.Contains(output, "First") || !strings.Contains(output, "Second") {
		t.Error("should contain both items")
	}
}

func TestFormatFeed_Empty(t *testing.T) {
	output := NewTerminalFormatter().FormatFeed(nil)
	if !strings.Contains(strings.ToLower(output), "no") {
		t.Error("should indicate no items")
	}
}
