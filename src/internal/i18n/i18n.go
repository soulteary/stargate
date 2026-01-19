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
	// LangFR is French
	LangFR Language = "fr"
	// LangIT is Italian
	LangIT Language = "it"
	// LangJA is Japanese
	LangJA Language = "ja"
	// LangDE is German
	LangDE Language = "de"
	// LangKO is Korean
	LangKO Language = "ko"
)

var (
	currentLang Language = LangEN
	mu          sync.RWMutex
)

// SetLanguage sets the current language
func SetLanguage(lang Language) {
	mu.Lock()
	defer mu.Unlock()
	if lang == LangEN || lang == LangZH || lang == LangFR || lang == LangIT || lang == LangJA || lang == LangDE || lang == LangKO {
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
		"error.user_not_in_list":        "User not found in allow list",
		"error.authentication_failed":    "Authentication failed",
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
		"error.user_not_in_list":        "用户不在允许列表中",
		"error.authentication_failed":    "认证失败",
		// Success messages
		"success.login": "登录成功",
	},
	LangFR: {
		// Error messages
		"error.auth_required":           "Authentification requise",
		"error.invalid_password":        "Mot de passe invalide",
		"error.session_store_failed":    "Erreur interne du serveur : échec d'accès au stockage de session",
		"error.authenticate_failed":     "Erreur interne du serveur : échec de l'authentification de la session",
		"error.missing_session_id":      "ID de session manquant",
		"error.config_invalid":          "Erreur de configuration : valeur invalide pour la variable d'environnement '%s' : '%s'",
		"error.config_invalid_values":   "Erreur de configuration : valeur invalide pour la variable d'environnement '%s' : '%s'.\n  Valeurs acceptées : %v\n  Veuillez vérifier votre configuration de variable d'environnement et réessayer.",
		"error.config_required":         "Erreur de configuration : la variable d'environnement '%s' est requise mais n'est pas définie.\n  Veuillez vérifier votre configuration de variable d'environnement et réessayer.",
		"error.config_required_not_set": "non définie (requis)",
		"error.user_not_in_list":        "Utilisateur non trouvé dans la liste d'autorisation",
		"error.authentication_failed":    "Échec de l'authentification",
		// Success messages
		"success.login": "Connexion réussie",
	},
	LangIT: {
		// Error messages
		"error.auth_required":           "Autenticazione richiesta",
		"error.invalid_password":        "Password non valida",
		"error.session_store_failed":    "Errore interno del server: impossibile accedere al deposito delle sessioni",
		"error.authenticate_failed":     "Errore interno del server: impossibile autenticare la sessione",
		"error.missing_session_id":      "ID sessione mancante",
		"error.config_invalid":          "Errore di configurazione: valore non valido per la variabile d'ambiente '%s': '%s'",
		"error.config_invalid_values":   "Errore di configurazione: valore non valido per la variabile d'ambiente '%s': '%s'.\n  Valori accettati: %v\n  Si prega di controllare la configurazione della variabile d'ambiente e riprovare.",
		"error.config_required":         "Errore di configurazione: la variabile d'ambiente '%s' è richiesta ma non è impostata.\n  Si prega di controllare la configurazione della variabile d'ambiente e riprovare.",
		"error.config_required_not_set": "non impostata (richiesto)",
		"error.user_not_in_list":        "Utente non trovato nell'elenco consentiti",
		"error.authentication_failed":    "Autenticazione fallita",
		// Success messages
		"success.login": "Accesso riuscito",
	},
	LangJA: {
		// Error messages
		"error.auth_required":           "認証が必要です",
		"error.invalid_password":        "パスワードが無効です",
		"error.session_store_failed":    "内部サーバーエラー：セッションストアへのアクセスに失敗しました",
		"error.authenticate_failed":     "内部サーバーエラー：セッションの認証に失敗しました",
		"error.missing_session_id":      "セッションIDが不足しています",
		"error.config_invalid":          "設定エラー：環境変数 '%s' の値 '%s' が無効です",
		"error.config_invalid_values":   "設定エラー：環境変数 '%s' の値 '%s' が無効です。\n  受け入れられる値: %v\n  環境変数の設定を確認して再試行してください。",
		"error.config_required":         "設定エラー：環境変数 '%s' は必須ですが設定されていません。\n  環境変数の設定を確認して再試行してください。",
		"error.config_required_not_set": "設定されていません（必須）",
		"error.user_not_in_list":        "許可リストにユーザーが見つかりません",
		"error.authentication_failed":    "認証に失敗しました",
		// Success messages
		"success.login": "ログイン成功",
	},
	LangDE: {
		// Error messages
		"error.auth_required":           "Authentifizierung erforderlich",
		"error.invalid_password":        "Ungültiges Passwort",
		"error.session_store_failed":    "Interner Serverfehler: Fehler beim Zugriff auf den Sitzungsspeicher",
		"error.authenticate_failed":     "Interner Serverfehler: Fehler bei der Authentifizierung der Sitzung",
		"error.missing_session_id":      "Sitzungs-ID fehlt",
		"error.config_invalid":          "Konfigurationsfehler: Ungültiger Wert für Umgebungsvariable '%s': '%s'",
		"error.config_invalid_values":   "Konfigurationsfehler: Ungültiger Wert für Umgebungsvariable '%s': '%s'.\n  Akzeptierte Werte: %v\n  Bitte überprüfen Sie Ihre Umgebungsvariablen-Konfiguration und versuchen Sie es erneut.",
		"error.config_required":         "Konfigurationsfehler: Umgebungsvariable '%s' ist erforderlich, wurde aber nicht gesetzt.\n  Bitte überprüfen Sie Ihre Umgebungsvariablen-Konfiguration und versuchen Sie es erneut.",
		"error.config_required_not_set": "nicht gesetzt (erforderlich)",
		"error.user_not_in_list":        "Benutzer nicht in der Zulassungsliste gefunden",
		"error.authentication_failed":    "Authentifizierung fehlgeschlagen",
		// Success messages
		"success.login": "Anmeldung erfolgreich",
	},
	LangKO: {
		// Error messages
		"error.auth_required":           "인증이 필요합니다",
		"error.invalid_password":        "잘못된 비밀번호",
		"error.session_store_failed":    "내부 서버 오류: 세션 저장소에 액세스하지 못했습니다",
		"error.authenticate_failed":     "내부 서버 오류: 세션 인증에 실패했습니다",
		"error.missing_session_id":      "세션 ID가 없습니다",
		"error.config_invalid":          "구성 오류: 환경 변수 '%s'의 값 '%s'이(가) 유효하지 않습니다",
		"error.config_invalid_values":   "구성 오류: 환경 변수 '%s'의 값 '%s'이(가) 유효하지 않습니다.\n  허용되는 값: %v\n  환경 변수 구성을 확인하고 다시 시도하세요.",
		"error.config_required":         "구성 오류: 환경 변수 '%s'이(가) 필요하지만 설정되지 않았습니다.\n  환경 변수 구성을 확인하고 다시 시도하세요.",
		"error.config_required_not_set": "설정되지 않음 (필수)",
		"error.user_not_in_list":        "허용 목록에 사용자를 찾을 수 없습니다",
		"error.authentication_failed":    "인증 실패",
		// Success messages
		"success.login": "로그인 성공",
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
