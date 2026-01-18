// Package aggregator combines feeds from multiple sources into a unified view.
package aggregator

import "sort"

// Aggregator collects and merges feed items from multiple sources.
type Aggregator struct {
	items []FeedItem
}

// New creates a new Aggregator instance.
func New() *Aggregator {
	return &Aggregator{
		items: make([]FeedItem, 0),
	}
}

// AddItems adds feed items to the aggregator.
func (a *Aggregator) AddItems(items []FeedItem) {
	a.items = append(a.items, items...)
}

// GetFeed returns aggregated feed items based on options.
func (a *Aggregator) GetFeed(opts FeedOptions) []FeedItem {
	// Start with all items
	result := make([]FeedItem, 0, len(a.items))

	for _, item := range a.items {
		// Apply source filter
		if len(opts.Sources) > 0 && !containsSource(opts.Sources, item.Source) {
			continue
		}

		// Apply type filter
		if len(opts.Types) > 0 && !containsType(opts.Types, item.Type) {
			continue
		}

		// Apply date filters
		if !opts.Since.IsZero() && item.PublishedAt.Before(opts.Since) {
			continue
		}
		if !opts.Until.IsZero() && item.PublishedAt.After(opts.Until) {
			continue
		}

		result = append(result, item)
	}

	// Sort by PublishedAt descending (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].PublishedAt.After(result[j].PublishedAt)
	})

	// Apply limit
	if opts.Limit > 0 && len(result) > opts.Limit {
		result = result[:opts.Limit]
	}

	return result
}

func containsSource(sources []Source, source Source) bool {
	for _, s := range sources {
		if s == source {
			return true
		}
	}
	return false
}

func containsType(types []ItemType, itemType ItemType) bool {
	for _, t := range types {
		if t == itemType {
			return true
		}
	}
	return false
}
