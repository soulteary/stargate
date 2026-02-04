package handlers

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	logger "github.com/soulteary/logger-kit"
	"go.opentelemetry.io/otel/attribute"

	"github.com/soulteary/herald/pkg/herald"
	secure "github.com/soulteary/secure-kit"
	"github.com/soulteary/stargate/src/internal/auditlog"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/heraldtotp"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/stargate/src/internal/metrics"
	"github.com/soulteary/tracing-kit"
	"github.com/soulteary/warden/pkg/warden"
)

// log is the package-level logger instance
var log *logger.Logger

// SetLogger sets the logger for this package
func SetLogger(l *logger.Logger) {
	log = l
}

var (
	heraldClient         *herald.Client
	heraldClientInit     sync.Once
	heraldTOTPClient     *heraldtotp.Client
	heraldTOTPClientInit sync.Once
)

// InitHeraldClient initializes the Herald client if enabled
func InitHeraldClient(l *logger.Logger) {
	log = l
	heraldClientInit.Do(func() {
		if !config.HeraldEnabled.ToBool() {
			log.Debug().Msg("Herald is not enabled, skipping client initialization")
			return
		}

		heraldURL := config.HeraldURL.String()
		if heraldURL == "" {
			log.Warn().Msg("HERALD_URL is not set, Herald client will not be initialized")
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
			log.Debug().Msg("Herald client will use HMAC authentication")
		} else if config.HeraldAPIKey.String() != "" {
			log.Debug().Msg("Herald client will use API key authentication")
		} else {
			log.Warn().Msg("Neither HERALD_HMAC_SECRET nor HERALD_API_KEY is set. Herald client may not authenticate properly.")
		}

		// Add TLS/mTLS configuration if provided
		if caCertFile := config.HeraldTLSCACertFile.String(); caCertFile != "" {
			opts = opts.WithTLSCACert(caCertFile)
			log.Debug().Msg("Herald client will verify server certificate using CA cert")
		}
		if clientCert := config.HeraldTLSClientCert.String(); clientCert != "" {
			clientKey := config.HeraldTLSClientKey.String()
			if clientKey != "" {
				opts = opts.WithTLSClientCert(clientCert, clientKey)
				log.Debug().Msg("Herald client will use mTLS with client certificate")
			} else {
				log.Warn().Msg("HERALD_TLS_CLIENT_CERT_FILE is set but HERALD_TLS_CLIENT_KEY_FILE is not, mTLS will not be used")
			}
		}
		if serverName := config.HeraldTLSServerName.String(); serverName != "" {
			opts = opts.WithTLSServerName(serverName)
			log.Debug().Str("server_name", serverName).Msg("Herald client will use server name for TLS verification")
		}

		client, err := herald.NewClient(opts)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to initialize Herald client. Check HERALD_URL and HERALD_ENABLED configuration.")
			return
		}

		heraldClient = client
		log.Info().Msg("Herald client initialized successfully")
	})
}

// getHeraldClient returns the herald client.
// Note: InitHeraldClient must be called with a logger before this function is used.
func getHeraldClient() *herald.Client {
	return heraldClient
}

// InitHeraldTOTPClient initializes the herald-totp client if HERALD_TOTP_ENABLED and HERALD_TOTP_BASE_URL are set.
func InitHeraldTOTPClient(l *logger.Logger) {
	log = l
	heraldTOTPClientInit.Do(func() {
		if !config.HeraldTOTPEnabled.ToBool() {
			log.Debug().Msg("Herald TOTP is not enabled, skipping client initialization")
			return
		}
		baseURL := config.HeraldTOTPBaseURL.String()
		if baseURL == "" {
			log.Debug().Msg("HERALD_TOTP_BASE_URL is not set, herald-totp client will not be initialized")
			return
		}
		opts := heraldtotp.DefaultOptions().
			WithBaseURL(strings.TrimSuffix(baseURL, "/")).
			WithAPIKey(config.HeraldTOTPAPIKey.String()).
			WithTimeout(10 * time.Second)
		if hmacSecret := config.HeraldTOTPHMACSecret.String(); hmacSecret != "" {
			opts = opts.WithHMACSecret(hmacSecret)
		}
		client, err := heraldtotp.NewClient(opts)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to initialize herald-totp client")
			return
		}
		heraldTOTPClient = client
		log.Info().Msg("Herald TOTP client initialized successfully")
	})
}

// getHeraldTOTPClient returns the herald-totp client (may be nil if not configured).
func getHeraldTOTPClient() *heraldtotp.Client {
	return heraldTOTPClient
}

// generateUserID generates a user ID from phone/mail
func generateUserID(phone, mail string) string {
	identifier := phone
	if identifier == "" {
		identifier = mail
	}
	return "u_" + secure.GetSHA256Hash(identifier)[:16]
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

	var userID string                        // Declare userID at function scope
	var verifyRespAMR []string               // Store AMR from Herald response
	var wardenUserInfo *warden.AllowListUser // Reused across warden branch to avoid repeated GetUserInfo

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
			return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.user_not_in_list"))
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
		log.Debug().Str("phone", secure.MaskPhone(userPhone)).Str("mail", secure.MaskEmail(userMail)).Msg("Attempting Warden authentication")

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
			log.Warn().Str("phone", secure.MaskPhone(userPhone)).Str("mail", secure.MaskEmail(userMail)).Msg("Warden authentication failed")
			auditlog.LogLogin(ctx.Context(), "", "warden", ctx.IP(), false, "user_not_in_list")
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.user_not_in_list"))
		}
		wardenSpan.SetAttributes(
			attribute.Bool("warden.user_found", true),
			attribute.String("warden.user_id", userInfo.UserID),
		)
		wardenSpan.End()
		metrics.RecordWardenCall("get_user_info", "success", wardenDuration)
		wardenUserInfo = userInfo // Reuse for session write and audit to avoid repeated GetUserInfo

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

		// OTP enabled: Warden global OTP or Herald TOTP (per-user) when configured
		otpEnabled := config.WardenOTPEnabled.ToBool() ||
			(config.HeraldTOTPEnabled.ToBool() && config.HeraldTOTPBaseURL.String() != "")

		// Step 4: Verify code via Herald (if not using OTP)
		if !useOTP {
			// Check if Herald is enabled
			if !config.HeraldEnabled.ToBool() {
				// If Herald is not enabled and OTP is also not enabled, return error
				if !otpEnabled {
					return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.herald_not_configured_use_otp_or_contact"))
				}
				// If OTP is enabled, suggest user to use OTP
				return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.herald_not_configured_use_otp"))
			}

			// Verify challenge via Herald
			if challengeID == "" || verifyCode == "" {
				return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.verify_code_and_challenge_required"))
			}

			heraldClient := getHeraldClient()
			if heraldClient == nil {
				return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.herald_unavailable"))
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
				log.Error().Err(err).Msg("Failed to verify challenge")

				// Check if it's a connection error (Herald service unavailable)
				if heraldErr, ok := err.(*herald.HeraldError); ok {
					if heraldErr.StatusCode == 0 || heraldErr.Reason == "connection_failed" {
						// Herald service is unavailable, suggest OTP fallback if enabled
						if otpEnabled {
							return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_use_otp"))
						}
						return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_retry"))
					}
					// Herald client returns (verifyResp, err) on 4xx; use verifyResp.Reason for user-facing message
					if (heraldErr.StatusCode == http.StatusUnauthorized || heraldErr.StatusCode == http.StatusBadRequest) &&
						verifyResp != nil && !verifyResp.OK && verifyResp.Reason != "" {
						reason := verifyResp.Reason
						auditlog.LogVerifyCodeCheck(ctx.Context(), userID, ctx.IP(), false, reason)
						var errorMsg string
						switch reason {
						case "expired":
							errorMsg = i18n.T(ctx, "error.verify_code_expired")
						case "invalid":
							errorMsg = i18n.T(ctx, "error.verify_code_invalid")
							if verifyResp.RemainingAttempts != nil {
								errorMsg = i18n.Tf(ctx, "error.verify_code_invalid_with_attempts", *verifyResp.RemainingAttempts)
							}
						case "locked":
							errorMsg = i18n.T(ctx, "error.verify_code_locked")
						case "too_many_attempts":
							errorMsg = i18n.T(ctx, "error.verify_code_too_many")
						case "rate_limited":
							errorMsg = i18n.T(ctx, "error.verify_code_rate_limited")
							if verifyResp.NextResendIn != nil {
								errorMsg = i18n.Tf(ctx, "error.verify_code_rate_limited_with_wait", *verifyResp.NextResendIn)
							}
						case "send_failed":
							errorMsg = i18n.T(ctx, "error.verify_code_send_failed")
						case "unauthorized":
							errorMsg = i18n.T(ctx, "error.verify_code_unauthorized")
						default:
							errorMsg = i18n.T(ctx, "error.verify_code_failed")
						}
						return SendErrorResponse(ctx, fiber.StatusUnauthorized, errorMsg)
					}
					// Real auth failure (e.g. bad HMAC), not verification failure
					if heraldErr.StatusCode == http.StatusUnauthorized {
						return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.verify_code_unauthorized"))
					}
				}

				// Default error handling
				return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.verify_code_failed"))
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
				log.Warn().Str("reason", reason).Msg("Challenge verification failed")
				auditlog.LogVerifyCodeCheck(ctx.Context(), userID, ctx.IP(), false, reason)

				// Provide detailed error message based on reason
				var errorMsg string
				switch reason {
				case "expired":
					errorMsg = i18n.T(ctx, "error.verify_code_expired")
				case "invalid":
					errorMsg = i18n.T(ctx, "error.verify_code_invalid")
					// Add remaining attempts if available
					if verifyResp.RemainingAttempts != nil {
						errorMsg = i18n.Tf(ctx, "error.verify_code_invalid_with_attempts", *verifyResp.RemainingAttempts)
					}
				case "locked":
					errorMsg = i18n.T(ctx, "error.verify_code_locked")
				case "too_many_attempts":
					errorMsg = i18n.T(ctx, "error.verify_code_too_many")
				case "rate_limited":
					errorMsg = i18n.T(ctx, "error.verify_code_rate_limited")
					// Add wait time if available
					if verifyResp.NextResendIn != nil {
						errorMsg = i18n.Tf(ctx, "error.verify_code_rate_limited_with_wait", *verifyResp.NextResendIn)
					}
				case "send_failed":
					errorMsg = i18n.T(ctx, "error.verify_code_send_failed")
				case "unauthorized":
					errorMsg = i18n.T(ctx, "error.verify_code_unauthorized")
				default:
					errorMsg = i18n.T(ctx, "error.verify_code_failed")
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
			auditlog.LogVerifyCodeCheck(ctx.Context(), userID, ctx.IP(), true, "")

			// Verify user ID matches
			if verifyResp.UserID != userID {
				log.Warn().Str("expected", userID).Str("got", verifyResp.UserID).Msg("User ID mismatch")
				return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.verify_failed"))
			}

			// Store AMR (Authentication Method Reference) from Herald response for later use
			if len(verifyResp.AMR) > 0 {
				verifyRespAMR = verifyResp.AMR
			}
		} else if otpEnabled && useOTP {
			// If OTP is enabled and user chose to use OTP (TOTP / Authenticator)
			if otpCode == "" {
				return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.otp_code_required"))
			}

			// Prefer herald-totp (per-user TOTP) when configured
			totpClient := getHeraldTOTPClient()
			if totpClient != nil {
				// Check if user has TOTP enrolled; if not, require verification code login first, then bind in settings
				statusResp, err := totpClient.Status(loginCtx, userID)
				if err != nil {
					log.Warn().Err(err).Str("user_id", userID).Msg("herald-totp status check failed")
					return SendErrorResponse(ctx, fiber.StatusBadGateway, i18n.T(ctx, "error.herald_unavailable_retry"))
				}
				if statusResp == nil || !statusResp.TotpEnabled {
					metrics.RecordAuthRequest("warden_otp", "failure")
					auditlog.LogLogin(ctx.Context(), userID, "warden_otp", ctx.IP(), false, "totp_not_enrolled")
					return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.totp_not_enrolled"))
				}
				verifyReq := &heraldtotp.VerifyRequest{
					Subject: userID,
					Code:    otpCode,
				}
				if challengeID != "" {
					verifyReq.ChallengeID = challengeID
				}
				verifyResp, err := totpClient.Verify(loginCtx, verifyReq)
				if err != nil || verifyResp == nil || !verifyResp.OK {
					metrics.RecordAuthRequest("warden_otp", "failure")
					log.Warn().Err(err).Str("phone", secure.MaskPhone(userPhone)).Str("mail", secure.MaskEmail(userMail)).Msg("TOTP verification failed")
					auditlog.LogLogin(ctx.Context(), userID, "warden_otp", ctx.IP(), false, "otp_verification_failed")
					return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.otp_code_invalid"))
				}
				metrics.RecordAuthRequest("warden_otp", "success")
			} else {
				// Fallback: legacy global OTP secret (WARDEN_OTP_SECRET_KEY)
				otpSecret := auth.GetOTPSecret()
				if otpSecret == "" {
					log.Warn().Msg("OTP secret is not configured")
					return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.otp_config_error"))
				}
				if !auth.VerifyOTP(otpSecret, otpCode) {
					metrics.RecordAuthRequest("warden_otp", "failure")
					log.Warn().Str("phone", secure.MaskPhone(userPhone)).Str("mail", secure.MaskEmail(userMail)).Msg("OTP verification failed")
					auditlog.LogLogin(ctx.Context(), userID, "warden_otp", ctx.IP(), false, "otp_verification_failed")
					return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.otp_code_invalid"))
				}
				metrics.RecordAuthRequest("warden_otp", "success")
			}
		} else {
			// Neither Herald verification nor OTP was used
			// This should not happen if frontend validation works correctly
			// But we add this check as a safety measure
			if !otpEnabled {
				return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.provide_verify_code_or_otp"))
			}
			return SendErrorResponse(ctx, fiber.StatusBadRequest, i18n.T(ctx, "error.choose_verify_method"))
		}

		log.Info().Str("phone", secure.MaskPhone(userPhone)).Str("mail", secure.MaskEmail(userMail)).Msg("Warden authentication successful")
		authenticated = true
	} else {
		// Password authentication (default)
		if password == "" {
			metrics.RecordAuthRequest("password", "failure")
			auditlog.LogLogin(ctx.Context(), "", "password", ctx.IP(), false, "empty_password")
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.invalid_password"))
		}
		if !auth.CheckPassword(password) {
			metrics.RecordAuthRequest("password", "failure")
			auditlog.LogLogin(ctx.Context(), "", "password", ctx.IP(), false, "invalid_password")
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.invalid_password"))
		}
		authenticated = true
	}

	if !authenticated {
		return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T(ctx, "error.authentication_failed"))
	}

	sess, err := sessionGetter.Get(ctx)
	if err != nil {
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
	}

	// Set user information to session for warden authentication before authenticating
	if authMethod == "warden" {
		// Reuse wardenUserInfo from earlier in the request to avoid repeated GetUserInfo
		if wardenUserInfo != nil {
			// Store complete user information in session
			if wardenUserInfo.UserID != "" {
				sess.Set("user_id", wardenUserInfo.UserID)
			}
			if wardenUserInfo.Phone != "" {
				sess.Set("user_phone", wardenUserInfo.Phone)
			}
			if wardenUserInfo.Mail != "" {
				sess.Set("user_mail", wardenUserInfo.Mail)
			}
			if wardenUserInfo.Status != "" {
				sess.Set("user_status", wardenUserInfo.Status)
			}
			// Store scope and role for authorization headers
			if len(wardenUserInfo.Scope) > 0 {
				sess.Set("user_scope", wardenUserInfo.Scope)
			}
			if wardenUserInfo.Role != "" {
				sess.Set("user_role", wardenUserInfo.Role)
			}
			log.Debug().
				Str("user_id", wardenUserInfo.UserID).
				Str("phone", secure.MaskPhone(wardenUserInfo.Phone)).
				Str("mail", secure.MaskEmail(wardenUserInfo.Mail)).
				Strs("scope", wardenUserInfo.Scope).
				Str("role", wardenUserInfo.Role).
				Msg("Stored user info in session")
		} else {
			// Fallback: store basic info if wardenUserInfo was not set (should not happen after successful warden auth)
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
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.authenticate_failed"))
	}

	// Log successful login and session creation
	var loggedUserID string
	if authMethod == "warden" {
		if wardenUserInfo != nil && wardenUserInfo.UserID != "" {
			loggedUserID = wardenUserInfo.UserID
		} else {
			loggedUserID = userID
		}
		metrics.RecordAuthRequest(authMethod, "success")
		auditlog.LogLogin(ctx.Context(), loggedUserID, authMethod, ctx.IP(), true, "")
	} else {
		metrics.RecordAuthRequest("password", "success")
		auditlog.LogLogin(ctx.Context(), "", "password", ctx.IP(), true, "")
	}
	metrics.RecordSessionCreated()
	auditlog.LogSessionCreate(ctx.Context(), loggedUserID, ctx.IP())

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
					return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
				}
				sessionID = sess.ID()
			}
			if sessionID == "" {
				// If session ID is still empty, return error
				return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.missing_session_id"))
			}
		}
		proto := GetForwardedProto(ctx)
		if proto == "" {
			proto = ctx.Protocol()
		}
		redirectURL := fmt.Sprintf("%s://%s/_session_exchange?id=%s", proto, callback, sessionID)
		// When client accepts JSON (e.g. fetch with Accept: application/json), return 200 + redirect URL
		// so the client can navigate; with redirect: 'manual', 302 Location is opaque and unreadable.
		if strings.Contains(ctx.Get("Accept"), "application/json") {
			ctx.Set("Content-Type", "application/json")
			return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
				"success":  true,
				"redirect": redirectURL,
				"message":  i18n.T(ctx, "success.login"),
			})
		}
		return ctx.Redirect(redirectURL)
	}

	// If still no callback (origin domain is the auth service itself), return response based on request type
	if IsHTMLRequest(ctx) {
		// HTML request returns success message and adds meta refresh redirect to origin domain
		ctx.Set("Content-Type", "text/html; charset=utf-8")
		successMsg := i18n.T(ctx, "success.login")

		// Get origin host and protocol
		originHost := GetForwardedHost(ctx)
		proto := GetForwardedProto(ctx)
		redirectURL := fmt.Sprintf("%s://%s", proto, originHost)

		// Escape URL to ensure HTML safety
		escapedURL := html.EscapeString(redirectURL)

		// Build HTML with meta refresh
		clickIfNoRedirect := i18n.T(ctx, "info.click_if_no_redirect")
		htmlContent := fmt.Sprintf(`<html><head><meta charset="UTF-8"><meta http-equiv="refresh" content="0;url=%s"><title>%s</title></head><body><h1>%s</h1><p>%s</p><p><a href="%s">%s</a></p></body></html>`,
			escapedURL, successMsg, successMsg, successMsg, escapedURL, html.EscapeString(clickIfNoRedirect))
		return ctx.Status(fiber.StatusOK).SendString(htmlContent)
	}

	// API request returns JSON response
	ctx.Set("Content-Type", "application/json")
	response := fiber.Map{
		"success": true,
		"message": i18n.T(ctx, "success.login"),
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
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.session_store_failed"))
	}

	if auth.IsAuthenticated(sess) {
		// Use X-Forwarded-* headers to build correct redirect URL
		sessionID := sess.ID()
		if sessionID == "" {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.missing_session_id"))
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
	otpEnabled := config.WardenOTPEnabled.ToBool() ||
		(config.HeraldTOTPEnabled.ToBool() && config.HeraldTOTPBaseURL.String() != "")

	return ctx.Render(templateName, fiber.Map{
		"Callback":          callback,
		"SessionID":         sess.ID(),
		"Title":             config.LoginPageTitle.Value,
		"FooterText":        config.LoginPageFooterText.Value,
		"WardenEnabled":     config.WardenEnabled.ToBool(),
		"HeraldEnabled":     heraldEnabled,
		"OTPEnabled":        otpEnabled,
		"HeraldTOTPEnabled": config.HeraldTOTPEnabled.ToBool() && config.HeraldTOTPBaseURL.String() != "",
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
