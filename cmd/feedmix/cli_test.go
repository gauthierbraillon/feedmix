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
	defer os.RemoveAll(dir)

	binaryPath = filepath.Join(dir, "feedmix")

	// Get version from git for build
	versionCmd := exec.Command("git", "describe", "--tags", "--always", "--dirty")
	versionOutput, err := versionCmd.Output()
	version := "dev"
	if err == nil {
		version = strings.TrimSpace(string(versionOutput))
	}

	// Build with version injected via ldflags
	ldflags := fmt.Sprintf("-X main.version=%s", version)
	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		panic("failed to build: " + err.Error())
	}

	os.Exit(m.Run())
}

func runCLI(t *testing.T, env map[string]string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
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
	for _, want := range []string{"feedmix", "auth", "feed"} {
		if !strings.Contains(strings.ToLower(stdout), want) {
			t.Errorf("help should contain %q", want)
		}
	}
}

func TestRootCommand_HelpShowsVersion(t *testing.T) {
	helpOutput, _, _ := runCLI(t, nil, "--help")
	versionOutput, _, _ := runCLI(t, nil, "--version")

	// Extract version from --version output (e.g., "feedmix version v0.2.0")
	versionLine := strings.TrimSpace(versionOutput)
	parts := strings.Fields(versionLine)
	if len(parts) < 3 {
		t.Fatalf("unexpected version output format: %s", versionOutput)
	}
	actualVersion := parts[2] // "feedmix version v0.2.0" -> "v0.2.0"

	// Requirement: --help MUST show the same version as --version
	// This ensures users know which version they're running from help output
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


func TestAuthCommand_RequiresCredentials(t *testing.T) {
	_, stderr, exitCode := runCLI(t, nil, "auth")
	if exitCode == 0 {
		t.Error("should fail without credentials")
	}
	if !strings.Contains(stderr, "FEEDMIX_YOUTUBE") {
		t.Errorf("error should mention env vars, got: %s", stderr)
	}
}

func TestFeedCommand_NotAuthenticatedErrorShowsConfigPath(t *testing.T) {
	configDir, _ := os.MkdirTemp("", "feedmix-config")
	defer os.RemoveAll(configDir)

	_, stderr, exitCode := runCLI(t, map[string]string{"FEEDMIX_CONFIG_DIR": configDir}, "feed")

	if exitCode == 0 {
		t.Fatal("feed should fail when no token exists")
	}
	if !strings.Contains(stderr, configDir) {
		t.Errorf("error should include config path so user knows where to look, got: %s", stderr)
	}
}

func TestFeedCommand_DisplaysItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	configDir, _ := os.MkdirTemp("", "feedmix-config")
	defer os.RemoveAll(configDir)

	tokenData := `{"access_token":"test-token","token_type":"Bearer"}`
	_ = os.WriteFile(filepath.Join(configDir, "youtube_token.json"), []byte(tokenData), 0600)

	env := map[string]string{
		"FEEDMIX_CONFIG_DIR": configDir,
		"FEEDMIX_API_URL":    server.URL,
	}

	stdout, _, exitCode := runCLI(t, env, "feed")
	if exitCode != 0 {
		t.Errorf("feed should succeed, got exit code %d", exitCode)
	}
	if !strings.Contains(stdout, "Test Channel") {
		t.Errorf("should show feed item, got: %s", stdout)
	}
}

func TestFeedCommand_AggregatesMultipleChannels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	}))
	defer server.Close()

	configDir, _ := os.MkdirTemp("", "feedmix-config")
	defer os.RemoveAll(configDir)
	_ = os.WriteFile(filepath.Join(configDir, "youtube_token.json"), []byte(`{"access_token":"tok","token_type":"Bearer"}`), 0600)

	stdout, _, exitCode := runCLI(t, map[string]string{"FEEDMIX_CONFIG_DIR": configDir, "FEEDMIX_API_URL": server.URL}, "feed")

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

func TestFeedCommand_DisplaysVideoURLs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// First call: subscriptions
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

		// Second call: search for videos
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

		// Third call: video statistics
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
	}))
	defer server.Close()

	configDir, _ := os.MkdirTemp("", "feedmix-config")
	defer os.RemoveAll(configDir)

	tokenData := `{"access_token":"test-token","token_type":"Bearer"}`
	_ = os.WriteFile(filepath.Join(configDir, "youtube_token.json"), []byte(tokenData), 0600)

	env := map[string]string{
		"FEEDMIX_CONFIG_DIR": configDir,
		"FEEDMIX_API_URL":    server.URL,
	}

	stdout, _, exitCode := runCLI(t, env, "feed")
	if exitCode != 0 {
		t.Errorf("feed should succeed, got exit code %d", exitCode)
	}

	// Should display video URL, not channel URL
	expectedVideoURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	if !strings.Contains(stdout, expectedVideoURL) {
		t.Errorf("feed should display video URL %q, got: %s", expectedVideoURL, stdout)
	}

	// Should NOT display channel URL
	channelURL := "https://youtube.com/channel/UCxYz123ABC"
	if strings.Contains(stdout, channelURL) {
		t.Errorf("feed should NOT display channel URL %q (should show videos instead), got: %s", channelURL, stdout)
	}
}
