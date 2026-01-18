// Package display provides terminal output formatting for feedmix.
package display

import (
	"fmt"
	"strings"
	"time"

	"feedmix/internal/aggregator"
)

const separator = " • "

// TerminalFormatter formats feed items for terminal display.
type TerminalFormatter struct{}

// NewTerminalFormatter creates a new terminal formatter.
func NewTerminalFormatter() *TerminalFormatter {
	return &TerminalFormatter{}
}

// FormatItem formats a single feed item for display.
func (f *TerminalFormatter) FormatItem(item aggregator.FeedItem) string {
	var lines []string

	// Header: [SOURCE] Title
	header := fmt.Sprintf("[%s] %s", strings.ToUpper(string(item.Source)), item.Title)
	lines = append(lines, header)

	// Author and timestamp
	meta := fmt.Sprintf("  by %s%s%s", item.Author, separator, f.FormatTimestamp(item.PublishedAt))
	lines = append(lines, meta)

	// Engagement stats (if any)
	if engagement := f.formatEngagement(item.Engagement); engagement != "" {
		lines = append(lines, "  "+engagement)
	}

	// URL
	if item.URL != "" {
		lines = append(lines, "  "+item.URL)
	}

	return strings.Join(lines, "\n") + "\n"
}

// formatEngagement formats engagement stats into a single line.
func (f *TerminalFormatter) formatEngagement(e aggregator.Engagement) string {
	var parts []string

	if e.Views > 0 {
		parts = append(parts, fmt.Sprintf("%d views", e.Views))
	}
	if e.Likes > 0 {
		parts = append(parts, fmt.Sprintf("%d likes", e.Likes))
	}
	if e.Comments > 0 {
		parts = append(parts, fmt.Sprintf("%d comments", e.Comments))
	}

	return strings.Join(parts, separator)
}

// FormatFeed formats multiple feed items for display.
func (f *TerminalFormatter) FormatFeed(items []aggregator.FeedItem) string {
	if len(items) == 0 {
		return "No items to display.\n"
	}

	var formatted []string
	for _, item := range items {
		formatted = append(formatted, f.FormatItem(item))
	}

	return strings.Join(formatted, "\n---\n\n")
}

// FormatTimestamp formats a timestamp as relative time.
func (f *TerminalFormatter) FormatTimestamp(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return pluralize(int(diff.Minutes()), "minute")
	case diff < 24*time.Hour:
		return pluralize(int(diff.Hours()), "hour")
	case diff < 7*24*time.Hour:
		return pluralize(int(diff.Hours()/24), "day")
	default:
		return t.Format("Jan 2, 2006")
	}
}

// pluralize returns "N unit ago" or "N units ago" based on count.
func pluralize(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s ago", unit)
	}
	return fmt.Sprintf("%d %ss ago", n, unit)
}

// TruncateText truncates text to maxLen, adding "..." if truncated.
func (f *TerminalFormatter) TruncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return "..."
	}
	return text[:maxLen-3] + "..."
}
