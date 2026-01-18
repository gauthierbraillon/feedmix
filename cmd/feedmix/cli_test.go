// Package main tests document the expected behavior of the feedmix CLI.
//
// TDD Cycle: RED -> GREEN -> REFACTOR
//
// These are BLACK BOX tests - they test the CLI by executing the binary
// and checking stdout/stderr output.
//
// External dependencies mocked:
// - HTTP APIs (YouTube, LinkedIn) via FEEDMIX_TEST_SERVER env var
// - Token storage via FEEDMIX_CONFIG_DIR env var
//
// Test requirements (this file serves as documentation):
// - CLI has root command with version info
// - "auth" command initiates OAuth flow for a provider
// - "feed" command displays aggregated feed
// - Commands validate required arguments
// - Error messages are helpful
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

// TestMain builds the binary once before running tests.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "feedmix-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	binaryPath = filepath.Join(dir, "feedmix")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	os.Exit(m.Run())
}

// runCLI executes the CLI binary with given arguments and environment.
func runCLI(t *testing.T, env map[string]string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(binaryPath, args...)

	// Set up environment
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	exitCode = 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("failed to run command: %v", err)
	}

	return outBuf.String(), errBuf.String(), exitCode
}

// runCLISimple runs CLI without custom environment.
func runCLISimple(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	return runCLI(t, nil, args...)
}

// TestRootCommand_Help verifies help output shows available commands.
func TestRootCommand_Help(t *testing.T) {
	stdout, _, _ := runCLISimple(t, "--help")
	output := strings.ToLower(stdout)

	expects := []string{"feedmix", "usage", "auth", "feed"}
	for _, want := range expects {
		if !strings.Contains(output, want) {
			t.Errorf("help should contain %q, got:\n%s", want, stdout)
		}
	}
}

// TestRootCommand_Version verifies version output.
func TestRootCommand_Version(t *testing.T) {
	stdout, _, _ := runCLISimple(t, "--version")

	if !strings.Contains(stdout, "feedmix") || !strings.Contains(stdout, "0.") {
		t.Errorf("version should show feedmix and version, got:\n%s", stdout)
	}
}

// TestAuthCommand_RequiresProvider verifies auth needs a provider argument.
func TestAuthCommand_RequiresProvider(t *testing.T) {
	_, stderr, exitCode := runCLISimple(t, "auth")

	if exitCode == 0 {
		t.Error("should fail without provider argument")
	}
	if !strings.Contains(strings.ToLower(stderr), "provider") {
		t.Errorf("error should mention provider, got:\n%s", stderr)
	}
}

// TestAuthCommand_RejectsInvalidProvider verifies only youtube/linkedin accepted.
func TestAuthCommand_RejectsInvalidProvider(t *testing.T) {
	_, stderr, exitCode := runCLISimple(t, "auth", "twitter")

	if exitCode == 0 {
		t.Error("should fail with invalid provider")
	}
	if !strings.Contains(strings.ToLower(stderr), "invalid") {
		t.Errorf("error should mention invalid, got:\n%s", stderr)
	}
}

// TestFeedCommand_Help verifies feed help shows filter options.
func TestFeedCommand_Help(t *testing.T) {
	stdout, _, _ := runCLISimple(t, "feed", "--help")
	output := strings.ToLower(stdout)

	expects := []string{"source", "limit"}
	for _, want := range expects {
		if !strings.Contains(output, want) {
			t.Errorf("feed help should contain %q, got:\n%s", want, stdout)
		}
	}
}

// TestFeedCommand_DisplaysItems verifies feed fetches and displays items.
// External HTTP API is mocked via test server.
func TestFeedCommand_DisplaysItems(t *testing.T) {
	// Create mock API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "subscriptions") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"snippet": map[string]interface{}{
							"resourceId": map[string]interface{}{"channelId": "UC123"},
							"title":      "Test Channel",
							"thumbnails": map[string]interface{}{
								"default": map[string]interface{}{"url": "http://example.com/thumb.jpg"},
							},
							"publishedAt": "2024-01-01T00:00:00Z",
						},
					},
				},
			})
		}
	}))
	defer server.Close()

	// Create temp config dir with mock token
	configDir, err := os.MkdirTemp("", "feedmix-config")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(configDir)

	// Write mock token
	tokenData := `{"access_token":"test-token","token_type":"Bearer"}`
	os.WriteFile(filepath.Join(configDir, "youtube_token.json"), []byte(tokenData), 0600)

	env := map[string]string{
		"FEEDMIX_CONFIG_DIR": configDir,
		"FEEDMIX_API_URL":    server.URL,
	}

	stdout, _, exitCode := runCLI(t, env, "feed", "--source", "youtube")

	if exitCode != 0 {
		t.Errorf("feed command should succeed, got exit code %d", exitCode)
	}

	// Verify actual feed content is displayed (not just placeholder)
	if !strings.Contains(stdout, "Test Channel") {
		t.Errorf("output should contain feed item 'Test Channel', got:\n%s", stdout)
	}
}

// TestConfigCommand_Help verifies config shows options.
func TestConfigCommand_Help(t *testing.T) {
	stdout, _, _ := runCLISimple(t, "config", "--help")

	if !strings.Contains(strings.ToLower(stdout), "config") {
		t.Errorf("should show config help, got:\n%s", stdout)
	}
}
