// Package display tests document the expected behavior of terminal output formatting.
//
// TDD Cycle: RED -> GREEN -> REFACTOR
//
// Test requirements (this file serves as documentation):
// - Feed items are formatted for terminal display
// - Different item types have appropriate formatting
// - Timestamps are human-readable
// - Long text is truncated appropriately
// - Colors/styles are applied (when supported)
package display

import (
	"strings"
	"testing"
	"time"

	"feedmix/internal/aggregator"
)

// TestFormatFeedItem documents basic item formatting:
// - Title is displayed prominently
// - Source and type are indicated
// - Timestamp is human-readable
// - URL is included
func TestFormatFeedItem(t *testing.T) {
	item := aggregator.FeedItem{
		ID:          "test123",
		Source:      aggregator.SourceYouTube,
		Type:        aggregator.ItemTypeVideo,
		Title:       "Test Video Title",
		Author:      "Test Author",
		URL:         "https://youtube.com/watch?v=test123",
		PublishedAt: time.Now().Add(-2 * time.Hour),
		Engagement: aggregator.Engagement{
			Views: 1000,
			Likes: 50,
		},
	}

	formatter := NewTerminalFormatter()
	output := formatter.FormatItem(item)

	// Should contain title
	if !strings.Contains(output, "Test Video Title") {
		t.Error("output should contain title")
	}

	// Should indicate source
	if !strings.Contains(strings.ToLower(output), "youtube") {
		t.Error("output should indicate YouTube source")
	}

	// Should contain author
	if !strings.Contains(output, "Test Author") {
		t.Error("output should contain author")
	}

	// Should contain URL
	if !strings.Contains(output, "https://youtube.com/watch?v=test123") {
		t.Error("output should contain URL")
	}
}

// TestFormatFeedItem_LinkedIn documents LinkedIn post formatting:
// - Shows post text instead of title
// - Shows engagement (likes, comments, shares)
func TestFormatFeedItem_LinkedIn(t *testing.T) {
	item := aggregator.FeedItem{
		ID:          "li123",
		Source:      aggregator.SourceLinkedIn,
		Type:        aggregator.ItemTypePost,
		Title:       "LinkedIn Post",
		Description: "This is the post content that should be displayed",
		Author:      "Jane Doe",
		PublishedAt: time.Now().Add(-1 * time.Hour),
		Engagement: aggregator.Engagement{
			Likes:    25,
			Comments: 10,
			Shares:   5,
		},
	}

	formatter := NewTerminalFormatter()
	output := formatter.FormatItem(item)

	// Should indicate LinkedIn
	if !strings.Contains(strings.ToLower(output), "linkedin") {
		t.Error("output should indicate LinkedIn source")
	}

	// Should show engagement
	if !strings.Contains(output, "25") { // likes
		t.Error("output should show like count")
	}
}

// TestFormatTimestamp documents relative time formatting:
// - Recent times show "X minutes ago", "X hours ago"
// - Older times show date
func TestFormatTimestamp(t *testing.T) {
	formatter := NewTerminalFormatter()

	tests := []struct {
		name     string
		time     time.Time
		contains string
	}{
		{
			name:     "minutes ago",
			time:     time.Now().Add(-30 * time.Minute),
			contains: "min",
		},
		{
			name:     "hours ago",
			time:     time.Now().Add(-3 * time.Hour),
			contains: "hour",
		},
		{
			name:     "days ago",
			time:     time.Now().Add(-2 * 24 * time.Hour),
			contains: "day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatTimestamp(tt.time)
			if !strings.Contains(strings.ToLower(result), tt.contains) {
				t.Errorf("expected %q to contain %q", result, tt.contains)
			}
		})
	}
}

// TestTruncateText documents text truncation:
// - Text longer than max length is truncated
// - Truncated text ends with "..."
// - Short text is not modified
func TestTruncateText(t *testing.T) {
	formatter := NewTerminalFormatter()

	tests := []struct {
		name      string
		input     string
		maxLen    int
		expected  string
		truncated bool
	}{
		{
			name:      "short text unchanged",
			input:     "Hello",
			maxLen:    10,
			expected:  "Hello",
			truncated: false,
		},
		{
			name:      "long text truncated",
			input:     "This is a very long text that should be truncated",
			maxLen:    20,
			truncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.TruncateText(tt.input, tt.maxLen)

			if tt.truncated {
				if len(result) > tt.maxLen {
					t.Errorf("result length %d exceeds max %d", len(result), tt.maxLen)
				}
				if !strings.HasSuffix(result, "...") {
					t.Error("truncated text should end with ...")
				}
			} else {
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

// TestFormatFeed documents full feed formatting:
// - Multiple items are formatted with separators
// - Empty feed produces appropriate message
func TestFormatFeed(t *testing.T) {
	items := []aggregator.FeedItem{
		{
			ID:          "item1",
			Source:      aggregator.SourceYouTube,
			Type:        aggregator.ItemTypeVideo,
			Title:       "First Video",
			Author:      "Author 1",
			PublishedAt: time.Now(),
		},
		{
			ID:          "item2",
			Source:      aggregator.SourceLinkedIn,
			Type:        aggregator.ItemTypePost,
			Title:       "Second Post",
			Author:      "Author 2",
			PublishedAt: time.Now(),
		},
	}

	formatter := NewTerminalFormatter()
	output := formatter.FormatFeed(items)

	// Should contain both items
	if !strings.Contains(output, "First Video") {
		t.Error("output should contain first item")
	}
	if !strings.Contains(output, "Second Post") {
		t.Error("output should contain second item")
	}
}

// TestFormatFeed_Empty documents empty feed handling:
// - Returns appropriate "no items" message
func TestFormatFeed_Empty(t *testing.T) {
	formatter := NewTerminalFormatter()
	output := formatter.FormatFeed([]aggregator.FeedItem{})

	if output == "" {
		t.Error("empty feed should produce a message, not empty string")
	}

	if !strings.Contains(strings.ToLower(output), "no") {
		t.Error("empty feed message should indicate no items")
	}
}
