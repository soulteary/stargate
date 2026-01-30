package i18n

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	kit "github.com/soulteary/i18n-kit"
)

// Language type alias for backward compatibility
type Language = kit.Language

// Language constants for backward compatibility
const (
	LangEN = kit.LangEN
	LangZH = kit.LangZH
	LangFR = kit.LangFR
	LangIT = kit.LangIT
	LangJA = kit.LangJA
	LangDE = kit.LangDE
	LangKO = kit.LangKO
)

// bundle is the global translation bundle
var bundle *kit.Bundle

func init() {
	bundle = kit.NewBundle(kit.LangEN)

	// Add English translations
	bundle.AddTranslations(kit.LangEN, map[string]string{
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
		"error.authentication_failed":   "Authentication failed",
		// Herald verification errors
		"error.verify_code_expired":                "Verification code has expired. Please request a new code.",
		"error.verify_code_invalid":                "Invalid verification code. Please check and try again.",
		"error.verify_code_invalid_with_attempts":  "Invalid verification code. %d attempts remaining. Please check and try again.",
		"error.verify_code_locked":                 "Verification code has been locked due to too many failed attempts. Please request a new code.",
		"error.verify_code_too_many":               "Too many verification attempts. Please request a new code.",
		"error.verify_code_rate_limited":           "Too many requests. Please wait a moment and try again.",
		"error.verify_code_rate_limited_with_wait": "Too many requests. Please wait %d seconds and try again.",
		"error.verify_code_send_failed":            "Failed to send verification code. Please try again later.",
		"error.verify_code_unauthorized":           "Authentication service error. Please contact administrator.",
		"error.verify_code_failed":                 "Verification failed. Please try again.",
		"error.step_up_required":                   "Additional authentication required for this resource.",
		// Herald/OTP and verify flow
		"error.herald_not_configured":                    "Verification code service is not configured.",
		"error.herald_not_configured_use_otp":            "Verification code service is not configured. Please use OTP.",
		"error.herald_not_configured_use_otp_or_contact": "Verification code service is not configured. Please use OTP or contact administrator.",
		"error.herald_unavailable":                       "Verification code service is unavailable.",
		"error.herald_unavailable_use_otp":               "Verification code service is temporarily unavailable. Please use OTP.",
		"error.herald_unavailable_retry":                 "Verification code service is temporarily unavailable. Please try again later.",
		"error.verify_code_and_challenge_required":       "Verification code and challenge_id are required.",
		"error.verify_failed":                            "Verification failed.",
		"error.otp_code_required":                        "OTP code is required.",
		"error.otp_config_error":                         "OTP configuration error.",
		"error.otp_code_invalid":                         "Invalid OTP code.",
		"error.provide_verify_code_or_otp":               "Please provide verification code or use OTP.",
		"error.choose_verify_method":                     "Please choose verification method: code or OTP.",
		"error.rate_limited_retry":                       "Too many requests. Please try again later.",
		"error.send_verify_code_failed":                  "Failed to send verification code: %s",
		"success.login":                                  "Login successful",
		"success.verify_code_sent":                       "Verification code sent",
		"info.click_if_no_redirect":                      "Click here if the page does not redirect automatically",
	})

	// Add Chinese translations
	bundle.AddTranslations(kit.LangZH, map[string]string{
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
		"error.authentication_failed":   "认证失败",
		// Herald verification errors
		"error.verify_code_expired":                "验证码已过期，请重新获取验证码",
		"error.verify_code_invalid":                "验证码错误，请检查后重试",
		"error.verify_code_invalid_with_attempts":  "验证码错误，还剩 %d 次尝试机会，请检查后重试",
		"error.verify_code_locked":                 "验证码已被锁定（尝试次数过多），请重新获取验证码",
		"error.verify_code_too_many":               "验证尝试次数过多，请重新获取验证码",
		"error.verify_code_rate_limited":           "请求过于频繁，请稍后再试",
		"error.verify_code_rate_limited_with_wait": "请求过于频繁，请等待 %d 秒后重试",
		"error.verify_code_send_failed":            "发送验证码失败，请稍后重试",
		"error.verify_code_unauthorized":           "验证服务错误，请联系管理员",
		"error.verify_code_failed":                 "验证失败，请重试",
		"error.step_up_required":                   "访问此资源需要二次验证",
		// Herald/OTP and verify flow
		"error.herald_not_configured":                    "验证码服务未配置",
		"error.herald_not_configured_use_otp":            "验证码服务未配置，请使用 OTP 验证",
		"error.herald_not_configured_use_otp_or_contact": "验证码服务未配置，请使用 OTP 或联系管理员",
		"error.herald_unavailable":                       "验证码服务不可用",
		"error.herald_unavailable_use_otp":               "验证码服务暂时不可用，请使用 OTP 验证",
		"error.herald_unavailable_retry":                 "验证码服务暂时不可用，请稍后重试",
		"error.verify_code_and_challenge_required":       "验证码和 challenge_id 不能为空",
		"error.verify_failed":                            "验证失败",
		"error.otp_code_required":                        "OTP 验证码不能为空",
		"error.otp_config_error":                         "OTP 配置错误",
		"error.otp_code_invalid":                         "OTP 验证码错误",
		"error.provide_verify_code_or_otp":               "请提供验证码或使用 OTP",
		"error.choose_verify_method":                     "请选择验证方式：验证码或 OTP",
		"error.rate_limited_retry":                       "请求过于频繁，请稍后重试",
		"error.send_verify_code_failed":                  "发送验证码失败: %s",
		"success.login":                                  "登录成功",
		"success.verify_code_sent":                       "验证码已发送",
		"info.click_if_no_redirect":                      "点击这里如果页面没有自动跳转",
	})

	// Add French translations
	bundle.AddTranslations(kit.LangFR, map[string]string{
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
		"error.authentication_failed":   "Échec de l'authentification",
		// Herald verification errors
		"error.verify_code_expired":                      "Le code de vérification a expiré. Veuillez demander un nouveau code.",
		"error.verify_code_invalid":                      "Code de vérification invalide. Veuillez vérifier et réessayer.",
		"error.verify_code_invalid_with_attempts":        "Code de vérification invalide. %d tentatives restantes. Veuillez vérifier et réessayer.",
		"error.verify_code_locked":                       "Le code de vérification a été verrouillé en raison de trop de tentatives échouées. Veuillez demander un nouveau code.",
		"error.verify_code_too_many":                     "Trop de tentatives de vérification. Veuillez demander un nouveau code.",
		"error.verify_code_rate_limited":                 "Trop de demandes. Veuillez attendre un moment et réessayer.",
		"error.verify_code_rate_limited_with_wait":       "Trop de demandes. Veuillez attendre %d secondes et réessayer.",
		"error.verify_code_send_failed":                  "Échec de l'envoi du code de vérification. Veuillez réessayer plus tard.",
		"error.verify_code_unauthorized":                 "Erreur du service d'authentification. Veuillez contacter l'administrateur.",
		"error.verify_code_failed":                       "Échec de la vérification. Veuillez réessayer.",
		"error.step_up_required":                         "Authentification supplémentaire requise pour cette ressource.",
		"error.herald_not_configured":                    "Le service de code de vérification n'est pas configuré.",
		"error.herald_not_configured_use_otp":            "Le service de code de vérification n'est pas configuré. Veuillez utiliser OTP.",
		"error.herald_not_configured_use_otp_or_contact": "Le service de code de vérification n'est pas configuré. Veuillez utiliser OTP ou contacter l'administrateur.",
		"error.herald_unavailable":                       "Le service de code de vérification est indisponible.",
		"error.herald_unavailable_use_otp":               "Le service de code de vérification est temporairement indisponible. Veuillez utiliser OTP.",
		"error.herald_unavailable_retry":                 "Le service de code de vérification est temporairement indisponible. Veuillez réessayer plus tard.",
		"error.verify_code_and_challenge_required":       "Le code de vérification et le challenge_id sont obligatoires.",
		"error.verify_failed":                            "Échec de la vérification.",
		"error.otp_code_required":                        "Le code OTP est obligatoire.",
		"error.otp_config_error":                         "Erreur de configuration OTP.",
		"error.otp_code_invalid":                         "Code OTP invalide.",
		"error.provide_verify_code_or_otp":               "Veuillez fournir le code de vérification ou utiliser OTP.",
		"error.choose_verify_method":                     "Veuillez choisir la méthode de vérification : code ou OTP.",
		"error.rate_limited_retry":                       "Trop de demandes. Veuillez réessayer plus tard.",
		"error.send_verify_code_failed":                  "Échec de l'envoi du code de vérification : %s",
		"success.login":                                  "Connexion réussie",
		"success.verify_code_sent":                       "Code de vérification envoyé",
		"info.click_if_no_redirect":                      "Cliquez ici si la page ne redirige pas automatiquement",
	})

	// Add Italian translations
	bundle.AddTranslations(kit.LangIT, map[string]string{
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
		"error.authentication_failed":   "Autenticazione fallita",
		// Herald verification errors
		"error.verify_code_expired":                      "Il codice di verifica è scaduto. Si prega di richiedere un nuovo codice.",
		"error.verify_code_invalid":                      "Codice di verifica non valido. Si prega di controllare e riprovare.",
		"error.verify_code_invalid_with_attempts":        "Codice di verifica non valido. %d tentativi rimanenti. Si prega di controllare e riprovare.",
		"error.verify_code_locked":                       "Il codice di verifica è stato bloccato a causa di troppi tentativi falliti. Si prega di richiedere un nuovo codice.",
		"error.verify_code_too_many":                     "Troppi tentativi di verifica. Si prega di richiedere un nuovo codice.",
		"error.verify_code_rate_limited":                 "Troppe richieste. Si prega di attendere un momento e riprovare.",
		"error.verify_code_rate_limited_with_wait":       "Troppe richieste. Si prega di attendere %d secondi e riprovare.",
		"error.verify_code_send_failed":                  "Invio del codice di verifica non riuscito. Si prega di riprovare più tardi.",
		"error.verify_code_unauthorized":                 "Errore del servizio di autenticazione. Si prega di contattare l'amministratore.",
		"error.verify_code_failed":                       "Verifica fallita. Si prega di riprovare.",
		"error.step_up_required":                         "Autenticazione aggiuntiva richiesta per questa risorsa.",
		"error.herald_not_configured":                    "Il servizio di codice di verifica non è configurato.",
		"error.herald_not_configured_use_otp":            "Il servizio di codice di verifica non è configurato. Si prega di utilizzare OTP.",
		"error.herald_not_configured_use_otp_or_contact": "Il servizio di codice di verifica non è configurato. Si prega di utilizzare OTP o contattare l'amministratore.",
		"error.herald_unavailable":                       "Il servizio di codice di verifica non è disponibile.",
		"error.herald_unavailable_use_otp":               "Il servizio di codice di verifica è temporaneamente non disponibile. Si prega di utilizzare OTP.",
		"error.herald_unavailable_retry":                 "Il servizio di codice di verifica è temporaneamente non disponibile. Si prega di riprovare più tardi.",
		"error.verify_code_and_challenge_required":       "Il codice di verifica e il challenge_id sono obbligatori.",
		"error.verify_failed":                            "Verifica fallita.",
		"error.otp_code_required":                        "Il codice OTP è obbligatorio.",
		"error.otp_config_error":                         "Errore di configurazione OTP.",
		"error.otp_code_invalid":                         "Codice OTP non valido.",
		"error.provide_verify_code_or_otp":               "Si prega di fornire il codice di verifica o utilizzare OTP.",
		"error.choose_verify_method":                     "Si prega di scegliere il metodo di verifica: codice o OTP.",
		"error.rate_limited_retry":                       "Troppe richieste. Si prega di riprovare più tardi.",
		"error.send_verify_code_failed":                  "Invio del codice di verifica non riuscito: %s",
		"success.login":                                  "Accesso riuscito",
		"success.verify_code_sent":                       "Codice di verifica inviato",
		"info.click_if_no_redirect":                      "Clicca qui se la pagina non reindirizza automaticamente",
	})

	// Add Japanese translations
	bundle.AddTranslations(kit.LangJA, map[string]string{
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
		"error.authentication_failed":   "認証に失敗しました",
		// Herald verification errors
		"error.verify_code_expired":                      "確認コードの有効期限が切れました。新しいコードをリクエストしてください。",
		"error.verify_code_invalid":                      "確認コードが無効です。確認して再試行してください。",
		"error.verify_code_invalid_with_attempts":        "確認コードが無効です。残り %d 回の試行があります。確認して再試行してください。",
		"error.verify_code_locked":                       "失敗した試行が多すぎるため、確認コードがロックされました。新しいコードをリクエストしてください。",
		"error.verify_code_too_many":                     "確認試行が多すぎます。新しいコードをリクエストしてください。",
		"error.verify_code_rate_limited":                 "リクエストが多すぎます。しばらく待ってから再試行してください。",
		"error.verify_code_rate_limited_with_wait":       "リクエストが多すぎます。%d 秒待ってから再試行してください。",
		"error.verify_code_send_failed":                  "確認コードの送信に失敗しました。後でもう一度お試しください。",
		"error.verify_code_unauthorized":                 "認証サービスエラー。管理者に連絡してください。",
		"error.verify_code_failed":                       "確認に失敗しました。再試行してください。",
		"error.step_up_required":                         "このリソースには追加の認証が必要です。",
		"error.herald_not_configured":                    "確認コードサービスが設定されていません。",
		"error.herald_not_configured_use_otp":            "確認コードサービスが設定されていません。OTPをご利用ください。",
		"error.herald_not_configured_use_otp_or_contact": "確認コードサービスが設定されていません。OTPをご利用いただくか、管理者にお問い合わせください。",
		"error.herald_unavailable":                       "確認コードサービスは利用できません。",
		"error.herald_unavailable_use_otp":               "確認コードサービスは一時的に利用できません。OTPをご利用ください。",
		"error.herald_unavailable_retry":                 "確認コードサービスは一時的に利用できません。しばらくしてから再試行してください。",
		"error.verify_code_and_challenge_required":       "確認コードとchallenge_idは必須です。",
		"error.verify_failed":                            "確認に失敗しました。",
		"error.otp_code_required":                        "OTPコードは必須です。",
		"error.otp_config_error":                         "OTP設定エラー。",
		"error.otp_code_invalid":                         "無効なOTPコードです。",
		"error.provide_verify_code_or_otp":               "確認コードを入力するか、OTPをご利用ください。",
		"error.choose_verify_method":                     "確認方法を選択してください：コードまたはOTP。",
		"error.rate_limited_retry":                       "リクエストが多すぎます。しばらくしてから再試行してください。",
		"error.send_verify_code_failed":                  "確認コードの送信に失敗しました: %s",
		"success.login":                                  "ログイン成功",
		"success.verify_code_sent":                       "確認コードを送信しました",
		"info.click_if_no_redirect":                      "ページが自動的にリダイレクトしない場合はここをクリックしてください",
	})

	// Add German translations
	bundle.AddTranslations(kit.LangDE, map[string]string{
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
		"error.authentication_failed":   "Authentifizierung fehlgeschlagen",
		// Herald verification errors
		"error.verify_code_expired":                      "Der Bestätigungscode ist abgelaufen. Bitte fordern Sie einen neuen Code an.",
		"error.verify_code_invalid":                      "Ungültiger Bestätigungscode. Bitte überprüfen Sie und versuchen Sie es erneut.",
		"error.verify_code_invalid_with_attempts":        "Ungültiger Bestätigungscode. %d Versuche verbleibend. Bitte überprüfen Sie und versuchen Sie es erneut.",
		"error.verify_code_locked":                       "Der Bestätigungscode wurde aufgrund zu vieler fehlgeschlagener Versuche gesperrt. Bitte fordern Sie einen neuen Code an.",
		"error.verify_code_too_many":                     "Zu viele Bestätigungsversuche. Bitte fordern Sie einen neuen Code an.",
		"error.verify_code_rate_limited":                 "Zu viele Anfragen. Bitte warten Sie einen Moment und versuchen Sie es erneut.",
		"error.verify_code_rate_limited_with_wait":       "Zu viele Anfragen. Bitte warten Sie %d Sekunden und versuchen Sie es erneut.",
		"error.verify_code_send_failed":                  "Senden des Bestätigungscodes fehlgeschlagen. Bitte versuchen Sie es später erneut.",
		"error.verify_code_unauthorized":                 "Authentifizierungsdienstfehler. Bitte kontaktieren Sie den Administrator.",
		"error.verify_code_failed":                       "Bestätigung fehlgeschlagen. Bitte versuchen Sie es erneut.",
		"error.step_up_required":                         "Zusätzliche Authentifizierung für diese Ressource erforderlich.",
		"error.herald_not_configured":                    "Der Bestätigungscode-Dienst ist nicht konfiguriert.",
		"error.herald_not_configured_use_otp":            "Der Bestätigungscode-Dienst ist nicht konfiguriert. Bitte verwenden Sie OTP.",
		"error.herald_not_configured_use_otp_or_contact": "Der Bestätigungscode-Dienst ist nicht konfiguriert. Bitte verwenden Sie OTP oder kontaktieren Sie den Administrator.",
		"error.herald_unavailable":                       "Der Bestätigungscode-Dienst ist nicht verfügbar.",
		"error.herald_unavailable_use_otp":               "Der Bestätigungscode-Dienst ist vorübergehend nicht verfügbar. Bitte verwenden Sie OTP.",
		"error.herald_unavailable_retry":                 "Der Bestätigungscode-Dienst ist vorübergehend nicht verfügbar. Bitte versuchen Sie es später erneut.",
		"error.verify_code_and_challenge_required":       "Bestätigungscode und challenge_id sind erforderlich.",
		"error.verify_failed":                            "Bestätigung fehlgeschlagen.",
		"error.otp_code_required":                        "OTP-Code ist erforderlich.",
		"error.otp_config_error":                         "OTP-Konfigurationsfehler.",
		"error.otp_code_invalid":                         "Ungültiger OTP-Code.",
		"error.provide_verify_code_or_otp":               "Bitte geben Sie den Bestätigungscode ein oder verwenden Sie OTP.",
		"error.choose_verify_method":                     "Bitte wählen Sie die Bestätigungsmethode: Code oder OTP.",
		"error.rate_limited_retry":                       "Zu viele Anfragen. Bitte versuchen Sie es später erneut.",
		"error.send_verify_code_failed":                  "Senden des Bestätigungscodes fehlgeschlagen: %s",
		"success.login":                                  "Anmeldung erfolgreich",
		"success.verify_code_sent":                       "Bestätigungscode gesendet",
		"info.click_if_no_redirect":                      "Klicken Sie hier, wenn die Seite nicht automatisch weitergeleitet wird",
	})

	// Add Korean translations
	bundle.AddTranslations(kit.LangKO, map[string]string{
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
		"error.authentication_failed":   "인증 실패",
		// Herald verification errors
		"error.verify_code_expired":                      "인증 코드가 만료되었습니다. 새 코드를 요청하세요.",
		"error.verify_code_invalid":                      "잘못된 인증 코드입니다. 확인 후 다시 시도하세요.",
		"error.verify_code_invalid_with_attempts":        "잘못된 인증 코드입니다. %d회 시도 기회가 남았습니다. 확인 후 다시 시도하세요.",
		"error.verify_code_locked":                       "실패한 시도가 너무 많아 인증 코드가 잠겼습니다. 새 코드를 요청하세요.",
		"error.verify_code_too_many":                     "인증 시도가 너무 많습니다. 새 코드를 요청하세요.",
		"error.verify_code_rate_limited":                 "요청이 너무 많습니다. 잠시 후 다시 시도하세요.",
		"error.verify_code_rate_limited_with_wait":       "요청이 너무 많습니다. %d초 후 다시 시도하세요.",
		"error.verify_code_send_failed":                  "인증 코드 전송에 실패했습니다. 나중에 다시 시도하세요.",
		"error.verify_code_unauthorized":                 "인증 서비스 오류. 관리자에게 문의하세요.",
		"error.verify_code_failed":                       "인증에 실패했습니다. 다시 시도하세요.",
		"error.step_up_required":                         "이 리소스에 대한 추가 인증이 필요합니다.",
		"error.herald_not_configured":                    "인증 코드 서비스가 구성되지 않았습니다.",
		"error.herald_not_configured_use_otp":            "인증 코드 서비스가 구성되지 않았습니다. OTP를 사용하세요.",
		"error.herald_not_configured_use_otp_or_contact": "인증 코드 서비스가 구성되지 않았습니다. OTP를 사용하거나 관리자에게 문의하세요.",
		"error.herald_unavailable":                       "인증 코드 서비스를 사용할 수 없습니다.",
		"error.herald_unavailable_use_otp":               "인증 코드 서비스를 일시적으로 사용할 수 없습니다. OTP를 사용하세요.",
		"error.herald_unavailable_retry":                 "인증 코드 서비스를 일시적으로 사용할 수 없습니다. 나중에 다시 시도하세요.",
		"error.verify_code_and_challenge_required":       "인증 코드와 challenge_id가 필요합니다.",
		"error.verify_failed":                            "인증에 실패했습니다.",
		"error.otp_code_required":                        "OTP 코드가 필요합니다.",
		"error.otp_config_error":                         "OTP 구성 오류.",
		"error.otp_code_invalid":                         "잘못된 OTP 코드입니다.",
		"error.provide_verify_code_or_otp":               "인증 코드를 입력하거나 OTP를 사용하세요.",
		"error.choose_verify_method":                     "인증 방법을 선택하세요: 코드 또는 OTP.",
		"error.rate_limited_retry":                       "요청이 너무 많습니다. 나중에 다시 시도하세요.",
		"error.send_verify_code_failed":                  "인증 코드 전송에 실패했습니다: %s",
		"success.login":                                  "로그인 성공",
		"success.verify_code_sent":                       "인증 코드가 전송되었습니다",
		"info.click_if_no_redirect":                      "페이지가 자동으로 리디렉션되지 않으면 여기를 클릭하세요",
	})
}

// T returns the translated string for the given key using the language from Fiber context.
// If the key is not found, it returns the key itself.
func T(c *fiber.Ctx, key string) string {
	return kit.TFromFiber(c, key)
}

// Tf returns a formatted translated string using the language from Fiber context.
func Tf(c *fiber.Ctx, key string, args ...interface{}) string {
	return fmt.Sprintf(T(c, key), args...)
}

// TWithLang returns the translated string for the given key using the specified language.
// This is useful for contexts where Fiber context is not available (e.g., configuration validation).
func TWithLang(lang Language, key string) string {
	return bundle.GetTranslation(lang, key)
}

// TfWithLang returns a formatted translated string using the specified language.
func TfWithLang(lang Language, key string, args ...interface{}) string {
	return fmt.Sprintf(TWithLang(lang, key), args...)
}

// TStatic returns the translated string using the default language (English).
// This is useful for contexts where Fiber context is not available,
// such as during configuration validation at startup.
func TStatic(key string) string {
	return bundle.GetTranslation(kit.LangEN, key)
}

// TfStatic returns a formatted translated string using the default language (English).
func TfStatic(key string, args ...interface{}) string {
	return fmt.Sprintf(TStatic(key), args...)
}

// GetBundle returns the global translation bundle.
// This is used by the middleware to inject the bundle into the Fiber context.
func GetBundle() *kit.Bundle {
	return bundle
}

// SetLanguage sets the default language for the bundle.
// This is used during application initialization to set the fallback language.
// Note: This is kept for backward compatibility with config initialization.
func SetLanguage(lang Language) {
	bundle.SetFallback(lang)
}

// GetLanguage returns the current default/fallback language.
func GetLanguage() Language {
	return bundle.GetFallback()
}
