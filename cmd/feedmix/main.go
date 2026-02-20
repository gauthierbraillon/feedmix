// Package main provides the feedmix CLI entry point.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/gauthierbraillon/feedmix/internal/aggregator"
	"github.com/gauthierbraillon/feedmix/internal/display"
	"github.com/gauthierbraillon/feedmix/internal/substack"
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
		Short:   "Aggregate feeds from YouTube and Substack",
		Long:    fmt.Sprintf("Feedmix aggregates your YouTube subscriptions and Substack newsletters into a unified feed.\n\nVersion: %s", version),
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
		Short: "Display unified feed",
		Long:  "Display your YouTube subscriptions and Substack newsletters in a unified feed.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			refreshToken := os.Getenv("FEEDMIX_YOUTUBE_REFRESH_TOKEN")
			if refreshToken == "" {
				return fmt.Errorf("missing credentials: set FEEDMIX_YOUTUBE_REFRESH_TOKEN (run 'feedmix config' for setup instructions)")
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

			substackURLs := parseSubstackURLs(os.Getenv("FEEDMIX_SUBSTACK_URLS"))
			if len(substackURLs) > 0 {
				substackClient := substack.NewClient()
				var substackMu sync.Mutex
				var substackWg sync.WaitGroup
				for _, pubURL := range substackURLs {
					substackWg.Add(1)
					go func(pubURL string) {
						defer substackWg.Done()
						posts, err := substackClient.FetchPosts(ctx, pubURL, 5)
						if err != nil {
							fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to fetch Substack feed from %s: %v\n", pubURL, err)
							return
						}
						items := make([]aggregator.FeedItem, 0, len(posts))
						for _, post := range posts {
							items = append(items, aggregator.FeedItem{
								ID:          post.ID,
								Source:      aggregator.SourceSubstack,
								Type:        aggregator.ItemTypeArticle,
								Title:       post.Title,
								Description: post.Description,
								Author:      post.Author,
								URL:         post.URL,
								PublishedAt: post.PublishedAt,
							})
						}
						substackMu.Lock()
						agg.AddItems(items)
						substackMu.Unlock()
					}(pubURL)
				}
				substackWg.Wait()
			}

			items := agg.GetFeed(aggregator.FeedOptions{Limit: limit})
			formatter := display.NewTerminalFormatter()
			fmt.Fprint(cmd.OutOrStdout(), formatter.FormatFeed(items))

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum items to display")
	return cmd
}

func credStatus(val string) string {
	if val != "" {
		return "✓ set"
	}
	return "✗ not set"
}

func resolveCredential(envVal, embedded string) string {
	if envVal != "" {
		return envVal
	}
	return embedded
}

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show configuration and setup instructions",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Configuration directory: %s\n\n", getConfigDir())

			ytID := resolveCredential(os.Getenv("FEEDMIX_YOUTUBE_CLIENT_ID"), clientID)
			ytSecret := resolveCredential(os.Getenv("FEEDMIX_YOUTUBE_CLIENT_SECRET"), clientSecret)
			ytToken := os.Getenv("FEEDMIX_YOUTUBE_REFRESH_TOKEN")

			fmt.Fprintf(out, "YouTube (required)\n")
			fmt.Fprintf(out, "  FEEDMIX_YOUTUBE_CLIENT_ID      %s\n", credStatus(ytID))
			fmt.Fprintf(out, "  FEEDMIX_YOUTUBE_CLIENT_SECRET  %s\n", credStatus(ytSecret))
			fmt.Fprintf(out, "  FEEDMIX_YOUTUBE_REFRESH_TOKEN  %s\n", credStatus(ytToken))

			if ytID == "" || ytSecret == "" || ytToken == "" {
				fmt.Fprint(out, "\n  To get credentials:\n")
				fmt.Fprint(out, "    1. Create OAuth credentials (Desktop app):\n")
				fmt.Fprint(out, "       https://console.cloud.google.com/apis/credentials\n")
				fmt.Fprint(out, "    2. Enable YouTube Data API v3:\n")
				fmt.Fprint(out, "       https://console.cloud.google.com/apis/library\n")
				fmt.Fprint(out, "    3. Get a refresh token:\n")
				fmt.Fprint(out, "       https://developers.google.com/oauthplayground\n")
				fmt.Fprint(out, "       • Gear icon → Use your own OAuth credentials → enter Client ID + Secret\n")
				fmt.Fprint(out, "       • Select scope: https://www.googleapis.com/auth/youtube.readonly\n")
				fmt.Fprint(out, "       • Authorize APIs → Exchange authorization code → copy Refresh token\n")
				fmt.Fprint(out, "    4. Export in your shell (add to ~/.bashrc or ~/.zshrc):\n")
				if ytID == "" {
					fmt.Fprint(out, "       export FEEDMIX_YOUTUBE_CLIENT_ID=<client-id>\n")
				}
				if ytSecret == "" {
					fmt.Fprint(out, "       export FEEDMIX_YOUTUBE_CLIENT_SECRET=<client-secret>\n")
				}
				if ytToken == "" {
					fmt.Fprint(out, "       export FEEDMIX_YOUTUBE_REFRESH_TOKEN=<refresh-token>\n")
				}
			}

			substackURLs := parseSubstackURLs(os.Getenv("FEEDMIX_SUBSTACK_URLS"))
			fmt.Fprint(out, "\nSubstack (optional)\n")
			if len(substackURLs) == 0 {
				fmt.Fprint(out, "  FEEDMIX_SUBSTACK_URLS  ✗ not configured\n")
				fmt.Fprint(out, "\n  Set to a comma-separated list of Substack publications.\n")
				fmt.Fprint(out, "  Both URL formats are accepted:\n")
				fmt.Fprint(out, "    https://example.substack.com      (subdomain)\n")
				fmt.Fprint(out, "    https://substack.com/@example      (@username)\n")
				fmt.Fprint(out, "\n  To persist across sessions, add to your shell config:\n")
				fmt.Fprint(out, "    # bash\n")
				fmt.Fprint(out, "    echo 'export FEEDMIX_SUBSTACK_URLS=https://example.substack.com' >> ~/.bashrc\n")
				fmt.Fprint(out, "    # zsh\n")
				fmt.Fprint(out, "    echo 'export FEEDMIX_SUBSTACK_URLS=https://example.substack.com' >> ~/.zshrc\n")
			} else {
				fmt.Fprintf(out, "  FEEDMIX_SUBSTACK_URLS  ✓ %d configured\n", len(substackURLs))
				for _, u := range substackURLs {
					fmt.Fprintf(out, "    • %s\n", u)
				}
			}
			return nil
		},
	}
}

func parseSubstackURLs(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	urls := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			urls = append(urls, p)
		}
	}
	return urls
}
