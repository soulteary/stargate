package handlers

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

// After the delivered code was accepted, the enrollment start is the only
// request to Herald: a failure of any request after that point costs the user
// the code, so there must be no second one.
func TestTOTPEnrollStartMakesOneHeraldCallAfterProof(t *testing.T) {
	fake := setupReauthDeployment(t)
	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")
	sendReauthCode(t, store, bob, "deliver_via=email")
	before := len(fake.requests())

	ctx := reauthRequest(t, bob, "POST", "/totp/enroll", "code="+reauthTestDeliveredCode, TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode())
	require.Contains(t, string(ctx.Response().Body()), "enroll-bob")
	require.Equal(t,
		[]string{"/v1/totp/status", "/v2/otp/verifications", "/v1/totp/enroll/start"},
		fake.requests()[before:])
}

// A user who has TOTP proves with a TOTP code and is sent to the revoke page.
// The status that selected the proof decides that; Herald is not asked again.
func TestTOTPEnrollWithEnrolledTOTPMakesNoHeraldCallAfterProof(t *testing.T) {
	fake := setupReauthDeployment(t)
	fake.totpEnabled["alice"], fake.totpCodes["alice"] = true, "111111"
	store := setupTestStore()
	alice := wardenReauthSession(t, store, "alice")

	// Without a valid TOTP code the request does not reach the redirect.
	ctx := reauthRequest(t, alice, "POST", "/totp/enroll", "code=999999", TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	require.Empty(t, string(ctx.Response().Header.Peek("Location")))
	before := len(fake.requests())

	ctx = reauthRequest(t, alice, "POST", "/totp/enroll", "code=111111", TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode())
	require.Equal(t, "/totp/revoke", string(ctx.Response().Header.Peek("Location")))
	require.Equal(t, []string{"/v1/totp/status", "/v1/totp/verify"}, fake.requests()[before:])
}

// When Herald cannot report the TOTP status, the request fails before the
// delivered code is verified. The code is not used up and starts the
// enrollment once Herald answers again.
func TestTOTPEnrollStatusFailureBeforeProofKeepsDeliveredCode(t *testing.T) {
	fake := setupReauthDeployment(t)
	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")
	challengeID := sendReauthCode(t, store, bob, "deliver_via=email")

	fake.mu.Lock()
	fake.statusFails = true
	fake.mu.Unlock()
	ctx := reauthRequest(t, bob, "POST", "/totp/enroll", "code="+reauthTestDeliveredCode, TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusBadGateway, ctx.Response().StatusCode())
	require.Zero(t, fake.count("/v2/otp/verifications"))
	require.Zero(t, fake.count("/v1/totp/enroll/start"))
	require.False(t, fake.challenge(challengeID).consumed)
	require.Equal(t, challengeID, reauthSessionValue(t, store, bob, reauthChallengeIDKey))

	fake.mu.Lock()
	fake.statusFails = false
	fake.mu.Unlock()
	ctx = reauthRequest(t, bob, "POST", "/totp/enroll", "code="+reauthTestDeliveredCode, TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode())
	require.Contains(t, string(ctx.Response().Body()), "enroll-bob")
	require.True(t, fake.challenge(challengeID).consumed)
}

// With HERALD_TOTP_ENABLED=false the proof is chosen without the TOTP status,
// so the start request still asks for it after the proof, as before.
func TestTOTPEnrollChecksStatusAfterProofWhenProofWasChosenWithoutIt(t *testing.T) {
	fake := setupReauthDeployment(t)
	setConfigValue(t, &config.HeraldTOTPEnabled, "false")
	fake.totpEnabled["alice"] = true
	store := setupTestStore()

	bob := wardenReauthSession(t, store, "bob")
	sendReauthCode(t, store, bob, "deliver_via=email")
	before := len(fake.requests())
	ctx := reauthRequest(t, bob, "POST", "/totp/enroll", "code="+reauthTestDeliveredCode, TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode())
	require.Equal(t,
		[]string{"/v2/otp/verifications", "/v1/totp/status", "/v1/totp/enroll/start"},
		fake.requests()[before:])

	alice := wardenReauthSession(t, store, "alice")
	sendReauthCode(t, store, alice, "deliver_via=email")
	ctx = reauthRequest(t, alice, "POST", "/totp/enroll", "code="+reauthTestDeliveredCode, TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode())
	require.Equal(t, "/totp/revoke", string(ctx.Response().Header.Peek("Location")))
	require.Equal(t, 1, fake.count("/v1/totp/enroll/start"))
}
