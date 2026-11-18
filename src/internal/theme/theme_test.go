package theme

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validManifest = "contract: 1\nui_contract: 1\nid: test-theme\nname: Test\nversion: 1.0.0\nauthor: Test\nlicense: MIT\nlayout: split\n"

func fixture(t *testing.T, manifest string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "theme.yml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeImage(t *testing.T, dir string) []byte {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.White)
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "space.png"), b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestSnapshotOwnsValidatedResources(t *testing.T) {
	dir := fixture(t, validManifest+"brand:\n  image: space.png\n  heading: '<script>alert(1)</script>'\ntokens:\n  accent: '#123456'\nmodes:\n  dark:\n    background: '#ffffff'\n")
	original := writeImage(t, dir)
	s, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.ID() != "test-theme" || s.Name() != "Test" || s.Version() != "1.0.0" || s.Layout() != "split" || s.Brand().Image != "space.png" {
		t.Fatal("metadata lost")
	}
	if s.ImageURL() != s.BaseURL()+"/assets/space.png" {
		t.Fatal("image URL")
	}
	css, mime, ok := s.Resource("tokens.css")
	if !ok || mime != "text/css; charset=utf-8" || !bytes.Contains(css, []byte("--sg-accent:#123456;")) || !bytes.Contains(css, []byte("@media(prefers-color-scheme:dark)")) {
		t.Fatalf("CSS: %s", css)
	}
	css[0] = 'x'
	again, _, _ := s.Resource("tokens.css")
	if again[0] == 'x' {
		t.Fatal("mutable CSS")
	}
	imageBytes, mime, ok := s.Resource("assets/space.png")
	if !ok || mime != "image/png" || !bytes.Equal(imageBytes, original) {
		t.Fatal("image resource")
	}
	imageBytes[0] = 0
	if err = os.WriteFile(filepath.Join(dir, "assets", "space.png"), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	imageBytes, _, _ = s.Resource("assets/space.png")
	if !bytes.Equal(imageBytes, original) {
		t.Fatal("snapshot changed with file or caller")
	}
	for _, name := range []string{"theme.yml", "LICENSE.txt", "../theme.yml", "assets/no.png"} {
		if _, _, ok = s.Resource(name); ok {
			t.Fatal("private file served", name)
		}
	}
	if _, err = Load(dir); err == nil {
		t.Fatal("invalid replacement accepted")
	}
}

func TestDigestIncludesManifestAndImage(t *testing.T) {
	dir := fixture(t, validManifest+"brand:\n  image: space.png\n")
	writeImage(t, dir)
	first, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	same, err := Load(dir)
	if err != nil || same.BaseURL() != first.BaseURL() {
		t.Fatal("unstable digest")
	}
	var b bytes.Buffer
	if err = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 3, 3))); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "assets", "space.png"), b.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := Load(dir)
	if err != nil || second.BaseURL() == first.BaseURL() {
		t.Fatal("image omitted from digest")
	}
	if err = os.WriteFile(filepath.Join(dir, "theme.yml"), []byte(validManifest+"brand:\n  image: space.png\n  heading: Changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	third, err := Load(dir)
	if err != nil || third.BaseURL() == second.BaseURL() {
		t.Fatal("manifest omitted from digest")
	}
}

func TestManifestRejections(t *testing.T) {
	cases := map[string]string{
		"unknown mode":       validManifest + "modes:\n  sepia:\n    text: '#111827'\n",
		"brand text":         validManifest + "brand:\n  heading: " + strings.Repeat("a", 257) + "\n",
		"contrast":           validManifest + "tokens:\n  text: '#f3f4f6'\n",
		"mode contrast":      validManifest + "modes:\n  dark:\n    text: '#f3f4f6'\n",
		"unknown":            validManifest + "script: evil.js\n",
		"multiple documents": validManifest + "---\nid: other\n",
		"syntax":             validManifest + "tokens: [\n",
		"contract":           strings.Replace(validManifest, "contract: 1", "contract: 2", 1),
		"UI contract":        strings.Replace(validManifest, "ui_contract: 1", "ui_contract: 2", 1),
		"id":                 strings.Replace(validManifest, "test-theme", "../outside", 1),
		"long id":            strings.Replace(validManifest, "test-theme", strings.Repeat("a", 33), 1),
		"missing name":       strings.Replace(validManifest, "name: Test\n", "", 1),
		"long author":        strings.Replace(validManifest, "author: Test", "author: "+strings.Repeat("a", 129), 1),
		"layout":             strings.Replace(validManifest, "layout: split", "layout: arbitrary", 1),
		"image path":         validManifest + "brand:\n  image: ../space.png\n",
		"token":              validManifest + "tokens:\n  arbitrary: red\n",
		"CSS injection":      validManifest + "tokens:\n  text: 'red;display:none'\n",
		"radius":             validManifest + "tokens:\n  radius: 33px\n",
		"radius unit":        validManifest + "tokens:\n  radius: 1rem\n",
		"scale":              validManifest + "tokens:\n  font-scale: NaN\n",
		"large scale":        validManifest + "tokens:\n  font-scale: '2'\n",
		"mode":               validManifest + "modes:\n  dark:\n    text: 'url(https://example.com)'\n",
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(fixture(t, raw)); err == nil {
				t.Fatal("accepted invalid theme")
			}
		})
	}
	for _, tokens := range []string{"radius: 0px", "radius: 32px", "font-scale: '0.8'", "font-scale: '1.5'"} {
		if _, err := Load(fixture(t, validManifest+"tokens:\n  "+tokens+"\n")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFilesystemRejections(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		if _, err := Load(filepath.Join(t.TempDir(), "missing")); err == nil {
			t.Fatal("accepted missing directory")
		}
	})
	t.Run("file", func(t *testing.T) {
		dir := fixture(t, validManifest)
		if _, err := Load(filepath.Join(dir, "theme.yml")); err == nil {
			t.Fatal("accepted file")
		}
	})
	t.Run("no manifest", func(t *testing.T) {
		if _, err := Load(t.TempDir()); err == nil {
			t.Fatal("accepted empty directory")
		}
	})
	t.Run("large manifest", func(t *testing.T) {
		if _, err := Load(fixture(t, strings.Repeat(" ", manifestLimit+1))); err == nil {
			t.Fatal("accepted large manifest")
		}
	})
	for _, name := range []string{"theme.css", "evil.js", "templates", ".env"} {
		t.Run(name, func(t *testing.T) {
			dir := fixture(t, validManifest)
			if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(dir); err == nil {
				t.Fatal("accepted undeclared file")
			}
		})
	}
	t.Run("license", func(t *testing.T) {
		dir := fixture(t, validManifest)
		if err := os.WriteFile(filepath.Join(dir, "LICENSE.txt"), []byte("license"), 0600); err != nil {
			t.Fatal(err)
		}
		s, err := Load(dir)
		if err != nil {
			t.Fatal(err)
		}
		if s.ImageURL() != "" {
			t.Fatal("unexpected image")
		}
	})
	for _, kind := range []string{"root", "manifest", "assets", "image", "license"} {
		t.Run("link "+kind, func(t *testing.T) {
			dir := fixture(t, validManifest+"brand:\n  image: space.png\n")
			writeImage(t, dir)
			target := filepath.Join(t.TempDir(), "outside")
			if err := os.WriteFile(target, []byte(validManifest), 0600); err != nil {
				t.Fatal(err)
			}
			name := map[string]string{"root": dir, "manifest": filepath.Join(dir, "theme.yml"), "assets": filepath.Join(dir, "assets"), "image": filepath.Join(dir, "assets", "space.png"), "license": filepath.Join(dir, "LICENSE.txt")}[kind]
			if err := os.RemoveAll(name); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, name); err != nil {
				t.Skip(err)
			}
			if _, err := Load(dir); err == nil {
				t.Fatal("accepted link")
			}
		})
	}
	for _, kind := range []string{"missing", "invalid", "large", "extension", "directory", "extra"} {
		t.Run("image "+kind, func(t *testing.T) {
			raw := validManifest + "brand:\n  image: space.png\n"
			if kind == "extension" {
				raw = validManifest + "brand:\n  image: space.svg\n"
			}
			dir := fixture(t, raw)
			data := writeImage(t, dir)
			name := filepath.Join(dir, "assets", "space.png")
			var err error
			switch kind {
			case "missing":
				err = os.Remove(name)
			case "invalid":
				err = os.WriteFile(name, []byte("not an image"), 0600)
			case "large":
				err = os.WriteFile(name, make([]byte, assetLimit+1), 0600)
			case "extension":
				err = os.WriteFile(filepath.Join(dir, "assets", "space.svg"), data, 0600)
			case "directory":
				err = os.Remove(name)
				if err == nil {
					err = os.Mkdir(name, 0700)
				}
			case "extra":
				err = os.WriteFile(filepath.Join(dir, "assets", "extra.png"), data, 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = Load(dir); err == nil {
				t.Fatal("accepted invalid asset")
			}
		})
	}
}

func TestConfiguredSelection(t *testing.T) {
	for _, directory := range []string{"", filepath.Join(t.TempDir(), "missing")} {
		if s, err := LoadConfigured(directory, true); s != nil || err != nil {
			t.Fatal("safe mode read a theme")
		}
	}
	if s, err := LoadConfigured("", false); s != nil || err != nil {
		t.Fatal("default selection")
	}
	if _, err := LoadConfigured("missing", false); err == nil {
		t.Fatal("invalid explicit selection ignored")
	}
	if s, err := LoadConfigured(fixture(t, validManifest), false); s == nil || err != nil {
		t.Fatal(err)
	}
}
