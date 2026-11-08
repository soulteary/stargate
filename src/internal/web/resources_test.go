package web

import (
	"io/fs"
	"strings"
	"testing"
)

func TestBuiltInResourcesAreExplicitAndScoped(t *testing.T) {
	for _, name := range []string{"login.html", "login.warden.html", "account.html", "totp_start.html", "totp_enroll.html", "totp_revoke.html"} {
		if data, err := fs.ReadFile(Templates(), name); err != nil || !strings.Contains(string(data), "auth-ui.js") {
			t.Fatalf("template %s unavailable: %v", name, err)
		}
	}
	for _, name := range []string{"auth-ui.js", "auth-ui.css", "qrcode.min.js", "favicon.ico"} {
		if data, err := fs.ReadFile(Assets(), name); err != nil || len(data) == 0 {
			t.Fatalf("asset %s unavailable: %v", name, err)
		}
	}
	for _, name := range []string{"../account.html", "account.html", "config.json", ".env"} {
		if _, err := fs.ReadFile(Assets(), name); err == nil {
			t.Fatalf("non-asset exposed: %s", name)
		}
	}
}
