// Package browser provides cross-platform browser opening functionality.
package browser

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

// Open opens the specified URL in the default browser.
// It validates the URL before passing it to the system browser to prevent command injection.
func Open(urlString string) error {
	// Validate URL to prevent command injection (fixes G204/CWE-78)
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Whitelist allowed schemes to prevent malicious URLs
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s (only http and https allowed)", parsedURL.Scheme)
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", urlString) // #nosec G204 -- URL validated above
	case "darwin":
		cmd = exec.Command("open", urlString) // #nosec G204 -- URL validated above
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", urlString) // #nosec G204 -- URL validated above
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
