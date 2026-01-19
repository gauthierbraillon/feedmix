// Package main provides the feedmix CLI entry point.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/gauthierbraillon/feedmix/internal/aggregator"
	"github.com/gauthierbraillon/feedmix/internal/display"
	"github.com/gauthierbraillon/feedmix/internal/youtube"
	"github.com/gauthierbraillon/feedmix/pkg/browser"
	"github.com/gauthierbraillon/feedmix/pkg/oauth"
)

var version = "0.1.0"

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
		Long:    "Feedmix aggregates your YouTube subscriptions into a unified feed.",
		Version: version,
	}

	rootCmd.SetVersionTemplate("feedmix version {{.Version}}\n")
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newFeedCmd())
	rootCmd.AddCommand(newConfigCmd())

	return rootCmd
}

func newAuthCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with YouTube",
		Long:  "Initiate OAuth authentication flow for YouTube.",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientID := os.Getenv("FEEDMIX_YOUTUBE_CLIENT_ID")
			clientSecret := os.Getenv("FEEDMIX_YOUTUBE_CLIENT_SECRET")

			if clientID == "" || clientSecret == "" {
				return fmt.Errorf("missing credentials: set FEEDMIX_YOUTUBE_CLIENT_ID and FEEDMIX_YOUTUBE_CLIENT_SECRET")
			}

			redirectURL := fmt.Sprintf("http://localhost:%d/callback", port)
			config := oauth.YouTubeOAuthConfig(clientID, clientSecret, redirectURL)
			flow := oauth.NewFlow(config)
			authURL, state := flow.GenerateAuthURL()

			fmt.Fprintln(cmd.OutOrStdout(), "Opening browser for authorization...")

			if err := browser.Open(authURL); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Visit: %s\n", authURL)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Waiting for authorization...")
			callbackServer := oauth.NewCallbackServer(port)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			code, err := callbackServer.WaitForCallback(ctx, state, 5*time.Minute)
			if err != nil {
				return fmt.Errorf("authorization failed: %w", err)
			}

			token, err := flow.ExchangeCode(ctx, code)
			if err != nil {
				return fmt.Errorf("token exchange failed: %w", err)
			}

			storage := oauth.NewTokenStorage(getConfigDir())
			if err := storage.Save("youtube", token); err != nil {
				return fmt.Errorf("failed to save token: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Successfully authenticated!")
			return nil
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port for OAuth callback server")
	return cmd
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

			storage := oauth.NewTokenStorage(getConfigDir())
			token, err := storage.Load("youtube")
			if err != nil {
				return fmt.Errorf("not authenticated (run 'feedmix auth')")
			}

			opts := []youtube.ClientOption{}
			if url := os.Getenv("FEEDMIX_API_URL"); url != "" {
				opts = append(opts, youtube.WithBaseURL(url))
			}
			client := youtube.NewClient(token, opts...)

			subs, err := client.FetchSubscriptions(ctx)
			if err != nil {
				return err
			}

			agg := aggregator.New()
			for _, sub := range subs {
				agg.AddItems([]aggregator.FeedItem{{
					ID:          sub.ChannelID,
					Source:      aggregator.SourceYouTube,
					Type:        aggregator.ItemTypeVideo,
					Title:       sub.ChannelTitle,
					Description: sub.Description,
					Author:      sub.ChannelTitle,
					AuthorID:    sub.ChannelID,
					URL:         fmt.Sprintf("https://youtube.com/channel/%s", sub.ChannelID),
					Thumbnail:   sub.Thumbnail,
					PublishedAt: sub.SubscribedAt,
				}})
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
