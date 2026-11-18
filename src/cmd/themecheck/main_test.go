package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	var out bytes.Buffer
	if code := run(nil, &out); code != 2 || !strings.Contains(out.String(), "usage:") {
		t.Fatal(code, out.String())
	}
	out.Reset()
	dir := t.TempDir()
	raw := "contract: 1\nui_contract: 1\nid: test\nname: Test\nversion: 1.0.0\nauthor: Test\nlicense: MIT\nlayout: centered\n"
	if err := os.WriteFile(filepath.Join(dir, "theme.yml"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{dir}, &out); code != 0 || !strings.Contains(out.String(), "ok test") {
		t.Fatal(code, out.String())
	}
	out.Reset()
	if code := run([]string{dir, filepath.Join(dir, "missing")}, &out); code != 1 || !strings.Contains(out.String(), "ok test") || !strings.Contains(out.String(), "FAIL") {
		t.Fatal(code, out.String())
	}
}

func TestJSON(t *testing.T) {
	dir := t.TempDir()
	raw := "contract: 1\nui_contract: 1\nid: test\nname: Test\nversion: 1.0.0\nauthor: Author\nlicense: MIT\nlayout: split\n"
	if err := os.WriteFile(filepath.Join(dir, "theme.yml"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := run([]string{"-json", dir}, &out); code != 0 {
		t.Fatal(code, out.String())
	}
	for _, text := range []string{`"id":"test"`, `"author":"Author"`, `"license":"MIT"`, `"contract":1`, `"ui_contract":1`, `"digest":`} {
		if !strings.Contains(out.String(), text) {
			t.Fatal(out.String())
		}
	}
	if code := run([]string{"-json"}, &out); code != 2 {
		t.Fatal(code)
	}
	if code := run([]string{"-json", dir + "/missing"}, &out); code != 1 {
		t.Fatal(code)
	}
	if code := run([]string{"-json", dir}, brokenWriter{}); code != 1 {
		t.Fatal(code)
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, os.ErrClosed }
