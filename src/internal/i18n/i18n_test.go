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

func TestSetLanguage_InvalidLanguage(t *testing.T) {
	// Set to a known state first
	SetLanguage(LangEN)
	originalLang := GetLanguage()

	// Try to set invalid language
	SetLanguage(Language("fr"))
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
			testza.AssertTrue(t, lang == LangEN || lang == LangZH, "language should be valid")
			_ = T("error.auth_required")
		}()
	}

	// Test concurrent writes
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				SetLanguage(LangEN)
			} else {
				SetLanguage(LangZH)
			}
		}(i)
	}

	wg.Wait()

	// Final state should be valid
	finalLang := GetLanguage()
	testza.AssertTrue(t, finalLang == LangEN || finalLang == LangZH, "final language should be valid")
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
			if i%2 == 0 {
				SetLanguage(LangEN)
			} else {
				SetLanguage(LangZH)
			}
		}(i)
	}

	wg.Wait()

	// Final state should be valid
	finalLang := GetLanguage()
	testza.AssertTrue(t, finalLang == LangEN || finalLang == LangZH, "final language should be valid")
}
