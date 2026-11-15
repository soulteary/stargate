package config

import (
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"
)

// WARDEN_HMAC_AUDIENCE is only meaningful for signed requests and is signed
// byte for byte, so a value the Warden SDK would refuse is refused at startup
// with the names of the settings involved.
func TestWardenHMACAudienceIsRejectedWhenUnusable(t *testing.T) {
	const pairReason = "requires WARDEN_HMAC_KEY_ID and WARDEN_HMAC_SECRET"
	const formReason = "must be 1 to 255 printable ASCII characters without spaces or commas"
	signed := func(audience string) map[string]string {
		return map[string]string{"WARDEN_HMAC_KEY_ID": "stargate", "WARDEN_HMAC_SECRET": "warden-hmac-secret-value", "WARDEN_HMAC_AUDIENCE": audience}
	}

	for _, tc := range []struct {
		name   string
		values map[string]string
		key    string
		reason string
	}{
		{"without HMAC credentials", map[string]string{"WARDEN_HMAC_AUDIENCE": "warden-a"}, "WARDEN_HMAC_AUDIENCE", pairReason},
		{"with an API key only", map[string]string{"WARDEN_API_KEY": "warden-api-key-value", "WARDEN_HMAC_AUDIENCE": "warden-a"}, "WARDEN_HMAC_AUDIENCE", pairReason},
		{"with a key ID only", map[string]string{"WARDEN_HMAC_KEY_ID": "stargate", "WARDEN_HMAC_AUDIENCE": "warden-a"}, "WARDEN_HMAC_SECRET", "HMAC key ID and secret must be configured together"},
		{"inner space", signed("warden a"), "WARDEN_HMAC_AUDIENCE", formReason},
		{"comma", signed("warden-a,warden-b"), "WARDEN_HMAC_AUDIENCE", formReason},
		{"control character", signed("warden\ta"), "WARDEN_HMAC_AUDIENCE", formReason},
		{"non-ASCII", signed("wärden"), "WARDEN_HMAC_AUDIENCE", formReason},
		{"too long", signed(strings.Repeat("a", 256)), "WARDEN_HMAC_AUDIENCE", formReason},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := initializeForValidationError(t, tc.values)

			testza.AssertEqual(t, tc.key, err.KeyName)
			testza.AssertEqual(t, tc.reason, err.Reason)
			testza.AssertContains(t, err.Error(), "environment variable '"+tc.key+"': "+tc.reason+".")
		})
	}
}

// The Warden server lower-cases the names in WARDEN_HMAC_AUDIENCES and removes
// the space around them. The stored value is normalized the same way.
func TestWardenHMACAudienceIsNormalized(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values map[string]string
		want   string
	}{
		{"unset", nil, ""},
		{"blank without HMAC credentials", map[string]string{"WARDEN_HMAC_AUDIENCE": "  "}, ""},
		{"lower case", map[string]string{"WARDEN_HMAC_KEY_ID": "stargate", "WARDEN_HMAC_SECRET": "warden-hmac-secret-value", "WARDEN_HMAC_AUDIENCE": "warden-a"}, "warden-a"},
		{"mixed case and surrounding space", map[string]string{"WARDEN_HMAC_KEY_ID": "stargate", "WARDEN_HMAC_SECRET": "warden-hmac-secret-value", "WARDEN_HMAC_AUDIENCE": " Warden-A.Example "}, "warden-a.example"},
		{"longest accepted", map[string]string{"WARDEN_HMAC_KEY_ID": "stargate", "WARDEN_HMAC_SECRET": "warden-hmac-secret-value", "WARDEN_HMAC_AUDIENCE": strings.Repeat("a", 255)}, strings.Repeat("a", 255)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateGlobalState(t)
			t.Setenv("AUTH_HOST", "auth.example.com")
			t.Setenv("PASSWORDS", "plaintext:test123")
			for name, value := range tc.values {
				t.Setenv(name, value)
			}

			testza.AssertNoError(t, Initialize(testLogger()))
			testza.AssertEqual(t, tc.want, WardenHMACAudience.Value)
		})
	}
}
