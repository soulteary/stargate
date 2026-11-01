package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/template/html/v3"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func authenticatedTOTPContext(t *testing.T, store *session.Store, path, body string) fiber.Ctx {
	t.Helper()
	app := fiber.New(fiber.Config{Views: html.New("../web/templates", ".html")})
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Request().SetRequestURI(path)
	ctx.Request().Header.SetMethod("POST")
	ctx.Request().Header.SetContentType("application/x-www-form-urlencoded")
	ctx.Request().SetBodyString(body)
	ctx.Locals("i18n-bundle", i18n.GetBundle())
	ctx.Locals("i18n-language", i18n.LangEN)
	sess, err := store.Get(ctx)
	require.NoError(t, err)
	sess.Set("user_id", "user-1")
	sess.Set(totpEnrollmentIDKey, "enroll-1")
	sess.Set(totpEnrollmentSubjectKey, "user-1")
	sess.Set(totpEnrollmentStartedKey, time.Now().Unix())
	require.NoError(t, auth.Authenticate(sess))
	next := rebindSessionContext(t, app, ctx, sess)
	sess.Release()
	t.Cleanup(func() { app.ReleaseCtx(next); app.ReleaseCtx(ctx) })
	return next
}

func TestTOTPEnrollmentProviderFailuresPreserveBinding(t *testing.T) {
	previous := heraldClient
	t.Cleanup(func() { heraldClient = previous })
	for _, tc := range []struct {
		name    string
		status  int
		payload any
		want    int
	}{
		{"provider rejects code", 400, map[string]any{"error": "invalid_code"}, 400},
		{"wrong subject", 200, herald.TOTPEnrollConfirmResponse{Subject: "other-user", TotpEnabled: true}, 502},
		{"not enabled", 200, herald.TOTPEnrollConfirmResponse{Subject: "user-1"}, 502},
		{"confirmed", 200, herald.TOTPEnrollConfirmResponse{Subject: "user-1", TotpEnabled: true, BackupCodes: []string{"test-recovery-code"}}, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(tc.payload)
			}))
			defer server.Close()
			var err error
			heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
			require.NoError(t, err)
			store := setupTestStore()
			ctx := authenticatedTOTPContext(t, store, "/totp/enroll/confirm", "enroll_id=enroll-1&code=123456")
			require.NoError(t, TOTPEnrollConfirmAPI(store)(ctx))
			require.Equal(t, tc.want, ctx.Response().StatusCode())
			if tc.want == 200 {
				require.Contains(t, string(ctx.Response().Body()), "test-recovery-code")
			} else {
				require.NotContains(t, string(ctx.Response().Body()), "test-recovery-code")
			}
		})
	}
}

func TestTOTPEnrollmentStartPersistsProviderEnrollment(t *testing.T) {
	previous := heraldClient
	t.Cleanup(func() { heraldClient = previous })
	for _, fail := range []string{"status", "start", ""} {
		t.Run("failure="+fail, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				status := strings.HasSuffix(r.URL.Path, "/status")
				if (status && fail == "status") || (!status && fail == "start") {
					w.WriteHeader(400)
					_, _ = w.Write([]byte(`{"error":"unavailable"}`))
					return
				}
				if status {
					_, _ = w.Write([]byte(`{"totp_enabled":false}`))
					return
				}
				_, _ = w.Write([]byte(`{"enroll_id":"new-enrollment","otpauth_uri":"otpauth://totp/test?secret=TEST"}`))
			}))
			defer server.Close()
			var err error
			heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
			require.NoError(t, err)
			store := setupTestStore()
			ctx := authenticatedTOTPContext(t, store, "/totp/enroll", "")
			require.NoError(t, TOTPEnrollRoute(store)(ctx))
			if fail != "" {
				require.Equal(t, 502, ctx.Response().StatusCode())
				return
			}
			require.Equal(t, 200, ctx.Response().StatusCode())
			require.Contains(t, string(ctx.Response().Body()), "new-enrollment")
		})
	}
}

func TestTOTPRevokeRequiresFreshCodeAndHandlesProviderFailure(t *testing.T) {
	previous := heraldClient
	t.Cleanup(func() { heraldClient = previous })
	for _, tc := range []struct {
		name           string
		verify, revoke bool
		want           int
	}{
		{"invalid code", false, false, 401}, {"revoke fails", true, false, 502}, {"revoked", true, true, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/verify") {
					_ = json.NewEncoder(w).Encode(map[string]any{"ok": tc.verify})
					return
				}
				calls++
				if !tc.revoke {
					w.WriteHeader(400)
					_, _ = w.Write([]byte(`{"error":"rejected"}`))
					return
				}
				_, _ = w.Write([]byte(`{"ok":true,"subject":"user-1"}`))
			}))
			defer server.Close()
			var err error
			heraldClient, err = herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
			require.NoError(t, err)
			store := setupTestStore()
			ctx := authenticatedTOTPContext(t, store, "/totp/revoke", "code=123456")
			require.NoError(t, TOTPRevokeConfirmAPI(store)(ctx))
			require.Equal(t, tc.want, ctx.Response().StatusCode())
			if tc.verify {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
			}
		})
	}
}
