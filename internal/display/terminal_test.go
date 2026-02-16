package display

import (
	"strings"
	"testing"
	"time"

	"github.com/gauthierbraillon/feedmix/internal/aggregator"
)

func TestAC300_TerminalFeed_ShowsVideoTitle(t *testing.T) {
	item := aggregator.FeedItem{
		ID:          "test123",
		Source:      aggregator.SourceYouTube,
		Type:        aggregator.ItemTypeVideo,
		Title:       "How to Build CLI Tools in Go",
		Author:      "Tech Channel",
		URL:         "https://youtube.com/watch?v=test123",
		PublishedAt: time.Now(),
	}

	output := NewTerminalFormatter().FormatItem(item)

	if !strings.Contains(output, "How to Build CLI Tools in Go") {
		t.Error("user should see video title in terminal output")
	}
}

func TestAC300_TerminalFeed_ShowsAuthorName(t *testing.T) {
	item := aggregator.FeedItem{
		Title:       "Test Video",
		Author:      "CodeMaster",
		Source:      aggregator.SourceYouTube,
		PublishedAt: time.Now(),
	}

	output := NewTerminalFormatter().FormatItem(item)

	if !strings.Contains(output, "CodeMaster") {
		t.Error("user should see author name in terminal output")
	}
}

func TestAC300_TerminalFeed_ShowsSourceIndicator(t *testing.T) {
	item := aggregator.FeedItem{
		Title:       "Test Video",
		Source:      aggregator.SourceYouTube,
		PublishedAt: time.Now(),
	}

	output := NewTerminalFormatter().FormatItem(item)

	if !strings.Contains(strings.ToLower(output), "youtube") {
		t.Error("user should see content source (YouTube) in terminal output")
	}
}

func TestAC301_TerminalFeed_ShowsRelativeTimestamps(t *testing.T) {
	formatter := NewTerminalFormatter()
	testCases := []struct {
		name      string
		timestamp time.Time
		contains  string
	}{
		{"recent minutes", time.Now().Add(-30 * time.Minute), "min"},
		{"recent hours", time.Now().Add(-3 * time.Hour), "hour"},
		{"recent days", time.Now().Add(-48 * time.Hour), "day"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := formatter.FormatTimestamp(tc.timestamp)
			if !strings.Contains(strings.ToLower(output), tc.contains) {
				t.Errorf("user should see relative time (%s) for %s content", tc.contains, tc.name)
			}
		})
	}
}

func TestAC302_TerminalFeed_ShowsClickableURLs(t *testing.T) {
	item := aggregator.FeedItem{
		Title:       "Test Video",
		URL:         "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		Source:      aggregator.SourceYouTube,
		PublishedAt: time.Now(),
	}

	output := NewTerminalFormatter().FormatItem(item)

	if !strings.Contains(output, "https://www.youtube.com/watch?v=dQw4w9WgXcQ") {
		t.Error("user should see clickable video URL in terminal output")
	}
}

func TestAC303_TerminalFeed_TruncatesLongText(t *testing.T) {
	formatter := NewTerminalFormatter()
	longText := "This is a very long text that should be truncated because it exceeds the maximum length"

	truncated := formatter.TruncateText(longText, 20)

	if len(truncated) > 20 {
		t.Errorf("user should see truncated text (max 20 chars), got %d chars", len(truncated))
	}
	if !strings.HasSuffix(truncated, "...") {
		t.Error("user should see ellipsis indicating text was truncated")
	}
}

func TestAC303_TerminalFeed_PreservesShortText(t *testing.T) {
	formatter := NewTerminalFormatter()
	shortText := "Short"

	output := formatter.TruncateText(shortText, 20)

	if output != "Short" {
		t.Errorf("user should see full text when under limit, got: %s", output)
	}
}

func TestAC304_TerminalFeed_ShowsMultipleItems(t *testing.T) {
	items := []aggregator.FeedItem{
		{ID: "1", Title: "First Video", Author: "Author A", Source: aggregator.SourceYouTube, PublishedAt: time.Now()},
		{ID: "2", Title: "Second Video", Author: "Author B", Source: aggregator.SourceYouTube, PublishedAt: time.Now()},
	}

	output := NewTerminalFormatter().FormatFeed(items)

	if !strings.Contains(output, "First Video") {
		t.Error("user should see first video in feed")
	}
	if !strings.Contains(output, "Second Video") {
		t.Error("user should see second video in feed")
	}
}

func TestAC305_TerminalFeed_ShowsEmptyFeedMessage(t *testing.T) {
	output := NewTerminalFormatter().FormatFeed(nil)

	if !strings.Contains(strings.ToLower(output), "no") {
		t.Error("user should see message indicating no content available")
	}
}
