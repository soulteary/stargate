package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/redis/go-redis/v9"
	"github.com/soulteary/herald/pkg/herald"
	i18nkit "github.com/soulteary/i18n-kit/v4"
	i18nfiber "github.com/soulteary/i18n-kit/v4/fiberadapter"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

// uniformTestCode is the code the fake Herald accepts for a live challenge.
const uniformTestCode = "123456"

// uniformWarden is a fake Warden directory served over HTTP. Like Warden it
// answers 404 for an unlisted identifier, returns inactive records with their
// status, and refuses identifiers above its length limit with 400.
type uniformWarden struct {
	mu      sync.Mutex
	users   []warden.AllowListUser
	down    bool
	calls   int
	lookups []uniformWardenLookup
}

// uniformWardenLookup is the identifier of one lookup the directory received.
type uniformWardenLookup struct{ phone, mail string }

// lookupsSince returns the lookups received after the first count ones.
func (w *uniformWarden) lookupsSince(count int) []uniformWardenLookup {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]uniformWardenLookup(nil), w.lookups[count:]...)
}

func (w *uniformWarden) add(users ...warden.AllowListUser) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.users = append(w.users, users...)
}

// replace swaps the record of one user, as a directory update would.
func (w *uniformWarden) replace(user warden.AllowListUser) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for index := range w.users {
		if w.users[index].UserID == user.UserID {
			w.users[index] = user
		}
	}
}

func (w *uniformWarden) setDown(down bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.down = down
}

func (w *uniformWarden) callCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.calls
}

func (w *uniformWarden) serve(rw http.ResponseWriter, r *http.Request) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls++
	phone, mail := r.URL.Query().Get("phone"), r.URL.Query().Get("mail")
	w.lookups = append(w.lookups, uniformWardenLookup{phone: phone, mail: mail})
	if w.down {
		rw.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if len(phone) > 512 || len(mail) > 512 {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	for _, user := range w.users {
		if (phone != "" && phone == user.Phone) || (mail != "" && strings.EqualFold(mail, user.Mail)) {
			rw.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(rw).Encode(user)
			return
		}
	}
	rw.WriteHeader(http.StatusNotFound)
}

type uniformHeraldChallenge struct {
	userID, channel, destination string
	active                       bool
}

type uniformHeraldCall struct {
	path, userID, challengeID, idempotencyKey string
}

// uniformHerald is a fake Herald served over HTTP. A challenge is bound to
// user, purpose and channel, is redeemed once, and is revoked when a newer one
// is created for the same user, channel and destination. Per-user modes make
// it answer the way Herald does for a cooldown, a locked user or an outage.
// The mode "resend-refused" creates the first challenge and answers the
// cooldown while that challenge can still be redeemed.
type uniformHerald struct {
	mu          sync.Mutex
	createMode  map[string]string // "", "cooldown", "resend-refused", "locked", "error", "unreachable"
	verifyMode  map[string]string // "", "expired", "locked", "error", "unreachable"
	totpMode    map[string]string // "", "error", "unreachable"
	totpEnabled map[string]bool
	challenges  map[string]*uniformHeraldChallenge
	calls       []uniformHeraldCall
}

func newUniformHerald() *uniformHerald {
	return &uniformHerald{
		createMode:  map[string]string{},
		verifyMode:  map[string]string{},
		totpMode:    map[string]string{},
		totpEnabled: map[string]bool{},
		challenges:  map[string]*uniformHeraldChallenge{},
	}
}

func (h *uniformHerald) set(modes map[string]string, userID, mode string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	modes[userID] = mode
}

// clearChallenges forgets every challenge, as if all had expired.
func (h *uniformHerald) clearChallenges() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.challenges = map[string]*uniformHeraldChallenge{}
}

func (h *uniformHerald) enrollTOTP(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.totpEnabled[userID] = true
}

// callsTo returns the recorded calls to path, for one user when userID is set.
func (h *uniformHerald) callsTo(path, userID string) []uniformHeraldCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	var matched []uniformHeraldCall
	for _, call := range h.calls {
		if call.path == path && (userID == "" || call.userID == userID) {
			matched = append(matched, call)
		}
	}
	return matched
}

func (h *uniformHerald) total() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.calls)
}

// activeChallenge returns the id of the user's challenge that can be redeemed.
func (h *uniformHerald) activeChallenge(userID string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, challenge := range h.challenges {
		if challenge.userID == userID && challenge.active {
			return id
		}
	}
	return ""
}

func (h *uniformHerald) serve(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
	call := uniformHeraldCall{path: r.URL.Path, idempotencyKey: r.Header.Get("Idempotency-Key")}
	reply := func(status int, payload map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(payload)
	}
	// An abruptly closed connection is what an unreachable Herald looks like
	// to the client.
	drop := func() {
		if hijacker, ok := w.(http.Hijacker); ok {
			if conn, _, err := hijacker.Hijack(); err == nil {
				_ = conn.Close()
			}
		}
	}
	switch r.URL.Path {
	case "/v1/otp/challenges":
		call.userID = body["user_id"]
		h.calls = append(h.calls, call)
		mode := h.createMode[call.userID]
		if mode == "resend-refused" {
			mode = ""
			for _, existing := range h.challenges {
				if existing.active && existing.userID == body["user_id"] && existing.channel == body["channel"] && existing.destination == body["destination"] {
					mode = "cooldown"
				}
			}
		}
		switch mode {
		case "cooldown":
			w.Header().Set("Retry-After", "57")
			reply(http.StatusTooManyRequests, map[string]any{"ok": false, "reason": "resend_cooldown", "next_resend_in": 57})
		case "locked":
			reply(http.StatusForbidden, map[string]any{"ok": false, "reason": "user_locked"})
		case "error":
			reply(http.StatusInternalServerError, map[string]any{"ok": false, "reason": "internal_error"})
		case "unreachable":
			drop()
		default:
			for _, existing := range h.challenges {
				if existing.userID == body["user_id"] && existing.channel == body["channel"] && existing.destination == body["destination"] {
					existing.active = false
				}
			}
			raw := make([]byte, 16)
			_, _ = rand.Read(raw)
			id := "ch_" + base64.RawURLEncoding.EncodeToString(raw)
			h.challenges[id] = &uniformHeraldChallenge{userID: body["user_id"], channel: body["channel"], destination: body["destination"], active: true}
			reply(http.StatusOK, map[string]any{"challenge_id": id, "expires_in": 300, "next_resend_in": 60, "debug_code": uniformTestCode})
		}
	case "/v2/otp/verifications":
		call.userID, call.challengeID = body["expected_user_id"], body["challenge_id"]
		h.calls = append(h.calls, call)
		switch h.verifyMode[call.userID] {
		case "expired":
			reply(http.StatusUnauthorized, map[string]any{"ok": false, "reason": "expired"})
		case "locked":
			reply(http.StatusUnauthorized, map[string]any{"ok": false, "reason": "locked", "remaining_attempts": 0})
		case "error":
			reply(http.StatusInternalServerError, map[string]any{"ok": false, "reason": "internal_error"})
		case "unreachable":
			drop()
		default:
			challenge := h.challenges[body["challenge_id"]]
			switch {
			case challenge == nil || !challenge.active:
				reply(http.StatusUnauthorized, map[string]any{"ok": false, "reason": "expired"})
			case body["code"] != uniformTestCode || body["expected_user_id"] != challenge.userID ||
				body["expected_purpose"] != "login" || body["expected_channel"] != challenge.channel:
				reply(http.StatusUnauthorized, map[string]any{"ok": false, "reason": "invalid", "remaining_attempts": 4})
			default:
				challenge.active = false
				reply(http.StatusOK, map[string]any{"ok": true, "user_id": challenge.userID, "purpose": "login", "channel": challenge.channel, "challenge_id": body["challenge_id"]})
			}
		}
	case "/v1/totp/status":
		call.userID = r.URL.Query().Get("subject")
		h.calls = append(h.calls, call)
		switch h.totpMode[call.userID] {
		case "error":
			reply(http.StatusBadGateway, map[string]any{"ok": false, "reason": "proxy_failed"})
		case "unreachable":
			drop()
		default:
			reply(http.StatusOK, map[string]any{"subject": call.userID, "totp_enabled": h.totpEnabled[call.userID]})
		}
	case "/v1/totp/verify":
		call.userID, call.challengeID = body["subject"], body["challenge_id"]
		h.calls = append(h.calls, call)
		reply(http.StatusOK, map[string]any{"ok": h.totpEnabled[call.userID] && body["code"] == uniformTestCode})
	default:
		h.calls = append(h.calls, call)
		reply(http.StatusNotFound, map[string]any{"ok": false, "reason": "not_found"})
	}
}

// uniformContextStore wraps the challenge context store to inject the two
// failures a Redis outage produces.
type uniformContextStore struct {
	ChallengeContextStore
	mu           sync.Mutex
	failWriteFor string // user id whose contexts cannot be stored
	failReads    bool
}

func (s *uniformContextStore) PutIfAbsent(ctx context.Context, value challengeContext, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	refused := s.failWriteFor != "" && value.UserID == s.failWriteFor
	s.mu.Unlock()
	if refused {
		return false, errors.New("injected context store write failure")
	}
	return s.ChallengeContextStore.PutIfAbsent(ctx, value, ttl)
}

func (s *uniformContextStore) Get(ctx context.Context, challengeID string) (challengeContext, bool, error) {
	s.mu.Lock()
	refused := s.failReads
	s.mu.Unlock()
	if refused {
		return challengeContext{}, false, errors.New("injected context store read failure")
	}
	return s.ChallengeContextStore.Get(ctx, challengeID)
}

func (s *uniformContextStore) setFailReads(fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failReads = fail
}

func (s *uniformContextStore) setFailWriteFor(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failWriteFor = userID
}

// uniformDeployment is the production route chain for the two public
// endpoints (same-origin check, per-client rate limit, handler) in front of
// fake Warden and Herald servers. Rate limits and challenge contexts are kept
// in the Redis-backed stores on miniredis, so tests can inspect the keys that
// are written. Tests move time with advance, which keeps the stores and the
// clock for challenge expiry in step.
type uniformDeployment struct {
	t        *testing.T
	app      *fiber.App
	redis    *miniredis.Miniredis
	warden   *uniformWarden
	herald   *uniformHerald
	contexts *uniformContextStore
	sessions *session.Store

	clockMu sync.Mutex
	clock   time.Time
}

// now is the time the handlers see for challenge expiry.
func (d *uniformDeployment) now() time.Time {
	d.clockMu.Lock()
	defer d.clockMu.Unlock()
	return d.clock
}

// advance moves the stores and the expiry clock forward together.
func (d *uniformDeployment) advance(duration time.Duration) {
	d.redis.FastForward(duration)
	d.clockMu.Lock()
	defer d.clockMu.Unlock()
	d.clock = d.clock.Add(duration)
}

// context returns the challenge context stored under a handle.
func (d *uniformDeployment) context(handle string) (challengeContext, bool) {
	d.t.Helper()
	binding, found, err := d.contexts.Get(context.Background(), handle)
	require.NoError(d.t, err)
	return binding, found
}

// contextTTL returns the remaining lifetime of the context stored under a handle.
func (d *uniformDeployment) contextTTL(handle string) time.Duration {
	return d.redis.TTL("stargate:session:challenge:context:" + handle)
}

func newUniformDeployment(t *testing.T) *uniformDeployment {
	t.Helper()
	d := &uniformDeployment{t: t, warden: &uniformWarden{}, herald: newUniformHerald(), sessions: setupTestStore(), clock: time.Unix(1800000000, 0)}
	previousNow := uniformNow
	uniformNow = d.now
	t.Cleanup(func() { uniformNow = previousNow })

	wardenServer := httptest.NewServer(http.HandlerFunc(d.warden.serve))
	t.Cleanup(wardenServer.Close)
	heraldServer := httptest.NewServer(http.HandlerFunc(d.herald.serve))
	t.Cleanup(heraldServer.Close)

	for variable, value := range map[*config.EnvVariable]string{
		&config.LoginUniformResponses:      "true",
		&config.LoginUniformResendCooldown: "60s",
		&config.WardenEnabled:              "true",
		&config.WardenURL:                  wardenServer.URL,
		&config.WardenSourceMaxAge:         "",
		&config.HeraldEnabled:              "true",
		&config.HeraldTOTPEnabled:          "true",
		&config.LoginSMSEnabled:            "true",
		&config.LoginEmailEnabled:          "true",
		&config.LoginDingtalkRequireUserID: "false",
		&config.RateLimitLoginMax:          "1000",
		&config.RateLimitVerificationMax:   "1000",
		&config.RateLimitWindow:            "1m",
		&config.Debug:                      "false",
	} {
		setConfigValue(t, variable, value)
	}

	auth.ResetWardenClientForTesting()
	require.NoError(t, auth.InitWardenClient(testLogger()))
	t.Cleanup(auth.ResetWardenClientForTesting)

	previousHerald := heraldClient
	t.Cleanup(func() { heraldClient = previousHerald })
	client, err := herald.NewClient(herald.DefaultOptions().WithBaseURL(heraldServer.URL))
	require.NoError(t, err)
	heraldClient = client

	d.redis = miniredis.RunT(t)
	// No command or dial retries: a test that stops Redis should fail fast.
	redisClient := redis.NewClient(&redis.Options{Addr: d.redis.Addr(), MaxRetries: -1, DialerRetries: 1, DialerRetryTimeout: time.Millisecond})
	t.Cleanup(func() { _ = redisClient.Close() })
	d.contexts = &uniformContextStore{ChallengeContextStore: NewChallengeContextStore(redisClient)}
	SetRateLimitStore(NewRateLimitStore(redisClient))
	SetChallengeContextStore(d.contexts)
	t.Cleanup(func() {
		SetRateLimitStore(nil)
		SetChallengeContextStore(nil)
	})

	resetUniformDigestKeyForTesting()
	t.Cleanup(resetUniformDigestKeyForTesting)

	d.app = fiber.New()
	d.app.Use(i18nfiber.Middleware(i18nfiber.Config{MiddlewareConfig: i18nkit.MiddlewareConfig{
		Bundle:   i18n.GetBundle(),
		Detector: i18nkit.NewDetector(i18nkit.DetectorConfig{Default: i18n.LangEN}),
	}}))
	sameOrigin := RequireSameOrigin()
	d.app.Post("/_login", sameOrigin, LoginRateLimit(), LoginAPI(d.sessions))
	d.app.Post("/_send_verify_code", sameOrigin, VerificationRateLimit(), SendVerifyCodeAPI())
	return d
}

// uniformReply is everything a client can observe in one response, except the
// Date header.
type uniformReply struct {
	status  int
	headers string
	body    string
}

func (d *uniformDeployment) post(path, form string, headers map[string]string) uniformReply {
	d.t.Helper()
	request := httptest.NewRequest(http.MethodPost, "http://auth.example.com"+path, strings.NewReader(form))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := d.app.Test(request, fiber.TestConfig{Timeout: 10 * time.Second, FailOnTimeout: true})
	require.NoError(d.t, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(d.t, err)
	require.NoError(d.t, response.Body.Close())
	lines := make([]string, 0, len(response.Header))
	for name, values := range response.Header {
		if name == "Date" {
			continue
		}
		lines = append(lines, name+": "+strings.Join(values, ", "))
	}
	sort.Strings(lines)
	return uniformReply{status: response.StatusCode, headers: strings.Join(lines, "\n"), body: string(body)}
}

var acceptJSON = map[string]string{"Accept": "application/json"}

func (d *uniformDeployment) send(form string) uniformReply {
	d.t.Helper()
	return d.post("/_send_verify_code", form, acceptJSON)
}

func (d *uniformDeployment) login(form string) uniformReply {
	d.t.Helper()
	return d.post("/_login", "auth_method=warden&"+form, acceptJSON)
}

// sendHandle sends a code and returns the handle of a 200 reply.
func (d *uniformDeployment) sendHandle(form string) string {
	d.t.Helper()
	reply := d.send(form)
	require.Equal(d.t, fiber.StatusOK, reply.status, reply.body)
	_, handle := maskHandle(d.t, reply)
	require.NotEmpty(d.t, handle)
	return handle
}

// reset clears the rate-limit and context stores and the challenges Herald
// holds, so that the next requests start from the same state as the first ones
// of the test.
func (d *uniformDeployment) reset() {
	d.redis.FlushAll()
	d.herald.clearChallenges()
}

var uniformHandlePattern = regexp.MustCompile(`^ch_[A-Za-z0-9_-]{22}$`)

const maskedHandle = "ch_0000000000000000000000"

// maskHandle replaces the challenge_id of a send reply, the one value that
// differs between two otherwise identical replies, after checking its shape.
// The replacement has the same length, so Content-Length stays comparable.
func maskHandle(t *testing.T, reply uniformReply) (uniformReply, string) {
	t.Helper()
	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(reply.body), &decoded), reply.body)
	handle, _ := decoded["challenge_id"].(string)
	if handle == "" {
		return reply, ""
	}
	require.Regexp(t, uniformHandlePattern, handle)
	require.Equal(t, 1, strings.Count(reply.body, handle), reply.body)
	reply.body = strings.Replace(reply.body, handle, maskedHandle, 1)
	return reply, handle
}

func jsonFields(t *testing.T, body string) map[string]any {
	t.Helper()
	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &decoded), body)
	return decoded
}
