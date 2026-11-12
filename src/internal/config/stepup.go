package config

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// StepUpMatcher handles step-up authentication path matching
type StepUpMatcher struct {
	patterns []*regexp.Regexp
	enabled  bool
}

var stepUpMatcher *StepUpMatcher

// ValidStepUpPathPattern reports whether a configured step-up pattern is a
// local absolute path without control characters, ambiguous dot segments, or
// backslashes.
func ValidStepUpPathPattern(raw string) bool {
	decoded, err := url.PathUnescape(raw)
	if err != nil || !utf8.ValidString(decoded) || strings.IndexFunc(decoded, unicode.IsControl) >= 0 {
		return false
	}

	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "\\#") {
		return false
	}
	// Do not parse the pattern as a request URI: '?' is a documented
	// single-character glob here, not a query delimiter.
	decoded, _ = url.PathUnescape(raw)
	for _, segment := range strings.Split(decoded, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

// InitStepUpMatcher initializes the step-up path matcher
func InitStepUpMatcher() {
	enabled := StepUpEnabled.ToBool()
	patterns := make([]*regexp.Regexp, 0)

	if enabled && StepUpPaths.Value != "" {
		// Parse comma-separated path patterns
		pathStrs := strings.Split(StepUpPaths.Value, ",")
		for _, pathStr := range pathStrs {
			pathStr = strings.TrimSpace(pathStr)
			if pathStr == "" {
				continue
			}

			regexPattern := stepUpRegexPattern(pathStr)

			pattern, err := regexp.Compile(regexPattern)
			if err != nil {
				// Log error but continue with other patterns
				continue
			}
			patterns = append(patterns, pattern)
		}
	}

	stepUpMatcher = &StepUpMatcher{
		patterns: patterns,
		enabled:  enabled,
	}
}

// StripPathParameters removes `;name=value` path parameters from every path
// segment. Servlet-style backends ignore them when routing, so `/admin;x` and
// `/admin;x/users` reach the same handlers as `/admin` and `/admin/users`.
// Segments are stripped individually and never joined: `/a;x/dmin` becomes
// `/a/dmin`, not `/admin`.
func StripPathParameters(path string) string {
	if !strings.Contains(path, ";") {
		return path
	}
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if idx := strings.IndexByte(segment, ';'); idx >= 0 {
			segments[i] = segment[:idx]
		}
	}
	return strings.Join(segments, "/")
}

// stepUpRegexPattern compiles to a case-insensitive expression. Backends that
// route case-insensitively would otherwise serve `/Admin` without step-up when
// `/admin` is protected; matching too much is the safe direction.
func stepUpRegexPattern(path string) string {
	// Explicit glob patterns retain their exact glob semantics. Plain paths
	// match the path itself and descendants separated by '/', not unrelated
	// paths that happen to share the same string prefix.
	if strings.ContainsAny(path, "*?") {
		return "(?i)^" + strings.ReplaceAll(
			strings.ReplaceAll(regexp.QuoteMeta(path), "\\*", ".*"),
			"\\?", ".",
		) + "$"
	}

	if path == "/" {
		return "(?i)^/.*$"
	}
	path = strings.TrimSuffix(path, "/")
	return "(?i)^" + regexp.QuoteMeta(path) + "(?:/.*)?$"
}

// GetStepUpMatcher returns the step-up matcher instance
func GetStepUpMatcher() *StepUpMatcher {
	if stepUpMatcher == nil {
		InitStepUpMatcher()
	}
	return stepUpMatcher
}

// RequiresStepUp checks if the given path requires step-up authentication.
// Matching is case-insensitive and ignores `;` path parameters.
func (m *StepUpMatcher) RequiresStepUp(path string) bool {
	if !m.enabled {
		return false
	}

	if len(m.patterns) == 0 {
		return false
	}

	// Match the path both as received and with path parameters removed, so a
	// parameter can neither hide a protected path nor break a configured
	// pattern that itself contains ';'.
	stripped := StripPathParameters(path)
	for _, pattern := range m.patterns {
		if pattern.MatchString(path) || (stripped != path && pattern.MatchString(stripped)) {
			return true
		}
	}

	return false
}
