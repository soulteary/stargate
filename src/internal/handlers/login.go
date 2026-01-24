package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"

	"github.com/soulteary/stargate/pkg/herald"
	"github.com/soulteary/stargate/src/internal/audit"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/stargate/src/internal/metrics"
	"github.com/soulteary/stargate/src/internal/tracing"
	"github.com/soulteary/stargate/src/internal/utils"
)

var (
	heraldClient     *herald.Client
	heraldClientInit sync.Once
)

// InitHeraldClient initializes the Herald client if enabled
func InitHeraldClient() {
	heraldClientInit.Do(func() {
		if !config.HeraldEnabled.ToBool() {
			logrus.Debug("Herald is not enabled, skipping client initialization")
			return
		}

		heraldURL := config.HeraldURL.String()
		if heraldURL == "" {
			logrus.Warn("HERALD_URL is not set, Herald client will not be initialized")
			return
		}

		opts := herald.DefaultOptions().
			WithBaseURL(heraldURL).
			WithAPIKey(config.HeraldAPIKey.String()).
			WithTimeout(10 * time.Second)

		// Add HMAC secret if configured (for service-to-service authentication)
		// HMAC takes precedence over API key if both are set
		hmacSecret := config.HeraldHMACSecret.String()
		if hmacSecret != "" {
			opts = opts.WithHMACSecret(hmacSecret)
			logrus.Debug("Herald client will use HMAC authentication")
		} else if config.HeraldAPIKey.String() != "" {
			logrus.Debug("Herald client will use API key authentication")
		} else {
			logrus.Warn("Neither HERALD_HMAC_SECRET nor HERALD_API_KEY is set. Herald client may not authenticate properly.")
		}

		// Add TLS/mTLS configuration if provided
		if caCertFile := config.HeraldTLSCACertFile.String(); caCertFile != "" {
			opts = opts.WithTLSCACert(caCertFile)
			logrus.Debug("Herald client will verify server certificate using CA cert")
		}
		if clientCert := config.HeraldTLSClientCert.String(); clientCert != "" {
			clientKey := config.HeraldTLSClientKey.String()
			if clientKey != "" {
				opts = opts.WithTLSClientCert(clientCert, clientKey)
				logrus.Debug("Herald client will use mTLS with client certificate")
			} else {
				logrus.Warn("HERALD_TLS_CLIENT_CERT_FILE is set but HERALD_TLS_CLIENT_KEY_FILE is not, mTLS will not be used")
			}
		}
		if serverName := config.HeraldTLSServerName.String(); serverName != "" {
			opts = opts.WithTLSServerName(serverName)
			logrus.Debugf("Herald client will use server name %s for TLS verification", serverName)
		}

		client, err := herald.NewClient(opts)
		if err != nil {
			logrus.Warnf("Failed to initialize Herald client: %v. Check HERALD_URL and HERALD_ENABLED configuration.", err)
			return
		}

		heraldClient = client
		logrus.Info("Herald client initialized successfully")
	})
}

// getHeraldClient returns the herald client, initializing it if necessary
func getHeraldClient() *herald.Client {
	InitHeraldClient()
	return heraldClient
}

// generateUserID generates a user ID from phone/mail
func generateUserID(phone, mail string) string {
	identifier := phone
	if identifier == "" {
		identifier = mail
	}
	hash := sha256.Sum256([]byte(identifier))
	return "u_" + hex.EncodeToString(hash[:])[:16]
}

// Authenticator defines an interface for authenticating sessions.
// This interface allows for easier testing by enabling mock implementations.
type Authenticator interface {
	Authenticate(sess *session.Session) error
}

// AuthAuthenticator wraps auth.Authenticate to implement Authenticator interface.
type AuthAuthenticator struct{}

// Authenticate marks a session as authenticated.
func (a *AuthAuthenticator) Authenticate(sess *session.Session) error {
	return auth.Authenticate(sess)
}

// loginAPIHandler is the internal handler that can be tested with mocked dependencies.
func loginAPIHandler(ctx *fiber.Ctx, sessionGetter SessionGetter, authenticator Authenticator) error {
	// Get trace context from middleware
	traceCtx := ctx.Locals("trace_context")
	if traceCtx == nil {
		traceCtx = ctx.Context()
	}
	spanCtx := traceCtx.(context.Context)

	// Start span for login
	loginCtx, loginSpan := tracing.StartSpan(spanCtx, "auth.login")
	defer loginSpan.End()

	password := ctx.FormValue("password")
	authMethod := ctx.FormValue("auth_method") // "password" or "warden"
	userPhone := ctx.FormValue("phone")
	userMail := ctx.FormValue("mail")

	loginSpan.SetAttributes(
		attribute.String("auth.method", authMethod),
	)

	var userID string          // Declare userID at function scope
	var verifyRespAMR []string // Store AMR from Herald response

	// Determine authentication method
	// If auth_method is not specified, default to password authentication for backward compatibility
	if authMethod == "" {
		authMethod = "password"
	}

	var authenticated bool

	if authMethod == "warden" {
		// Warden user list authentication
		// Check if at least one identifier is provided
		if userPhone == "" && userMail == "" {
			tracing.RecordError(loginSpan, fmt.Errorf("no identifier provided"))
			return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T("error.user_not_in_list"))
		}

		// Start span for Warden get user info
		wardenCtx, wardenSpan := tracing.StartSpan(loginCtx, "warden.get_user_info")
		wardenSpan.SetAttributes(
			attribute.String("warden.identifier_type", func() string {
				if userPhone != "" {
					return "phone"
				}
				return "mail"
			}()),
		)

		// Log the authentication attempt
		logrus.Debugf("Attempting Warden authentication: phone=%s, mail=%s", utils.MaskPhone(userPhone), utils.MaskEmail(userMail))

		// Step 1: Get complete user information from Warden (includes status check)
		wardenStartTime := time.Now()
		userInfo := auth.GetUserInfo(wardenCtx, userPhone, userMail)
		wardenDuration := time.Since(wardenStartTime)
		if userInfo == nil {
			wardenSpan.SetAttributes(attribute.Bool("warden.user_found", false))
			wardenSpan.End()
			tracing.RecordError(loginSpan, fmt.Errorf("user not found in Warden"))
			metrics.RecordWardenCall("get_user_info", "failure", wardenDuration)
			metrics.RecordAuthRequest("warden", "failure")
			logrus.Warnf("Warden authentication failed for: phone=%s, mail=%s", utils.MaskPhone(userPhone), utils.MaskEmail(userMail))
			audit.GetAuditLogger().LogLogin("", "warden", ctx.IP(), false, "user_not_in_list")
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.user_not_in_list"))
		}
		wardenSpan.SetAttributes(
			attribute.Bool("warden.user_found", true),
			attribute.String("warden.user_id", userInfo.UserID),
		)
		wardenSpan.End()
		metrics.RecordWardenCall("get_user_info", "success", wardenDuration)

		// Step 2: Use user_id from Warden if available, otherwise generate one
		userID = userInfo.UserID
		if userID == "" {
			userID = generateUserID(userPhone, userMail)
		}

		// Step 3: Get verification code and OTP code from form
		verifyCode := ctx.FormValue("verify_code")
		challengeID := ctx.FormValue("challenge_id")
		otpCode := ctx.FormValue("otp_code")
		useOTP := ctx.FormValue("use_otp") == "true"

		otpEnabled := config.WardenOTPEnabled.ToBool()

		// Step 4: Verify code via Herald (if not using OTP)
		if !useOTP {
			// Check if Herald is enabled
			if !config.HeraldEnabled.ToBool() {
				// If Herald is not enabled and OTP is also not enabled, return error
				if !otpEnabled {
					return SendErrorResponse(ctx, fiber.StatusInternalServerError, "验证码服务未配置，请使用 OTP 或联系管理员")
				}
				// If OTP is enabled, suggest user to use OTP
				return SendErrorResponse(ctx, fiber.StatusBadRequest, "验证码服务未配置，请使用 OTP 验证")
			}

			// Verify challenge via Herald
			if challengeID == "" || verifyCode == "" {
				return SendErrorResponse(ctx, fiber.StatusBadRequest, "验证码和 challenge_id 不能为空")
			}

			heraldClient := getHeraldClient()
			if heraldClient == nil {
				return SendErrorResponse(ctx, fiber.StatusInternalServerError, "验证码服务不可用")
			}

			verifyReq := &herald.VerifyChallengeRequest{
				ChallengeID: challengeID,
				Code:        verifyCode,
				ClientIP:    ctx.IP(),
			}

			// Start span for Herald verify challenge
			heraldCtx, heraldSpan := tracing.StartSpan(loginCtx, "herald.verify_challenge")
			heraldSpan.SetAttributes(
				attribute.String("herald.challenge_id", challengeID),
			)

			startTime := time.Now()
			verifyResp, err := heraldClient.VerifyChallenge(heraldCtx, verifyReq)
			duration := time.Since(startTime)
			if err != nil {
				tracing.RecordError(heraldSpan, err)
				heraldSpan.End()
				metrics.RecordHeraldCall("verify_challenge", "failure", duration)
				logrus.Errorf("Failed to verify challenge: %v", err)

				// Check if it's a connection error (Herald service unavailable)
				if heraldErr, ok := err.(*herald.HeraldError); ok {
					if heraldErr.StatusCode == 0 || heraldErr.Reason == "connection_failed" {
						// Herald service is unavailable, suggest OTP fallback if enabled
						if otpEnabled {
							return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "验证码服务暂时不可用，请使用 OTP 验证")
						}
						return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, "验证码服务暂时不可用，请稍后重试")
					}
					// Other Herald errors (unauthorized, etc.)
					if heraldErr.StatusCode == http.StatusUnauthorized {
						return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.verify_code_unauthorized"))
					}
				}

				// Default error handling
				return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.verify_code_failed"))
			}

			if !verifyResp.OK {
				heraldSpan.SetAttributes(
					attribute.String("herald.result", "failure"),
					attribute.String("herald.reason", verifyResp.Reason),
				)
				heraldSpan.End()
				metrics.RecordHeraldCall("verify_challenge", "failure", duration)
				reason := verifyResp.Reason
				if reason == "" {
					reason = "invalid"
				}
				logrus.Warnf("Challenge verification failed: reason=%s", reason)
				audit.GetAuditLogger().LogVerifyCodeCheck(userID, ctx.IP(), false, reason)

				// Provide detailed error message based on reason (as per Claude.md section 9)
				var errorMsg string
				switch reason {
				case "expired":
					errorMsg = i18n.T("error.verify_code_expired")
				case "invalid":
					errorMsg = i18n.T("error.verify_code_invalid")
					// Add remaining attempts if available
					if verifyResp.RemainingAttempts != nil {
						errorMsg = i18n.Tf("error.verify_code_invalid_with_attempts", *verifyResp.RemainingAttempts)
					}
				case "locked":
					errorMsg = i18n.T("error.verify_code_locked")
				case "too_many_attempts":
					errorMsg = i18n.T("error.verify_code_too_many")
				case "rate_limited":
					errorMsg = i18n.T("error.verify_code_rate_limited")
					// Add wait time if available
					if verifyResp.NextResendIn != nil {
						errorMsg = i18n.Tf("error.verify_code_rate_limited_with_wait", *verifyResp.NextResendIn)
					}
				case "send_failed":
					errorMsg = i18n.T("error.verify_code_send_failed")
				case "unauthorized":
					errorMsg = i18n.T("error.verify_code_unauthorized")
				default:
					errorMsg = i18n.T("error.verify_code_failed")
				}
				return SendErrorResponse(ctx, fiber.StatusUnauthorized, errorMsg)
			}

			// Log successful verification
			heraldSpan.SetAttributes(
				attribute.String("herald.result", "success"),
				attribute.String("herald.user_id", verifyResp.UserID),
			)
			heraldSpan.End()
			metrics.RecordHeraldCall("verify_challenge", "success", duration)
			audit.GetAuditLogger().LogVerifyCodeCheck(userID, ctx.IP(), true, "")

			// Verify user ID matches
			if verifyResp.UserID != userID {
				logrus.Warnf("User ID mismatch: expected=%s, got=%s", userID, verifyResp.UserID)
				return SendErrorResponse(ctx, fiber.StatusUnauthorized, "验证失败")
			}

			// Store AMR (Authentication Method Reference) from Herald response for later use
			if len(verifyResp.AMR) > 0 {
				verifyRespAMR = verifyResp.AMR
			}
		} else if otpEnabled && useOTP {
			// If OTP is enabled and user chose to use OTP
			if otpCode == "" {
				return SendErrorResponse(ctx, fiber.StatusBadRequest, "OTP 验证码不能为空")
			}

			// Get OTP secret
			otpSecret := auth.GetOTPSecret()
			if otpSecret == "" {
				logrus.Warn("OTP secret is not configured")
				return SendErrorResponse(ctx, fiber.StatusInternalServerError, "OTP 配置错误")
			}

			// Verify OTP code
			if !auth.VerifyOTP(otpSecret, otpCode) {
				metrics.RecordAuthRequest("warden_otp", "failure")
				logrus.Warnf("OTP verification failed: phone=%s, mail=%s", utils.MaskPhone(userPhone), utils.MaskEmail(userMail))
				audit.GetAuditLogger().LogLogin(userID, "warden_otp", ctx.IP(), false, "otp_verification_failed")
				return SendErrorResponse(ctx, fiber.StatusUnauthorized, "OTP 验证码错误")
			}
		} else {
			// Neither Herald verification nor OTP was used
			// This should not happen if frontend validation works correctly
			// But we add this check as a safety measure
			if !otpEnabled {
				return SendErrorResponse(ctx, fiber.StatusBadRequest, "请提供验证码或使用 OTP")
			}
			return SendErrorResponse(ctx, fiber.StatusBadRequest, "请选择验证方式：验证码或 OTP")
		}

		logrus.Infof("Warden authentication successful for: phone=%s, mail=%s", utils.MaskPhone(userPhone), utils.MaskEmail(userMail))
		authenticated = true
	} else {
		// Password authentication (default)
		if password == "" {
			metrics.RecordAuthRequest("password", "failure")
			audit.GetAuditLogger().LogLogin("", "password", ctx.IP(), false, "empty_password")
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.invalid_password"))
		}
		if !auth.CheckPassword(password) {
			metrics.RecordAuthRequest("password", "failure")
			audit.GetAuditLogger().LogLogin("", "password", ctx.IP(), false, "invalid_password")
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.invalid_password"))
		}
		authenticated = true
	}

	if !authenticated {
		return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.authentication_failed"))
	}

	sess, err := sessionGetter.Get(ctx)
	if err != nil {
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
	}

	// Set user information to session for warden authentication before authenticating
	if authMethod == "warden" {
		// Get complete user information from Warden
		userInfo := auth.GetUserInfo(ctx.Context(), userPhone, userMail)
		if userInfo != nil {
			// Store complete user information in session
			if userInfo.UserID != "" {
				sess.Set("user_id", userInfo.UserID)
			}
			if userInfo.Phone != "" {
				sess.Set("user_phone", userInfo.Phone)
			}
			if userInfo.Mail != "" {
				sess.Set("user_mail", userInfo.Mail)
			}
			if userInfo.Status != "" {
				sess.Set("user_status", userInfo.Status)
			}
			// Store scope and role for authorization headers
			if len(userInfo.Scope) > 0 {
				sess.Set("user_scope", userInfo.Scope)
			}
			if userInfo.Role != "" {
				sess.Set("user_role", userInfo.Role)
			}
			logrus.Debugf("Stored user info in session: user_id=%s, phone=%s, mail=%s, scope=%v, role=%s",
				userInfo.UserID, utils.MaskPhone(userInfo.Phone), utils.MaskEmail(userInfo.Mail), userInfo.Scope, userInfo.Role)
		} else {
			// Fallback: store basic info if GetUserInfo failed (should not happen after CheckUserInList)
			if userPhone != "" {
				sess.Set("user_phone", userPhone)
			}
			if userMail != "" {
				sess.Set("user_mail", userMail)
			}
		}

		// Store AMR (Authentication Method Reference) from Herald response if available
		if len(verifyRespAMR) > 0 {
			sess.Set("user_amr", verifyRespAMR)
		}
	}

	// Authenticate and save session (this will save all session data including user info)
	err = authenticator.Authenticate(sess)
	if err != nil {
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.authenticate_failed"))
	}

	// Log successful login and session creation
	var loggedUserID string
	if authMethod == "warden" {
		if userInfo := auth.GetUserInfo(ctx.Context(), userPhone, userMail); userInfo != nil && userInfo.UserID != "" {
			loggedUserID = userInfo.UserID
		} else {
			loggedUserID = userID
		}
		metrics.RecordAuthRequest(authMethod, "success")
		audit.GetAuditLogger().LogLogin(loggedUserID, authMethod, ctx.IP(), true, "")
	} else {
		metrics.RecordAuthRequest("password", "success")
		audit.GetAuditLogger().LogLogin("", "password", ctx.IP(), true, "")
	}
	metrics.RecordSessionCreated()
	audit.GetAuditLogger().LogSessionCreate(loggedUserID, ctx.IP())

	// Get callback parameter (priority: cookie, form data, query parameter)
	callbackFromCookie := GetCallbackFromCookie(ctx)
	callback := callbackFromCookie
	if callback == "" {
		callback = ctx.FormValue("callback")
	}
	if callback == "" {
		callback = ctx.Query("callback")
	}

	// If callback was retrieved from cookie, clear the cookie after successful login
	if callbackFromCookie != "" {
		ClearCallbackCookie(ctx)
	}

	// If no callback, try using origin host as callback
	if callback == "" {
		originHost := GetForwardedHost(ctx)
		// Only use origin host as callback if it's different from auth service domain
		if IsDifferentDomain(ctx) {
			callback = originHost
		}
	}

	// If callback exists, redirect to session exchange endpoint
	if callback != "" {
		// Get session ID (should already exist)
		sessionID := sess.ID()
		if sessionID == "" {
			// If ID is empty, try to get it from response cookie
			// In Fiber session, Save() will set session ID to response cookie
			cookieBytes := ctx.Response().Header.Peek("Set-Cookie")
			if len(cookieBytes) > 0 {
				cookieStr := string(cookieBytes)
				// Find session cookie (format: stargate_session=<session_id>; ...)
				cookieName := auth.SessionCookieName + "="
				if idx := strings.Index(cookieStr, cookieName); idx >= 0 {
					start := idx + len(cookieName)
					end := start
					for end < len(cookieStr) && cookieStr[end] != ';' && cookieStr[end] != ' ' {
						end++
					}
					sessionID = cookieStr[start:end]
				}
			}
			// If still empty, try to get session again
			if sessionID == "" {
				sess, err = sessionGetter.Get(ctx)
				if err != nil {
					return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
				}
				sessionID = sess.ID()
			}
			if sessionID == "" {
				// If session ID is still empty, return error
				return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.missing_session_id"))
			}
		}
		proto := GetForwardedProto(ctx)
		if proto == "" {
			proto = ctx.Protocol()
		}
		redirectURL := fmt.Sprintf("%s://%s/_session_exchange?id=%s", proto, callback, sessionID)
		return ctx.Redirect(redirectURL)
	}

	// If still no callback (origin domain is the auth service itself), return response based on request type
	if IsHTMLRequest(ctx) {
		// HTML request returns success message and adds meta refresh redirect to origin domain
		ctx.Set("Content-Type", "text/html; charset=utf-8")
		successMsg := i18n.T("success.login")

		// Get origin host and protocol
		originHost := GetForwardedHost(ctx)
		proto := GetForwardedProto(ctx)
		redirectURL := fmt.Sprintf("%s://%s", proto, originHost)

		// Escape URL to ensure HTML safety
		escapedURL := html.EscapeString(redirectURL)

		// Build HTML with meta refresh
		htmlContent := fmt.Sprintf(`<html><head><meta charset="UTF-8"><meta http-equiv="refresh" content="0;url=%s"><title>%s</title></head><body><h1>%s</h1><p>%s</p><p><a href="%s">点击这里如果页面没有自动跳转</a></p></body></html>`,
			escapedURL, successMsg, successMsg, successMsg, escapedURL)
		return ctx.Status(fiber.StatusOK).SendString(htmlContent)
	}

	// API request returns JSON response
	ctx.Set("Content-Type", "application/json")
	response := fiber.Map{
		"success": true,
		"message": i18n.T("success.login"),
	}
	// If session ID exists, add it to response
	if sessionID := sess.ID(); sessionID != "" {
		response["session_id"] = sessionID
	}
	return ctx.Status(fiber.StatusOK).JSON(response)
}

// LoginAPI handles POST requests to /_login for password authentication.
// It validates the password from the form data, creates a session if valid,
// and redirects to the callback URL (if provided) or returns a success response.
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func LoginAPI(store *session.Store) func(c *fiber.Ctx) error {
	sessionGetter := &SessionStoreAdapter{store: store}
	authenticator := &AuthAuthenticator{}
	return func(ctx *fiber.Ctx) error {
		return loginAPIHandler(ctx, sessionGetter, authenticator)
	}
}

// loginRouteHandler is the internal handler that can be tested with mocked dependencies.
func loginRouteHandler(ctx *fiber.Ctx, sessionGetter SessionGetter) error {
	// Get callback parameter (priority: URL query parameter, then cookie)
	// URL parameter takes priority as it represents the explicit intent of the current request
	callback := ctx.Query("callback")
	if callback == "" {
		callback = GetCallbackFromCookie(ctx)
	} else {
		// If URL has callback parameter, update cookie (if domain is different)
		SetCallbackCookie(ctx, callback)
	}

	sess, err := sessionGetter.Get(ctx)
	if err != nil {
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
	}

	if auth.IsAuthenticated(sess) {
		// Use X-Forwarded-* headers to build correct redirect URL
		sessionID := sess.ID()
		if sessionID == "" {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.missing_session_id"))
		}
		proto := GetForwardedProto(ctx)
		if proto == "" {
			proto = ctx.Protocol()
		}
		// If callback exists, redirect to callback's _session_exchange endpoint
		// If no callback, redirect to current host's root path
		if callback != "" {
			redirectURL := fmt.Sprintf("%s://%s/_session_exchange?id=%s", proto, callback, sessionID)
			return ctx.Redirect(redirectURL)
		}
		// When no callback, redirect to current host's root path
		host := GetForwardedHost(ctx)
		redirectURL := fmt.Sprintf("%s://%s/", proto, host)
		return ctx.Redirect(redirectURL)
	}

	// Select template based on Warden configuration
	templateName := "login"
	if config.WardenEnabled.ToBool() {
		templateName = "login.warden"
	}

	heraldEnabled := config.HeraldEnabled.ToBool()
	otpEnabled := config.WardenOTPEnabled.ToBool()

	return ctx.Render(templateName, fiber.Map{
		"Callback":      callback,
		"SessionID":     sess.ID(),
		"Title":         config.LoginPageTitle.Value,
		"FooterText":    config.LoginPageFooterText.Value,
		"WardenEnabled": config.WardenEnabled.ToBool(),
		"HeraldEnabled": heraldEnabled,
		"OTPEnabled":    otpEnabled,
	})
}

// LoginRoute handles GET requests to /_login for displaying the login page.
// If the user is already authenticated, it redirects to the session exchange endpoint.
// Otherwise, it renders the login page template.
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func LoginRoute(store *session.Store) func(c *fiber.Ctx) error {
	sessionGetter := &SessionStoreAdapter{store: store}
	return func(ctx *fiber.Ctx) error {
		return loginRouteHandler(ctx, sessionGetter)
	}
}

// Note: SendVerifyCodeAPI and sendVerifyCodeHandler have been removed.
// Verification code sending is now handled by the Herald service via the login flow.
