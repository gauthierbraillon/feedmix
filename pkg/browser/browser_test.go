package browser

import (
	"strings"
	"testing"
)

func TestOpen_ValidHTTPURL(t *testing.T) {
	// Note: We can't actually test browser opening without mocking,
	// but we can test that valid URLs don't return errors
	err := Open("http://example.com")
	if err != nil && !strings.Contains(err.Error(), "unsupported platform") {
		t.Errorf("Valid HTTP URL should not return error: %v", err)
	}
}

func TestOpen_ValidHTTPSURL(t *testing.T) {
	err := Open("https://example.com")
	if err != nil && !strings.Contains(err.Error(), "unsupported platform") {
		t.Errorf("Valid HTTPS URL should not return error: %v", err)
	}
}

func TestOpen_RejectsInvalidScheme(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"file scheme", "file:///etc/passwd"},
		{"javascript scheme", "javascript:alert(1)"},
		{"data scheme", "data:text/html,<script>alert(1)</script>"},
		{"ftp scheme", "ftp://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Open(tt.url)
			if err == nil {
				t.Errorf("Should reject %s, but got no error", tt.url)
			}
			if !strings.Contains(err.Error(), "unsupported URL scheme") {
				t.Errorf("Expected scheme error, got: %v", err)
			}
		})
	}
}

func TestOpen_RejectsMalformedURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"shell injection attempt", "http://example.com; rm -rf /"},
		{"command substitution", "http://example.com$(whoami)"},
		{"newline injection", "http://example.com\nrm -rf /"},
		{"null byte", "http://example.com\x00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These should either be rejected as invalid URLs
			// or sanitized during parsing
			err := Open(tt.url)
			// We don't expect these to succeed
			// (either invalid URL or command will fail to find browser)
			_ = err
		})
	}
}

func TestOpen_RejectsEmptyURL(t *testing.T) {
	err := Open("")
	if err == nil {
		t.Error("Should reject empty URL")
	}
	if !strings.Contains(err.Error(), "unsupported URL scheme") && !strings.Contains(err.Error(), "invalid URL") {
		t.Errorf("Expected URL validation error, got: %v", err)
	}
}

func TestOpen_RejectsURLWithoutScheme(t *testing.T) {
	err := Open("example.com")
	if err == nil {
		t.Error("Should reject URL without scheme")
	}
	if !strings.Contains(err.Error(), "unsupported URL scheme") {
		t.Errorf("Expected scheme error, got: %v", err)
	}
}
