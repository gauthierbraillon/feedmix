// Package main provides the feedmix CLI entry point.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/gauthierbraillon/feedmix/internal/aggregator"
	"github.com/gauthierbraillon/feedmix/internal/display"
	"github.com/gauthierbraillon/feedmix/internal/youtube"
	"github.com/gauthierbraillon/feedmix/pkg/oauth"
)

// version is set via ldflags at build time:
//
//	go build -ldflags="-X main.version=$(git describe --tags --always --dirty)"
//
// OR automatically from build info when installed via: go install github.com/user/repo/cmd/tool@v1.2.3
var version = "dev"

// clientID and clientSecret are embedded at build time via ldflags:
//
//	go build -ldflags="-X main.clientID=$ID -X main.clientSecret=$SECRET"
//
// Environment variables FEEDMIX_YOUTUBE_CLIENT_ID / FEEDMIX_YOUTUBE_CLIENT_SECRET take priority.
var (
	clientID     string
	clientSecret string
)

func init() {
	// Resolve actual version (ldflags or build info)
	buildInfo, _ := debug.ReadBuildInfo()
	version = resolveVersion(version, buildInfo)
}

// resolveVersion determines the correct version to use.
// Priority: 1) ldflags version, 2) build info version, 3) "dev"
func resolveVersion(ldflagsVersion string, buildInfo *debug.BuildInfo) string {
	// If version was set via ldflags (not "dev"), use it
	if ldflagsVersion != "dev" {
		return ldflagsVersion
	}

	// Try to get version from build info (set by go install)
	if buildInfo != nil && buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		return buildInfo.Main.Version
	}

	// Fall back to "dev"
	return "dev"
}

func main() {
	// Load .env file if it exists (silently ignore if not found)
	_ = godotenv.Load()

	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func getConfigDir() string {
	if dir := os.Getenv("FEEDMIX_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "feedmix")
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "feedmix",
		Short:   "Aggregate feeds from YouTube",
		Long:    fmt.Sprintf("Feedmix aggregates your YouTube subscriptions into a unified feed.\n\nVersion: %s", version),
		Version: version,
	}

	rootCmd.SetVersionTemplate("feedmix version {{.Version}}\n")
	rootCmd.AddCommand(newFeedCmd())
	rootCmd.AddCommand(newConfigCmd())

	return rootCmd
}

func newFeedCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "feed",
		Short: "Display YouTube feed",
		Long:  "Display your YouTube subscriptions feed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			refreshToken := os.Getenv("FEEDMIX_YOUTUBE_REFRESH_TOKEN")
			if refreshToken == "" {
				return fmt.Errorf("missing credentials: set FEEDMIX_YOUTUBE_REFRESH_TOKEN")
			}

			id := os.Getenv("FEEDMIX_YOUTUBE_CLIENT_ID")
			if id == "" {
				id = clientID
			}
			secret := os.Getenv("FEEDMIX_YOUTUBE_CLIENT_SECRET")
			if secret == "" {
				secret = clientSecret
			}

			config := oauth.YouTubeOAuthConfig(id, secret)
			if tokenURL := os.Getenv("FEEDMIX_OAUTH_TOKEN_URL"); tokenURL != "" {
				config.TokenURL = tokenURL
			}

			token, err := oauth.NewFlow(config).RefreshAccessToken(ctx, refreshToken)
			if err != nil {
				return fmt.Errorf("failed to refresh token: %w", err)
			}

			opts := []youtube.ClientOption{}
			if apiURL := os.Getenv("FEEDMIX_API_URL"); apiURL != "" {
				opts = append(opts, youtube.WithBaseURL(apiURL))
			}
			client := youtube.NewClient(token, opts...)

			subs, err := client.FetchSubscriptions(ctx)
			if err != nil {
				return err
			}

			agg := aggregator.New()
			var mu sync.Mutex
			var wg sync.WaitGroup
			for _, sub := range subs {
				wg.Add(1)
				go func(sub youtube.Subscription) {
					defer wg.Done()
					videos, err := client.FetchRecentVideos(ctx, sub.ChannelID, 5)
					if err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to fetch videos from %s: %v\n", sub.ChannelTitle, err)
						return
					}
					items := make([]aggregator.FeedItem, 0, len(videos))
					for _, video := range videos {
						items = append(items, aggregator.FeedItem{
							ID:          video.ID,
							Source:      aggregator.SourceYouTube,
							Type:        aggregator.ItemTypeVideo,
							Title:       video.Title,
							Description: video.Description,
							Author:      video.ChannelTitle,
							AuthorID:    video.ChannelID,
							URL:         video.URL,
							Thumbnail:   video.Thumbnail,
							PublishedAt: video.PublishedAt,
							Engagement: aggregator.Engagement{
								Views: video.ViewCount,
								Likes: video.LikeCount,
							},
						})
					}
					mu.Lock()
					agg.AddItems(items)
					mu.Unlock()
				}(sub)
			}
			wg.Wait()

			items := agg.GetFeed(aggregator.FeedOptions{Limit: limit})
			formatter := display.NewTerminalFormatter()
			fmt.Fprint(cmd.OutOrStdout(), formatter.FormatFeed(items))

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum items to display")
	return cmd
}

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "Config directory: %s\n", getConfigDir())
			return nil
		},
	}
}
