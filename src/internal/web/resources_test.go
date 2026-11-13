package web

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltInResourcesAreExplicitAndScoped(t *testing.T) {
	for _, name := range []string{"login.html", "login.warden.html", "account.html", "totp_start.html", "totp_enroll.html", "totp_revoke.html", "step_up.html"} {
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

// Every embedded file must also be allowed into the Docker build context.
// The source image compiles the go:embed directive inside that context, and
// the release image copies the template directory from it and serves that
// copy in preference to the embedded one. A file that is embedded but not
// allowed breaks the first and is missing from the second.
func TestBuiltInResourcesAreInTheDockerContext(t *testing.T) {
	ignore, err := os.ReadFile(filepath.Join("..", "..", "..", ".dockerignore"))
	if err != nil {
		t.Fatalf("read .dockerignore: %v", err)
	}
	allowed := map[string]bool{}
	for _, line := range strings.Split(string(ignore), "\n") {
		if rule := strings.TrimSpace(line); strings.HasPrefix(rule, "!") {
			allowed[strings.TrimPrefix(rule, "!")] = true
		}
	}
	checked := 0
	err = fs.WalkDir(resources, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		checked++
		if path := "src/internal/web/" + name; !allowed[path] {
			t.Errorf("%s is embedded but .dockerignore has no line !%s", name, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded resources: %v", err)
	}
	if checked == 0 {
		t.Fatal("no embedded resources found")
	}
}
