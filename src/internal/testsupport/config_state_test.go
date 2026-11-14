package testsupport

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

func TestConfigVariablesListIsComplete(t *testing.T) {
	listed := map[string]bool{}
	for _, variable := range ConfigVariables() {
		if listed[variable.Name] {
			t.Errorf("%s is listed twice in ConfigVariables", variable.Name)
		}
		listed[variable.Name] = true
	}
	declared, err := DeclaredConfigVariableNames("../config")
	if err != nil {
		t.Fatal(err)
	}
	if len(declared) == 0 {
		t.Fatal("found no EnvVariable declaration in ../config")
	}
	for _, name := range declared {
		if !listed[name] {
			t.Errorf("%s is declared in the config package and missing from ConfigVariables", name)
		}
	}
}

func TestRestorePutsBackValuesLanguageAndStepUpMatcher(t *testing.T) {
	initial := CaptureConfig()
	t.Cleanup(initial.Restore)

	config.StepUpEnabled.Value = "true"
	config.StepUpPaths.Value = "/admin"
	config.InitStepUpMatcher()
	i18n.SetLanguage(i18n.LangEN)
	before := CaptureConfig()

	config.AuthHost.Value = "changed.example.com"
	config.StepUpPaths.Value = "/other"
	config.InitStepUpMatcher()
	i18n.SetLanguage(i18n.LangDE)

	want := []string{"config AUTH_HOST", "config STEP_UP_PATHS", "i18n language"}
	if got := ConfigChanges(before, CaptureConfig()); !reflect.DeepEqual(got, want) {
		t.Fatalf("ConfigChanges = %v, want %v", got, want)
	}

	before.Restore()

	if got := ConfigChanges(before, CaptureConfig()); len(got) != 0 {
		t.Fatalf("state still differs after Restore: %v", got)
	}
	if !config.GetStepUpMatcher().RequiresStepUp("/admin") || config.GetStepUpMatcher().RequiresStepUp("/other") {
		t.Fatal("Restore did not rebuild the step-up matcher from the restored values")
	}
}

func TestDeclaredConfigVariableNamesReadsOnlyEnvVariableLiterals(t *testing.T) {
	dir := t.TempDir()
	source := `package sample

func helper() {}

const fixed = "CONST"

var (
	First    = EnvVariable{Name: "FIRST", Required: true}
	Second   = EnvVariable{Required: true, Name: "SECOND"}
	Unnamed  = EnvVariable{Required: true}
	Computed = EnvVariable{Name: fixed}
	Bare     = EnvVariable{"positional"}
	Other    = Different{Name: "OTHER"}
	Remote   = pkg.EnvVariable{Name: "REMOTE"}
	Plain    = "PLAIN"
)
`
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	// A test file is skipped without being parsed.
	if err := os.WriteFile(filepath.Join(dir, "sample_test.go"), []byte("not Go"), 0o600); err != nil {
		t.Fatal(err)
	}

	names, err := DeclaredConfigVariableNames(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"FIRST", "SECOND"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
}

func TestDeclaredConfigVariableNamesReportsUnreadableSources(t *testing.T) {
	if _, err := DeclaredConfigVariableNames("["); err == nil {
		t.Fatal("a malformed directory pattern was accepted")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.go"), []byte("not Go"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := DeclaredConfigVariableNames(dir); err == nil {
		t.Fatal("a file that does not parse was accepted")
	}
}
