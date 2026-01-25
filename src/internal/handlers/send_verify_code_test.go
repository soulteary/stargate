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
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

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
	err := config.Initialize()
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
	err := config.Initialize()
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
	testza.AssertContains(t, string(ctx.Response().Body()), "验证码服务未配置")
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
	err := config.Initialize()
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
	err := config.Initialize()
	testza.AssertNoError(t, err)

	resetHeraldClientForTesting()
	auth.ResetWardenClientForTesting()

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
