package redact

import (
	"regexp"
)

// scrubPatterns defines regular expressions for token patterns, secret headers, and URL credentials.
// Note: This credential scrubbing is a defense-in-depth safety mechanism to prevent accidental leakage
// of credentials in debug and error envelopes.
var (
	// GitHub token prefixes (ghp_, gho_, ghu_, ghs_, ghr_, github_pat_)
	githubTokenRegex = regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{30,}\b`)
	githubPatRegex   = regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}\b`)

	// Authorization header and Bearer token scrubbing
	authHeaderRegex = regexp.MustCompile(`(?i)(authorization\s*:\s*(?:bearer\s+|basic\s+|token\s+)?)[A-Za-z0-9_\-\.\+/=]{8,}\b`)
	bearerTokenRegex = regexp.MustCompile(`(?i)\b(bearer\s+)[A-Za-z0-9_\-\.\+/=]{12,}\b`)

	// URL with embedded basic auth: https://user:pass@github.com
	urlAuthRegex = regexp.MustCompile(`(https?://[^:\s/]+):([^@\s/]+)@`)
)

// Scrub sanitizes a string by replacing credential patterns with "[REDACTED]".
func Scrub(s string) string {
	if s == "" {
		return ""
	}

	// Scrub URL basic authentication credentials
	s = urlAuthRegex.ReplaceAllString(s, `${1}:[REDACTED]@`)

	// Scrub Authorization headers
	s = authHeaderRegex.ReplaceAllString(s, `${1}[REDACTED]`)

	// Scrub stand-alone Bearer tokens
	s = bearerTokenRegex.ReplaceAllString(s, `${1}[REDACTED]`)

	// Scrub GitHub tokens
	s = githubTokenRegex.ReplaceAllString(s, "[REDACTED]")
	s = githubPatRegex.ReplaceAllString(s, "[REDACTED]")

	return s
}