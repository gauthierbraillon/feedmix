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

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "feedmix-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	binaryPath = filepath.Join(dir, "feedmix")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
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
