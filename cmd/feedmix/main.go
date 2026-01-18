// Package main provides the feedmix CLI entry point.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"feedmix/internal/aggregator"
	"feedmix/internal/display"
	"feedmix/internal/youtube"
	"feedmix/pkg/browser"
	"feedmix/pkg/oauth"
)

var version = "0.1.0"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// getConfigDir returns the configuration directory path.
func getConfigDir() string {
	if dir := os.Getenv("FEEDMIX_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "feedmix")
}

// getAPIURL returns the API base URL (overridable for testing).
func getAPIURL(provider string) string {
	if url := os.Getenv("FEEDMIX_API_URL"); url != "" {
		return url
	}
	if provider == "youtube" {
		return "https://www.googleapis.com"
	}
	return "https://api.linkedin.com"
}

// newRootCmd creates the root command for feedmix CLI.
func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "feedmix",
		Short:   "Aggregate feeds from YouTube and LinkedIn",
		Long:    "Feedmix aggregates your social feeds from YouTube and LinkedIn into a unified view.",
		Version: version,
	}

	rootCmd.SetVersionTemplate("feedmix version {{.Version}}\n")

	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newFeedCmd())
	rootCmd.AddCommand(newConfigCmd())

	return rootCmd
}

// newAuthCmd creates the auth subcommand.
func newAuthCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "auth <provider>",
		Short: "Authenticate with a provider (youtube or linkedin)",
		Long:  "Initiate OAuth authentication flow for YouTube or LinkedIn.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := args[0]
			if provider != "youtube" && provider != "linkedin" {
				return fmt.Errorf("invalid provider %q: must be 'youtube' or 'linkedin'", provider)
			}

			// Get OAuth credentials from environment
			providerUpper := strings.ToUpper(provider)
			clientID := os.Getenv("FEEDMIX_" + providerUpper + "_CLIENT_ID")
			clientSecret := os.Getenv("FEEDMIX_" + providerUpper + "_CLIENT_SECRET")

			if clientID == "" || clientSecret == "" {
				return fmt.Errorf("missing credentials: set FEEDMIX_%s_CLIENT_ID and FEEDMIX_%s_CLIENT_SECRET environment variables", providerUpper, providerUpper)
			}

			redirectURL := fmt.Sprintf("http://localhost:%d/callback", port)

			// Get OAuth config for provider
			var config oauth.Config
			switch provider {
			case "youtube":
				config = oauth.YouTubeOAuthConfig(clientID, clientSecret, redirectURL)
			case "linkedin":
				config = oauth.LinkedInOAuthConfig(clientID, clientSecret, redirectURL)
			}

			// Create OAuth flow
			flow := oauth.NewFlow(config)
			authURL, state := flow.GenerateAuthURL()

			fmt.Fprintf(cmd.OutOrStdout(), "Authenticating with %s...\n", provider)
			fmt.Fprintf(cmd.OutOrStdout(), "Opening browser for authorization...\n")

			// Open browser
			if err := browser.Open(authURL); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Could not open browser. Please visit:\n%s\n", authURL)
			}

			// Start callback server
			fmt.Fprintf(cmd.OutOrStdout(), "Waiting for authorization...\n")
			callbackServer := oauth.NewCallbackServer(port)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			code, err := callbackServer.WaitForCallback(ctx, state, 5*time.Minute)
			if err != nil {
				return fmt.Errorf("authorization failed: %w", err)
			}

			// Exchange code for token
			fmt.Fprintf(cmd.OutOrStdout(), "Exchanging authorization code...\n")
			token, err := flow.ExchangeCode(ctx, code)
			if err != nil {
				return fmt.Errorf("token exchange failed: %w", err)
			}

			// Save token
			configDir := getConfigDir()
			storage := oauth.NewTokenStorage(configDir)
			if err := storage.Save(provider, token); err != nil {
				return fmt.Errorf("failed to save token: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Successfully authenticated with %s!\n", provider)
			fmt.Fprintf(cmd.OutOrStdout(), "Token saved to: %s\n", configDir)
			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port for OAuth callback server")

	return cmd
}

// newFeedCmd creates the feed subcommand.
func newFeedCmd() *cobra.Command {
	var source string
	var limit int
	var itemType string

	cmd := &cobra.Command{
		Use:   "feed",
		Short: "Display aggregated feed",
		Long:  "Display your aggregated feed from YouTube and LinkedIn.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			configDir := getConfigDir()
			storage := oauth.NewTokenStorage(configDir)
			agg := aggregator.New()

			// Fetch from YouTube if no source filter or source is youtube
			if source == "" || source == "youtube" {
				items, err := fetchYouTubeItems(ctx, storage)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "YouTube: %v\n", err)
				} else {
					agg.AddItems(items)
				}
			}

			// Get aggregated feed
			opts := aggregator.FeedOptions{Limit: limit}
			if source != "" {
				opts.Sources = []aggregator.Source{aggregator.Source(source)}
			}
			items := agg.GetFeed(opts)

			// Display
			formatter := display.NewTerminalFormatter()
			output := formatter.FormatFeed(items)
			fmt.Fprint(cmd.OutOrStdout(), output)

			return nil
		},
	}

	cmd.Flags().StringVarP(&source, "source", "s", "", "Filter by source (youtube, linkedin)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of items to display")
	cmd.Flags().StringVarP(&itemType, "type", "t", "", "Filter by type (video, post, like)")

	return cmd
}

// fetchYouTubeItems fetches feed items from YouTube.
func fetchYouTubeItems(ctx context.Context, storage *oauth.TokenStorage) ([]aggregator.FeedItem, error) {
	token, err := storage.Load("youtube")
	if err != nil {
		return nil, fmt.Errorf("not authenticated (run 'feedmix auth youtube')")
	}

	opts := []youtube.ClientOption{}
	if url := os.Getenv("FEEDMIX_API_URL"); url != "" {
		opts = append(opts, youtube.WithBaseURL(url))
	}
	client := youtube.NewClient(token, opts...)

	subs, err := client.FetchSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	var items []aggregator.FeedItem
	for _, sub := range subs {
		items = append(items, aggregator.FeedItem{
			ID:          sub.ChannelID,
			Source:      aggregator.SourceYouTube,
			Type:        aggregator.ItemTypeVideo,
			Title:       sub.ChannelTitle,
			Description: sub.Description,
			Author:      sub.ChannelTitle,
			AuthorID:    sub.ChannelID,
			Thumbnail:   sub.Thumbnail,
			PublishedAt: sub.SubscribedAt,
		})
	}

	return items, nil
}

// newConfigCmd creates the config subcommand.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "View or modify feedmix configuration settings.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "Config directory: %s\n", getConfigDir())
			return nil
		},
	}

	return cmd
}
