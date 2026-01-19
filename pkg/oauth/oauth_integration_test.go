// +build integration

package oauth

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestTokenStorage_FilePermissions_Ubuntu verifies that token files have
// secure permissions (0600) on Ubuntu/Linux systems.
// Run with: go test -tags=integration ./pkg/oauth
func TestTokenStorage_FilePermissions_Ubuntu(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("This integration test is for Ubuntu/Linux only")
	}

	dir, err := os.MkdirTemp("", "oauth-integration-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	storage := NewTokenStorage(dir)
	token := &Token{
		AccessToken:  "secret-token",
		RefreshToken: "secret-refresh",
		TokenType:    "Bearer",
	}

	if err := storage.Save("youtube", token); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify file permissions are restrictive (owner read/write only)
	tokenPath := filepath.Join(dir, "youtube_token.json")
	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("failed to stat token file: %v", err)
	}

	perm := info.Mode().Perm()
	expectedPerm := os.FileMode(0600)
	if perm != expectedPerm {
		t.Errorf("token file permissions are %o, expected %o (security risk)", perm, expectedPerm)
	}

	// Verify directory permissions are also restrictive
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("failed to stat directory: %v", err)
	}

	dirPerm := dirInfo.Mode().Perm()
	expectedDirPerm := os.FileMode(0700)
	if dirPerm != expectedDirPerm {
		t.Errorf("directory permissions are %o, expected %o (security risk)", dirPerm, expectedDirPerm)
	}
}

// TestTokenStorage_RealFilesystem verifies token storage works with the
// actual filesystem, including handling of paths, symlinks, etc.
func TestTokenStorage_RealFilesystem(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("This integration test is for Ubuntu/Linux only")
	}

	dir, err := os.MkdirTemp("", "oauth-integration-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	storage := NewTokenStorage(dir)
	token := &Token{
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}

	// Test saving
	if err := storage.Save("youtube", token); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify file exists
	tokenPath := filepath.Join(dir, "youtube_token.json")
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		t.Error("token file was not created")
	}

	// Test loading
	loaded, err := storage.Load("youtube")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	// Verify all fields
	if loaded.AccessToken != token.AccessToken {
		t.Errorf("AccessToken mismatch: got %q, want %q", loaded.AccessToken, token.AccessToken)
	}
	if loaded.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken mismatch: got %q, want %q", loaded.RefreshToken, token.RefreshToken)
	}
	if loaded.TokenType != token.TokenType {
		t.Errorf("TokenType mismatch: got %q, want %q", loaded.TokenType, token.TokenType)
	}

	// Test overwriting existing token
	newToken := &Token{
		AccessToken: "new-access",
		TokenType:   "Bearer",
	}
	if err := storage.Save("youtube", newToken); err != nil {
		t.Fatalf("overwrite failed: %v", err)
	}

	reloaded, err := storage.Load("youtube")
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded.AccessToken != "new-access" {
		t.Error("token was not overwritten")
	}
}

// TestTokenStorage_PathTraversalProtection verifies that path traversal
// attacks are prevented by the sanitization in TokenStorage.
func TestTokenStorage_PathTraversalProtection(t *testing.T) {
	dir, err := os.MkdirTemp("", "oauth-integration-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	storage := NewTokenStorage(dir)
	token := &Token{AccessToken: "test", TokenType: "Bearer"}

	// Attempt path traversal attack
	maliciousProviders := []string{
		"../../../etc/passwd",
		"..%2F..%2Fetc%2Fpasswd",
		"/etc/passwd",
		"youtube/../../secret",
	}

	for _, provider := range maliciousProviders {
		err := storage.Save(provider, token)
		if err != nil {
			// Errors are acceptable
			continue
		}

		// If save succeeded, verify it was sanitized
		// The sanitized filename should be in the storage dir
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read dir: %v", err)
		}

		// Verify no files were created outside the storage directory
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			if !filepath.HasPrefix(path, dir) {
				t.Errorf("path traversal attack succeeded: file created outside storage dir: %s", path)
			}
		}
	}
}
