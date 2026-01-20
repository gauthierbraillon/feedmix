// +build integration

package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestBinaryVersion_MatchesGitTag verifies that the binary version
// matches the git tag (source of truth for versioning).
// This is an integration test because it interacts with git (OS).
// Run with: go test -tags=integration ./cmd/feedmix -v
func TestBinaryVersion_MatchesGitTag(t *testing.T) {
	// Get version from git (source of truth)
	cmd := exec.Command("git", "describe", "--tags", "--always", "--dirty")
	output, err := cmd.Output()
	if err != nil {
		t.Skipf("Skipping test: git not available or not a git repo: %v", err)
	}
	gitVersion := strings.TrimSpace(string(output))

	// Get version from binary
	versionOutput, _, _ := runCLI(t, nil, "--version")
	versionLine := strings.TrimSpace(versionOutput)
	parts := strings.Fields(versionLine)
	if len(parts) < 3 {
		t.Fatalf("unexpected version output format: %s", versionOutput)
	}
	binaryVersion := parts[2] // "feedmix version v0.2.0" -> "v0.2.0"

	// Requirement: Binary version MUST match git tag (source of truth)
	// This ensures version is auto-generated from git, not hardcoded
	if binaryVersion != gitVersion {
		t.Errorf("Binary version %q does not match git tag %q.\n\nVersion must be injected at build time via:\n  go build -ldflags=\"-X main.version=$(git describe --tags --always --dirty)\" ./cmd/feedmix", binaryVersion, gitVersion)
	}
}
