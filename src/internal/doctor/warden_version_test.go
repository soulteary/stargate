package doctor

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// versionManifest builds a manifest through the JSON reader, so the
// warden_image field is exercised the way an operator supplies it.
func versionManifest(t *testing.T, image string, change func(*Manifest)) Manifest {
	t.Helper()
	m := fixture()
	m.Gateway["WARDEN_HMAC_AUDIENCE"] = "warden-a"
	m.Warden["WARDEN_HMAC_AUDIENCES"] = "warden-a"
	if change != nil {
		change(&m)
	}
	document := map[string]any{"gateway": m.Gateway}
	if m.Warden != nil {
		document["warden"] = m.Warden
	}
	if image != "" {
		document["warden_image"] = image
	}
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	read, err := ReadManifest(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("manifest with warden_image was rejected: %v", err)
	}
	return read
}

func TestWardenVersionFromImageReference(t *testing.T) {
	const digest = "@sha256:3bcec4df695cdb7ee8951e6a0ea378d45b9342df76f46cfa702db6e1c4f2608e"
	for _, tc := range []struct {
		name, image string
		change      func(*Manifest)
		want        string
	}{
		{"not supplied", "", nil, "unknown"},
		{"1.8.0", "ghcr.io/soulteary/warden:1.8.0", nil, "pass"},
		{"1.8.0 with digest", "ghcr.io/soulteary/warden:1.8.0" + digest, nil, "pass"},
		{"surrounding space", " ghcr.io/soulteary/warden:1.8.0 ", nil, "pass"},
		{"minor tag 1.8", "ghcr.io/soulteary/warden:1.8", nil, "pass"},
		{"later patch", "ghcr.io/soulteary/warden:1.8.3", nil, "pass"},
		{"later minor", "ghcr.io/soulteary/warden:1.10.0", nil, "pass"},
		{"later major", "ghcr.io/soulteary/warden:2.0.0", nil, "pass"},
		{"1.7.0", "ghcr.io/soulteary/warden:1.7.0", nil, "fail"},
		{"1.7.0 with digest", "ghcr.io/soulteary/warden:1.7.0" + digest, nil, "fail"},
		{"1.6.1", "ghcr.io/soulteary/warden:1.6.1", nil, "fail"},
		{"minor tag 1.7", "ghcr.io/soulteary/warden:1.7", nil, "fail"},
		{"0.9.0", "ghcr.io/soulteary/warden:0.9.0", nil, "fail"},
		{"digest only", "ghcr.io/soulteary/warden" + digest, nil, "unknown"},
		{"no tag", "ghcr.io/soulteary/warden", nil, "unknown"},
		{"latest", "ghcr.io/soulteary/warden:latest", nil, "unknown"},
		{"major tag", "ghcr.io/soulteary/warden:1", nil, "unknown"},
		{"v prefix", "ghcr.io/soulteary/warden:v1.7.0", nil, "unknown"},
		{"pre-release suffix", "ghcr.io/soulteary/warden:1.7.0-rc1", nil, "unknown"},
		{"four parts", "ghcr.io/soulteary/warden:1.7.0.1", nil, "unknown"},
		{"leading zero", "ghcr.io/soulteary/warden:1.07.0", nil, "unknown"},
		{"overlong number", "ghcr.io/soulteary/warden:1.1234567.0", nil, "unknown"},
		{"branch tag", "ghcr.io/soulteary/warden:main", nil, "unknown"},
		{"custom repository", "registry.example.test/custom/warden:1.7.0", nil, "unknown"},
		{"other registry", "docker.io/soulteary/warden:1.7.0", nil, "unknown"},
		{"registry port without tag", "localhost:5000/warden", nil, "unknown"},
		{"repository suffix", "ghcr.io/soulteary/warden-fork:1.7.0", nil, "unknown"},
		{"not a reference", "image-in-manifest", nil, "unknown"},
		{"mTLS with old release", "ghcr.io/soulteary/warden:1.7.0", func(m *Manifest) { m.Gateway["WARDEN_TLS_CLIENT_CERT_FILE"] = "not-read" }, "unknown"},
		{"server not supplied with old release", "ghcr.io/soulteary/warden:1.7.0", func(m *Manifest) { m.Warden = nil }, "fail"},
		{"audience not listed with old release", "ghcr.io/soulteary/warden:1.7.0", func(m *Manifest) { m.Warden["WARDEN_HMAC_AUDIENCES"] = "warden-b" }, "fail"},
		{"no audience with old release", "ghcr.io/soulteary/warden:1.7.0", func(m *Manifest) { delete(m.Gateway, "WARDEN_HMAC_AUDIENCE") }, ""},
		{"blank audience with old release", "ghcr.io/soulteary/warden:1.7.0", func(m *Manifest) { m.Gateway["WARDEN_HMAC_AUDIENCE"] = "  " }, ""},
		{"no audience and not supplied", "", func(m *Manifest) { delete(m.Gateway, "WARDEN_HMAC_AUDIENCE") }, ""},
		{"audience without credentials", "ghcr.io/soulteary/warden:1.7.0", func(m *Manifest) { delete(m.Gateway, "WARDEN_HMAC_SECRET") }, ""},
		{"invalid audience", "ghcr.io/soulteary/warden:1.7.0", func(m *Manifest) { m.Gateway["WARDEN_HMAC_AUDIENCE"] = "warden a" }, ""},
		{"Warden disabled", "ghcr.io/soulteary/warden:1.7.0", func(m *Manifest) {
			m.Gateway["WARDEN_ENABLED"] = "false"
			m.Gateway["PASSWORDS"] = "plaintext:synthetic-password"
		}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Inspect(versionManifest(t, tc.image, tc.change))
			if got := find(r, "warden.version").Status; got != tc.want {
				t.Fatalf("warden.version status=%q, want %q", got, tc.want)
			}
			data, _ := json.Marshal(r)
			for _, value := range []string{"sha256", "3bcec4df", "registry.example.test", "image-in-manifest", "warden-fork", "1.7.0", "1.6.1", "rc1"} {
				if bytes.Contains(data, []byte(value)) {
					t.Fatal("diagnostic exposed an input value")
				}
			}
		})
	}
}

func TestWardenVersionExitStatus(t *testing.T) {
	for _, tc := range []struct {
		name, image string
		code        int
	}{
		{"older release fails", "ghcr.io/soulteary/warden:1.7.0", 1},
		{"undetermined release is incomplete", "ghcr.io/soulteary/warden:latest", 3},
		{"release not supplied is incomplete", "", 3},
		{"1.8.0 passes", "ghcr.io/soulteary/warden:1.8.0", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if code := Inspect(versionManifest(t, tc.image, nil)).ExitCode(); code != tc.code {
				t.Fatalf("exit=%d, want %d", code, tc.code)
			}
		})
	}
	// Without an audience the gateway signs v2 requests, which every Warden
	// release this check distinguishes accepts: an older image is not a failure.
	m := versionManifest(t, "ghcr.io/soulteary/warden:1.7.0", func(m *Manifest) {
		delete(m.Gateway, "WARDEN_HMAC_AUDIENCE")
		delete(m.Warden, "WARDEN_HMAC_AUDIENCES")
	})
	if code := Inspect(m).ExitCode(); code != 0 {
		t.Fatalf("no audience exit=%d", code)
	}
}

func TestWardenVersionFromEnvironmentIsUnknown(t *testing.T) {
	values := map[string]string{"AUTH_HOST": "auth.example.test", "WARDEN_ENABLED": "true", "WARDEN_URL": "https://directory.example.test", "WARDEN_HMAC_KEY_ID": "gateway", "WARDEN_HMAC_SECRET": "synthetic-warden-secret-123456789", "WARDEN_HMAC_AUDIENCE": "warden-a"}
	var out, errOut bytes.Buffer
	code := Run([]string{"--json"}, func(key string) (string, bool) { v, ok := values[key]; return v, ok }, &out, &errOut)
	var report Report
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if c := find(report, "warden.version"); code != 3 || c.Status != "unknown" || !strings.Contains(c.Message, "warden_image") {
		t.Fatalf("exit=%d status=%q", code, c.Status)
	}
}
