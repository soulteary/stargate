package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/template/html/v3"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

const (
	reauthTestPassword      = "test123"
	reauthTestDeliveredCode = "654321"
)

func setConfigValue(t *testing.T, variable *config.EnvVariable, value string) {
	t.Helper()
	previous := variable.Value
	t.Cleanup(func() { variable.Value = previous })
	variable.Value = value
}

type fakeHeraldChallenge struct {
	userID, channel, destination, purpose string
	consumed                              bool
}

// fakeHerald implements the Herald endpoints the re-authentication flows use.
// It enforces what Herald enforces: TOTP codes are per subject, and a delivered
// code is single-use and bound to user, purpose and channel.
type fakeHerald struct {
	mu          sync.Mutex
	totpEnabled map[string]bool
	totpCodes   map[string]string
	statusFails bool
	challenges  map[string]*fakeHeraldChallenge
	calls       map[string]int
}

func newFakeHerald(t *testing.T) *fakeHerald {
	t.Helper()
	fake := &fakeHerald{
		totpEnabled: map[string]bool{},
		totpCodes:   map[string]string{},
		challenges:  map[string]*fakeHeraldChallenge{},
		calls:       map[string]int{},
	}
	server := httptest.NewServer(http.HandlerFunc(fake.serve))
	t.Cleanup(server.Close)
	previous := heraldClient
	t.Cleanup(func() { heraldClient = previous })
	client, err := herald.NewClient(herald.DefaultOptions().WithBaseURL(server.URL))
	require.NoError(t, err)
	heraldClient = client
	return fake
}

func (f *fakeHerald) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[r.URL.Path]++
	var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
	w.Header().Set("Content-Type", "application/json")
	reply := func(status int, payload any) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(payload)
	}
	switch r.URL.Path {
	case "/v1/totp/status":
		if f.statusFails {
			reply(http.StatusBadGateway, map[string]any{"error": "unavailable"})
			return
		}
		reply(http.StatusOK, map[string]any{"subject": r.URL.Query().Get("subject"), "totp_enabled": f.totpEnabled[r.URL.Query().Get("subject")]})
	case "/v1/totp/verify":
		code, enrolled := f.totpCodes[body["subject"]]
		reply(http.StatusOK, map[string]any{"ok": enrolled && body["code"] == code})
	case "/v1/totp/enroll/start":
		reply(http.StatusOK, map[string]any{"enroll_id": "enroll-" + body["subject"], "otpauth_uri": "otpauth://totp/test?secret=TEST"})
	case "/v1/totp/enroll/confirm":
		subject := body["enroll_id"][len("enroll-"):]
		reply(http.StatusOK, map[string]any{"subject": subject, "totp_enabled": true, "backup_codes": []string{"backup-code-1"}})
	case "/v1/totp/revoke":
		reply(http.StatusOK, map[string]any{"ok": true, "subject": body["subject"]})
	case "/v1/otp/challenges":
		id := fmt.Sprintf("challenge-%d", len(f.challenges)+1)
		f.challenges[id] = &fakeHeraldChallenge{userID: body["user_id"], channel: body["channel"], destination: body["destination"], purpose: body["purpose"]}
		reply(http.StatusOK, map[string]any{"challenge_id": id, "expires_in": 300})
	case "/v2/otp/verifications":
		challenge := f.challenges[body["challenge_id"]]
		if challenge == nil || challenge.consumed || body["code"] != reauthTestDeliveredCode ||
			body["expected_user_id"] != challenge.userID || body["expected_purpose"] != challenge.purpose || body["expected_channel"] != challenge.channel {
			reply(http.StatusUnauthorized, map[string]any{"ok": false, "reason": "invalid"})
			return
		}
		challenge.consumed = true
		reply(http.StatusOK, map[string]any{"ok": true, "user_id": challenge.userID, "purpose": challenge.purpose, "channel": challenge.channel, "challenge_id": body["challenge_id"]})
	default:
		reply(http.StatusNotFound, map[string]any{"error": "not found"})
	}
}

func (f *fakeHerald) count(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[path]
}

func (f *fakeHerald) challenge(id string) fakeHeraldChallenge {
	f.mu.Lock()
	defer f.mu.Unlock()
	return *f.challenges[id]
}

// setupReauthDeployment configures a Warden deployment with Herald OTP and
// TOTP, a shared password, and two directory users.
func setupReauthDeployment(t *testing.T) *fakeHerald {
	t.Helper()
	setupRefreshTest(t)
	setConfigValue(t, &config.AuthRefreshEnabled, "false")
	setConfigValue(t, &config.HeraldEnabled, "true")
	setConfigValue(t, &config.HeraldTOTPEnabled, "true")
	setConfigValue(t, &config.Passwords, "plaintext:"+reauthTestPassword)
	setConfigValue(t, &config.LoginSMSEnabled, "true")
	setConfigValue(t, &config.LoginEmailEnabled, "true")
	setConfigValue(t, &config.LoginDingtalkRequireUserID, "false")
	setConfigValue(t, &config.WardenSourceMaxAge, "")
	SetChallengeContextStore(nil)
	t.Cleanup(func() { SetChallengeContextStore(nil) })
	directory := map[string]*warden.AllowListUser{
		"alice@example.com": {UserID: "alice", Mail: "alice@example.com", Status: "active"},
		"bob@example.com":   {UserID: "bob", Mail: "bob@example.com", Status: "active"},
	}
	lookupRefreshUser = func(_ context.Context, _, mail string) (*warden.AllowListUser, time.Time, error) {
		if user, ok := directory[mail]; ok {
			return user, time.Time{}, nil
		}
		return nil, time.Time{}, auth.ErrUserUnavailable
	}
	return newFakeHerald(t)
}

// newReauthSession stores an authenticated session and returns its id.
func newReauthSession(t *testing.T, store *session.Store, values map[string]any) string {
	t.Helper()
	seed, app := createTestContext("GET", "/", nil, "")
	defer app.ReleaseCtx(seed)
	sess, err := store.Get(seed)
	require.NoError(t, err)
	defer sess.Release()
	for key, value := range values {
		sess.Set(key, value)
	}
	require.NoError(t, auth.Authenticate(sess))
	return sess.ID()
}

func wardenReauthSession(t *testing.T, store *session.Store, user string) string {
	t.Helper()
	return newReauthSession(t, store, map[string]any{"auth_method": "warden", "user_id": user, "user_mail": user + "@example.com"})
}

func passwordReauthSession(t *testing.T, store *session.Store) string {
	t.Helper()
	return newReauthSession(t, store, map[string]any{"auth_method": "password"})
}

// reauthRequest runs handler as the next request of the given session.
func reauthRequest(t *testing.T, sessionID, method, path, body string, handler func(fiber.Ctx) error) fiber.Ctx {
	t.Helper()
	app := fiber.New(fiber.Config{Views: html.New("../web/templates", ".html")})
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	t.Cleanup(func() { app.ReleaseCtx(ctx) })
	ctx.Request().SetRequestURI(path)
	ctx.Request().Header.SetMethod(method)
	ctx.Request().Header.SetContentType("application/x-www-form-urlencoded")
	ctx.Request().SetBodyString(body)
	ctx.Request().Header.SetCookie(auth.SessionCookieName, sessionID)
	ctx.Locals("i18n-bundle", i18n.GetBundle())
	ctx.Locals("i18n-language", i18n.LangEN)
	require.NoError(t, handler(ctx))
	return ctx
}

func reauthSessionValue(t *testing.T, store *session.Store, sessionID, key string) any {
	t.Helper()
	sess, err := store.GetByID(context.Background(), sessionID)
	require.NoError(t, err)
	defer sess.Release()
	return sess.Get(key)
}

func setReauthSessionValue(t *testing.T, store *session.Store, sessionID, key string, value any) {
	t.Helper()
	sess, err := store.GetByID(context.Background(), sessionID)
	require.NoError(t, err)
	defer sess.Release()
	sess.Set(key, value)
	require.NoError(t, sess.Save())
}

// sendReauthCode requests a delivered code for the session and returns the
// challenge id the gateway bound to it.
func sendReauthCode(t *testing.T, store *session.Store, sessionID, body string) string {
	t.Helper()
	ctx := reauthRequest(t, sessionID, "POST", "/_send_reauth_code", body, ReauthSendCodeAPI(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode(), string(ctx.Response().Body()))
	challengeID, _ := reauthSessionValue(t, store, sessionID, reauthChallengeIDKey).(string)
	require.NotEmpty(t, challengeID)
	return challengeID
}

func TestReauthProofSelection(t *testing.T) {
	fake := setupReauthDeployment(t)
	fake.totpEnabled["alice"] = true
	store := setupTestStore()
	proofOf := func(sessionID string) (string, error) {
		sess, err := store.GetByID(context.Background(), sessionID)
		require.NoError(t, err)
		defer sess.Release()
		return reauthProofFor(context.Background(), sess)
	}

	proof, err := proofOf(wardenReauthSession(t, store, "alice"))
	require.NoError(t, err)
	require.Equal(t, reauthProofTOTP, proof)

	proof, err = proofOf(wardenReauthSession(t, store, "bob"))
	require.NoError(t, err)
	require.Equal(t, reauthProofCode, proof)

	proof, err = proofOf(passwordReauthSession(t, store))
	require.NoError(t, err)
	require.Equal(t, reauthProofPassword, proof)

	// HERALD_TOTP_ENABLED=false: the TOTP service is not consulted.
	config.HeraldTOTPEnabled.Value = "false"
	statusCalls := fake.count("/v1/totp/status")
	proof, err = proofOf(wardenReauthSession(t, store, "alice"))
	require.NoError(t, err)
	require.Equal(t, reauthProofCode, proof)
	require.Equal(t, statusCalls, fake.count("/v1/totp/status"))
	config.HeraldTOTPEnabled.Value = "true"

	// An unknown TOTP state must not downgrade to a delivered code.
	fake.statusFails = true
	_, err = proofOf(wardenReauthSession(t, store, "alice"))
	require.ErrorIs(t, err, errReauthProviderUnavailable)
	fake.statusFails = false

	// A user session without Herald has no per-user proof and gets none.
	heraldClient = nil
	proof, err = proofOf(wardenReauthSession(t, store, "bob"))
	require.NoError(t, err)
	require.Equal(t, reauthProofNone, proof)

	config.Passwords.Value = ""
	proof, err = proofOf(passwordReauthSession(t, store))
	require.NoError(t, err)
	require.Equal(t, reauthProofNone, proof)
}

// The shared password must not pass step-up for a session that belongs to a
// Warden user, whether or not that user has TOTP enrolled.
func TestStepUpRejectsSharedPasswordForWardenSession(t *testing.T) {
	fake := setupReauthDeployment(t)
	fake.totpEnabled["alice"], fake.totpCodes["alice"] = true, "111111"
	store := setupTestStore()
	for _, user := range []string{"alice", "bob"} {
		for _, body := range []string{"password=" + reauthTestPassword, "password=" + reauthTestPassword + "&code=" + reauthTestPassword} {
			sessionID := wardenReauthSession(t, store, user)
			ctx := reauthRequest(t, sessionID, "POST", "/_step_up", body+"&callback=/admin", StepUpAPI(store))
			require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode(), user+" "+body)
			require.Nil(t, reauthSessionValue(t, store, sessionID, "step_up_verified_at"), user+" "+body)
			require.Nil(t, reauthSessionValue(t, store, sessionID, reauthVerifiedAtKey), user+" "+body)
		}
	}
}

func TestStepUpAcceptsTOTPCodeOfSessionUser(t *testing.T) {
	fake := setupReauthDeployment(t)
	fake.totpEnabled["alice"], fake.totpCodes["alice"] = true, "111111"
	fake.totpEnabled["bob"], fake.totpCodes["bob"] = true, "222222"
	store := setupTestStore()
	alice := wardenReauthSession(t, store, "alice")

	// Bob's valid code is not a proof for Alice's session.
	ctx := reauthRequest(t, alice, "POST", "/_step_up", "code=222222&callback=/admin", StepUpAPI(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	require.Nil(t, reauthSessionValue(t, store, alice, "step_up_verified_at"))

	ctx = reauthRequest(t, alice, "POST", "/_step_up", "code=111111&callback=/admin", StepUpAPI(store))
	require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode())
	require.NotNil(t, reauthSessionValue(t, store, alice, "step_up_verified_at"))
	require.Nil(t, reauthSessionValue(t, store, alice, reauthVerifiedAtKey))
	require.Zero(t, fake.count("/v1/otp/challenges"))
}

func TestStepUpAcceptsDeliveredCodeOnce(t *testing.T) {
	fake := setupReauthDeployment(t)
	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")

	// Without a challenge requested by this session a code is never accepted.
	ctx := reauthRequest(t, bob, "POST", "/_step_up", "code="+reauthTestDeliveredCode, StepUpAPI(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())

	challengeID := sendReauthCode(t, store, bob, "deliver_via=email")
	ctx = reauthRequest(t, bob, "POST", "/_step_up", "code=000000", StepUpAPI(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())

	ctx = reauthRequest(t, bob, "POST", "/_step_up", "code="+reauthTestDeliveredCode+"&callback=/admin", StepUpAPI(store))
	require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode())
	require.NotNil(t, reauthSessionValue(t, store, bob, "step_up_verified_at"))
	require.True(t, fake.challenge(challengeID).consumed)

	// Replaying the consumed code fails, also when the session is made to
	// point at the consumed challenge again.
	setReauthSessionValue(t, store, bob, "step_up_verified_at", int64(0))
	for _, rebind := range []bool{false, true} {
		if rebind {
			setReauthSessionValue(t, store, bob, reauthChallengeIDKey, challengeID)
			require.NoError(t, getChallengeContextStore().Delete(context.Background(), challengeID))
			_, err := getChallengeContextStore().PutIfAbsent(context.Background(), challengeContext{ChallengeID: challengeID, UserID: "bob", Channel: "email", Destination: "bob@example.com", Purpose: reauthChallengePurpose}, time.Minute)
			require.NoError(t, err)
		}
		ctx = reauthRequest(t, bob, "POST", "/_step_up", "code="+reauthTestDeliveredCode, StepUpAPI(store))
		require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
		require.Equal(t, int64(0), reauthSessionValue(t, store, bob, "step_up_verified_at"))
	}
}

func TestStepUpPasswordOnlyDeploymentAcceptsSharedPassword(t *testing.T) {
	setupReauthDeployment(t)
	config.WardenEnabled.Value, config.HeraldEnabled.Value, config.HeraldTOTPEnabled.Value = "false", "false", "false"
	heraldClient = nil
	store := setupTestStore()
	sessionID := passwordReauthSession(t, store)

	ctx := reauthRequest(t, sessionID, "POST", "/_step_up", "password=wrong", StepUpAPI(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())

	ctx = reauthRequest(t, sessionID, "POST", "/_step_up", "password="+reauthTestPassword+"&callback=/admin", StepUpAPI(store))
	require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode())
	require.NotNil(t, reauthSessionValue(t, store, sessionID, "step_up_verified_at"))
	require.Nil(t, reauthSessionValue(t, store, sessionID, reauthVerifiedAtKey))
}

// A code is sent to the contact Warden holds for the session's user. Nothing
// the client submits can redirect it.
func TestReauthCodeGoesToWardenRegisteredDestination(t *testing.T) {
	fake := setupReauthDeployment(t)
	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")

	challengeID := sendReauthCode(t, store, bob, "deliver_via=email&mail=attacker@evil.example&phone=13800138000&destination=attacker@evil.example&user_id=alice")
	sent := fake.challenge(challengeID)
	require.Equal(t, "bob", sent.userID)
	require.Equal(t, "email", sent.channel)
	require.Equal(t, "bob@example.com", sent.destination)
	require.Equal(t, reauthHeraldPurpose, sent.purpose)
	binding, found, err := getChallengeContextStore().Get(context.Background(), challengeID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, reauthChallengePurpose, binding.Purpose)
	require.Equal(t, "bob@example.com", binding.Destination)
}

func TestReauthCodeIsOnlySentWhenItIsTheSessionProof(t *testing.T) {
	fake := setupReauthDeployment(t)
	fake.totpEnabled["alice"] = true
	store := setupTestStore()

	ctx := reauthRequest(t, "", "POST", "/_send_reauth_code", "", ReauthSendCodeAPI(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	for name, sessionID := range map[string]string{"TOTP user": wardenReauthSession(t, store, "alice"), "password session": passwordReauthSession(t, store)} {
		ctx = reauthRequest(t, sessionID, "POST", "/_send_reauth_code", "deliver_via=email", ReauthSendCodeAPI(store))
		require.Equal(t, fiber.StatusBadRequest, ctx.Response().StatusCode(), name)
	}
	require.Zero(t, fake.count("/v1/otp/challenges"))
}

// A delivered code proves the user it was sent to, in the session that asked
// for it. It is not a proof for another user's session, nor for a second
// session of the same user.
func TestReauthCodeIsBoundToUserAndSession(t *testing.T) {
	fake := setupReauthDeployment(t)
	store := setupTestStore()
	alice := wardenReauthSession(t, store, "alice")
	bob := wardenReauthSession(t, store, "bob")
	aliceOther := wardenReauthSession(t, store, "alice")
	challengeID := sendReauthCode(t, store, alice, "deliver_via=email")

	ctx := reauthRequest(t, bob, "POST", "/_step_up", "code="+reauthTestDeliveredCode, StepUpAPI(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	ctx = reauthRequest(t, aliceOther, "POST", "/_step_up", "code="+reauthTestDeliveredCode, StepUpAPI(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())

	// Even a session that names Alice's challenge as its own is refused.
	setReauthSessionValue(t, store, bob, reauthChallengeIDKey, challengeID)
	ctx = reauthRequest(t, bob, "POST", "/_step_up", "code="+reauthTestDeliveredCode, StepUpAPI(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	require.Nil(t, reauthSessionValue(t, store, bob, "step_up_verified_at"))
	require.False(t, fake.challenge(challengeID).consumed)
	require.Zero(t, fake.count("/v2/otp/verifications"))

	ctx = reauthRequest(t, alice, "POST", "/_step_up", "code="+reauthTestDeliveredCode, StepUpAPI(store))
	require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode())
}

// A login challenge and a re-authentication challenge are not interchangeable.
func TestLoginAndReauthChallengesAreNotInterchangeable(t *testing.T) {
	fake := setupReauthDeployment(t)
	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")
	challengeID := sendReauthCode(t, store, bob, "deliver_via=email")

	binding, err := loadChallengeContext(context.Background(), challengeID)
	require.NoError(t, err)
	login, app := createTestContext("POST", "/_login", map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "challenge_id="+challengeID+"&verify_code="+reauthTestDeliveredCode)
	defer app.ReleaseCtx(login)
	identity := &loginIdentity{userID: "bob", wardenUserInfo: &warden.AllowListUser{UserID: "bob", Mail: "bob@example.com", Status: "active"}}
	ok, err := identity.verifyChallenge(login, context.Background(), binding, true)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, fiber.StatusUnauthorized, login.Response().StatusCode())
	require.Zero(t, fake.count("/v2/otp/verifications"))

	// A login challenge context placed in the session is not accepted either.
	require.NoError(t, getChallengeContextStore().Delete(context.Background(), challengeID))
	_, err = getChallengeContextStore().PutIfAbsent(context.Background(), challengeContext{ChallengeID: challengeID, UserID: "bob", Channel: "email", Destination: "bob@example.com"}, time.Minute)
	require.NoError(t, err)
	ctx := reauthRequest(t, bob, "POST", "/_step_up", "code="+reauthTestDeliveredCode, StepUpAPI(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	require.Zero(t, fake.count("/v2/otp/verifications"))
}

// A session that is minutes old is not enough to bind a new authenticator.
// Enrollment needs a proof marker that is at most five minutes old.
func TestTOTPEnrollRequiresFreshProofDespiteRecentLogin(t *testing.T) {
	fake := setupReauthDeployment(t)
	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")
	createdAt, ok := reauthSessionValue(t, store, bob, "created_at").(int64)
	require.True(t, ok)
	require.WithinDuration(t, time.Now(), time.Unix(createdAt, 0), time.Minute)
	setReauthSessionValue(t, store, bob, totpEnrollmentIDKey, "enroll-bob")
	setReauthSessionValue(t, store, bob, totpEnrollmentSubjectKey, "bob")
	setReauthSessionValue(t, store, bob, totpEnrollmentStartedKey, time.Now().Unix())

	for _, marker := range []any{nil, time.Now().Add(-6 * time.Minute).Unix(), time.Now().Add(time.Minute).Unix()} {
		if marker != nil {
			setReauthSessionValue(t, store, bob, reauthVerifiedAtKey, marker)
		}
		ctx := reauthRequest(t, bob, "POST", "/totp/enroll", "", TOTPEnrollRoute(store))
		require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
		ctx = reauthRequest(t, bob, "POST", "/totp/enroll/confirm", "enroll_id=enroll-bob&code=123456", TOTPEnrollConfirmAPI(store))
		require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
		require.Contains(t, string(ctx.Response().Body()), "recent_authentication_required")
	}
	require.Zero(t, fake.count("/v1/totp/enroll/start"))
	require.Zero(t, fake.count("/v1/totp/enroll/confirm"))

	// A record that is recent enough covers the confirmation of the enrollment
	// it was written with. It does not start another enrollment.
	setReauthSessionValue(t, store, bob, reauthVerifiedAtKey, time.Now().Add(-4*time.Minute).Unix())
	ctx := reauthRequest(t, bob, "POST", "/totp/enroll", "", TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	require.Zero(t, fake.count("/v1/totp/enroll/start"))
	ctx = reauthRequest(t, bob, "POST", "/totp/enroll/confirm", "enroll_id=enroll-bob&code=123456", TOTPEnrollConfirmAPI(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode())
	require.Contains(t, string(ctx.Response().Body()), "backup-code-1")
}

func TestTOTPEnrollStartVerifiesDeliveredCode(t *testing.T) {
	fake := setupReauthDeployment(t)
	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")

	// The shared password is not a proof for a Warden user.
	for _, body := range []string{"password=" + reauthTestPassword, "password=" + reauthTestPassword + "&code=" + reauthTestPassword} {
		ctx := reauthRequest(t, bob, "POST", "/totp/enroll", body, TOTPEnrollRoute(store))
		require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode(), body)
	}
	require.Zero(t, fake.count("/v1/totp/enroll/start"))

	sendReauthCode(t, store, bob, "deliver_via=email")
	ctx := reauthRequest(t, bob, "POST", "/totp/enroll", "code="+reauthTestDeliveredCode, TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode())
	require.Contains(t, string(ctx.Response().Body()), "enroll-bob")
	require.NotNil(t, reauthSessionValue(t, store, bob, reauthVerifiedAtKey))

	ctx = reauthRequest(t, bob, "POST", "/totp/enroll/confirm", "enroll_id=enroll-bob&code=123456", TOTPEnrollConfirmAPI(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode())
	require.Contains(t, string(ctx.Response().Body()), "backup-code-1")
}

// With TOTP already enrolled, starting enrollment still ends at the revoke
// page, but only after a current TOTP code.
func TestTOTPEnrollStartWithEnrolledTOTPRequiresCodeThenRedirects(t *testing.T) {
	fake := setupReauthDeployment(t)
	fake.totpEnabled["alice"], fake.totpCodes["alice"] = true, "111111"
	store := setupTestStore()
	alice := wardenReauthSession(t, store, "alice")

	for _, body := range []string{"", "password=" + reauthTestPassword, "password=" + reauthTestPassword + "&code=" + reauthTestPassword, "code=999999"} {
		ctx := reauthRequest(t, alice, "POST", "/totp/enroll", body, TOTPEnrollRoute(store))
		require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode(), body)
	}
	ctx := reauthRequest(t, alice, "POST", "/totp/enroll", "code=111111", TOTPEnrollRoute(store))
	require.Equal(t, fiber.StatusFound, ctx.Response().StatusCode())
	require.Equal(t, "/totp/revoke", string(ctx.Response().Header.Peek("Location")))
	require.Zero(t, fake.count("/v1/totp/enroll/start"))
}

// The shared password must not remove a Warden user's TOTP.
func TestTOTPRevokeRejectsSharedPasswordForWardenSession(t *testing.T) {
	fake := setupReauthDeployment(t)
	fake.totpEnabled["alice"], fake.totpCodes["alice"] = true, "111111"
	fake.totpCodes["bob"] = "222222"
	store := setupTestStore()
	alice := wardenReauthSession(t, store, "alice")

	for _, body := range []string{"password=" + reauthTestPassword, "password=" + reauthTestPassword + "&code=" + reauthTestPassword, "code=222222"} {
		ctx := reauthRequest(t, alice, "POST", "/totp/revoke", body, TOTPRevokeConfirmAPI(store))
		require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode(), body)
		require.Contains(t, string(ctx.Response().Body()), "recent_authentication_required")
	}
	require.Zero(t, fake.count("/v1/totp/revoke"))

	ctx := reauthRequest(t, alice, "POST", "/totp/revoke", "code=111111", TOTPRevokeConfirmAPI(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode(), string(ctx.Response().Body()))
	require.Equal(t, 1, fake.count("/v1/totp/revoke"))
}

// A user whose TOTP state is "not enrolled" proves a revoke request with a
// delivered code; the shared password is refused here as well.
func TestTOTPRevokeWithoutEnrolledTOTPUsesDeliveredCode(t *testing.T) {
	fake := setupReauthDeployment(t)
	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")

	ctx := reauthRequest(t, bob, "POST", "/totp/revoke", "password="+reauthTestPassword, TOTPRevokeConfirmAPI(store))
	require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
	require.Zero(t, fake.count("/v1/totp/revoke"))

	sendReauthCode(t, store, bob, "deliver_via=email")
	ctx = reauthRequest(t, bob, "POST", "/totp/revoke", "code="+reauthTestDeliveredCode, TOTPRevokeConfirmAPI(store))
	require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode(), string(ctx.Response().Body()))
	require.Equal(t, 1, fake.count("/v1/totp/revoke"))
}

// A user session that has no per-user proof available is refused, and the
// shared password is not offered as a substitute.
func TestReauthRefusedWhenNoPerUserProofIsAvailable(t *testing.T) {
	setupReauthDeployment(t)
	store := setupTestStore()
	bob := wardenReauthSession(t, store, "bob")
	heraldClient = nil

	ctx := reauthRequest(t, bob, "POST", "/_step_up", "password="+reauthTestPassword, StepUpAPI(store))
	require.Equal(t, fiber.StatusForbidden, ctx.Response().StatusCode())
	require.Nil(t, reauthSessionValue(t, store, bob, "step_up_verified_at"))
	ctx = reauthRequest(t, bob, "GET", "/_step_up", "", StepUpRoute(store))
	require.Equal(t, fiber.StatusForbidden, ctx.Response().StatusCode())
}

func TestReauthPagesRenderTheApplicableProof(t *testing.T) {
	fake := setupReauthDeployment(t)
	fake.totpEnabled["alice"] = true
	store := setupTestStore()
	const totpField, codeField, passwordField = `maxlength="19"`, `id="reauthSendBtn"`, `name="password"`
	pages := []struct {
		path    string
		handler func(fiber.Ctx) error
	}{
		{"/_step_up?callback=/admin", StepUpRoute(store)},
		{"/totp/enroll", TOTPEnrollPageRoute(store)},
		{"/totp/revoke", TOTPRevokeRoute(store)},
	}
	for _, page := range pages {
		for _, tc := range []struct {
			name, sessionID, want string
		}{
			{"TOTP user", wardenReauthSession(t, store, "alice"), totpField},
			{"OTP user", wardenReauthSession(t, store, "bob"), codeField},
			{"password session", passwordReauthSession(t, store), passwordField},
		} {
			if tc.want == passwordField && page.path == "/totp/revoke" {
				continue // a shared-password session has no user and no TOTP to revoke
			}
			ctx := reauthRequest(t, tc.sessionID, "GET", page.path, "", page.handler)
			body := string(ctx.Response().Body())
			require.Equal(t, fiber.StatusOK, ctx.Response().StatusCode(), page.path+" "+tc.name)
			for _, field := range []string{totpField, codeField, passwordField} {
				require.Equal(t, field == tc.want, strings.Contains(body, field), page.path+" "+tc.name+" "+field)
			}
		}
	}

	// The enrollment page asks for the proof also when the session holds the
	// record of a recent one.
	bob := wardenReauthSession(t, store, "bob")
	setReauthSessionValue(t, store, bob, reauthVerifiedAtKey, time.Now().Unix())
	body := string(reauthRequest(t, bob, "GET", "/totp/enroll", "", TOTPEnrollPageRoute(store)).Response().Body())
	require.Contains(t, body, `action="/totp/enroll"`)
	require.Contains(t, body, codeField)
}
