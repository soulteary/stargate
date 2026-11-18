// Package theme loads declarative authentication themes. Themes supply data,
// never templates, scripts or stylesheets; authentication markup stays in core.
package theme

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

const (
	Contract      = 1
	manifestLimit = 64 << 10
	assetLimit    = 2 << 20
)

// Manifest is version one of the theme format. Unknown fields are errors.
type Manifest struct {
	Contract   int                          `yaml:"contract"`
	ID         string                       `yaml:"id"`
	Name       string                       `yaml:"name"`
	Version    string                       `yaml:"version"`
	Author     string                       `yaml:"author"`
	License    string                       `yaml:"license"`
	UIContract int                          `yaml:"ui_contract"`
	Layout     string                       `yaml:"layout"`
	Tokens     map[string]string            `yaml:"tokens"`
	Modes      map[string]map[string]string `yaml:"modes"`
	Brand      Brand                        `yaml:"brand"`
}

// Brand describes optional decorative content, escaped by html/template.
type Brand struct {
	Image   string `yaml:"image"`
	Heading string `yaml:"heading"`
	Caption string `yaml:"caption"`
	Eyebrow string `yaml:"eyebrow"`
}

type asset struct {
	data []byte
	mime string
}

// Snapshot owns all validated bytes. Editing a directory cannot change it.
type Snapshot struct {
	manifest Manifest
	digest   string
	css      []byte
	assets   map[string]asset
}

func (s *Snapshot) ID() string      { return s.manifest.ID }
func (s *Snapshot) Author() string  { return s.manifest.Author }
func (s *Snapshot) License() string { return s.manifest.License }
func (s *Snapshot) Name() string    { return s.manifest.Name }
func (s *Snapshot) Version() string { return s.manifest.Version }
func (s *Snapshot) Layout() string  { return s.manifest.Layout }
func (s *Snapshot) Brand() Brand    { return s.manifest.Brand }

func (s *Snapshot) BaseURL() string { return "/_theme/" + s.digest }
func (s *Snapshot) ImageURL() string {
	if s.manifest.Brand.Image == "" {
		return ""
	}
	return s.BaseURL() + "/assets/" + s.manifest.Brand.Image
}

// Resource returns a copy so callers cannot modify a validated snapshot.
func (s *Snapshot) Resource(name string) ([]byte, string, bool) {
	if name == "tokens.css" {
		return bytes.Clone(s.css), "text/css; charset=utf-8", true
	}
	a, ok := s.assets[name]
	return bytes.Clone(a.data), a.mime, ok
}

var (
	identifier = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	assetName  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)
	hexColor   = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	defaults   = map[string]string{
		"background": "#f3f4f6", "surface": "#ffffff", "text": "#111827",
		"muted": "#4b5563", "accent": "#254fa0", "accent-text": "#ffffff",
		"border": "#9ca3af", "danger": "#b91c1c", "focus": "#2563eb",
		"art": "#101820", "art-text": "#f9fafb", "radius": "14px", "font-scale": "1",
	}
)

// LoadConfigured ignores all external resources in safe mode. An explicitly
// selected invalid theme is an error rather than a silent fallback.
func LoadConfigured(directory string, safe bool) (*Snapshot, error) {
	if safe || directory == "" {
		return nil, nil
	}
	return Load(directory)
}

// Load reads a single directory through os.Root. Only theme.yml and its declared
// image are accepted; links and unrecognized files are rejected.
func Load(directory string) (*Snapshot, error) {
	info, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("theme directory must be a directory, not a link")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	raw, err := readRegular(root, "theme.yml", manifestLimit)
	if err != nil {
		return nil, fmt.Errorf("theme.yml: %w", err)
	}
	var m Manifest
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err = decoder.Decode(&m); err != nil {
		return nil, fmt.Errorf("theme.yml: %w", err)
	}
	var extra any
	if err = decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("theme.yml: expected one YAML document")
	}
	if err = validate(m); err != nil {
		return nil, fmt.Errorf("theme.yml: %w", err)
	}
	s := &Snapshot{manifest: m, assets: map[string]asset{}}
	allowed := map[string]bool{"theme.yml": true, "LICENSE.txt": true}
	if m.Brand.Image != "" {
		allowed["assets"] = true
		name := "assets/" + m.Brand.Image
		allowed[name] = true
		data, readErr := readRegular(root, name, assetLimit)
		if readErr != nil {
			return nil, fmt.Errorf("%s: %w", name, readErr)
		}
		cfg, format, decodeErr := image.DecodeConfig(bytes.NewReader(data))
		expected := map[string]string{".png": "png", ".jpg": "jpeg", ".jpeg": "jpeg"}[strings.ToLower(filepath.Ext(name))]
		if decodeErr != nil || expected == "" || format != expected || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 8192 || cfg.Height > 8192 || int64(cfg.Width)*int64(cfg.Height) > 32_000_000 {
			return nil, fmt.Errorf("%s: expected PNG or JPEG, at most 8192px per side and 32 million pixels", name)
		}
		mime := "image/" + format
		s.assets[name] = asset{data: data, mime: mime}
	}
	// Do not admit undeclared code or files into a theme package.
	if err = checkEntries(root, ".", allowed); err != nil {
		return nil, err
	}
	hash := sha256.New()
	_, _ = hash.Write(raw)
	if a, ok := s.assets["assets/"+m.Brand.Image]; ok {
		_, _ = hash.Write(a.data)
	}
	s.digest = hex.EncodeToString(hash.Sum(nil))
	s.css = []byte(s.stylesheet())
	return s, nil
}

func readRegular(root *os.Root, name string, limit int64) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("expected a regular file of at most %d bytes", limit)
	}
	// Check the intermediate directory too; Root itself prevents escaping it.
	if strings.HasPrefix(name, "assets/") {
		dir, statErr := root.Lstat("assets")
		if statErr != nil {
			return nil, statErr
		}
		if !dir.IsDir() || dir.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("assets must be a directory, not a link")
		}
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	actual, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !actual.Mode().IsRegular() {
		return nil, fmt.Errorf("expected a regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file exceeds %d bytes", limit)
	}
	return data, nil
}

func checkEntries(root *os.Root, directory string, allowed map[string]bool) error {
	f, err := root.Open(directory)
	if err != nil {
		return err
	}
	entries, err := f.ReadDir(65)
	_ = f.Close()
	if err != nil && err != io.EOF {
		return err
	}
	if len(entries) > 64 {
		return fmt.Errorf("too many theme files")
	}
	for _, e := range entries {
		name := e.Name()
		if directory != "." {
			name = directory + "/" + name
		}
		if !allowed[name] {
			return fmt.Errorf("unsupported theme file: %s", name)
		}
		if e.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("theme links are not supported: %s", name)
		}
		switch name {
		case "assets":
			if !e.IsDir() {
				return fmt.Errorf("assets must be a directory")
			}
			if err = checkEntries(root, name, allowed); err != nil {
				return err
			}
		case "LICENSE.txt":
			if _, err = readRegular(root, name, manifestLimit); err != nil {
				return err
			}
		}
	}
	return nil
}

func validate(m Manifest) error {
	if m.Contract != Contract || m.UIContract != Contract {
		return fmt.Errorf("contract and ui_contract must be 1")
	}
	if len(m.ID) > 32 || !identifier.MatchString(m.ID) {
		return fmt.Errorf("invalid id")
	}
	for _, field := range []struct{ name, value string }{{"name", m.Name}, {"version", m.Version}, {"author", m.Author}, {"license", m.License}} {
		if strings.TrimSpace(field.value) == "" || len(field.value) > 128 || !utf8.ValidString(field.value) {
			return fmt.Errorf("invalid %s", field.name)
		}
	}
	if m.Layout != "centered" && m.Layout != "split" {
		return fmt.Errorf("layout must be centered or split")
	}
	if m.Brand.Image != "" && !assetName.MatchString(m.Brand.Image) {
		return fmt.Errorf("invalid brand image name")
	}
	for _, value := range []string{m.Brand.Heading, m.Brand.Caption, m.Brand.Eyebrow} {
		if len(value) > 256 || !utf8.ValidString(value) || strings.ContainsAny(value, "\x00\r\n") {
			return fmt.Errorf("invalid brand text")
		}
	}
	if err := validateTokens(m.Tokens); err != nil {
		return err
	}
	for mode, tokens := range m.Modes {
		if mode != "light" && mode != "dark" {
			return fmt.Errorf("unsupported mode: %s", mode)
		}
		if err := validateTokens(tokens); err != nil {
			return err
		}
	}
	if err := validateContrast(m.Tokens, nil); err != nil {
		return err
	}
	for mode, tokens := range m.Modes {
		if err := validateContrast(m.Tokens, tokens); err != nil {
			return fmt.Errorf("%s: %w", mode, err)
		}
	}
	return nil
}

func validateTokens(tokens map[string]string) error {
	for key, value := range tokens {
		if _, ok := defaults[key]; !ok {
			return fmt.Errorf("unknown token: %s", key)
		}
		switch key {
		case "radius":
			n, err := strconv.Atoi(strings.TrimSuffix(value, "px"))
			if err != nil || n < 0 || n > 32 || value != strconv.Itoa(n)+"px" {
				return fmt.Errorf("invalid radius")
			}
		case "font-scale":
			n, err := strconv.ParseFloat(value, 64)
			if err != nil || !(n >= 0.8 && n <= 1.5) || len(value) > 4 {
				return fmt.Errorf("invalid font-scale")
			}
		default:
			if !hexColor.MatchString(value) {
				return fmt.Errorf("invalid color: %s", key)
			}
		}
	}
	return nil
}

func (s *Snapshot) stylesheet() string {
	rule := func(override map[string]string) string {
		keys := make([]string, 0, len(defaults))
		for key := range defaults {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString(".auth-themed{")
		for _, key := range keys {
			value := defaults[key]
			if v, ok := s.manifest.Tokens[key]; ok {
				value = v
			}
			if v, ok := override[key]; ok {
				value = v
			}
			fmt.Fprintf(&b, "--sg-%s:%s;", key, value)
		}
		b.WriteString("}")
		return b.String()
	}
	css := rule(nil)
	for _, mode := range []string{"light", "dark"} {
		if tokens, ok := s.manifest.Modes[mode]; ok {
			css += "@media(prefers-color-scheme:" + mode + "){" + rule(tokens) + "}"
		}
	}
	return css
}

func validateContrast(base, override map[string]string) error {
	values := map[string]string{}
	for key, value := range defaults {
		values[key] = value
	}
	for key, value := range base {
		values[key] = value
	}
	for key, value := range override {
		values[key] = value
	}
	for _, pair := range [][2]string{{"text", "background"}, {"text", "surface"}, {"muted", "background"}, {"muted", "surface"}, {"accent", "background"}, {"accent", "surface"}, {"accent-text", "accent"}, {"danger", "background"}, {"danger", "surface"}, {"art-text", "art"}} {
		a, b := luminance(values[pair[0]]), luminance(values[pair[1]])
		if a < b {
			a, b = b, a
		}
		if (a+0.05)/(b+0.05) < 4.5 {
			return fmt.Errorf("%s and %s need at least 4.5:1 contrast", pair[0], pair[1])
		}
	}
	return nil
}

func luminance(value string) float64 {
	raw, _ := hex.DecodeString(value[1:]) // validateTokens has checked every color.
	channel := func(c byte) float64 {
		v := float64(c) / 255
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(raw[0]) + 0.7152*channel(raw[1]) + 0.0722*channel(raw[2])
}
