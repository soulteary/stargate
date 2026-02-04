package config

import (
	"strings"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	logger "github.com/soulteary/logger-kit"
)

// testLogger creates a logger instance for testing
func testLogger() *logger.Logger {
	return logger.New(logger.Config{
		Level:       logger.DebugLevel,
		Format:      logger.FormatJSON,
		ServiceName: "config-test",
	})
}

func TestEnvVariable_String(t *testing.T) {
	v := EnvVariable{
		Value: "test-value",
	}
	testza.AssertEqual(t, "test-value", v.String())
}

func TestEnvVariable_ToDuration(t *testing.T) {
	tests := []struct {
		value    string
		expected time.Duration
	}{
		{"", 0},
		{"5m", 5 * time.Minute},
		{"1h", time.Hour},
		{"30s", 30 * time.Second},
		{"1m30s", 90 * time.Second},
		{"invalid", 0},
		{"x", 0},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			v := EnvVariable{Value: tt.value}
			got := v.ToDuration()
			testza.AssertEqual(t, tt.expected, got)
		})
	}
}

func TestEnvVariable_ToBool(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"", false},
		{"other", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			v := EnvVariable{Value: tt.value}
			testza.AssertEqual(t, tt.expected, v.ToBool())
		})
	}
}

func TestEnvVariable_Validate_Required_NotSet(t *testing.T) {
	v := EnvVariable{
		Name:      "REQUIRED_VAR",
		Required:  true,
		Validator: ValidateNotEmptyString,
	}

	err := v.Validate()
	testza.AssertNotNil(t, err)
	testza.AssertEqual(t, "REQUIRED_VAR", err.(*ValidationError).KeyName)
}

func TestEnvVariable_Validate_Required_Set(t *testing.T) {
	t.Setenv("TEST_VAR", "test-value")
	v := EnvVariable{
		Name:      "TEST_VAR",
		Required:  true,
		Validator: ValidateNotEmptyString,
	}

	err := v.Validate()
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "test-value", v.Value)
}

func TestEnvVariable_Validate_Optional_NotSet_WithDefault(t *testing.T) {
	v := EnvVariable{
		Name:         "OPTIONAL_VAR",
		Required:     false,
		DefaultValue: "default-value",
		Validator:    ValidateAny,
	}

	err := v.Validate()
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "default-value", v.Value)
}

func TestEnvVariable_Validate_Optional_Set(t *testing.T) {
	t.Setenv("OPTIONAL_VAR", "custom-value")
	v := EnvVariable{
		Name:         "OPTIONAL_VAR",
		Required:     false,
		DefaultValue: "default-value",
		Validator:    ValidateAny,
	}

	err := v.Validate()
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "custom-value", v.Value)
}

func TestValidateNotEmptyString(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"test", true},
		{"", false},
		{" ", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			v := EnvVariable{Value: tt.value}
			result := ValidateNotEmptyString(v)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestValidateAny(t *testing.T) {
	v := EnvVariable{Value: "any-value"}
	result := ValidateAny(v)
	testza.AssertTrue(t, result)
}

func TestValidateStrictPossibleValues(t *testing.T) {
	tests := []struct {
		value        string
		possibleVals []string
		expected     bool
	}{
		{"value1", []string{"value1", "value2"}, true},
		{"value2", []string{"value1", "value2"}, true},
		{"value3", []string{"value1", "value2"}, false},
		{"VALUE1", []string{"value1", "value2"}, false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			v := EnvVariable{
				Value:          tt.value,
				PossibleValues: tt.possibleVals,
			}
			result := ValidateStrictPossibleValues(v)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestValidateCaseInsensitivePossibleValues(t *testing.T) {
	tests := []struct {
		value        string
		possibleVals []string
		expected     bool
	}{
		{"true", []string{"true", "false"}, true},
		{"True", []string{"true", "false"}, true},
		{"TRUE", []string{"true", "false"}, true},
		{"false", []string{"true", "false"}, true},
		{"other", []string{"true", "false"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			v := EnvVariable{
				Value:          tt.value,
				PossibleValues: tt.possibleVals,
			}
			result := ValidateCaseInsensitivePossibleValues(v)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestValidatePasswords_Plaintext_Valid(t *testing.T) {
	tests := []struct {
		name      string
		passwords string
		expected  bool
	}{
		{"single password", "plaintext:pass1", true},
		{"multiple passwords", "plaintext:pass1|pass2|pass3", true},
		{"with spaces", "plaintext:pass1 | pass2", true},
		{"bcrypt", "bcrypt:$2a$10$k8fBIpJInrE70BzYy5rO/OUSt1w2.IX0bWhiMdb2mJEhjheVHDhvK", true},
		{"md5", "md5:65a8e27d8879283831b664bd8b7f0ad4", true},
		{"sha512", "sha512:374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := EnvVariable{Value: tt.passwords}
			result := ValidatePasswords(v)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestValidatePasswords_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		passwords string
		expected  bool
	}{
		{"unsupported algorithm", "unknown:pass1", false},
		{"missing algorithm", "pass1", false},
		{"empty password", "plaintext:", false},
		{"empty password in list", "plaintext:pass1||pass3", false},
		{"missing colon", "plaintextpass1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := EnvVariable{Value: tt.passwords}
			result := ValidatePasswords(v)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestInitialize_Success(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "false")
	t.Setenv("LOGIN_PAGE_TITLE", "Test Title")
	t.Setenv("LOGIN_PAGE_FOOTER_TEXT", "Test Footer")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
}

func TestInitialize_MissingRequired(t *testing.T) {
	t.Setenv("AUTH_HOST", "")
	t.Setenv("PASSWORDS", "")

	err := Initialize(testLogger())
	testza.AssertNotNil(t, err)
}

func TestValidationError_Error(t *testing.T) {
	err := NewValidationError("TEST_VAR", "invalid-value", []string{"value1", "value2"})

	errorStr := err.Error()
	testza.AssertContains(t, errorStr, "TEST_VAR")
	testza.AssertContains(t, errorStr, "invalid-value")
}

func TestValidationError_String(t *testing.T) {
	err := NewValidationError("TEST_VAR", "invalid-value", []string{"value1", "value2"})

	errorStr := err.String()
	testza.AssertContains(t, errorStr, "TEST_VAR")
	testza.AssertContains(t, errorStr, "invalid-value")
}

func TestInitialize_Language_EN(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("LANGUAGE", "en")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "en", Language.Value)
}

func TestInitialize_Language_ZH(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("LANGUAGE", "zh")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "zh", Language.Value)
}

func TestInitialize_Language_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		lang     string
		expected string
	}{
		{"uppercase EN", "EN", "en"},
		{"uppercase ZH", "ZH", "zh"},
		{"mixed case En", "En", "en"},
		{"mixed case Zh", "Zh", "zh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AUTH_HOST", "auth.example.com")
			t.Setenv("PASSWORDS", "plaintext:test123")
			t.Setenv("LANGUAGE", tt.lang)

			err := Initialize(testLogger())
			testza.AssertNoError(t, err)
			testza.AssertEqual(t, tt.expected, strings.ToLower(Language.Value))
		})
	}
}

func TestInitialize_Language_Default(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	// Don't set LANGUAGE to test default

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	// Default should be "en"
	testza.AssertEqual(t, "en", Language.Value)
}

func TestInitialize_AllConfigVariables(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("DEBUG", "true")
	t.Setenv("LOGIN_PAGE_TITLE", "Custom Title")
	t.Setenv("LOGIN_PAGE_FOOTER_TEXT", "Custom Footer")
	t.Setenv("USER_HEADER_NAME", "X-Custom-User")
	t.Setenv("COOKIE_DOMAIN", ".example.com")
	t.Setenv("LANGUAGE", "zh")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)

	testza.AssertEqual(t, "auth.example.com", AuthHost.Value)
	testza.AssertEqual(t, "plaintext:test123", Passwords.Value)
	testza.AssertEqual(t, "true", Debug.Value)
	testza.AssertEqual(t, "Custom Title", LoginPageTitle.Value)
	testza.AssertEqual(t, "Custom Footer", LoginPageFooterText.Value)
	testza.AssertEqual(t, "X-Custom-User", UserHeaderName.Value)
	testza.AssertEqual(t, ".example.com", CookieDomain.Value)
	testza.AssertEqual(t, "zh", Language.Value)
}

func TestInitialize_WithDefaults(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	// Don't set optional variables to test defaults

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)

	testza.AssertEqual(t, "false", Debug.Value)
	testza.AssertEqual(t, "Stargate - Login", LoginPageTitle.Value)
	testza.AssertEqual(t, "Copyright © 2024 - Stargate", LoginPageFooterText.Value)
	testza.AssertEqual(t, "X-Forwarded-User", UserHeaderName.Value)
	testza.AssertEqual(t, "", CookieDomain.Value)
	testza.AssertEqual(t, "en", Language.Value)
}

func TestInitialize_Language_FR(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("LANGUAGE", "fr")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "fr", Language.Value)
}

func TestInitialize_Language_IT(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("LANGUAGE", "it")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "it", Language.Value)
}

func TestInitialize_Language_JA(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("LANGUAGE", "ja")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "ja", Language.Value)
}

func TestInitialize_Language_DE(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("LANGUAGE", "de")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "de", Language.Value)
}

func TestInitialize_Language_KO(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("LANGUAGE", "ko")

	err := Initialize(testLogger())
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "ko", Language.Value)
}

func TestValidationError_String_WithWildcard(t *testing.T) {
	err := NewValidationError("TEST_VAR", "invalid-value", []string{"*"})

	errorStr := err.String()
	testza.AssertContains(t, errorStr, "TEST_VAR")
	testza.AssertContains(t, errorStr, "invalid-value")
}
