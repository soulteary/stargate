package config

import "testing"

func TestValidateSessionExchangeRedirectPath(t *testing.T) {
	for _, value := range []string{"/", "/auth/login/example/", "/auth/login/provider:instance/"} {
		if !ValidateSessionExchangeRedirectPath(EnvVariable{Value: value}) {
			t.Errorf("rejected same-origin landing path %q", value)
		}
	}
	for _, value := range []string{"", "login", "https://evil.example/", "//evil.example/", "/\\evil.example", "/%2fevil.example", "/%252fevil.example", "/ok?next=evil", "/ok#fragment", "/ok\r\nLocation: //evil.example", "/space here", "/a/../_login", "/a/./b", "/_session_exchange", "/_session_exchange/", "/_login", "/_login/", "/_session_exchange//", "/a//b"} {
		if ValidateSessionExchangeRedirectPath(EnvVariable{Value: value}) {
			t.Errorf("accepted unsafe or looping landing path %q", value)
		}
	}
}

func TestSessionExchangeRedirectPathLoadsAndValidatesEnvironment(t *testing.T) {
	original := SessionExchangeRedirectPath.Value
	t.Cleanup(func() { SessionExchangeRedirectPath.Value = original })
	t.Setenv(SessionExchangeRedirectPath.Name, "/auth/login/example/")
	if err := SessionExchangeRedirectPath.Validate(); err != nil {
		t.Fatal(err)
	}
	if SessionExchangeRedirectPath.Value != "/auth/login/example/" {
		t.Fatal("landing path was not loaded from the environment")
	}
	t.Setenv(SessionExchangeRedirectPath.Name, "//evil.example/")
	if err := SessionExchangeRedirectPath.Validate(); err == nil {
		t.Fatal("unsafe configured landing path must fail startup validation")
	}
}
