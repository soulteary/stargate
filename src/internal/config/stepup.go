package config

import (
	"net/url"
	"regexp"
	"strings"
)

// StepUpMatcher handles step-up authentication path matching
type StepUpMatcher struct {
	patterns []*regexp.Regexp
	enabled  bool
}

var stepUpMatcher *StepUpMatcher

// ValidStepUpPathPattern reports whether a configured step-up pattern is a
// local absolute path without ambiguous dot segments or backslashes.
func ValidStepUpPathPattern(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "\\#") {
		return false
	}
	// Do not parse the pattern as a request URI: '?' is a documented
	// single-character glob here, not a query delimiter.
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return false
	}
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

func stepUpRegexPattern(path string) string {
	// Explicit glob patterns retain their exact glob semantics. Plain paths
	// match the path itself and descendants separated by '/', not unrelated
	// paths that happen to share the same string prefix.
	if strings.ContainsAny(path, "*?") {
		return "^" + strings.ReplaceAll(
			strings.ReplaceAll(regexp.QuoteMeta(path), "\\*", ".*"),
			"\\?", ".",
		) + "$"
	}

	if path == "/" {
		return "^/.*$"
	}
	path = strings.TrimSuffix(path, "/")
	return "^" + regexp.QuoteMeta(path) + "(?:/.*)?$"
}

// GetStepUpMatcher returns the step-up matcher instance
func GetStepUpMatcher() *StepUpMatcher {
	if stepUpMatcher == nil {
		InitStepUpMatcher()
	}
	return stepUpMatcher
}

// RequiresStepUp checks if the given path requires step-up authentication
func (m *StepUpMatcher) RequiresStepUp(path string) bool {
	if !m.enabled {
		return false
	}

	if len(m.patterns) == 0 {
		return false
	}

	for _, pattern := range m.patterns {
		if pattern.MatchString(path) {
			return true
		}
	}

	return false
}
