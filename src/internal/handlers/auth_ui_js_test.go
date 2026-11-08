package handlers

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAuthenticationJavaScript(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is unavailable; browser/static asset checks run on CI")
	}
	command := exec.Command(node, "--test", filepath.Join("..", "..", "..", ".github", "tests", "auth-ui.test.cjs"))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("authentication script regressions: %v\n%s", err, output)
	}
}
