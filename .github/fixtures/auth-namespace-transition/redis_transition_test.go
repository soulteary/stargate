package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestMigrationRedisServe(t *testing.T) {
	root := os.Getenv("MIGRATION_ROOT")
	if !filepath.IsAbs(root) {
		t.Fatal("owned local fixture root required")
	}
	mr, e := miniredis.Run()
	if e != nil {
		t.Fatal("miniredis collaborator startup failed")
	}
	defer mr.Close()
	if mr.Set("unrelated:do-not-delete", "preserved") != nil {
		t.Fatal("collaborator sentinel failed")
	}
	stop := make(chan struct{})
	var once sync.Once
	var mu sync.Mutex
	advanced := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("X-Fixture-Key") != "synthetic-namespace-control" {
			w.WriteHeader(401)
			return
		}
		switch r.URL.Path {
		case "/advance-301-seconds":
			mu.Lock()
			defer mu.Unlock()
			if advanced {
				w.WriteHeader(409)
				return
			}
			mr.FastForward(301 * time.Second)
			advanced = true
			w.WriteHeader(204)
		case "/stop":
			if value, e := mr.Get("unrelated:do-not-delete"); e != nil || value != "preserved" {
				w.WriteHeader(500)
				return
			}
			w.WriteHeader(204)
			once.Do(func() { close(stop) })
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	raw, _ := json.Marshal(map[string]string{"redis_addr": mr.Addr(), "control_url": server.URL})
	if e := publishMigrationReady(root, raw); e != nil {
		t.Fatal("atomic readiness publish failed")
	}

	select {
	case <-stop:
	case <-time.After(120 * time.Second):
		t.Fatal("owned collaborator stop timeout")
	}
	if value, e := mr.Get("unrelated:do-not-delete"); e != nil || value != "preserved" {
		t.Fatal("unrelated collaborator data removed")
	}
}

// Only the final rename makes readiness visible to the polling reader.
func publishMigrationReady(root string, raw []byte) error {
	pending := filepath.Join(root, "redis-ready.pending.private.json")
	final := filepath.Join(root, "redis-ready.private.json")
	if _, e := os.Lstat(final); !os.IsNotExist(e) {
		return fmt.Errorf("final readiness already exists or unknown")
	}
	f, e := os.OpenFile(pending, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	n, writeErr := f.Write(raw)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil || n != len(raw) {
		return fmt.Errorf("pending readiness incomplete")
	}
	return os.Rename(pending, final)
}

func TestNamespaceReadyPublication(t *testing.T) {
	root := t.TempDir()
	raw := []byte(`{"redis_addr":"127.0.0.1:1234","control_url":"http://127.0.0.1:1235"}`)
	if e := publishMigrationReady(root, raw); e != nil {
		t.Fatal("complete publication failed")
	}
	got, e := os.ReadFile(filepath.Join(root, "redis-ready.private.json"))
	if e != nil || !bytes.Equal(got, raw) {
		t.Fatal("published bytes differ")
	}
	if _, e := os.Lstat(filepath.Join(root, "redis-ready.pending.private.json")); !os.IsNotExist(e) {
		t.Fatal("pending was not atomically moved")
	}
	if e := publishMigrationReady(root, raw); e == nil {
		t.Fatal("existing final must not be reused")
	}
	blocked := t.TempDir()
	if e := os.WriteFile(filepath.Join(blocked, "redis-ready.pending.private.json"), []byte("partial"), 0600); e != nil {
		t.Fatal("control setup failed")
	}
	if e := publishMigrationReady(blocked, raw); e == nil {
		t.Fatal("spent pending must not be reused")
	}
	if _, e := os.Lstat(filepath.Join(blocked, "redis-ready.private.json")); !os.IsNotExist(e) {
		t.Fatal("incomplete pending was published")
	}
}
