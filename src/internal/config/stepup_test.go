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

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	InitStepUpMatcher()

	matcher := GetStepUpMatcher()
	testza.AssertNotNil(t, matcher)
	testza.AssertFalse(t, matcher.RequiresStepUp("/admin"))
}

func TestInitStepUpMatcher_Enabled_SinglePath(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", "/admin*")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	InitStepUpMatcher()

	matcher := GetStepUpMatcher()
	testza.AssertNotNil(t, matcher)
	testza.AssertTrue(t, matcher.RequiresStepUp("/admin"))
	testza.AssertTrue(t, matcher.RequiresStepUp("/admin/users"))
	testza.AssertFalse(t, matcher.RequiresStepUp("/api"))
	testza.AssertFalse(t, matcher.RequiresStepUp("/"))
}

func TestInitStepUpMatcher_Enabled_MultiplePaths(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("STEP_UP_ENABLED", "true")
	t.Setenv("STEP_UP_PATHS", "/admin*,/api/secret*")

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
