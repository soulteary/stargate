package config

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// configVariables lists every setting of this package.
// TestConfigVariablesListIsComplete fails when one is missing.
var configVariables = []*EnvVariable{
	&Debug,
	&AuthHost,
	&LoginSupportURL,
	&LoginPageTitle,
	&LoginPageFooterText,
	&Passwords,
	&PasswordHeaderAuthEnabled,
	&PasswordHeaderRequireTrustedProxy,
	&UserHeaderName,
	&TrustedProxies,
	&ProxyHeader,
	&CookieDomain,
	&CookieSecure,
	&CallbackAllowedHosts,
	&SessionExchangeSecret,
	&SessionExchangeRedirectPath,
	&Language,
	&Port,
	&WardenURL,
	&WardenAPIKey,
	&WardenHMACKeyID,
	&WardenHMACSecret,
	&WardenTLSCACertFile,
	&WardenTLSClientCert,
	&WardenTLSClientKey,
	&WardenTLSServerName,
	&WardenEnabled,
	&HeaderAuthEnabled,
	&HeaderAuthSharedSecret,
	&HeaderAuthSecretHeader,
	&WardenCacheTTL,
	&HeraldURL,
	&HeraldAPIKey,
	&HeraldEnabled,
	&HeraldHMACSecret,
	&HeraldHMACKeyID,
	&HeraldClientIPHeader,
	&HeraldTLSCACertFile,
	&HeraldTLSClientCert,
	&HeraldTLSClientKey,
	&HeraldTLSServerName,
	&HeraldTOTPEnabled,
	&SessionStorageEnabled,
	&SessionStorageRedisAddr,
	&SessionStorageRedisPassword,
	&SessionStorageRedisDB,
	&SessionStorageRedisKeyPrefix,
	&SessionStorageRedisUsername,
	&SessionStorageRedisTLSEnabled,
	&SessionStorageRedisTLSCACertFile,
	&SessionStorageRedisTLSServerName,
	&AuditLogEnabled,
	&AuditLogFormat,
	&StepUpEnabled,
	&StepUpPaths,
	&OTLPEnabled,
	&OTLPEndpoint,
	&WardenSourceMaxAge,
	&WardenSourceClockSkew,
	&MetricsAPIKey,
	&AuthRefreshEnabled,
	&AuthRefreshInterval,
	&RequestContextTimeout,
	&RateLimitLoginMax,
	&RateLimitVerificationMax,
	&RateLimitWindow,
	&LoginSMSEnabled,
	&LoginEmailEnabled,
	&LoginDingtalkRequireUserID,
	&LoginUniformResponses,
	&LoginUniformResendCooldown,
}

// globalState is the package-level state that Initialize and the tests
// change. t.Setenv puts the environment back when a test ends, but not the
// values that Initialize copied from it.
type globalState struct {
	values        []string
	language      i18n.Language
	logger        *logger.Logger
	stepUpMatcher *StepUpMatcher
}

func captureGlobalState() globalState {
	state := globalState{
		values:        make([]string, len(configVariables)),
		language:      i18n.GetLanguage(),
		logger:        log,
		stepUpMatcher: stepUpMatcher,
	}
	for i, variable := range configVariables {
		state.values[i] = variable.Value
	}
	return state
}

func (state globalState) restore() {
	for i, variable := range configVariables {
		variable.Value = state.values[i]
	}
	i18n.SetLanguage(state.language)
	log = state.logger
	stepUpMatcher = state.stepUpMatcher
}

// isolateGlobalState puts the package-level state back when the test ends.
// Call it before the first t.Setenv, Initialize or assignment to that state.
// Cleanups run last in, first out, so the state comes back after the cleanups
// the test registers later have run.
func isolateGlobalState(t testing.TB) {
	t.Helper()
	state := captureGlobalState()
	t.Cleanup(state.restore)
}

// globalStateChanges names what differs between two states, in a fixed
// order. The logger is restored by isolateGlobalState but not compared: no
// test depends on which logger an earlier Initialize was given.
func globalStateChanges(before, after globalState) []string {
	var changes []string
	for i, variable := range configVariables {
		if before.values[i] != after.values[i] {
			changes = append(changes, variable.Name)
		}
	}
	if before.language != after.language {
		changes = append(changes, "i18n language")
	}
	if before.stepUpMatcher != after.stepUpMatcher {
		changes = append(changes, "step-up matcher")
	}
	sort.Strings(changes)
	return changes
}

func TestMain(m *testing.M) {
	initial := captureGlobalState()
	code := m.Run()
	// A test that leaves this state changed makes the tests after it depend on
	// the order they run in.
	if changes := globalStateChanges(initial, captureGlobalState()); code == 0 && len(changes) > 0 {
		fmt.Fprintf(os.Stderr, "FAIL: package-level state was not restored when the tests ended: %s\nCall isolateGlobalState(t) at the start of the test that changes it.\n", strings.Join(changes, ", "))
		code = 1
	}
	os.Exit(code)
}

func TestConfigVariablesListIsComplete(t *testing.T) {
	listed := map[string]bool{}
	for _, variable := range configVariables {
		listed[variable.Name] = true
	}
	declared := declaredConfigVariableNames(t, ".")
	if len(declared) == 0 {
		t.Fatal("found no EnvVariable declaration in this package")
	}
	for _, name := range declared {
		if !listed[name] {
			t.Errorf("%s is declared in this package and missing from configVariables", name)
		}
	}
}

// declaredConfigVariableNames reads the Name of every package-level
// "X = EnvVariable{...}" declaration in the non-test files of dir.
func declaredConfigVariableNames(t *testing.T, dir string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.VAR {
				continue
			}
			for _, spec := range general.Specs {
				for _, value := range spec.(*ast.ValueSpec).Values {
					if name, ok := envVariableName(value); ok {
						names = append(names, name)
					}
				}
			}
		}
	}
	return names
}

// envVariableName returns the Name field of an EnvVariable{...} literal.
func envVariableName(value ast.Expr) (string, bool) {
	literal, ok := value.(*ast.CompositeLit)
	if !ok {
		return "", false
	}
	if kind, ok := literal.Type.(*ast.Ident); !ok || kind.Name != "EnvVariable" {
		return "", false
	}
	for _, element := range literal.Elts {
		field, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, ok := field.Key.(*ast.Ident); !ok || key.Name != "Name" {
			continue
		}
		if name, ok := field.Value.(*ast.BasicLit); ok {
			unquoted, err := strconv.Unquote(name.Value)
			return unquoted, err == nil
		}
	}
	return "", false
}
