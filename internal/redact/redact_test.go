package redact

import (
	"strings"
	"testing"
)

func TestScrub(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		omits    []string
	}{
		{
			name:     "Classic GitHub Token (ghp_)",
			input:    "fatal: remote error using token ghp_1234567890abcdefghijklmnopqrstuvwxyzAB in url",
			contains: []string{"[REDACTED]", "fatal: remote error using token", "in url"},
			omits:    []string{"ghp_1234567890abcdefghijklmnopqrstuvwxyzAB"},
		},
		{
			name:     "Fine-grained GitHub Token (github_pat_)",
			input:    "gh auth error: github_pat_11ABCD1234_long_secure_token_value_xyz789 rejected",
			contains: []string{"[REDACTED]", "gh auth error:", "rejected"},
			omits:    []string{"github_pat_11ABCD1234_long_secure_token_value_xyz789"},
		},
		{
			name:     "Authorization Bearer Header",
			input:    "HTTP 401: Authorization: Bearer gho_9876543210zyxwvutsrqponmlkjihgfedcba",
			contains: []string{"Authorization: Bearer [REDACTED]", "HTTP 401:"},
			omits:    []string{"gho_9876543210zyxwvutsrqponmlkjihgfedcba"},
		},
		{
			name:     "Authorization with Token followed by trailing text",
			input:    "Authorization: token supersecrettoken123\nNext line of output",
			contains: []string{"Authorization: token [REDACTED]", "Next line of output"},
			omits:    []string{"supersecrettoken123"},
		},
		{
			name:     "Standalone Bearer Token",
			input:    "failed with bearer mysecretbearertoken456 in response",
			contains: []string{"bearer [REDACTED]", "failed with", "in response"},
			omits:    []string{"mysecretbearertoken456"},
		},
		{
			name:     "Embedded URL credentials",
			input:    "fatal: repository 'https://x-access-token:ghs_secret123456789012345678901234567890@github.com/org/repo.git' not found",
			contains: []string{"https://x-access-token:[REDACTED]@github.com/org/repo.git"},
			omits:    []string{"ghs_secret123456789012345678901234567890"},
		},
		{
			name:     "Clean error text unchanged",
			input:    "error: failed to push some refs to 'git@github.com:user/repo.git'",
			contains: []string{"error: failed to push some refs to 'git@github.com:user/repo.git'"},
			omits:    []string{"[REDACTED]"},
		},
		{
			name:     "Empty input",
			input:    "",
			contains: []string{""},
			omits:    []string{"[REDACTED]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Scrub(tt.input)

			for _, c := range tt.contains {
				if !strings.Contains(got, c) {
					t.Errorf("expected output to contain %q, but got %q", c, got)
				}
			}

			for _, o := range tt.omits {
				if strings.Contains(got, o) {
					t.Errorf("expected output to NOT contain %q, but got %q", o, got)
				}
			}
		})
	}
}