package handlers

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	forwardauth "github.com/soulteary/forwardauth-kit/v3"
	"github.com/soulteary/herald/pkg/herald"
	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/warden/pkg/warden"
)

// configVariables lists every setting of the config package.
// TestConfigVariablesListIsComplete fails when one is missing.
var configVariables = []*config.EnvVariable{
	&config.Debug,
	&config.AuthHost,
	&config.LoginSupportURL,
	&config.LoginPageTitle,
	&config.LoginPageFooterText,
	&config.Passwords,
	&config.PasswordHeaderAuthEnabled,
	&config.PasswordHeaderRequireTrustedProxy,
	&config.UserHeaderName,
	&config.TrustedProxies,
	&config.ProxyHeader,
	&config.CookieDomain,
	&config.CookieSecure,
	&config.CallbackAllowedHosts,
	&config.SessionExchangeSecret,
	&config.SessionExchangeRedirectPath,
	&config.Language,
	&config.Port,
	&config.WardenURL,
	&config.WardenAPIKey,
	&config.WardenHMACKeyID,
	&config.WardenHMACSecret,
	&config.WardenTLSCACertFile,
	&config.WardenTLSClientCert,
	&config.WardenTLSClientKey,
	&config.WardenTLSServerName,
	&config.WardenEnabled,
	&config.HeaderAuthEnabled,
	&config.HeaderAuthSharedSecret,
	&config.HeaderAuthSecretHeader,
	&config.WardenCacheTTL,
	&config.HeraldURL,
	&config.HeraldAPIKey,
	&config.HeraldEnabled,
	&config.HeraldHMACSecret,
	&config.HeraldHMACKeyID,
	&config.HeraldClientIPHeader,
	&config.HeraldTLSCACertFile,
	&config.HeraldTLSClientCert,
	&config.HeraldTLSClientKey,
	&config.HeraldTLSServerName,
	&config.HeraldTOTPEnabled,
	&config.SessionStorageEnabled,
	&config.SessionStorageRedisAddr,
	&config.SessionStorageRedisPassword,
	&config.SessionStorageRedisDB,
	&config.SessionStorageRedisKeyPrefix,
	&config.SessionStorageRedisUsername,
	&config.SessionStorageRedisTLSEnabled,
	&config.SessionStorageRedisTLSCACertFile,
	&config.SessionStorageRedisTLSServerName,
	&config.AuditLogEnabled,
	&config.AuditLogFormat,
	&config.StepUpEnabled,
	&config.StepUpPaths,
	&config.OTLPEnabled,
	&config.OTLPEndpoint,
	&config.WardenSourceMaxAge,
	&config.WardenSourceClockSkew,
	&config.MetricsAPIKey,
	&config.AuthRefreshEnabled,
	&config.AuthRefreshInterval,
	&config.RequestContextTimeout,
	&config.RateLimitLoginMax,
	&config.RateLimitVerificationMax,
	&config.RateLimitWindow,
	&config.LoginSMSEnabled,
	&config.LoginEmailEnabled,
	&config.LoginDingtalkRequireUserID,
	&config.LoginUniformResponses,
	&config.LoginUniformResendCooldown,
}

type lookupUserFunc = func(ctx context.Context, phone, mail string) (*warden.AllowListUser, time.Time, error)

// globalState is the package-level state that tests change and that t.Setenv
// does not put back: config.Initialize copies the environment into the config
// package's variables, and the handlers keep clients, stores and a
// ForwardAuth handler that were built from them.
type globalState struct {
	configValues            []string
	language                i18n.Language
	logger                  *logger.Logger
	forwardAuthHandler      *forwardauth.Handler
	heraldClient            *herald.Client
	heraldClientInitErr     error
	rateLimitStore          RateLimitStore
	challengeContexts       ChallengeContextStore
	uniformDigestKey        []byte
	lookupRefreshUser       lookupUserFunc
	lookupTrustedHeaderUser lookupUserFunc
	authorizationNow        func() time.Time
	uniformNow              func() time.Time
	uniformEntropy          io.Reader
}

func captureGlobalState() globalState {
	state := globalState{
		configValues:            make([]string, len(configVariables)),
		language:                i18n.GetLanguage(),
		logger:                  log,
		forwardAuthHandler:      forwardAuthHandler,
		heraldClient:            heraldClient,
		heraldClientInitErr:     heraldClientInitErr,
		rateLimitStore:          getRateLimitStore(),
		challengeContexts:       getChallengeContextStore(),
		lookupRefreshUser:       lookupRefreshUser,
		lookupTrustedHeaderUser: lookupTrustedHeaderUser,
		authorizationNow:        authorizationNow,
		uniformNow:              uniformNow,
		uniformEntropy:          uniformEntropy,
	}
	for i, variable := range configVariables {
		state.configValues[i] = variable.Value
	}
	uniformDigestKeyState.Lock()
	state.uniformDigestKey = uniformDigestKeyState.key
	uniformDigestKeyState.Unlock()
	return state
}

func (state globalState) restore() {
	for i, variable := range configVariables {
		variable.Value = state.configValues[i]
	}
	// The step-up matcher is derived from the values restored above.
	config.InitStepUpMatcher()
	i18n.SetLanguage(state.language)
	log = state.logger
	forwardAuthHandler = state.forwardAuthHandler

	heraldClient = state.heraldClient
	heraldClientInitErr = state.heraldClientInitErr
	heraldClientInit = sync.Once{}
	if state.heraldClient != nil || state.heraldClientInitErr != nil {
		// A sync.Once cannot be read, so a completed initialisation is
		// recognised by its result.
		heraldClientInit.Do(func() {})
	}

	resetRateLimitStateForTesting()
	SetRateLimitStore(state.rateLimitStore)
	SetChallengeContextStore(state.challengeContexts)
	uniformDigestKeyState.Lock()
	uniformDigestKeyState.key = state.uniformDigestKey
	uniformDigestKeyState.Unlock()
	lookupRefreshUser = state.lookupRefreshUser
	lookupTrustedHeaderUser = state.lookupTrustedHeaderUser
	authorizationNow = state.authorizationNow
	uniformNow = state.uniformNow
	uniformEntropy = state.uniformEntropy
}

// isolateGlobalState gives the test its own rate-limit and challenge-context
// stores and one-shot warnings, and puts the package-level state back when
// the test ends. Call it before the first t.Setenv, config.Initialize or
// assignment to that state. Cleanups run last in, first out, so the state
// comes back after the cleanups the test registers later have run.
//
// The Warden client of the auth package cannot be read from here. It is
// discarded when the test ends, which is the state TestMain starts from.
func isolateGlobalState(t testing.TB) {
	t.Helper()
	state := captureGlobalState()
	t.Cleanup(func() {
		state.restore()
		auth.ResetWardenClientForTesting()
	})
	resetRateLimitStateForTesting()
	SetChallengeContextStore(newMemoryChallengeContextStore())
}

// describe lists the part of the state that later tests depend on. The
// logger, the clocks and the lookup functions are restored by
// isolateGlobalState but not compared: replacing them with an equal value is
// harmless and cannot be told apart from a different one.
func (state globalState) describe() map[string]string {
	described := map[string]string{
		"i18n language":           fmt.Sprint(state.language),
		"ForwardAuth handler":     fmt.Sprintf("%p", state.forwardAuthHandler),
		"Herald client":           fmt.Sprintf("%p", state.heraldClient),
		"Herald client error":     fmt.Sprint(state.heraldClientInitErr),
		"rate-limit store":        describeStore(state.rateLimitStore),
		"challenge-context store": describeStore(state.challengeContexts),
		"uniform digest key":      fmt.Sprintf("%x", state.uniformDigestKey),
	}
	for i, variable := range configVariables {
		described["config "+variable.Name] = strconv.Quote(state.configValues[i])
	}
	return described
}

// describeStore tells a replaced process-local store from the first one only
// by what it holds: an empty one behaves like any other empty one.
func describeStore(store any) string {
	switch memory := store.(type) {
	case *memoryRateLimitStore:
		memory.mu.Lock()
		defer memory.mu.Unlock()
		return fmt.Sprintf("process-local with %d entries", len(memory.buckets))
	case *memoryChallengeContextStore:
		memory.mu.Lock()
		defer memory.mu.Unlock()
		return fmt.Sprintf("process-local with %d entries", len(memory.entries))
	}
	return fmt.Sprintf("%T %p", store, store)
}

// globalStateChanges names what differs between two states, in a fixed order.
func globalStateChanges(before, after globalState) []string {
	was, is := before.describe(), after.describe()
	var changes []string
	for name, value := range is {
		if was[name] != value {
			changes = append(changes, name)
		}
	}
	sort.Strings(changes)
	return changes
}

func TestConfigVariablesListIsComplete(t *testing.T) {
	listed := map[string]bool{}
	for _, variable := range configVariables {
		listed[variable.Name] = true
	}
	declared := declaredConfigVariableNames(t, "../config")
	if len(declared) == 0 {
		t.Fatal("found no EnvVariable declaration in ../config")
	}
	for _, name := range declared {
		if !listed[name] {
			t.Errorf("%s is declared in the config package and missing from configVariables", name)
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
