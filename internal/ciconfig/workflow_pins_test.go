package ciconfig_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

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
