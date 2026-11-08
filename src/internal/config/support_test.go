package config

import "testing"

func TestLoginSupportURL(t *testing.T) {
	for _, raw := range []string{"", "https://support.example/help#login", "mailto:help+login@example.com"} {
		if !ValidateLoginSupportURL(EnvVariable{Value: raw}) {
			t.Errorf("valid support URL rejected: %q", raw)
		}
	}
	for _, raw := range []string{"#", "http://example.com", "javascript:alert(1)", "//example.com", "https://user:pass@example.com", "https://", "https://example.com/%0aheader", "mailto:a@example.com?subject=secret", "mailto:a@example.com,b@example.com", "mailto:a%0a@example.com", "mailto:Name <a@example.com>"} {
		if ValidateLoginSupportURL(EnvVariable{Value: raw}) {
			t.Errorf("unsafe support URL accepted: %q", raw)
		}
	}
}
