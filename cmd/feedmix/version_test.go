package main

import (
	"runtime/debug"
	"testing"
)

// TestResolveVersion_PreferLdflags verifies that ldflags version takes precedence
func TestResolveVersion_PreferLdflags(t *testing.T) {
	// When version is set via ldflags (not "dev"), use it
	result := resolveVersion("v1.2.3", &debug.BuildInfo{
		Main: debug.Module{Version: "v0.0.0"},
	})

	if result != "v1.2.3" {
		t.Errorf("should prefer ldflags version, got: %s", result)
	}
}

// TestResolveVersion_FallbackToBuildInfo verifies that build info is used when ldflags is "dev"
// This is the requirement for go install to work correctly
func TestResolveVersion_FallbackToBuildInfo(t *testing.T) {
	// When version is "dev" (not set via ldflags), use build info
	// This happens with: go install github.com/user/repo/cmd/tool@v1.2.3
	result := resolveVersion("dev", &debug.BuildInfo{
		Main: debug.Module{Version: "v1.2.3"},
	})

	if result != "v1.2.3" {
		t.Errorf("should use build info version when ldflags is 'dev', got: %s", result)
	}
}

// TestResolveVersion_IgnoreDevel verifies that "(devel)" is treated as "dev"
func TestResolveVersion_IgnoreDevel(t *testing.T) {
	// When build info has "(devel)", fall back to "dev"
	result := resolveVersion("dev", &debug.BuildInfo{
		Main: debug.Module{Version: "(devel)"},
	})

	if result != "dev" {
		t.Errorf("should return 'dev' when build info is '(devel)', got: %s", result)
	}
}

// TestResolveVersion_NilBuildInfo handles nil build info gracefully
func TestResolveVersion_NilBuildInfo(t *testing.T) {
	result := resolveVersion("dev", nil)

	if result != "dev" {
		t.Errorf("should return 'dev' when build info is nil, got: %s", result)
	}
}
