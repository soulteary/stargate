package i18n

import (
	"fmt"
	"sync"
)

// Language represents the supported languages
type Language string

const (
	// LangEN is English (default)
	LangEN Language = "en"
	// LangZH is Chinese
	LangZH Language = "zh"
)

var (
	currentLang Language = LangEN
	mu          sync.RWMutex
)

// SetLanguage sets the current language
func SetLanguage(lang Language) {
	mu.Lock()
	defer mu.Unlock()
	if lang == LangEN || lang == LangZH {
		currentLang = lang
	}
}

// GetLanguage returns the current language
func GetLanguage() Language {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// Translations map
var translations = map[Language]map[string]string{
	LangEN: {
		// Error messages
		"error.auth_required":           "Authentication required",
		"error.invalid_password":        "Invalid password",
		"error.session_store_failed":    "Internal server error: failed to access session store",
		"error.authenticate_failed":     "Internal server error: failed to authenticate session",
		"error.missing_session_id":      "Missing session ID",
		"error.config_invalid":          "Configuration error: invalid value for environment variable '%s': '%s'",
		"error.config_invalid_values":   "Configuration error: invalid value for environment variable '%s': '%s'.\n  Accepted values: %v\n  Please check your environment variable configuration and try again.",
		"error.config_required":         "Configuration error: environment variable '%s' is required but not set.\n  Please check your environment variable configuration and try again.",
		"error.config_required_not_set": "not set (required)",
		// Success messages
		"success.login": "Login successful",
	},
	LangZH: {
		// Error messages
		"error.auth_required":           "需要身份验证",
		"error.invalid_password":        "密码无效",
		"error.session_store_failed":    "内部服务器错误：无法访问会话存储",
		"error.authenticate_failed":     "内部服务器错误：无法验证会话",
		"error.missing_session_id":      "缺少会话 ID",
		"error.config_invalid":          "配置错误: 环境变量 '%s' 的值 '%s' 无效。\n  请检查环境变量配置并重试。",
		"error.config_invalid_values":   "配置错误: 环境变量 '%s' 的值 '%s' 无效。\n  可接受的值: %v\n  请检查环境变量配置并重试。",
		"error.config_required":         "配置错误: 环境变量 '%s' 未设置（必填项）。\n  请检查环境变量配置并重试。",
		"error.config_required_not_set": "未设置（必填项）",
		// Success messages
		"success.login": "登录成功",
	},
}

// T returns the translated string for the given key
// If the key is not found, it returns the key itself
func T(key string) string {
	mu.RLock()
	lang := currentLang
	mu.RUnlock()

	if langMap, ok := translations[lang]; ok {
		if translation, ok := langMap[key]; ok {
			return translation
		}
	}

	// Fallback to English if translation not found
	if langMap, ok := translations[LangEN]; ok {
		if translation, ok := langMap[key]; ok {
			return translation
		}
	}

	// Return key if no translation found
	return key
}

// Tf returns a formatted translated string
func Tf(key string, args ...interface{}) string {
	return fmt.Sprintf(T(key), args...)
}
