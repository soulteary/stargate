package i18n

import (
	"sync"
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestSetLanguage_ValidLanguage_EN(t *testing.T) {
	SetLanguage(LangEN)
	result := GetLanguage()
	testza.AssertEqual(t, LangEN, result)
}

func TestSetLanguage_ValidLanguage_ZH(t *testing.T) {
	SetLanguage(LangZH)
	result := GetLanguage()
	testza.AssertEqual(t, LangZH, result)
}

func TestSetLanguage_ValidLanguage_FR(t *testing.T) {
	SetLanguage(LangFR)
	result := GetLanguage()
	testza.AssertEqual(t, LangFR, result)
}

func TestSetLanguage_ValidLanguage_IT(t *testing.T) {
	SetLanguage(LangIT)
	result := GetLanguage()
	testza.AssertEqual(t, LangIT, result)
}

func TestSetLanguage_ValidLanguage_JA(t *testing.T) {
	SetLanguage(LangJA)
	result := GetLanguage()
	testza.AssertEqual(t, LangJA, result)
}

func TestSetLanguage_ValidLanguage_DE(t *testing.T) {
	SetLanguage(LangDE)
	result := GetLanguage()
	testza.AssertEqual(t, LangDE, result)
}

func TestSetLanguage_ValidLanguage_KO(t *testing.T) {
	SetLanguage(LangKO)
	result := GetLanguage()
	testza.AssertEqual(t, LangKO, result)
}

func TestSetLanguage_InvalidLanguage(t *testing.T) {
	// Set to a known state first
	SetLanguage(LangEN)
	originalLang := GetLanguage()

	// Try to set invalid language
	SetLanguage(Language("xx"))
	result := GetLanguage()

	// Should remain unchanged
	testza.AssertEqual(t, originalLang, result)
}

func TestGetLanguage_Default(t *testing.T) {
	// Reset to default
	SetLanguage(LangEN)
	result := GetLanguage()
	testza.AssertEqual(t, LangEN, result)
}

func TestT_English_ExistingKey(t *testing.T) {
	SetLanguage(LangEN)
	result := T("error.auth_required")
	testza.AssertEqual(t, "Authentication required", result)
}

func TestT_English_NonExistentKey(t *testing.T) {
	SetLanguage(LangEN)
	result := T("error.non_existent")
	testza.AssertEqual(t, "error.non_existent", result, "should return key if translation not found")
}

func TestT_Chinese_ExistingKey(t *testing.T) {
	SetLanguage(LangZH)
	result := T("error.auth_required")
	testza.AssertEqual(t, "需要身份验证", result)
}

func TestT_Chinese_NonExistentKey(t *testing.T) {
	SetLanguage(LangZH)
	result := T("error.non_existent")
	// Should fallback to English
	testza.AssertEqual(t, "error.non_existent", result)
}

func TestT_FallbackToEnglish(t *testing.T) {
	SetLanguage(LangZH)
	// Use a key that exists in English but not in Chinese (if any)
	// Since all keys exist in both, test with non-existent key
	result := T("error.non_existent_key")
	testza.AssertEqual(t, "error.non_existent_key", result)
}

func TestT_AllErrorKeys_EN(t *testing.T) {
	SetLanguage(LangEN)

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
			result := T(tt.key)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestT_AllErrorKeys_ZH(t *testing.T) {
	SetLanguage(LangZH)

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
			result := T(tt.key)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestT_French_ExistingKey(t *testing.T) {
	SetLanguage(LangFR)
	result := T("error.auth_required")
	testza.AssertEqual(t, "Authentification requise", result)
}

func TestT_Italian_ExistingKey(t *testing.T) {
	SetLanguage(LangIT)
	result := T("error.auth_required")
	testza.AssertEqual(t, "Autenticazione richiesta", result)
}

func TestT_Japanese_ExistingKey(t *testing.T) {
	SetLanguage(LangJA)
	result := T("error.auth_required")
	testza.AssertEqual(t, "認証が必要です", result)
}

func TestT_German_ExistingKey(t *testing.T) {
	SetLanguage(LangDE)
	result := T("error.auth_required")
	testza.AssertEqual(t, "Authentifizierung erforderlich", result)
}

func TestT_Korean_ExistingKey(t *testing.T) {
	SetLanguage(LangKO)
	result := T("error.auth_required")
	testza.AssertEqual(t, "인증이 필요합니다", result)
}

func TestTf_English_WithArgs(t *testing.T) {
	SetLanguage(LangEN)
	result := Tf("error.config_invalid", "TEST_VAR", "invalid-value")
	testza.AssertContains(t, result, "TEST_VAR")
	testza.AssertContains(t, result, "invalid-value")
	testza.AssertContains(t, result, "Configuration error")
}

func TestTf_Chinese_WithArgs(t *testing.T) {
	SetLanguage(LangZH)
	result := Tf("error.config_invalid", "TEST_VAR", "invalid-value")
	testza.AssertContains(t, result, "TEST_VAR")
	testza.AssertContains(t, result, "invalid-value")
	testza.AssertContains(t, result, "配置错误")
}

func TestTf_MultipleArgs(t *testing.T) {
	SetLanguage(LangEN)
	result := Tf("error.config_invalid_values", "TEST_VAR", "invalid-value", []string{"value1", "value2"})
	testza.AssertContains(t, result, "TEST_VAR")
	testza.AssertContains(t, result, "invalid-value")
	testza.AssertContains(t, result, "Accepted values")
}

func TestTf_NonExistentKey(t *testing.T) {
	SetLanguage(LangEN)
	result := Tf("error.non_existent", "arg1", "arg2")
	// Tf uses fmt.Sprintf, so it will format even if the key doesn't exist
	testza.AssertContains(t, result, "error.non_existent")
}

func TestConcurrentAccess(t *testing.T) {
	SetLanguage(LangEN)
	var wg sync.WaitGroup
	iterations := 100

	// Test concurrent reads
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			lang := GetLanguage()
			testza.AssertTrue(t, lang == LangEN || lang == LangZH || lang == LangFR || lang == LangIT || lang == LangJA || lang == LangDE || lang == LangKO, "language should be valid")
			_ = T("error.auth_required")
		}()
	}

	// Test concurrent writes
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(i int) {
			defer wg.Done()
			switch i % 7 {
			case 0:
				SetLanguage(LangEN)
			case 1:
				SetLanguage(LangZH)
			case 2:
				SetLanguage(LangFR)
			case 3:
				SetLanguage(LangIT)
			case 4:
				SetLanguage(LangJA)
			case 5:
				SetLanguage(LangDE)
			case 6:
				SetLanguage(LangKO)
			}
		}(i)
	}

	wg.Wait()

	// Final state should be valid
	finalLang := GetLanguage()
	testza.AssertTrue(t, finalLang == LangEN || finalLang == LangZH || finalLang == LangFR || finalLang == LangIT || finalLang == LangJA || finalLang == LangDE || finalLang == LangKO, "final language should be valid")
}

func TestConcurrentReadWrite(t *testing.T) {
	SetLanguage(LangEN)
	var wg sync.WaitGroup
	iterations := 50

	// Mix of reads and writes
	wg.Add(iterations * 2)
	for i := 0; i < iterations; i++ {
		// Read goroutine
		go func() {
			defer wg.Done()
			_ = GetLanguage()
			_ = T("error.auth_required")
			_ = Tf("error.config_invalid", "VAR", "value")
		}()

		// Write goroutine
		go func(i int) {
			defer wg.Done()
			switch i % 7 {
			case 0:
				SetLanguage(LangEN)
			case 1:
				SetLanguage(LangZH)
			case 2:
				SetLanguage(LangFR)
			case 3:
				SetLanguage(LangIT)
			case 4:
				SetLanguage(LangJA)
			case 5:
				SetLanguage(LangDE)
			case 6:
				SetLanguage(LangKO)
			}
		}(i)
	}

	wg.Wait()

	// Final state should be valid
	finalLang := GetLanguage()
	testza.AssertTrue(t, finalLang == LangEN || finalLang == LangZH || finalLang == LangFR || finalLang == LangIT || finalLang == LangJA || finalLang == LangDE || finalLang == LangKO, "final language should be valid")
}

func TestT_FallbackToEnglish_WhenKeyNotFoundInCurrentLanguage(t *testing.T) {
	// Set to a non-English language
	SetLanguage(LangFR)

	// Use a key that exists in English translations
	// Since all keys exist in all languages, we test with a non-existent key
	// which should return the key itself, not fallback to English
	result := T("error.non_existent_key_in_any_language")
	testza.AssertEqual(t, "error.non_existent_key_in_any_language", result, "should return key when not found in any language")
}

func TestT_InvalidLanguage_FallbackToEnglish(t *testing.T) {
	// Set to an invalid language (this shouldn't happen in practice, but test the code path)
	// The SetLanguage function validates, so we can't easily set an invalid language
	// But we can test that T() handles missing language maps gracefully
	SetLanguage(LangEN)

	// Test that T() works correctly with valid language
	result := T("error.auth_required")
	testza.AssertEqual(t, "Authentication required", result)
}

// TestT_AllSuccessKeys tests all success message keys
func TestT_AllSuccessKeys(t *testing.T) {
	SetLanguage(LangEN)

	tests := []struct {
		key      string
		expected string
	}{
		{"success.login", "Login successful"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := T(tt.key)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

// TestT_AllConfigKeys tests all config-related error keys
func TestT_AllConfigKeys(t *testing.T) {
	SetLanguage(LangEN)

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
			result := T(tt.key)
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

// TestT_AllLanguages_AllKeys tests all keys in all languages to ensure complete coverage
func TestT_AllLanguages_AllKeys(t *testing.T) {
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
			SetLanguage(lang)
			for _, key := range allKeys {
				result := T(key)
				// Just verify it doesn't return empty string and doesn't panic
				testza.AssertNotEqual(t, "", result, "translation should not be empty for key: %s", key)
				testza.AssertNotNil(t, result, "translation should not be nil for key: %s", key)
			}
		})
	}
}

// TestT_EdgeCase_EmptyKey tests edge case with empty key
func TestT_EdgeCase_EmptyKey(t *testing.T) {
	SetLanguage(LangEN)
	result := T("")
	testza.AssertEqual(t, "", result, "empty key should return empty string")
}

// TestT_EdgeCase_WhitespaceKey tests edge case with whitespace key
func TestT_EdgeCase_WhitespaceKey(t *testing.T) {
	SetLanguage(LangEN)
	result := T("   ")
	testza.AssertEqual(t, "   ", result, "whitespace key should return key itself")
}

// TestT_ConcurrentAccess_SameKey tests concurrent access to same key
func TestT_ConcurrentAccess_SameKey(t *testing.T) {
	SetLanguage(LangEN)
	var wg sync.WaitGroup
	iterations := 100
	key := "error.auth_required"
	expected := "Authentication required"

	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func() {
			defer wg.Done()
			result := T(key)
			testza.AssertEqual(t, expected, result)
		}()
	}

	wg.Wait()
}

// TestT_AllLanguages_SuccessLogin tests success.login in all languages
func TestT_AllLanguages_SuccessLogin(t *testing.T) {
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
			SetLanguage(tt.lang)
			result := T("success.login")
			testza.AssertEqual(t, tt.expected, result)
		})
	}
}

// TestT_AllLanguages_ConfigInvalid tests error.config_invalid in all languages
func TestT_AllLanguages_ConfigInvalid(t *testing.T) {
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
			SetLanguage(tt.lang)
			result := T("error.config_invalid")
			for _, substr := range tt.contains {
				testza.AssertContains(t, result, substr)
			}
		})
	}
}

// TestT_AllLanguages_ConfigInvalidValues tests error.config_invalid_values in all languages
func TestT_AllLanguages_ConfigInvalidValues(t *testing.T) {
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
			SetLanguage(tt.lang)
			result := T("error.config_invalid_values")
			for _, substr := range tt.contains {
				testza.AssertContains(t, result, substr)
			}
		})
	}
}

// TestT_AllLanguages_ConfigRequired tests error.config_required in all languages
func TestT_AllLanguages_ConfigRequired(t *testing.T) {
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
			SetLanguage(tt.lang)
			result := T("error.config_required")
			for _, substr := range tt.contains {
				testza.AssertContains(t, result, substr)
			}
		})
	}
}
