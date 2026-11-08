package main

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStandaloneBinaryServesAuthenticationResources(t *testing.T) {
	if testing.Short() {
		t.Skip("compiled native integration; run explicitly without -short")
	}
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "stargate")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	build := exec.Command(goBinary, "build", "-o", binary, "./src/cmd/stargate")
	build.Dir = root
	if output, e := build.CombinedOutput(); e != nil {
		t.Fatalf("native build: %v\n%s", e, output)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	port := strings.TrimPrefix(address, "127.0.0.1:")
	_ = listener.Close()
	command := exec.Command(binary)
	command.Dir = t.TempDir()
	command.Env = []string{"AUTH_HOST=" + address, "PORT=" + port, "PASSWORDS=plaintext:synthetic-resource-test-password", "COOKIE_SECURE=false", "AUDIT_LOG_ENABLED=false", "OTLP_ENABLED=false", "LOG_LEVEL=error"}
	for _, key := range []string{"PATH", "SYSTEMROOT", "WINDIR", "TEMP", "TMP"} {
		if value, ok := os.LookupEnv(key); ok {
			command.Env = append(command.Env, key+"="+value)
		}
	}

	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	stopped := false
	stop := func() {
		if !stopped {
			_ = command.Process.Kill()
			_ = command.Wait()
			stopped = true
		}
	}
	defer stop()
	client := &http.Client{Timeout: time.Second}
	base := "http://" + address
	ready := false
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		r, e := client.Get(base + "/healthz")
		if e == nil {
			_ = r.Body.Close()
			if r.StatusCode == 200 {
				ready = true
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		stop()
		t.Fatalf("standalone did not start: %s", output.String())
	}
	for _, item := range []struct{ path, text string }{{"/_login", "/assets/auth-ui.js"}, {"/assets/auth-ui.js", "Authentication requests are never retried"}, {"/assets/auth-ui.css", ".card"}, {"/assets/qrcode.min.js", "QRCode"}} {
		r, e := client.Get(base + item.path)
		if e != nil {
			t.Fatal(e)
		}
		body, e := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if e != nil || r.StatusCode != 200 || !strings.Contains(string(body), item.text) {
			t.Fatalf("standalone resource %s: status%d err%v", item.path, r.StatusCode, e)
		}
	}
}
