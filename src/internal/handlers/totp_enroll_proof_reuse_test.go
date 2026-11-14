package handlers

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/stretchr/testify/require"
)

// A session cookie copied after its owner passed step-up must not be enough to
// bind an authenticator: the step-up proof is not a proof for enrollment.
func TestStepUpProofDoesNotAuthorizeTOTPEnrollment(t *testing.T) {
	fake := setupReauthDeployment(t)
	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")

	sendReauthCode(t, store, bob, "deliver_via=email")
	ctx := reauthRequest(t, bob, "POST", "/_step_up", "code="+reauthTestDeliveredCode+"&callback=/admin", StepUpAPI(store))
	require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode())
	require.NotNil(t, reauthSessionValue(t, store, bob, "step_up_verified_at"))

	// The holder of the copied cookie has no code to present.
	ctx = reauthRequest(t, bob, "POST", "/totp/enroll", "", TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	require.NotContains(t, string(ctx.Response().Body()), "otpauth://")
	require.Zero(t, fake.count("/v1/totp/enroll/start"))
	require.Nil(t, reauthSessionValue(t, store, bob, totpEnrollmentIDKey))
	page := reauthRequest(t, bob, "GET", "/totp/enroll", "", TOTPEnrollPageRoute(store))
	require.Contains(t, string(page.Response().Body()), `id="reauthSendBtn"`)
}

// The proof that removes TOTP does not start the enrollment of another
// authenticator in the same session.
func TestTOTPRevokeProofDoesNotAuthorizeEnrollment(t *testing.T) {
	fake := setupReauthDeployment(t)
	fake.totpEnabled["alice"], fake.totpCodes["alice"] = true, "111111"
	store := setupTestStore()
	alice := wardenReauthSession(t, store, "alice")

	ctx := reauthRequest(t, alice, "POST", "/totp/revoke", "code=111111", TOTPRevokeConfirmAPI(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode())
	fake.mu.Lock()
	fake.totpEnabled["alice"] = false
	fake.mu.Unlock()

	ctx = reauthRequest(t, alice, "POST", "/totp/enroll", "", TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	require.Zero(t, fake.count("/v1/totp/enroll/start"))
}

// A proof given to the enrollment form while TOTP is already enrolled ends at
// the revoke page and is not kept for an enrollment after the removal.
func TestTOTPEnrollProofIsDiscardedWhenTOTPIsAlreadyEnrolled(t *testing.T) {
	fake := setupReauthDeployment(t)
	fake.totpEnabled["alice"], fake.totpCodes["alice"] = true, "111111"
	store := setupTestStore()
	alice := wardenReauthSession(t, store, "alice")

	ctx := reauthRequest(t, alice, "POST", "/totp/enroll", "code=111111", TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode())
	require.Equal(t, "/totp/revoke", string(ctx.Response().Header.Peek("Location")))
	require.Nil(t, reauthSessionValue(t, store, alice, reauthVerifiedAtKey))

	fake.mu.Lock()
	fake.totpEnabled["alice"] = false
	fake.mu.Unlock()
	ctx = reauthRequest(t, alice, "POST", "/totp/enroll", "", TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	require.Zero(t, fake.count("/v1/totp/enroll/start"))
}

// The owner's flow: prove identity with the start request, then confirm. The
// proof starts one enrollment; a second start needs a new proof, and the
// enrollment already started can still be confirmed.
func TestTOTPEnrollProofStartsOneEnrollment(t *testing.T) {
	fake := setupReauthDeployment(t)
	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")

	sendReauthCode(t, store, bob, "deliver_via=email")
	ctx := reauthRequest(t, bob, "POST", "/totp/enroll", "code="+reauthTestDeliveredCode, TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode())
	require.Contains(t, string(ctx.Response().Body()), "enroll-bob")
	require.Equal(t, 1, fake.count("/v1/totp/enroll/start"))

	// The page asks for a proof again and a start without one is refused.
	page := reauthRequest(t, bob, "GET", "/totp/enroll", "", TOTPEnrollPageRoute(store))
	require.Contains(t, string(page.Response().Body()), `id="reauthSendBtn"`)
	for _, body := range []string{"", "code=" + reauthTestDeliveredCode} {
		ctx = reauthRequest(t, bob, "POST", "/totp/enroll", body, TOTPEnrollRoute(store))
		require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode(), body)
	}
	require.Equal(t, 1, fake.count("/v1/totp/enroll/start"))

	ctx = reauthRequest(t, bob, "POST", "/totp/enroll/confirm", "enroll_id=enroll-bob&code=123456", TOTPEnrollConfirmAPI(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode())
	require.Contains(t, string(ctx.Response().Body()), "backup-code-1")
	require.Nil(t, reauthSessionValue(t, store, bob, reauthVerifiedAtKey))
	require.Nil(t, reauthSessionValue(t, store, bob, reauthUsedAtKey))

	// A new proof starts a new enrollment.
	sendReauthCode(t, store, bob, "deliver_via=email")
	ctx = reauthRequest(t, bob, "POST", "/totp/enroll", "code="+reauthTestDeliveredCode, TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode())
	require.Equal(t, 2, fake.count("/v1/totp/enroll/start"))
}

// A delivered code is consumed when it is verified. When Herald then fails to
// start the enrollment, the user retries without a new code, once.
func TestTOTPEnrollProofSurvivesFailedStartOnce(t *testing.T) {
	fake := setupReauthDeployment(t)
	var failStart atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/totp/enroll/start" && failStart.Load() {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		fake.serve(w, r)
	}))
	t.Cleanup(server.Close)
	client, err := herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
	require.NoError(t, err)
	heraldClient = client // restored by the cleanup newFakeHerald registered

	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")
	sendReauthCode(t, store, bob, "deliver_via=email")
	failStart.Store(true)
	ctx := reauthRequest(t, bob, "POST", "/totp/enroll", "code="+reauthTestDeliveredCode, TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusBadGateway, ctx.Response().StatusCode())
	require.Nil(t, reauthSessionValue(t, store, bob, totpEnrollmentIDKey))

	failStart.Store(false)
	page := reauthRequest(t, bob, "GET", "/totp/enroll", "", TOTPEnrollPageRoute(store))
	require.NotContains(t, string(page.Response().Body()), `id="reauthSendBtn"`)
	ctx = reauthRequest(t, bob, "POST", "/totp/enroll", "", TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode())
	require.Contains(t, string(ctx.Response().Body()), "enroll-bob")

	ctx = reauthRequest(t, bob, "POST", "/totp/enroll", "", TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	require.Equal(t, 1, fake.count("/v1/totp/enroll/start"))
}

// The record that releases before this change wrote for any proof is not an
// enrollment proof.
func TestTOTPEnrollIgnoresProofRecordOfEarlierReleases(t *testing.T) {
	fake := setupReauthDeployment(t)
	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")
	setReauthSessionValue(t, store, bob, "reauth_verified_at", time.Now().Unix())
	setReauthSessionValue(t, store, bob, totpEnrollmentIDKey, "enroll-bob")
	setReauthSessionValue(t, store, bob, totpEnrollmentSubjectKey, "bob")
	setReauthSessionValue(t, store, bob, totpEnrollmentStartedKey, time.Now().Unix())

	ctx := reauthRequest(t, bob, "POST", "/totp/enroll", "", TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	ctx = reauthRequest(t, bob, "POST", "/totp/enroll/confirm", "enroll_id=enroll-bob&code=123456", TOTPEnrollConfirmAPI(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	require.Contains(t, string(ctx.Response().Body()), "recent_authentication_required")
	require.Zero(t, fake.count("/v1/totp/enroll/start"))
	require.Zero(t, fake.count("/v1/totp/enroll/confirm"))
}
