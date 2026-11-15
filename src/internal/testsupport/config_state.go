// Package testsupport holds helpers shared by the tests of several packages.
// Only test files import it. It has no init function and no package-level
// side effects, and the stargate binary does not link it.
package testsupport

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// ConfigVariables returns every setting of the config package. The test of
// this package fails when a setting declared there is missing here.
func ConfigVariables() []*config.EnvVariable {
	return []*config.EnvVariable{
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
		&config.WardenHMACAudience,
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
}

// ConfigState is the part of the config package's state that other packages
// can read: the value of every setting and the language that
// config.Initialize selects from them. t.Setenv puts the environment back
// when a test ends, but not the values config.Initialize copied from it.
type ConfigState struct {
	values   []string
	language i18n.Language
}

// CaptureConfig records the current configuration state.
func CaptureConfig() ConfigState {
	variables := ConfigVariables()
	state := ConfigState{values: make([]string, len(variables)), language: i18n.GetLanguage()}
	for i, variable := range variables {
		state.values[i] = variable.Value
	}
	return state
}

// Restore puts the recorded values and language back and rebuilds the
// step-up matcher, which is derived from the values.
func (state ConfigState) Restore() {
	for i, variable := range ConfigVariables() {
		variable.Value = state.values[i]
	}
	config.InitStepUpMatcher()
	i18n.SetLanguage(state.language)
}

// ConfigChanges names what differs between two states, in a fixed order.
func ConfigChanges(before, after ConfigState) []string {
	var changes []string
	for i, variable := range ConfigVariables() {
		if before.values[i] != after.values[i] {
			changes = append(changes, "config "+variable.Name)
		}
	}
	if before.language != after.language {
		changes = append(changes, "i18n language")
	}
	sort.Strings(changes)
	return changes
}

// DeclaredConfigVariableNames reads the Name of every package-level
// "X = EnvVariable{...}" declaration in the non-test files of dir.
func DeclaredConfigVariableNames(dir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return nil, err
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
	return names, nil
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
