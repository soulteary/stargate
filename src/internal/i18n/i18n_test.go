package i18n

import (
	"sync"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestTStatic_English_ExistingKey(t *testing.T) {
	result := TStatic("error.auth_required")
	testza.AssertEqual(t, "Authentication required", result)
}

func TestTStatic_English_NonExistentKey(t *testing.T) {
	result := TStatic("error.non_existent")
	testza.AssertEqual(t, "error.non_existent", result, "should return key if translation not found")
}

func TestTWithLang_English_ExistingKey(t *testing.T) {
	result := TWithLang(LangEN, "error.auth_required")
	testza.AssertEqual(t, "Authentication required", result)
}

func TestTWithLang_Chinese_ExistingKey(t *testing.T) {
	result := TWithLang(LangZH, "error.auth_required")
	testza.AssertEqual(t, "需要身份验证", result)
}

func TestTWithLang_Chinese_NonExistentKey(t *testing.T) {
	result := TWithLang(LangZH, "error.non_existent")
	// Should fallback to English, which also doesn't have it, so return key
	testza.AssertEqual(t, "error.non_existent", result)
}

func TestTWithLang_FallbackToEnglish(t *testing.T) {
	// Use a key that doesn't exist in any language
	result := TWithLang(LangZH, "error.non_existent_key")
	testza.AssertEqual(t, "error.non_existent_key", result)
}

func TestTWithLang_AllErrorKeys_EN(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"error.auth_required", "Authentication required"},
		{"error.invalid_password", "Invalid password"},
		{"error.session_store_failed", "Internal server error: failed to access session store"},
		{"error.authenticate_failed", "Internal server error: failed to authenticate session"},
		{"error.missing_session_id", "Missing session ID"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := TWithLang(LangEN, tt.key)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestTWithLang_AllErrorKeys_ZH(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"error.auth_required", "需要身份验证"},
		{"error.invalid_password", "密码无效"},
		{"error.session_store_failed", "内部服务器错误：无法访问会话存储"},
		{"error.authenticate_failed", "内部服务器错误：无法验证会话"},
		{"error.missing_session_id", "缺少会话 ID"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := TWithLang(LangZH, tt.key)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestTWithLang_French_ExistingKey(t *testing.T) {
	result := TWithLang(LangFR, "error.auth_required")
	testza.AssertEqual(t, "Authentification requise", result)
}

func TestTWithLang_Italian_ExistingKey(t *testing.T) {
	result := TWithLang(LangIT, "error.auth_required")
	testza.AssertEqual(t, "Autenticazione richiesta", result)
}

func TestTWithLang_Japanese_ExistingKey(t *testing.T) {
	result := TWithLang(LangJA, "error.auth_required")
	testza.AssertEqual(t, "認証が必要です", result)
}

func TestTWithLang_German_ExistingKey(t *testing.T) {
	result := TWithLang(LangDE, "error.auth_required")
	testza.AssertEqual(t, "Authentifizierung erforderlich", result)
}

func TestTWithLang_Korean_ExistingKey(t *testing.T) {
	result := TWithLang(LangKO, "error.auth_required")
	testza.AssertEqual(t, "인증이 필요합니다", result)
}

func TestTfWithLang_English_WithArgs(t *testing.T) {
	result := TfWithLang(LangEN, "error.config_invalid", "TEST_VAR", "invalid-value")
	testza.AssertContains(t, result, "TEST_VAR")
	testza.AssertContains(t, result, "invalid-value")
	testza.AssertContains(t, result, "Configuration error")
}

func TestTfWithLang_Chinese_WithArgs(t *testing.T) {
	result := TfWithLang(LangZH, "error.config_invalid", "TEST_VAR", "invalid-value")
	testza.AssertContains(t, result, "TEST_VAR")
	testza.AssertContains(t, result, "invalid-value")
	testza.AssertContains(t, result, "配置错误")
}

func TestTfWithLang_MultipleArgs(t *testing.T) {
	result := TfWithLang(LangEN, "error.config_invalid_values", "TEST_VAR", "invalid-value", []string{"value1", "value2"})
	testza.AssertContains(t, result, "TEST_VAR")
	testza.AssertContains(t, result, "invalid-value")
	testza.AssertContains(t, result, "Accepted values")
}

func TestTfStatic_NonExistentKey(t *testing.T) {
	result := TfStatic("error.non_existent", "arg1", "arg2")
	// TfStatic uses fmt.Sprintf, so it will format even if the key doesn't exist
	testza.AssertContains(t, result, "error.non_existent")
}

func TestConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup
	iterations := 100

	// Test concurrent reads
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			_ = TWithLang(LangEN, "error.auth_required")
			_ = TWithLang(LangZH, "error.auth_required")
		}()
	}

	wg.Wait()
}

func TestConcurrentReadWrite(t *testing.T) {
	var wg sync.WaitGroup
	iterations := 50

	// Mix of reads with different languages
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(i int) {
			defer wg.Done()
			switch i % 7 {
			case 0:
				_ = TWithLang(LangEN, "error.auth_required")
			case 1:
				_ = TWithLang(LangZH, "error.auth_required")
			case 2:
				_ = TWithLang(LangFR, "error.auth_required")
			case 3:
				_ = TWithLang(LangIT, "error.auth_required")
			case 4:
				_ = TWithLang(LangJA, "error.auth_required")
			case 5:
				_ = TWithLang(LangDE, "error.auth_required")
			case 6:
				_ = TWithLang(LangKO, "error.auth_required")
			}
			_ = TfWithLang(LangEN, "error.config_invalid", "VAR", "value")
		}(i)
	}

	wg.Wait()
}

func TestTWithLang_FallbackToEnglish_WhenKeyNotFoundInCurrentLanguage(t *testing.T) {
	// Use a key that doesn't exist in any language
	result := TWithLang(LangFR, "error.non_existent_key_in_any_language")
	testza.AssertEqual(t, "error.non_existent_key_in_any_language", result, "should return key when not found in any language")
}

// TestTWithLang_AllSuccessKeys tests all success message keys
func TestTWithLang_AllSuccessKeys(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"success.login", "Login successful"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := TWithLang(LangEN, tt.key)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

// TestTWithLang_AllConfigKeys tests all config-related error keys
func TestTWithLang_AllConfigKeys(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"error.config_invalid", "Configuration error: invalid value for environment variable '%s': '%s'"},
		{"error.config_invalid_values", "Configuration error: invalid value for environment variable '%s': '%s'.\n  Accepted values: %v\n  Please check your environment variable configuration and try again."},
		{"error.config_required", "Configuration error: environment variable '%s' is required but not set.\n  Please check your environment variable configuration and try again."},
		{"error.config_required_not_set", "not set (required)"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := TWithLang(LangEN, tt.key)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

// TestTWithLang_AllLanguages_AllKeys tests all keys in all languages to ensure complete coverage
func TestTWithLang_AllLanguages_AllKeys(t *testing.T) {
	allKeys := []string{
		"error.auth_required",
		"error.invalid_password",
		"error.session_store_failed",
		"error.authenticate_failed",
		"error.missing_session_id",
		"error.config_invalid",
		"error.config_invalid_values",
		"error.config_required",
		"error.config_required_not_set",
		"success.login",
	}

	// Test all keys for each language
	for _, lang := range []Language{LangEN, LangZH, LangFR, LangIT, LangJA, LangDE, LangKO} {
		t.Run("AllKeys_"+string(lang), func(t *testing.T) {
			for _, key := range allKeys {
				result := TWithLang(lang, key)
				// Just verify it doesn't return empty string and doesn't panic
				testza.AssertNotEqual(t, "", result, "translation should not be empty for key: %s", key)
				testza.AssertNotNil(t, result, "translation should not be nil for key: %s", key)
			}
		})
	}
}

// TestTStatic_EdgeCase_EmptyKey tests edge case with empty key
func TestTStatic_EdgeCase_EmptyKey(t *testing.T) {
	result := TStatic("")
	testza.AssertEqual(t, "", result, "empty key should return empty string")
}

// TestTStatic_EdgeCase_WhitespaceKey tests edge case with whitespace key
func TestTStatic_EdgeCase_WhitespaceKey(t *testing.T) {
	result := TStatic("   ")
	testza.AssertEqual(t, "   ", result, "whitespace key should return key itself")
}

// TestTWithLang_ConcurrentAccess_SameKey tests concurrent access to same key
func TestTWithLang_ConcurrentAccess_SameKey(t *testing.T) {
	var wg sync.WaitGroup
	iterations := 100
	key := "error.auth_required"
	expected := "Authentication required"

	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			result := TWithLang(LangEN, key)
			testza.AssertEqual(t, expected, result)
		}()
	}

	wg.Wait()
}

// TestTWithLang_AllLanguages_SuccessLogin tests success.login in all languages
func TestTWithLang_AllLanguages_SuccessLogin(t *testing.T) {
	tests := []struct {
		lang     Language
		expected string
	}{
		{LangEN, "Login successful"},
		{LangZH, "登录成功"},
		{LangFR, "Connexion réussie"},
		{LangIT, "Accesso riuscito"},
		{LangJA, "ログイン成功"},
		{LangDE, "Anmeldung erfolgreich"},
		{LangKO, "로그인 성공"},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			result := TWithLang(tt.lang, "success.login")
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

// TestTWithLang_AllLanguages_ConfigInvalid tests error.config_invalid in all languages
func TestTWithLang_AllLanguages_ConfigInvalid(t *testing.T) {
	tests := []struct {
		lang     Language
		contains []string
	}{
		{LangEN, []string{"Configuration error", "invalid value"}},
		{LangZH, []string{"配置错误"}},
		{LangFR, []string{"Erreur de configuration"}},
		{LangIT, []string{"Errore di configurazione"}},
		{LangJA, []string{"設定エラー"}},
		{LangDE, []string{"Konfigurationsfehler"}},
		{LangKO, []string{"구성 오류"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			result := TWithLang(tt.lang, "error.config_invalid")
			for _, substr := range tt.contains {
				testza.AssertContains(t, result, substr)
			}
		})
	}
}

// TestTWithLang_AllLanguages_ConfigInvalidValues tests error.config_invalid_values in all languages
func TestTWithLang_AllLanguages_ConfigInvalidValues(t *testing.T) {
	tests := []struct {
		lang     Language
		contains []string
	}{
		{LangEN, []string{"Configuration error", "Accepted values"}},
		{LangZH, []string{"配置错误", "可接受的值"}},
		{LangFR, []string{"Erreur de configuration", "Valeurs acceptées"}},
		{LangIT, []string{"Errore di configurazione", "Valori accettati"}},
		{LangJA, []string{"設定エラー", "受け入れられる値"}},
		{LangDE, []string{"Konfigurationsfehler", "Akzeptierte Werte"}},
		{LangKO, []string{"구성 오류", "허용되는 값"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			result := TWithLang(tt.lang, "error.config_invalid_values")
			for _, substr := range tt.contains {
				testza.AssertContains(t, result, substr)
			}
		})
	}
}

// TestTWithLang_AllLanguages_ConfigRequired tests error.config_required in all languages
func TestTWithLang_AllLanguages_ConfigRequired(t *testing.T) {
	tests := []struct {
		lang     Language
		contains []string
	}{
		{LangEN, []string{"Configuration error", "required but not set"}},
		{LangZH, []string{"配置错误", "未设置"}},
		{LangFR, []string{"Erreur de configuration", "requise mais n'est pas définie"}},
		{LangIT, []string{"Errore di configurazione", "richiesta ma non è impostata"}},
		{LangJA, []string{"設定エラー", "必須ですが設定されていません"}},
		{LangDE, []string{"Konfigurationsfehler", "erforderlich, wurde aber nicht gesetzt"}},
		{LangKO, []string{"구성 오류", "필요하지만 설정되지 않았습니다"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			result := TWithLang(tt.lang, "error.config_required")
			for _, substr := range tt.contains {
				testza.AssertContains(t, result, substr)
			}
		})
	}
}

// TestGetBundle tests that GetBundle returns a non-nil bundle
func TestGetBundle(t *testing.T) {
	bundle := GetBundle()
	testza.AssertNotNil(t, bundle, "bundle should not be nil")
}

// TestLanguageConstants tests that language constants are correctly defined
func TestLanguageConstants(t *testing.T) {
	testza.AssertEqual(t, Language("en"), LangEN)
	testza.AssertEqual(t, Language("zh"), LangZH)
	testza.AssertEqual(t, Language("fr"), LangFR)
	testza.AssertEqual(t, Language("it"), LangIT)
	testza.AssertEqual(t, Language("ja"), LangJA)
	testza.AssertEqual(t, Language("de"), LangDE)
	testza.AssertEqual(t, Language("ko"), LangKO)
}
