package config

import (
	"regexp"
	"strings"
)

// StepUpMatcher handles step-up authentication path matching
type StepUpMatcher struct {
	patterns []*regexp.Regexp
	enabled  bool
}

var stepUpMatcher *StepUpMatcher

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

			// Convert glob pattern to regex
			// Simple conversion: * -> .*, ? -> ., ^ and $ for exact match
			regexPattern := "^" + strings.ReplaceAll(
				strings.ReplaceAll(regexp.QuoteMeta(pathStr), "\\*", ".*"),
				"\\?", ".",
			) + "$"

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
