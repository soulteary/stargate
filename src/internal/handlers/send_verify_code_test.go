package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v2"
	"github.com/soulteary/herald/pkg/herald"
	logger "github.com/soulteary/logger-kit"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// testLoggerSendVerifyCode creates a logger instance for testing
func testLoggerSendVerifyCode() *logger.Logger {
	return logger.New(logger.Config{
		Level:       logger.DebugLevel,
		Format:      logger.FormatJSON,
		ServiceName: "send-verify-code-test",
	})
}

func resetHeraldClientForTesting() {
	heraldClient = nil
	heraldClientInit = sync.Once{}
}

func setupSendVerifyCodeBaseEnv(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
}

func TestSendVerifyCodeAPI_NoIdentifier(t *testing.T) {
	setupSendVerifyCodeBaseEnv(t)
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("WARDEN_ENABLED", "true")
	err := config.Initialize(testLoggerSendVerifyCode())
	testza.AssertNoError(t, err)

	resetHeraldClientForTesting()
	auth.ResetWardenClientForTesting()

	ctx, app := createTestContext("POST", "/_send_verify_code", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Accept":       "application/json",
	}, "")
	defer app.ReleaseCtx(ctx)

	handler := SendVerifyCodeAPI()
	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusBadRequest, ctx.Response().StatusCode())
	testza.AssertContains(t, string(ctx.Response().Body()), i18n.TStatic("error.user_not_in_list"))
}

func TestSendVerifyCodeAPI_HeraldDisabled(t *testing.T) {
	setupSendVerifyCodeBaseEnv(t)
	t.Setenv("HERALD_ENABLED", "false")
	err := config.Initialize(testLoggerSendVerifyCode())
	testza.AssertNoError(t, err)

	resetHeraldClientForTesting()
	auth.ResetWardenClientForTesting()

	ctx, app := createTestContext("POST", "/_send_verify_code", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Accept":       "application/json",
	}, "phone=13800138000")
	defer app.ReleaseCtx(ctx)

	handler := SendVerifyCodeAPI()
	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusBadRequest, ctx.Response().StatusCode())
	testza.AssertContains(t, string(ctx.Response().Body()), i18n.TStatic("error.herald_not_configured"))
}

func TestSendVerifyCodeAPI_WardenUserNotFound(t *testing.T) {
	setupSendVerifyCodeBaseEnv(t)
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("WARDEN_ENABLED", "true")

	wardenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer wardenServer.Close()

	t.Setenv("WARDEN_URL", wardenServer.URL)
	auth.ResetWardenClientForTesting()
	resetHeraldClientForTesting()
	testLog := testLoggerSendVerifyCode()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	auth.InitWardenClient(testLog)
	SetLogger(testLog)

	ctx, app := createTestContext("POST", "/_send_verify_code", map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Accept":       "application/json",
	}, "phone=13800138000")
	defer app.ReleaseCtx(ctx)

	handler := SendVerifyCodeAPI()
	err = handler(ctx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	testza.AssertContains(t, string(ctx.Response().Body()), i18n.TStatic("error.user_not_in_list"))
}

func TestSendVerifyCodeAPI_Success(t *testing.T) {
	setupSendVerifyCodeBaseEnv(t)
	t.Setenv("LANGUAGE", "zh")
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("HERALD_API_KEY", "api-key")
	t.Setenv("WARDEN_ENABLED", "true")

	wardenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		user := struct {
			Phone  string `json:"phone"`
			Mail   string `json:"mail"`
			UserID string `json:"user_id"`
			Status string `json:"status"`
		}{
			Phone:  "13800138000",
			Mail:   "user@example.com",
			UserID: "",
			Status: "active",
		}
		_ = json.NewEncoder(w).Encode(user)
	}))
	defer wardenServer.Close()

	expectedUserID := generateUserID("13800138000", "user@example.com")

	var receivedRequest *herald.CreateChallengeRequest
	heraldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/otp/challenges" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		testza.AssertEqual(t, http.MethodPost, r.Method)
		testza.AssertEqual(t, "api-key", r.Header.Get("X-API-Key"))

		bodyBytes, err := io.ReadAll(r.Body)
		testza.AssertNoError(t, err)

		var req herald.CreateChallengeRequest
		err = json.Unmarshal(bodyBytes, &req)
		testza.AssertNoError(t, err)
		receivedRequest = &req

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(herald.CreateChallengeResponse{
			ChallengeID:  "challenge-1",
			ExpiresIn:    120,
			NextResendIn: 30,
		})
	}))
	defer heraldServer.Close()

	t.Setenv("WARDEN_URL", wardenServer.URL)
	t.Setenv("HERALD_URL", heraldServer.URL)
	auth.ResetWardenClientForTesting()
	resetHeraldClientForTesting()
	testLog := testLoggerSendVerifyCode()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	auth.InitWardenClient(testLog)
	InitHeraldClient(testLog)

	app := fiber.New()
	app.Post("/_send_verify_code", SendVerifyCodeAPI())

	req := httptest.NewRequest("POST", "/_send_verify_code", strings.NewReader("phone=13800138000"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")

	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	testza.AssertNoError(t, err)

	var body struct {
		Success     bool   `json:"success"`
		ChallengeID string `json:"challenge_id"`
		ExpiresIn   int    `json:"expires_in"`
	}
	err = json.Unmarshal(bodyBytes, &body)
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, body.Success)
	testza.AssertEqual(t, "challenge-1", body.ChallengeID)
	testza.AssertEqual(t, 120, body.ExpiresIn)

	testza.AssertNotNil(t, receivedRequest)
	testza.AssertEqual(t, expectedUserID, receivedRequest.UserID)
	testza.AssertEqual(t, "sms", receivedRequest.Channel)
	testza.AssertEqual(t, "13800138000", receivedRequest.Destination)
	testza.AssertEqual(t, "login", receivedRequest.Purpose)
	testza.AssertEqual(t, "fr-FR", receivedRequest.Locale)
}

// TestSendVerifyCodeAPI_IdempotencyKeyPassthrough verifies that Idempotency-Key request header is forwarded to Herald.
func TestSendVerifyCodeAPI_IdempotencyKeyPassthrough(t *testing.T) {
	setupSendVerifyCodeBaseEnv(t)
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("WARDEN_ENABLED", "true")

	wardenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := struct {
			Phone  string `json:"phone"`
			Mail   string `json:"mail"`
			UserID string `json:"user_id"`
			Status string `json:"status"`
		}{
			Phone:  "13800138000",
			Mail:   "user@example.com",
			UserID: "",
			Status: "active",
		}
		_ = json.NewEncoder(w).Encode(user)
	}))
	defer wardenServer.Close()

	var receivedIdempotencyKey string
	heraldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/otp/challenges" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		receivedIdempotencyKey = r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(herald.CreateChallengeResponse{
			ChallengeID:  "ch-idem",
			ExpiresIn:    300,
			NextResendIn: 60,
		})
	}))
	defer heraldServer.Close()

	t.Setenv("WARDEN_URL", wardenServer.URL)
	t.Setenv("HERALD_URL", heraldServer.URL)
	auth.ResetWardenClientForTesting()
	resetHeraldClientForTesting()
	testLog := testLoggerSendVerifyCode()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	auth.InitWardenClient(testLog)
	InitHeraldClient(testLog)

	app := fiber.New()
	app.Post("/_send_verify_code", SendVerifyCodeAPI())

	idemKey := "req-uuid-12345"
	req := httptest.NewRequest("POST", "/_send_verify_code", strings.NewReader("phone=13800138000"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Idempotency-Key", idemKey)

	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
	testza.AssertEqual(t, idemKey, receivedIdempotencyKey, "Idempotency-Key should be forwarded to Herald")
}

// TestSendVerifyCodeAPI_LocaleFromConfig verifies getLocaleFromConfig is used when Accept-Language is not set.
func TestSendVerifyCodeAPI_LocaleFromConfig(t *testing.T) {
	setupSendVerifyCodeBaseEnv(t)
	t.Setenv("LANGUAGE", "de")
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("HERALD_API_KEY", "api-key")
	t.Setenv("WARDEN_ENABLED", "true")

	wardenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		user := struct {
			Phone  string `json:"phone"`
			Mail   string `json:"mail"`
			UserID string `json:"user_id"`
			Status string `json:"status"`
		}{
			Phone:  "13800138000",
			Mail:   "user@example.com",
			UserID: "",
			Status: "active",
		}
		_ = json.NewEncoder(w).Encode(user)
	}))
	defer wardenServer.Close()

	var receivedLocale string
	heraldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/otp/challenges" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		var req herald.CreateChallengeRequest
		_ = json.Unmarshal(bodyBytes, &req)
		receivedLocale = req.Locale

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(herald.CreateChallengeResponse{
			ChallengeID:  "ch-locale",
			ExpiresIn:    120,
			NextResendIn: 30,
		})
	}))
	defer heraldServer.Close()

	t.Setenv("WARDEN_URL", wardenServer.URL)
	t.Setenv("HERALD_URL", heraldServer.URL)
	auth.ResetWardenClientForTesting()
	resetHeraldClientForTesting()
	testLog := testLoggerSendVerifyCode()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	auth.InitWardenClient(testLog)
	InitHeraldClient(testLog)

	app := fiber.New()
	app.Post("/_send_verify_code", SendVerifyCodeAPI())

	// No Accept-Language header so locale comes from config (LANGUAGE=de -> de-DE)
	req := httptest.NewRequest("POST", "/_send_verify_code", strings.NewReader("phone=13800138000"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
	testza.AssertEqual(t, "de-DE", receivedLocale, "locale should come from getLocaleFromConfig when Accept-Language is not set")
}

// TestSendVerifyCodeAPI_GetLocaleFromConfig_AllLanguages covers all getLocaleFromConfig branches:
// zh->zh-CN, en->en-US, fr->fr-FR, it->it-IT, ja->ja-JP, de->de-DE, ko->ko-KR, default->en-US.
func TestSendVerifyCodeAPI_GetLocaleFromConfig_AllLanguages(t *testing.T) {
	// Config validates LANGUAGE to [en zh fr it ja de ko], so we only test those; default branch is unreachable under current config.
	langToLocale := map[string]string{
		"zh": "zh-CN",
		"en": "en-US",
		"fr": "fr-FR",
		"it": "it-IT",
		"ja": "ja-JP",
		"de": "de-DE",
		"ko": "ko-KR",
	}
	for lang, wantLocale := range langToLocale {
		t.Run("LANGUAGE="+lang, func(t *testing.T) {
			setupSendVerifyCodeBaseEnv(t)
			t.Setenv("LANGUAGE", lang)
			t.Setenv("HERALD_ENABLED", "true")
			t.Setenv("HERALD_API_KEY", "api-key")
			t.Setenv("WARDEN_ENABLED", "true")
			t.Setenv("LOGIN_SMS_ENABLED", "true")

			wardenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/user" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				user := struct {
					Phone  string `json:"phone"`
					Mail   string `json:"mail"`
					UserID string `json:"user_id"`
					Status string `json:"status"`
				}{
					Phone:  "13800138000",
					Mail:   "user@example.com",
					UserID: "",
					Status: "active",
				}
				_ = json.NewEncoder(w).Encode(user)
			}))
			defer wardenServer.Close()

			var receivedLocale string
			heraldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/otp/challenges" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				bodyBytes, _ := io.ReadAll(r.Body)
				var req herald.CreateChallengeRequest
				_ = json.Unmarshal(bodyBytes, &req)
				receivedLocale = req.Locale
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(herald.CreateChallengeResponse{
					ChallengeID:  "ch-1",
					ExpiresIn:    120,
					NextResendIn: 30,
				})
			}))
			defer heraldServer.Close()

			t.Setenv("WARDEN_URL", wardenServer.URL)
			t.Setenv("HERALD_URL", heraldServer.URL)
			auth.ResetWardenClientForTesting()
			resetHeraldClientForTesting()
			testLog := testLoggerSendVerifyCode()
			err := config.Initialize(testLog)
			testza.AssertNoError(t, err)
			auth.InitWardenClient(testLog)
			InitHeraldClient(testLog)

			app := fiber.New()
			app.Post("/_send_verify_code", SendVerifyCodeAPI())
			req := httptest.NewRequest("POST", "/_send_verify_code", strings.NewReader("phone=13800138000"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Accept", "application/json")
			// No Accept-Language so locale comes from getLocaleFromConfig()

			resp, err := app.Test(req)
			testza.AssertNoError(t, err)
			testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
			testza.AssertEqual(t, wantLocale, receivedLocale, "getLocaleFromConfig(LANGUAGE=%s) should produce %s", lang, wantLocale)
		})
	}
}

// TestSendVerifyCodeAPI_HeraldRateLimited covers the rate_limited branch when Herald returns 429.
func TestSendVerifyCodeAPI_HeraldRateLimited(t *testing.T) {
	setupSendVerifyCodeBaseEnv(t)
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("WARDEN_ENABLED", "true")
	t.Setenv("LOGIN_SMS_ENABLED", "true")

	wardenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		user := struct {
			Phone  string `json:"phone"`
			Mail   string `json:"mail"`
			UserID string `json:"user_id"`
			Status string `json:"status"`
		}{
			Phone:  "13800138000",
			Mail:   "user@example.com",
			UserID: "",
			Status: "active",
		}
		_ = json.NewEncoder(w).Encode(user)
	}))
	defer wardenServer.Close()

	heraldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/otp/challenges" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"reason":"rate_limited"}`))
	}))
	defer heraldServer.Close()

	t.Setenv("WARDEN_URL", wardenServer.URL)
	t.Setenv("HERALD_URL", heraldServer.URL)
	auth.ResetWardenClientForTesting()
	resetHeraldClientForTesting()
	testLog := testLoggerSendVerifyCode()
	err := config.Initialize(testLog)
	testza.AssertNoError(t, err)
	auth.InitWardenClient(testLog)
	InitHeraldClient(testLog)

	app := fiber.New()
	app.Post("/_send_verify_code", SendVerifyCodeAPI())
	req := httptest.NewRequest("POST", "/_send_verify_code", strings.NewReader("phone=13800138000"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, http.StatusTooManyRequests, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	testza.AssertContains(t, bodyStr, "rate_limited")
	testza.AssertContains(t, bodyStr, "success")
}
