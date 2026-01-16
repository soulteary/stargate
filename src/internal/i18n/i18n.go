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
		"error.auth_required":            "Authentication required",
		"error.invalid_password":         "Invalid password",
		"error.invalid_callback":         "Invalid callback host",
		"error.session_store_failed":     "Internal server error: failed to access session store",
		"error.authenticate_failed":      "Internal server error: failed to authenticate session",
		"error.missing_session_id":       "Missing session ID",
		"error.config_invalid":           "Configuration error: invalid value for environment variable '%s': '%s'",
		"error.config_invalid_values":    "Configuration error: invalid value for environment variable '%s': '%s'.\n  Accepted values: %v\n  Please check your environment variable configuration and try again.",
		"error.config_required":          "Configuration error: environment variable '%s' is required but not set.\n  Please check your environment variable configuration and try again.",
		"error.config_required_not_set":  "not set (required)",
		"error.oidc_not_configured":      "OIDC is not configured",
		"error.state_generation_failed":  "Failed to generate security token",
		"error.oidc_missing_code":        "Authorization code is missing",
		"error.oidc_invalid_state":       "Invalid security token",
		"error.oidc_token_exchange_failed": "Failed to exchange authorization code",
		"error.oidc_missing_id_token":    "ID token is missing from response",
		"error.oidc_token_verification_failed": "Failed to verify authentication token",
		"error.password_login_disabled":  "Password login is disabled when OIDC is enabled",
		"error.oidc_error":               "Authentication Error",
		// Success messages
		"success.login":      "Login successful",
		// Login messages
		"login.oidc_button":  "Login with %s",
	},
	LangZH: {
		// Error messages
		"error.auth_required":            "需要身份验证",
		"error.invalid_password":         "密码无效",
		"error.invalid_callback":         "回调地址无效",
		"error.session_store_failed":     "内部服务器错误：无法访问会话存储",
		"error.authenticate_failed":      "内部服务器错误：无法验证会话",
		"error.missing_session_id":       "缺少会话 ID",
		"error.config_invalid":           "配置错误: 环境变量 '%s' 的值 '%s' 无效。\n  请检查环境变量配置并重试。",
		"error.config_invalid_values":    "配置错误: 环境变量 '%s' 的值 '%s' 无效。\n  可接受的值: %v\n  请检查环境变量配置并重试。",
		"error.config_required":          "配置错误: 环境变量 '%s' 未设置（必填项）。\n  请检查环境变量配置并重试。",
		"error.config_required_not_set":  "未设置（必填项）",
		"error.oidc_not_configured":      "OIDC 未配置",
		"error.state_generation_failed":  "生成安全令牌失败",
		"error.oidc_missing_code":        "缺少授权码",
		"error.oidc_invalid_state":       "无效的安全令牌",
		"error.oidc_token_exchange_failed": "交换授权码失败",
		"error.oidc_missing_id_token":    "响应中缺少 ID 令牌",
		"error.oidc_token_verification_failed": "验证认证令牌失败",
		"error.password_login_disabled":  "启用 OIDC 时禁用密码登录",
		"error.oidc_error":               "认证错误",
		// Success messages
		"success.login":      "登录成功",
		// Login messages
		"login.oidc_button":  "使用 %s 登录",
	},
	LangFR: {
		// Error messages
		"error.auth_required":            "Authentification requise",
		"error.invalid_password":         "Mot de passe invalide",
		"error.invalid_callback":         "Hôte de rappel invalide",
		"error.session_store_failed":     "Erreur interne du serveur : échec d'accès au stockage de session",
		"error.authenticate_failed":      "Erreur interne du serveur : échec de l'authentification de la session",
		"error.missing_session_id":       "ID de session manquant",
		"error.config_invalid":           "Erreur de configuration : valeur invalide pour la variable d'environnement '%s' : '%s'",
		"error.config_invalid_values":    "Erreur de configuration : valeur invalide pour la variable d'environnement '%s' : '%s'.\n  Valeurs acceptées : %v\n  Veuillez vérifier votre configuration de variable d'environnement et réessayer.",
		"error.config_required":          "Erreur de configuration : la variable d'environnement '%s' est requise mais n'est pas définie.\n  Veuillez vérifier votre configuration de variable d'environnement et réessayer.",
		"error.config_required_not_set":  "non définie (requis)",
		"error.oidc_not_configured":      "OIDC n'est pas configuré",
		"error.state_generation_failed":  "Échec de la génération du jeton de sécurité",
		"error.oidc_missing_code":        "Code d'autorisation manquant",
		"error.oidc_invalid_state":       "Jeton de sécurité invalide",
		"error.oidc_token_exchange_failed": "Échec de l'échange du code d'autorisation",
		"error.oidc_missing_id_token":    "Jeton ID manquant dans la réponse",
		"error.oidc_token_verification_failed": "Échec de la vérification du jeton d'authentification",
		"error.password_login_disabled":  "La connexion par mot de passe est désactivée lorsque OIDC est activé",
		"error.oidc_error":               "Erreur d'authentification",
		// Success messages
		"success.login":      "Connexion réussie",
		// Login messages
		"login.oidc_button":  "Se connecter avec %s",
	},
	LangIT: {
		// Error messages
		"error.auth_required":            "Autenticazione richiesta",
		"error.invalid_password":         "Password non valida",
		"error.invalid_callback":         "Host di callback non valido",
		"error.session_store_failed":     "Errore interno del server: impossibile accedere al deposito delle sessioni",
		"error.authenticate_failed":      "Errore interno del server: impossibile autenticare la sessione",
		"error.missing_session_id":       "ID sessione mancante",
		"error.config_invalid":           "Errore di configurazione: valore non valido per la variabile d'ambiente '%s': '%s'",
		"error.config_invalid_values":    "Errore di configurazione: valore non valido per la variabile d'ambiente '%s': '%s'.\n  Valori accettati: %v\n  Si prega di controllare la configurazione della variabile d'ambiente e riprovare.",
		"error.config_required":          "Errore di configurazione: la variabile d'ambiente '%s' è richiesta ma non è impostata.\n  Si prega di controllare la configurazione della variabile d'ambiente e riprovare.",
		"error.config_required_not_set":  "non impostata (richiesto)",
		"error.oidc_not_configured":      "OIDC non è configurato",
		"error.state_generation_failed":  "Impossibile generare il token di sicurezza",
		"error.oidc_missing_code":        "Codice di autorizzazione mancante",
		"error.oidc_invalid_state":       "Token di sicurezza non valido",
		"error.oidc_token_exchange_failed": "Impossibile scambiare il codice di autorizzazione",
		"error.oidc_missing_id_token":    "Token ID mancante nella risposta",
		"error.oidc_token_verification_failed": "Impossibile verificare il token di autenticazione",
		"error.password_login_disabled":  "L'accesso con password è disabilitato quando OIDC è abilitato",
		"error.oidc_error":               "Errore di autenticazione",
		// Success messages
		"success.login":      "Accesso riuscito",
		// Login messages
		"login.oidc_button":  "Accedi con %s",
	},
	LangJA: {
		// Error messages
		"error.auth_required":            "認証が必要です",
		"error.invalid_password":         "パスワードが無効です",
		"error.invalid_callback":         "無効なコールバックホスト",
		"error.session_store_failed":     "内部サーバーエラー：セッションストアへのアクセスに失敗しました",
		"error.authenticate_failed":      "内部サーバーエラー：セッションの認証に失敗しました",
		"error.missing_session_id":       "セッションIDが不足しています",
		"error.config_invalid":           "設定エラー：環境変数 '%s' の値 '%s' が無効です",
		"error.config_invalid_values":    "設定エラー：環境変数 '%s' の値 '%s' が無効です。\n  受け入れられる値: %v\n  環境変数の設定を確認して再試行してください。",
		"error.config_required":          "設定エラー：環境変数 '%s' は必須ですが設定されていません。\n  環境変数の設定を確認して再試行してください。",
		"error.config_required_not_set":  "設定されていません（必須）",
		"error.oidc_not_configured":      "OIDC が設定されていません",
		"error.state_generation_failed":  "セキュリティトークンの生成に失敗しました",
		"error.oidc_missing_code":        "認証コードが不足しています",
		"error.oidc_invalid_state":       "無効なセキュリティトークン",
		"error.oidc_token_exchange_failed": "認証コードの交換に失敗しました",
		"error.oidc_missing_id_token":    "応答に ID トークンが不足しています",
		"error.oidc_token_verification_failed": "認証トークンの検証に失敗しました",
		"error.password_login_disabled":  "OIDC が有効な場合、パスワードログインは無効になります",
		"error.oidc_error":               "認証エラー",
		// Success messages
		"success.login":      "ログイン成功",
		// Login messages
		"login.oidc_button":  "%sでログイン",
	},
	LangDE: {
		// Error messages
		"error.auth_required":            "Authentifizierung erforderlich",
		"error.invalid_password":         "Ungültiges Passwort",
		"error.invalid_callback":         "Ungültiger Callback-Host",
		"error.session_store_failed":     "Interner Serverfehler: Fehler beim Zugriff auf den Sitzungsspeicher",
		"error.authenticate_failed":      "Interner Serverfehler: Fehler bei der Authentifizierung der Sitzung",
		"error.missing_session_id":       "Sitzungs-ID fehlt",
		"error.config_invalid":           "Konfigurationsfehler: Ungültiger Wert für Umgebungsvariable '%s': '%s'",
		"error.config_invalid_values":    "Konfigurationsfehler: Ungültiger Wert für Umgebungsvariable '%s': '%s'.\n  Akzeptierte Werte: %v\n  Bitte überprüfen Sie Ihre Umgebungsvariablen-Konfiguration und versuchen Sie es erneut.",
		"error.config_required":          "Konfigurationsfehler: Umgebungsvariable '%s' ist erforderlich, wurde aber nicht gesetzt.\n  Bitte überprüfen Sie Ihre Umgebungsvariablen-Konfiguration und versuchen Sie es erneut.",
		"error.config_required_not_set":  "nicht gesetzt (erforderlich)",
		"error.oidc_not_configured":      "OIDC ist nicht konfiguriert",
		"error.state_generation_failed":  "Fehler beim Generieren des Sicherheitstokens",
		"error.oidc_missing_code":        "Autorisierungscode fehlt",
		"error.oidc_invalid_state":       "Ungültiges Sicherheitstoken",
		"error.oidc_token_exchange_failed": "Fehler beim Austausch des Autorisierungscodes",
		"error.oidc_missing_id_token":    "ID-Token in der Antwort fehlt",
		"error.oidc_token_verification_failed": "Fehler beim Verifizieren des Authentifizierungstokens",
		"error.password_login_disabled":  "Passwort-Anmeldung ist deaktiviert, wenn OIDC aktiviert ist",
		"error.oidc_error":               "Authentifizierungsfehler",
		// Success messages
		"success.login":      "Anmeldung erfolgreich",
		// Login messages
		"login.oidc_button":  "Anmelden mit %s",
	},
	LangKO: {
		// Error messages
		"error.auth_required":            "인증이 필요합니다",
		"error.invalid_password":         "잘못된 비밀번호",
		"error.invalid_callback":         "잘못된 콜백 호스트",
		"error.session_store_failed":     "내부 서버 오류: 세션 저장소에 액세스하지 못했습니다",
		"error.authenticate_failed":      "내부 서버 오류: 세션 인증에 실패했습니다",
		"error.missing_session_id":       "세션 ID가 없습니다",
		"error.config_invalid":           "구성 오류: 환경 변수 '%s'의 값 '%s'이(가) 유효하지 않습니다",
		"error.config_invalid_values":    "구성 오류: 환경 변수 '%s'의 값 '%s'이(가) 유효하지 않습니다.\n  허용되는 값: %v\n  환경 변수 구성을 확인하고 다시 시도하세요.",
		"error.config_required":          "구성 오류: 환경 변수 '%s'이(가) 필요하지만 설정되지 않았습니다.\n  환경 변수 구성을 확인하고 다시 시도하세요.",
		"error.config_required_not_set":  "설정되지 않음 (필수)",
		"error.oidc_not_configured":      "OIDC가 구성되지 않았습니다",
		"error.state_generation_failed":  "보안 토큰 생성 실패",
		"error.oidc_missing_code":        "인증 코드가 누락되었습니다",
		"error.oidc_invalid_state":       "잘못된 보안 토큰",
		"error.oidc_token_exchange_failed": "인증 코드 교환 실패",
		"error.oidc_missing_id_token":    "응답에서 ID 토큰이 누락되었습니다",
		"error.oidc_token_verification_failed": "인증 토큰 검증 실패",
		"error.password_login_disabled":  "OIDC가 활성화된 경우 비밀번호 로그인이 비활성화됩니다",
		"error.oidc_error":               "인증 오류",
		// Success messages
		"success.login":      "로그인 성공",
		// Login messages
		"login.oidc_button":  "%s로 로그인",
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
