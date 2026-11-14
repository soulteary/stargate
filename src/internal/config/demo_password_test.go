package config

import (
	"errors"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func setDemoPasswordTestEnv(t *testing.T, passwords, cookieSecure string) {
	t.Helper()
	isolateGlobalState(t)
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", passwords)
	if cookieSecure != "" {
		t.Setenv("COOKIE_SECURE", cookieSecure)
	}
}

func TestInitializeRejectsPublishedDemoPasswordsWhenCookieSecure(t *testing.T) {
	for _, passwords := range []string{
		"plaintext:test1234|test1337",
		"plaintext:test1234",
		"plaintext:test1337",
		"PLAINTEXT:own-password|test1337",
	} {
		for _, cookieSecure := range []string{"", "true", "TRUE"} {
			t.Run(passwords+" COOKIE_SECURE="+cookieSecure, func(t *testing.T) {
				setDemoPasswordTestEnv(t, passwords, cookieSecure)

				err := Initialize(testLogger())
				testza.AssertNotNil(t, err)
				testza.AssertContains(t, err.Error(), "PASSWORDS")
				testza.AssertContains(t, err.Error(), "published demo passwords")
				// The rule is reported as a reason, not as a provided value.
				var validationErr *ValidationError
				testza.AssertTrue(t, errors.As(err, &validationErr))
				testza.AssertContains(t, validationErr.Reason, "published demo passwords")
				testza.AssertEqual(t, "", validationErr.ProvidedValue)
				// The configured value never appears in the error.
				testza.AssertNotContains(t, err.Error(), "own-password")
			})
		}
	}
}

func TestInitializeAcceptsPublishedDemoPasswordsForLocalHTTP(t *testing.T) {
	setDemoPasswordTestEnv(t, "plaintext:test1234|test1337", "false")

	testza.AssertNoError(t, Initialize(testLogger()))
}

// Only the two published values are refused. Any other plaintext password
// keeps the startup warning, and a value that merely contains a demo password
// is not one.
func TestInitializeAcceptsOtherPlaintextPasswordsWhenCookieSecure(t *testing.T) {
	for _, passwords := range []string{
		"plaintext:test123",
		"plaintext:test12345|xtest1337",
		"plaintext:Test1234",
	} {
		t.Run(passwords, func(t *testing.T) {
			setDemoPasswordTestEnv(t, passwords, "true")

			testza.AssertNoError(t, Initialize(testLogger()))
		})
	}
}

// A bcrypt value is compared as an opaque hash, never as one of the demo
// passwords, even when it is spelled like one.
func TestUsesPublishedDemoPasswordIgnoresOtherAlgorithms(t *testing.T) {
	original := Passwords.Value
	t.Cleanup(func() { Passwords.Value = original })

	Passwords.Value = "bcrypt:test1234"
	testza.AssertFalse(t, usesPublishedDemoPassword())
	Passwords.Value = ""
	testza.AssertFalse(t, usesPublishedDemoPassword())
	Passwords.Value = "plaintext:test1337"
	testza.AssertTrue(t, usesPublishedDemoPassword())
}
