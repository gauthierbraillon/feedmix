package ciconfig_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func fileContains(t *testing.T, path string, want ...string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("required file missing: %s", path)
	}
	content := string(data)
	for _, s := range want {
		if !strings.Contains(content, s) {
			t.Errorf("%s: missing required content %q", filepath.Base(path), s)
		}
	}
}

func TestDiscoverability_RequiredFilesExist(t *testing.T) {
	required := []string{
		"../../LICENSE",
		"../../CHANGELOG.md",
		"../../CONTRIBUTING.md",
		"../../ARCHITECTURE.md",
		"../../SECURITY.md",
		"../../llms.txt",
		"../../.github/ISSUE_TEMPLATE/bug.yml",
		"../../.github/PULL_REQUEST_TEMPLATE.md",
	}
	for _, path := range required {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("required file missing: %s", path)
		}
	}
}

func TestDiscoverability_READMEHasRequiredSections(t *testing.T) {
	fileContains(t, "../../README.md",
		"RSS",
		"privacy",
		"lightweight",
		"Why Feedmix",
		"License",
	)
}

func TestDiscoverability_LicenseIsMIT(t *testing.T) {
	fileContains(t, "../../LICENSE", "MIT")
}

func TestDiscoverability_CHANGELOGHasVersions(t *testing.T) {
	fileContains(t, "../../CHANGELOG.md", "v0.1.0", "v0.2.0", "v0.5.0")
}

func TestDiscoverability_LLMSTxtHasProjectSummary(t *testing.T) {
	fileContains(t, "../../llms.txt",
		"feedmix",
		"YouTube",
		"terminal",
		"go install",
	)
}

var pinnedSHA = regexp.MustCompile(`@[0-9a-f]{40}`)

func TestWorkflowActions_PinnedToCommitSHA(t *testing.T) {
	workflows, err := filepath.Glob("../../.github/workflows/*.yml")
	if err != nil {
		t.Fatal(err)
	}
	if len(workflows) == 0 {
		t.Fatal("no workflow files found")
	}

	for _, path := range workflows {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if !strings.Contains(line, "uses:") {
				continue
			}
			if !pinnedSHA.MatchString(line) {
				t.Errorf("%s:%d: action not pinned to commit SHA: %s",
					filepath.Base(path), i+1, strings.TrimSpace(line))
			}
		}
	}
}
