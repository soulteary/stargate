package config

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestInitStepUpMatcher_Disabled(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "false")
	t.Setenv("STEP_UP_PATHS", "")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	InitStepUpMatcher()

	matcher := GetStepUpMatcher()
	testza.AssertNotNil(t, matcher)
	testza.AssertFalse(t, matcher.RequiresStepUp("/admin"))
	testza.AssertFalse(t, matcher.RequiresStepUp("/api/secret"))
}

func TestInitStepUpMatcher_Enabled_EmptyPaths(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", "")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")

	err := Initialize(testLogger())
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "STEP_UP_PATHS")
}

func TestInitializeStepUpRejectsDelimiterOnlyPaths(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", " , , ")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")

	err := Initialize(testLogger())
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "STEP_UP_PATHS")
}

func TestInitializeStepUpAcceptsQuestionMarkGlob(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", "/users/?/settings")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")

	testza.AssertNoError(t, Initialize(testLogger()))
	matcher := GetStepUpMatcher()
	testza.AssertTrue(t, matcher.RequiresStepUp("/users/a/settings"))
	testza.AssertFalse(t, matcher.RequiresStepUp("/users/ab/settings"))
}

func TestValidStepUpPathPattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		valid   bool
	}{
		{name: "question mark glob", pattern: "/users/?/settings", valid: true},
		{name: "star glob", pattern: "/admin/*", valid: true},
		{name: "literal newline", pattern: "/ad\nmin", valid: false},
		{name: "trailing literal newline", pattern: "/admin\n", valid: false},
		{name: "embedded NUL", pattern: "/ad\x00min", valid: false},
		{name: "embedded control", pattern: "/ad\x1fmin", valid: false},
		{name: "Unicode control", pattern: "/ad\u0085min", valid: false},
		{name: "encoded NUL", pattern: "/admin/%00", valid: false},
		{name: "encoded LF uppercase", pattern: "/admin/%0A", valid: false},
		{name: "encoded LF lowercase", pattern: "/admin/%0a", valid: false},
		{name: "encoded CR uppercase", pattern: "/admin/%0D", valid: false},
		{name: "encoded CR lowercase", pattern: "/admin/%0d", valid: false},
		{name: "backslash", pattern: `/admin\\users`, valid: false},
		{name: "fragment", pattern: "/admin#users", valid: false},
		{name: "dot segment", pattern: "/admin/../public", valid: false},
		{name: "encoded dot segment", pattern: "/admin/%2e%2e/public", valid: false},
		{name: "invalid escape", pattern: "/admin/%zz", valid: false},
		{name: "non-local absolute path", pattern: "//evil.example/admin", valid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testza.AssertEqual(t, test.valid, ValidStepUpPathPattern(test.pattern))
		})
	}
}

func TestInitStepUpMatcher_Enabled_SinglePath(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", "/admin")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	InitStepUpMatcher()

	matcher := GetStepUpMatcher()
	testza.AssertNotNil(t, matcher)
	testza.AssertTrue(t, matcher.RequiresStepUp("/admin"))
	testza.AssertTrue(t, matcher.RequiresStepUp("/admin/users"))
	testza.AssertFalse(t, matcher.RequiresStepUp("/administrator"))
	testza.AssertFalse(t, matcher.RequiresStepUp("/api"))
	testza.AssertFalse(t, matcher.RequiresStepUp("/"))
}

func TestInitStepUpMatcher_Enabled_MultiplePaths(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", "/admin,/api/secret")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	InitStepUpMatcher()

	matcher := GetStepUpMatcher()
	testza.AssertNotNil(t, matcher)
	testza.AssertTrue(t, matcher.RequiresStepUp("/admin"))
	testza.AssertTrue(t, matcher.RequiresStepUp("/api/secret"))
	testza.AssertTrue(t, matcher.RequiresStepUp("/api/secret/data"))
	testza.AssertFalse(t, matcher.RequiresStepUp("/api/public"))
}

func TestInitializeStepUpRequiresTrustedProxy(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", "/admin")
	t.Setenv("TRUSTED_PROXIES", "")

	err := Initialize(testLogger())
	testza.AssertNotNil(t, err)
	testza.AssertContains(t, err.Error(), "TRUSTED_PROXIES")
}

func TestInitializeRejectsUnsafeStepUpPathPatterns(t *testing.T) {
	for _, paths := range []string{"admin", "//evil.example/admin", "/admin#users", `/admin\\users`, "/admin/../public", "/admin/%2e%2e/public"} {
		t.Run(paths, func(t *testing.T) {
			t.Setenv("AUTH_HOST", "auth.example.com")
			t.Setenv("PASSWORDS", "plaintext:test123")
			t.Setenv("STEP_UP_ENABLED", "true")
			t.Setenv("STEP_UP_PATHS", paths)
			t.Setenv("TRUSTED_PROXIES", "127.0.0.1")

			err := Initialize(testLogger())
			testza.AssertNotNil(t, err)
			testza.AssertContains(t, err.Error(), "STEP_UP_PATHS")
		})
	}
}

func TestGetStepUpMatcher_NilInitializes(t *testing.T) {
	// Ensure stepUpMatcher is nil by not calling InitStepUpMatcher in a fresh package state.
	// GetStepUpMatcher() calls InitStepUpMatcher() if nil, so we need to have env set.
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "false")

	_ = Initialize(testLogger())
	// Do not call InitStepUpMatcher; GetStepUpMatcher should call it internally.
	matcher := GetStepUpMatcher()
	testza.AssertNotNil(t, matcher)
}
