package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "feedmix-test")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	binaryPath = filepath.Join(dir, "feedmix")

	versionCmd := exec.Command("git", "describe", "--tags", "--always", "--dirty")
	versionOutput, err := versionCmd.Output()
	version := "dev"
	if err == nil {
		version = strings.TrimSpace(string(versionOutput))
	}

	ldflags := fmt.Sprintf("-X main.version=%s", version)
	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		panic("failed to build: " + err.Error())
	}

	os.Exit(m.Run())
}

// runCLI runs the feedmix binary with the given env and args.
// Explicit env values override inherited env; an empty string value unsets the var.
func runCLI(t *testing.T, env map[string]string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	for _, e := range os.Environ() {
		key := strings.SplitN(e, "=", 2)[0]
		if _, overridden := env[key]; !overridden {
			cmd.Env = append(cmd.Env, e)
		}
	}
	for k, v := range env {
		if v != "" {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	return outBuf.String(), errBuf.String(), exitCode
}

func TestRootCommand_Help(t *testing.T) {
	stdout, _, _ := runCLI(t, nil, "--help")
	if !strings.Contains(strings.ToLower(stdout), "feedmix") {
		t.Errorf("help should contain feedmix, got: %s", stdout)
	}
}

func TestRootCommand_HelpShowsVersion(t *testing.T) {
	helpOutput, _, _ := runCLI(t, nil, "--help")
	versionOutput, _, _ := runCLI(t, nil, "--version")

	versionLine := strings.TrimSpace(versionOutput)
	parts := strings.Fields(versionLine)
	if len(parts) < 3 {
		t.Fatalf("unexpected version output format: %s", versionOutput)
	}
	actualVersion := parts[2]

	if !strings.Contains(helpOutput, actualVersion) {
		t.Errorf("--help should display version %q (same as --version), but it's missing.\nHelp output: %s", actualVersion, helpOutput)
	}
}

func TestRootCommand_Version(t *testing.T) {
	stdout, _, _ := runCLI(t, nil, "--version")
	if !strings.Contains(stdout, "feedmix") {
		t.Errorf("version should show feedmix, got: %s", stdout)
	}
}

func TestFeedCommand_RequiresRefreshToken(t *testing.T) {
	_, stderr, exitCode := runCLI(t, map[string]string{"FEEDMIX_YOUTUBE_REFRESH_TOKEN": ""}, "feed")

	if exitCode == 0 {
		t.Error("feed should fail without refresh token")
	}
	if !strings.Contains(stderr, "FEEDMIX_YOUTUBE_REFRESH_TOKEN") {
		t.Errorf("error should tell user which env var to set, got: %s", stderr)
	}
}

func mockFeedServer(youtubeHandler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "test-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
			return
		}
		youtubeHandler(w, r)
	}))
}

func feedEnv(server *httptest.Server) map[string]string {
	return map[string]string{
		"FEEDMIX_YOUTUBE_REFRESH_TOKEN":  "test-refresh-token",
		"FEEDMIX_YOUTUBE_CLIENT_ID":      "test-id",
		"FEEDMIX_YOUTUBE_CLIENT_SECRET":  "test-secret",
		"FEEDMIX_OAUTH_TOKEN_URL":        server.URL,
		"FEEDMIX_API_URL":                server.URL,
	}
}

func TestFeedCommand_DisplaysItems(t *testing.T) {
	server := mockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"snippet": map[string]interface{}{
						"resourceId":  map[string]interface{}{"channelId": "UC123"},
						"title":       "Test Channel",
						"thumbnails":  map[string]interface{}{"default": map[string]interface{}{"url": "http://example.com/thumb.jpg"}},
						"publishedAt": "2024-01-01T00:00:00Z",
					},
				},
			},
		})
	})
	defer server.Close()

	stdout, _, exitCode := runCLI(t, feedEnv(server), "feed")
	if exitCode != 0 {
		t.Errorf("feed should succeed, got exit code %d", exitCode)
	}
	if !strings.Contains(stdout, "Test Channel") {
		t.Errorf("should show feed item, got: %s", stdout)
	}
}

func TestFeedCommand_AggregatesMultipleChannels(t *testing.T) {
	server := mockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/subscriptions") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{"snippet": map[string]interface{}{"resourceId": map[string]interface{}{"channelId": "UC_channel_A"}, "title": "Channel A", "thumbnails": map[string]interface{}{"default": map[string]interface{}{"url": ""}}, "publishedAt": "2024-01-01T00:00:00Z"}},
					{"snippet": map[string]interface{}{"resourceId": map[string]interface{}{"channelId": "UC_channel_B"}, "title": "Channel B", "thumbnails": map[string]interface{}{"default": map[string]interface{}{"url": ""}}, "publishedAt": "2024-01-01T00:00:00Z"}},
				},
			})
			return
		}

		channelID := r.URL.Query().Get("channelId")
		videoID := "vid_a"
		title := "Video from Channel A"
		if channelID == "UC_channel_B" {
			videoID = "vid_b"
			title = "Video from Channel B"
		}

		if strings.Contains(r.URL.Path, "/search") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": map[string]interface{}{"videoId": videoID}, "snippet": map[string]interface{}{"title": title, "channelId": channelID, "channelTitle": "Ch", "publishedAt": "2024-01-15T00:00:00Z", "thumbnails": map[string]interface{}{"default": map[string]interface{}{"url": ""}}}},
				},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": []interface{}{}})
	})
	defer server.Close()

	stdout, _, exitCode := runCLI(t, feedEnv(server), "feed")

	if exitCode != 0 {
		t.Fatalf("feed should succeed with multiple channels, exit code %d\noutput: %s", exitCode, stdout)
	}
	if !strings.Contains(stdout, "Video from Channel A") {
		t.Errorf("feed should include videos from Channel A, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Video from Channel B") {
		t.Errorf("feed should include videos from Channel B, got: %s", stdout)
	}
}

const substackRSSXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <channel>
    <title>Test Newsletter</title>
    <item>
      <title>My Substack Article</title>
      <link>https://testnewsletter.substack.com/p/my-article</link>
      <dc:creator>Test Author</dc:creator>
      <pubDate>Mon, 01 Jan 2024 12:00:00 +0000</pubDate>
      <description>An interesting article.</description>
      <guid>https://testnewsletter.substack.com/p/my-article</guid>
    </item>
  </channel>
</rss>`

// TestFeedCommand_ShowsSubstackItems documents Substack integration:
// - FEEDMIX_SUBSTACK_URLS set to a publication URL → posts appear in unified feed
func TestFeedCommand_ShowsSubstackItems(t *testing.T) {
	rssServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprint(w, substackRSSXML)
	}))
	defer rssServer.Close()

	youtubeServer := mockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": []interface{}{}})
	})
	defer youtubeServer.Close()

	env := feedEnv(youtubeServer)
	env["FEEDMIX_SUBSTACK_URLS"] = rssServer.URL

	stdout, stderr, exitCode := runCLI(t, env, "feed")
	if exitCode != 0 {
		t.Fatalf("feed should succeed with Substack, exit code %d\nstderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "My Substack Article") {
		t.Errorf("feed should display Substack article title, got: %s", stdout)
	}
}

// TestFeedCommand_WorksWithoutSubstack documents optional Substack integration:
// - FEEDMIX_SUBSTACK_URLS not set → feed runs normally, no error
func TestFeedCommand_WorksWithoutSubstack(t *testing.T) {
	server := mockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": []interface{}{}})
	})
	defer server.Close()

	env := feedEnv(server)
	env["FEEDMIX_SUBSTACK_URLS"] = ""

	_, stderr, exitCode := runCLI(t, env, "feed")
	if exitCode != 0 {
		t.Errorf("feed should succeed without Substack URLs, got exit code %d\nstderr: %s", exitCode, stderr)
	}
}

func TestConfigCommand_ShowsYouTubeStatusWhenSet(t *testing.T) {
	env := map[string]string{
		"FEEDMIX_YOUTUBE_CLIENT_ID":     "my-id",
		"FEEDMIX_YOUTUBE_CLIENT_SECRET": "my-secret",
		"FEEDMIX_YOUTUBE_REFRESH_TOKEN": "my-token",
	}
	stdout, _, exitCode := runCLI(t, env, "config")
	if exitCode != 0 {
		t.Fatalf("config should succeed, got exit code %d", exitCode)
	}
	if strings.Count(stdout, "✓") < 3 {
		t.Errorf("should show ✓ for all 3 YouTube credentials, got: %s", stdout)
	}
}

func TestConfigCommand_ShowsSetupInstructionsWhenCredsMissing(t *testing.T) {
	env := map[string]string{
		"FEEDMIX_YOUTUBE_CLIENT_ID":     "",
		"FEEDMIX_YOUTUBE_CLIENT_SECRET": "",
		"FEEDMIX_YOUTUBE_REFRESH_TOKEN": "",
	}
	stdout, _, exitCode := runCLI(t, env, "config")
	if exitCode != 0 {
		t.Fatalf("config should succeed even with no credentials, got exit code %d", exitCode)
	}
	if !strings.Contains(stdout, "console.cloud.google.com") {
		t.Errorf("should show Google Cloud Console URL, got: %s", stdout)
	}
	if !strings.Contains(stdout, "oauthplayground") {
		t.Errorf("should show OAuth Playground URL, got: %s", stdout)
	}
	if strings.Count(stdout, "✗") < 3 {
		t.Errorf("should show ✗ for all 3 missing YouTube credentials, got: %s", stdout)
	}
	if !strings.Contains(stdout, "export FEEDMIX_YOUTUBE_REFRESH_TOKEN") {
		t.Errorf("should show export syntax, not .env instructions, got: %s", stdout)
	}
	if !strings.Contains(stdout, "~/.bashrc") {
		t.Errorf("should reference shell config file for persistence, got: %s", stdout)
	}
}

func TestConfigCommand_ShowsSubstackWhenConfigured(t *testing.T) {
	env := map[string]string{"FEEDMIX_SUBSTACK_URLS": "https://simonwillison.substack.com"}
	stdout, _, exitCode := runCLI(t, env, "config")
	if exitCode != 0 {
		t.Fatalf("config should succeed, got exit code %d", exitCode)
	}
	if !strings.Contains(stdout, "simonwillison.substack.com") {
		t.Errorf("should show configured Substack URL, got: %s", stdout)
	}
}

func TestConfigCommand_ShowsSubstackSetupWhenNotConfigured(t *testing.T) {
	env := map[string]string{"FEEDMIX_SUBSTACK_URLS": ""}
	stdout, _, exitCode := runCLI(t, env, "config")
	if exitCode != 0 {
		t.Fatalf("config should succeed, got exit code %d", exitCode)
	}
	if !strings.Contains(stdout, "FEEDMIX_SUBSTACK_URLS") {
		t.Errorf("should show FEEDMIX_SUBSTACK_URLS env var name, got: %s", stdout)
	}
	if !strings.Contains(stdout, "export FEEDMIX_SUBSTACK_URLS") {
		t.Errorf("should show export syntax, not .env instructions, got: %s", stdout)
	}
	if !strings.Contains(stdout, "~/.bashrc") {
		t.Errorf("should reference shell config file for persistence, got: %s", stdout)
	}
}

func TestFeedCommand_ErrorMentionsConfigCommand(t *testing.T) {
	_, stderr, exitCode := runCLI(t, map[string]string{"FEEDMIX_YOUTUBE_REFRESH_TOKEN": ""}, "feed")
	if exitCode == 0 {
		t.Error("feed should fail without refresh token")
	}
	if !strings.Contains(stderr, "feedmix config") {
		t.Errorf("error should mention 'feedmix config', got: %s", stderr)
	}
}

func TestFeedCommand_DisplaysVideoURLs(t *testing.T) {
	server := mockFeedServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/subscriptions") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"snippet": map[string]interface{}{
							"resourceId":  map[string]interface{}{"channelId": "UCxYz123ABC"},
							"title":       "Tech Channel",
							"thumbnails":  map[string]interface{}{"default": map[string]interface{}{"url": "http://example.com/thumb.jpg"}},
							"publishedAt": "2024-01-01T00:00:00Z",
						},
					},
				},
			})
			return
		}

		if strings.Contains(r.URL.Path, "/search") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"id": map[string]interface{}{"videoId": "dQw4w9WgXcQ"},
						"snippet": map[string]interface{}{
							"title":        "Amazing Video",
							"description":  "Great content",
							"channelId":    "UCxYz123ABC",
							"channelTitle": "Tech Channel",
							"publishedAt":  "2024-01-15T12:00:00Z",
							"thumbnails":   map[string]interface{}{"default": map[string]interface{}{"url": "http://example.com/video-thumb.jpg"}},
						},
					},
				},
			})
			return
		}

		if strings.Contains(r.URL.Path, "/videos") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"id": "dQw4w9WgXcQ",
						"statistics": map[string]interface{}{
							"viewCount": "1000000",
							"likeCount": "50000",
						},
						"contentDetails": map[string]interface{}{
							"duration": "PT3M30S",
						},
					},
				},
			})
		}
	})
	defer server.Close()

	stdout, _, exitCode := runCLI(t, feedEnv(server), "feed")
	if exitCode != 0 {
		t.Errorf("feed should succeed, got exit code %d", exitCode)
	}

	expectedVideoURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	if !strings.Contains(stdout, expectedVideoURL) {
		t.Errorf("feed should display video URL %q, got: %s", expectedVideoURL, stdout)
	}

	channelURL := "https://youtube.com/channel/UCxYz123ABC"
	if strings.Contains(stdout, channelURL) {
		t.Errorf("feed should NOT display channel URL %q (should show videos instead), got: %s", channelURL, stdout)
	}
}
