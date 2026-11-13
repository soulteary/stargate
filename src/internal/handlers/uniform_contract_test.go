package handlers

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

// These tests are the contract of LOGIN_UNIFORM_RESPONSES: for one request
// shape, status, headers and body must not depend on whether the identifier
// is listed, on the state of the listed account, or on what Warden-dependent
// services answer. Only the Date header and the value of challenge_id are
// excluded from the comparison. The rate-limit headers are compared as they
// are: every identity starts from empty stores, so they must be the same
// function of the request index. Response time is outside this contract.

const (
	uniformSendOK       = `{"challenge_id":"` + maskedHandle + `","message":"If this account can receive codes, a code has been sent.","next_resend_in":60,"success":true}`
	uniformLoginFailure = "Sign-in failed. Check the account and the code, or request a new code."
)

func activeUser(id, phone, mail string) warden.AllowListUser {
	return warden.AllowListUser{UserID: id, Phone: phone, Mail: mail, Status: "active"}
}

func TestUniformSendResponsesDoNotDependOnTheIdentifier(t *testing.T) {
	d := newUniformDeployment(t)
	// With SMS disabled a user without a mail has no usable destination, and
	// with the userid requirement a user without a binding has no DingTalk
	// destination.
	setConfigValue(t, &config.LoginSMSEnabled, "false")
	setConfigValue(t, &config.LoginDingtalkRequireUserID, "true")
	d.warden.add(
		activeUser("user-usable", "13800000001", "usable@example.com"),
		warden.AllowListUser{UserID: "user-inactive", Phone: "13800000002", Mail: "inactive@example.com", Status: "inactive"},
		activeUser("user-phone-only", "13800000003", ""),
		activeUser("user-no-dingtalk", "13800000004", "no-dingtalk@example.com"),
		activeUser("user-cooldown", "13800000005", "cooldown@example.com"),
		activeUser("user-locked", "13800000006", "locked@example.com"),
		activeUser("user-error", "13800000007", "error@example.com"),
		activeUser("user-unreachable", "13800000008", "unreachable@example.com"),
		activeUser("user-store-failure", "13800000009", "store-failure@example.com"),
	)
	d.herald.set(d.herald.createMode, "user-cooldown", "cooldown")
	d.herald.set(d.herald.createMode, "user-locked", "locked")
	d.herald.set(d.herald.createMode, "user-error", "error")
	d.herald.set(d.herald.createMode, "user-unreachable", "unreachable")
	d.contexts.setFailWriteFor("user-store-failure")

	cases := []struct {
		name, form, userID string
		// creates is the number of Herald creation requests of an admitted
		// request; -1 accepts one or more.
		creates int
	}{
		{name: "unknown", form: "phone=13900000000&deliver_via=email"},
		{name: "inactive", form: "phone=13800000002&deliver_via=email", userID: "user-inactive"},
		{name: "known usable", form: "phone=13800000001&deliver_via=email", userID: "user-usable", creates: 1},
		{name: "known without a usable destination", form: "phone=13800000003&deliver_via=email", userID: "user-phone-only"},
		{name: "known with DingTalk unbound", form: "phone=13800000004&deliver_via=dingtalk", userID: "user-no-dingtalk"},
		{name: "Herald 429 cooldown", form: "phone=13800000005&deliver_via=email", userID: "user-cooldown", creates: 1},
		{name: "Herald 403 user_locked", form: "phone=13800000006&deliver_via=email", userID: "user-locked", creates: 1},
		{name: "Herald 5xx", form: "phone=13800000007&deliver_via=email", userID: "user-error", creates: 1},
		{name: "Herald unreachable", form: "phone=13800000008&deliver_via=email", userID: "user-unreachable", creates: -1},
		{name: "context store write failure", form: "phone=13800000009&deliver_via=email", userID: "user-store-failure", creates: 1},
	}
	phases := []struct {
		name     string
		advance  time.Duration
		admitted bool
	}{
		{name: "first request", admitted: true},
		{name: "second request inside the window", advance: 10 * time.Second},
		{name: "request after the window", advance: 51 * time.Second, admitted: true},
	}
	headers := map[string]string{"Accept": "application/json", "Idempotency-Key": "client-chosen-key"}

	reference := make([]uniformReply, len(phases))
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d.reset()
			handles := map[string]bool{}
			for phase, step := range phases {
				d.advance(step.advance)
				wardenBefore := d.warden.callCount()
				createsBefore := len(d.herald.callsTo("/v1/otp/challenges", ""))
				heraldBefore := d.herald.total()

				reply, handle := maskHandle(t, d.post("/_send_verify_code", tc.form, headers))
				if index == 0 {
					reference[phase] = reply
				}
				require.Equal(t, reference[phase], reply, step.name)

				creates := len(d.herald.callsTo("/v1/otp/challenges", "")) - createsBefore
				require.Equal(t, creates, d.herald.total()-heraldBefore, "only challenge creation may reach Herald")
				if !step.admitted {
					require.Empty(t, handle, step.name)
					require.Zero(t, d.warden.callCount()-wardenBefore, "a refused request must not reach Warden")
					require.Zero(t, creates, "a refused request must not reach Herald")
					continue
				}
				require.NotEmpty(t, handle, step.name)
				require.False(t, handles[handle], "every admitted request gets a new handle")
				handles[handle] = true
				require.Equal(t, 1, d.warden.callCount()-wardenBefore, step.name)
				if tc.creates < 0 {
					require.GreaterOrEqual(t, creates, 1, step.name)
				} else {
					require.Equal(t, tc.creates, creates, step.name)
				}
			}
			for _, call := range d.herald.callsTo("/v1/otp/challenges", "") {
				require.Empty(t, call.idempotencyKey, "the client's Idempotency-Key must not reach Herald")
				require.NotEmpty(t, call.userID)
			}
			if tc.userID == "" {
				require.Zero(t, d.herald.total(), "an unknown identifier must never reach Herald")
			}
		})
	}

	// The shared shape: 200 with a handle, then 429 for the rest of the
	// window, then 200 again.
	require.Equal(t, fiber.StatusOK, reference[0].status)
	require.Equal(t, uniformSendOK, reference[0].body)
	require.NotContains(t, reference[0].headers, "Retry-After")
	require.Equal(t, fiber.StatusTooManyRequests, reference[1].status)
	require.Equal(t, `{"message":"Too many requests. Please try again later.","next_resend_in":50,"reason":"rate_limited","success":false}`, reference[1].body)
	require.Contains(t, reference[1].headers, "Retry-After: 50")
	require.Equal(t, fiber.StatusOK, reference[2].status)
	require.Equal(t, uniformSendOK, reference[2].body)
	require.Empty(t, d.herald.callsTo("/v1/otp/challenges", "user-inactive"), "an inactive identifier must never reach Herald")
}

// The identifier may be a phone, a mail or both. Each form is its own request
// shape; within one the answer is the same for every combination of listed
// and unlisted identifiers.
func TestUniformSendTreatsEveryIdentifierFormAlike(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(
		activeUser("user-a", "13800000001", "a@example.com"),
		warden.AllowListUser{UserID: "user-inactive", Phone: "13800000002", Mail: "inactive@example.com", Status: "inactive"},
	)
	for shape, forms := range map[string][]string{
		"mail": {"mail=nobody@example.com", "mail=a@example.com", "mail=inactive@example.com", "mail=A@Example.COM"},
		"phone and mail": {
			"phone=13900000000&mail=nobody@example.com",
			"phone=13800000001&mail=a@example.com",
			"phone=13800000001&mail=nobody@example.com",
			"phone=13900000000&mail=a@example.com",
			"phone=13800000002&mail=a@example.com",
		},
	} {
		t.Run(shape, func(t *testing.T) {
			var reference []uniformReply
			for index, form := range forms {
				d.reset()
				first, handle := maskHandle(t, d.send(form))
				require.NotEmpty(t, handle, form)
				second, _ := maskHandle(t, d.send(form))
				replies := []uniformReply{first, second}
				if index == 0 {
					reference = replies
					require.Equal(t, fiber.StatusOK, first.status)
					require.Equal(t, uniformSendOK, first.body)
					require.Equal(t, fiber.StatusTooManyRequests, second.status)
				}
				require.Equal(t, reference, replies, form)
			}
		})
	}
}

func TestUniformLoginResponsesDoNotDependOnTheIdentifier(t *testing.T) {
	d := newUniformDeployment(t)
	phoneOf := map[string]string{}
	for index, name := range []string{"live", "expired", "locked", "verify-error", "verify-unreachable", "totp-enrolled", "totp-error", "totp-unreachable", "no-challenge"} {
		phoneOf["user-"+name] = strconv.Itoa(13800000010 + index)
		d.warden.add(activeUser("user-"+name, phoneOf["user-"+name], name+"@example.com"))
	}
	d.warden.add(warden.AllowListUser{UserID: "user-inactive", Phone: "13800000002", Mail: "inactive@example.com", Status: "inactive"})
	d.herald.set(d.herald.verifyMode, "user-expired", "expired")
	d.herald.set(d.herald.verifyMode, "user-locked", "locked")
	d.herald.set(d.herald.verifyMode, "user-verify-error", "error")
	d.herald.set(d.herald.verifyMode, "user-verify-unreachable", "unreachable")
	d.herald.enrollTOTP("user-totp-enrolled")
	d.herald.set(d.herald.totpMode, "user-totp-error", "error")
	d.herald.set(d.herald.totpMode, "user-totp-unreachable", "unreachable")
	d.herald.set(d.herald.createMode, "user-no-challenge", "error")

	const accepts = 3
	cases := []struct {
		name, phone, userID string
		// Expected Herald requests over the whole sequence. A negative number
		// accepts that many or more: the HTTP client may repeat a request on
		// a connection the server dropped.
		verifications, totpStatus, totpVerifications int
	}{
		{name: "unknown", phone: "13900000000"},
		{name: "inactive", phone: "13800000002", userID: "user-inactive"},
		{name: "known, wrong code", userID: "user-live", verifications: accepts, totpStatus: accepts},
		{name: "known, challenge expired at Herald", userID: "user-expired", verifications: accepts, totpStatus: accepts},
		{name: "known, challenge locked at Herald", userID: "user-locked", verifications: accepts, totpStatus: accepts},
		{name: "known, Herald 5xx on verification", userID: "user-verify-error", verifications: accepts, totpStatus: accepts},
		{name: "known, Herald unreachable on verification", userID: "user-verify-unreachable", verifications: -accepts, totpStatus: accepts},
		{name: "known, TOTP enrolled, wrong code", userID: "user-totp-enrolled", verifications: accepts, totpStatus: accepts, totpVerifications: accepts},
		{name: "known, TOTP status 5xx", userID: "user-totp-error", verifications: accepts, totpStatus: accepts},
		{name: "known, TOTP status unreachable", userID: "user-totp-unreachable", verifications: accepts, totpStatus: -accepts},
		{name: "known, no challenge was created", userID: "user-no-challenge", totpStatus: accepts},
	}
	var reference []uniformReply
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d.reset()
			phone := tc.phone
			if phone == "" {
				phone = phoneOf[tc.userID]
			}
			require.NotEmpty(t, phone)
			heraldBefore := d.herald.total()

			sent, handle := maskHandle(t, d.send("phone="+phone+"&deliver_via=sms"))
			require.NotEmpty(t, handle)
			replies := []uniformReply{sent}
			form := "auth_method=warden&phone=" + phone
			for _, accept := range []string{"application/json", "application/xml", "text/plain"} {
				headers := map[string]string{"Accept": accept}
				replies = append(replies,
					d.post("/_login", form, headers),
					d.post("/_login", form+"&challenge_id=ch_AAAAAAAAAAAAAAAAAAAAAA&verify_code=000000", headers),
					d.post("/_login", form+"&challenge_id="+handle+"&verify_code=000000", headers),
					d.post("/_login", form+"&use_otp=true", headers),
					d.post("/_login", form+"&use_otp=true&otp_code=000000", headers),
				)
			}
			// Past the lifetime of the challenge context and its audit grace
			// the handle is unknown to Stargate.
			d.advance(11 * time.Minute)
			replies = append(replies, d.post("/_login", form+"&challenge_id="+handle+"&verify_code=000000", acceptJSON))

			if index == 0 {
				reference = replies
			}
			require.Equal(t, len(reference), len(replies))
			for step := range replies {
				require.Equal(t, reference[step], replies[step], "step %d", step)
			}

			count := func(path string) int { return len(d.herald.callsTo(path, tc.userID)) }
			expect := func(want, got int, what string) {
				t.Helper()
				if want < 0 {
					require.GreaterOrEqual(t, got, -want, what)
				} else {
					require.Equal(t, want, got, what)
				}
			}
			if tc.userID == "" {
				require.Equal(t, heraldBefore, d.herald.total(), "an unknown identifier must never reach Herald")
				return
			}
			expect(tc.verifications, count("/v2/otp/verifications"), "code verifications")
			expect(tc.totpStatus, count("/v1/totp/status"), "TOTP status requests")
			expect(tc.totpVerifications, count("/v1/totp/verify"), "TOTP verifications")
			for _, call := range d.herald.callsTo("/v2/otp/verifications", tc.userID) {
				require.NotEqual(t, handle, call.challengeID, "Herald is addressed by its own challenge id")
				require.Regexp(t, uniformHandlePattern, call.challengeID)
			}
		})
	}

	// The shared shape.
	require.Equal(t, uniformSendOK, reference[0].body)
	bodies := map[string]string{
		"application/json": `{"code":401,"error":"` + uniformLoginFailure + `","reason":"authentication_failed"}`,
		"application/xml":  `<errors><error code="401">` + uniformLoginFailure + `</error></errors>`,
		"text/plain":       uniformLoginFailure,
	}
	for offset, accept := range []string{"application/json", "application/xml", "text/plain"} {
		steps := reference[1+offset*5 : 6+offset*5]
		require.Equal(t, fiber.StatusBadRequest, steps[0].status, "a request without a code is a shape error")
		require.Equal(t, fiber.StatusBadRequest, steps[3].status, "TOTP without a code is a shape error")
		for _, step := range []int{1, 2, 4} {
			require.Equal(t, fiber.StatusUnauthorized, steps[step].status, "%s step %d", accept, step)
			require.Equal(t, bodies[accept], steps[step].body, "%s step %d", accept, step)
			require.Contains(t, steps[step].headers, "Content-Type: "+accept)
			require.NotContains(t, steps[step].headers, "Set-Cookie")
			require.NotContains(t, steps[step].headers, "Retry-After")
		}
	}
	last := reference[len(reference)-1]
	require.Equal(t, fiber.StatusUnauthorized, last.status)
	require.Equal(t, bodies["application/json"], last.body)
	require.Equal(t, `{"code":400,"error":"Verification code and challenge_id are required.","reason":"verification_required"}`, reference[1].body)
	require.Equal(t, `{"code":400,"error":"OTP code is required."}`, reference[4].body)
}

// Requests that are malformed, or that the configuration cannot serve, keep
// the answers of the default mode. They are given before the directory lookup
// and are therefore the same for a listed and an unlisted identifier.
func TestUniformRequestShapeErrorsAreAnsweredBeforeTheLookup(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	for _, tc := range []struct {
		name      string
		configure func(t *testing.T)
		path      string
		form      string // %s is the phone
		status    int
		body      string
	}{
		{name: "login without a code", path: "/_login", form: "auth_method=warden&phone=%s", status: 400,
			body: `{"code":400,"error":"Verification code and challenge_id are required.","reason":"verification_required"}`},
		{name: "login with a handle and no code", path: "/_login", form: "auth_method=warden&phone=%s&challenge_id=ch_AAAAAAAAAAAAAAAAAAAAAA", status: 400,
			body: `{"code":400,"error":"Verification code and challenge_id are required.","reason":"verification_required"}`},
		{name: "TOTP without a code", path: "/_login", form: "auth_method=warden&phone=%s&use_otp=true", status: 400,
			body: `{"code":400,"error":"OTP code is required."}`},
		{name: "TOTP disabled", path: "/_login", form: "auth_method=warden&phone=%s&use_otp=true&otp_code=000000", status: 400,
			configure: func(t *testing.T) { setConfigValue(t, &config.HeraldTOTPEnabled, "false") },
			body:      `{"code":400,"error":"Please provide verification code or use OTP."}`},
		{name: "Herald disabled", path: "/_login", form: "auth_method=warden&phone=%s&challenge_id=ch_AAAAAAAAAAAAAAAAAAAAAA&verify_code=000000", status: 500,
			configure: func(t *testing.T) {
				setConfigValue(t, &config.HeraldEnabled, "false")
				setConfigValue(t, &config.HeraldTOTPEnabled, "false")
			},
			body: `{"code":500,"error":"Verification code service is not configured. Please use OTP or contact administrator."}`},
		{name: "Herald disabled, TOTP offered", path: "/_login", form: "auth_method=warden&phone=%s&challenge_id=ch_AAAAAAAAAAAAAAAAAAAAAA&verify_code=000000", status: 400,
			configure: func(t *testing.T) { setConfigValue(t, &config.HeraldEnabled, "false") },
			body:      `{"code":400,"error":"Verification code service is not configured. Please use OTP."}`},
		{name: "no Herald client, code", path: "/_login", form: "auth_method=warden&phone=%s&challenge_id=ch_AAAAAAAAAAAAAAAAAAAAAA&verify_code=000000", status: 500,
			configure: func(t *testing.T) { withoutHeraldClient(t) },
			body:      `{"code":500,"error":"Verification code service is unavailable."}`},
		{name: "no Herald client, TOTP", path: "/_login", form: "auth_method=warden&phone=%s&use_otp=true&otp_code=000000", status: 503,
			configure: func(t *testing.T) { withoutHeraldClient(t) },
			body:      `{"code":503,"error":"Verification code service is temporarily unavailable. Please try again later.","reason":"service_unavailable"}`},
		{name: "context store unavailable", path: "/_login", form: "auth_method=warden&phone=%s&challenge_id=ch_AAAAAAAAAAAAAAAAAAAAAA&verify_code=000000", status: 503,
			configure: func(t *testing.T) {
				d.contexts.setFailReads(true)
				t.Cleanup(func() { d.contexts.setFailReads(false) })
			},
			body: `{"code":503,"error":"verification context is unavailable"}`},
		{name: "send with SMS disabled", path: "/_send_verify_code", form: "phone=%s&deliver_via=sms", status: 400,
			configure: func(t *testing.T) { setConfigValue(t, &config.LoginSMSEnabled, "false") },
			body:      `{"message":"SMS login is disabled.","reason":"channel_disabled","success":false}`},
		{name: "send with email disabled", path: "/_send_verify_code", form: "phone=%s&deliver_via=email", status: 400,
			configure: func(t *testing.T) { setConfigValue(t, &config.LoginEmailEnabled, "false") },
			body:      `{"message":"Email login is disabled.","reason":"channel_disabled","success":false}`},
		{name: "send without a Herald client", path: "/_send_verify_code", form: "phone=%s&deliver_via=sms", status: 503,
			configure: func(t *testing.T) { withoutHeraldClient(t) },
			body:      `{"message":"Verification code service is temporarily unavailable. Please use OTP.","reason":"connection_failed","success":false}`},
		{name: "send without a Herald client or TOTP", path: "/_send_verify_code", form: "phone=%s&deliver_via=sms", status: 503,
			configure: func(t *testing.T) {
				withoutHeraldClient(t)
				setConfigValue(t, &config.HeraldTOTPEnabled, "false")
			},
			body: `{"message":"Verification code service is temporarily unavailable. Please try again later.","reason":"connection_failed","success":false}`},
		{name: "send with Herald disabled", path: "/_send_verify_code", form: "phone=%s&deliver_via=sms", status: 400,
			configure: func(t *testing.T) { setConfigValue(t, &config.HeraldEnabled, "false") },
			body:      `{"message":"Verification code service is not configured.","reason":"herald_not_configured","success":false}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.configure != nil {
				tc.configure(t)
			}
			wardenBefore, heraldBefore := d.warden.callCount(), d.herald.total()
			var replies []uniformReply
			for _, phone := range []string{"13900000000", "13800000001"} {
				d.reset()
				replies = append(replies, d.post(tc.path, strings.ReplaceAll(tc.form, "%s", phone), acceptJSON))
			}
			require.Equal(t, replies[0], replies[1])
			require.Equal(t, tc.status, replies[0].status, replies[0].body)
			require.Equal(t, tc.body, replies[0].body)
			require.Equal(t, wardenBefore, d.warden.callCount(), "the answer must be given before the directory lookup")
			require.Equal(t, heraldBefore, d.herald.total())
		})
	}

	// The identifier checks come first, as in the default mode. An invalid
	// phone number is refused also next to a listed mail: the request is not
	// handled by the mail instead.
	wardenBefore, heraldBefore := d.warden.callCount(), d.herald.total()
	for _, tc := range []struct{ path, form, reason string }{
		{"/_login", "auth_method=warden&phone=12345", "identifier_invalid"},
		{"/_login", "auth_method=warden", "identifier_required"},
		{"/_login", "auth_method=warden&phone=12345&use_otp=true&otp_code=000000", "identifier_invalid"},
		{"/_login", "auth_method=warden&challenge_id=ch_x&verify_code=000000", "identifier_required"},
		{"/_login", "auth_method=warden&phone=12345&challenge_id=&verify_code=", "identifier_invalid"},
		{"/_login", "auth_method=warden&mail=&phone=&use_otp=true&otp_code=000000", "identifier_required"},
		{"/_login", "auth_method=warden&phone=12345&mail=a@example.com&challenge_id=ch_x&verify_code=000000", "identifier_invalid"},
		{"/_login", "auth_method=warden&phone=12345&mail=a@example.com&use_otp=true&otp_code=000000", "identifier_invalid"},
		{"/_send_verify_code", "phone=12345&mail=a@example.com&deliver_via=email", "invalid_phone_format"},
	} {
		d.reset()
		reply := d.post(tc.path, tc.form, acceptJSON)
		require.Equal(t, fiber.StatusBadRequest, reply.status, tc.form)
		require.Equal(t, tc.reason, jsonFields(t, reply.body)["reason"], tc.form)
	}
	require.Equal(t, wardenBefore, d.warden.callCount())
	require.Equal(t, heraldBefore, d.herald.total())
}

// A directory that cannot answer is reported as such. That answer does not
// say whether the identifier is listed: the lookup failed for all of them.
func TestUniformDirectoryFailureAnswersTheSameForEveryIdentifier(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	handles := map[string]string{}
	for _, phone := range []string{"13900000000", "13800000001"} {
		d.reset()
		handles[phone] = d.sendHandle("phone=" + phone + "&deliver_via=sms")
	}

	const sendUnavailable = `{"message":"Verification code service is temporarily unavailable. Please try again later.","reason":"service_unavailable","success":false}`
	const loginUnavailable = `{"code":503,"error":"directory authorization is unavailable"}`
	check := func(t *testing.T, form func(phone string) (path, body string), want string) {
		t.Helper()
		heraldBefore := d.herald.total()
		var replies []uniformReply
		for _, phone := range []string{"13900000000", "13800000001"} {
			d.reset()
			path, body := form(phone)
			replies = append(replies, d.post(path, body, acceptJSON))
		}
		require.Equal(t, replies[0], replies[1])
		require.Equal(t, fiber.StatusServiceUnavailable, replies[0].status, replies[0].body)
		require.Equal(t, want, replies[0].body)
		require.Equal(t, heraldBefore, d.herald.total())
	}

	t.Run("Warden outage", func(t *testing.T) {
		d.warden.setDown(true)
		t.Cleanup(func() { d.warden.setDown(false) })
		check(t, func(phone string) (string, string) {
			return "/_send_verify_code", "phone=" + phone + "&deliver_via=sms"
		}, sendUnavailable)
		check(t, func(phone string) (string, string) {
			return "/_login", "auth_method=warden&phone=" + phone + "&challenge_id=" + handles[phone] + "&verify_code=000000"
		}, loginUnavailable)
		check(t, func(phone string) (string, string) {
			return "/_login", "auth_method=warden&phone=" + phone + "&use_otp=true&otp_code=000000"
		}, loginUnavailable)
	})
}

// With the flag off the responses are the ones that exist today, so the flag
// is what removes the differences.
func TestDefaultModeStillAnswersByIdentifier(t *testing.T) {
	d := newUniformDeployment(t)
	setConfigValue(t, &config.LoginUniformResponses, "false")
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	d.herald.enrollTOTP("user-b")
	d.warden.add(activeUser("user-b", "13800000002", "b@example.com"))

	unknown := d.send("phone=13900000000&deliver_via=sms")
	require.Equal(t, fiber.StatusUnauthorized, unknown.status)
	require.Equal(t, `{"message":"User not found in allow list","reason":"user_not_in_list","success":false}`, unknown.body)

	known := d.send("phone=13800000001&deliver_via=sms")
	require.Equal(t, fiber.StatusOK, known.status)
	fields := jsonFields(t, known.body)
	heraldID := d.herald.activeChallenge("user-a")
	require.NotEmpty(t, heraldID)
	require.Equal(t, heraldID, fields["challenge_id"], "the default mode returns the id Herald issued")
	require.Equal(t, float64(300), fields["expires_in"])
	require.Equal(t, "Verification code sent", fields["message"])

	// No cooldown of Stargate's own: the second request reaches Herald.
	require.Equal(t, fiber.StatusOK, d.send("phone=13800000001&deliver_via=sms").status)
	require.Len(t, d.herald.callsTo("/v1/otp/challenges", "user-a"), 2)
	require.Equal(t, fiber.StatusUnauthorized, d.send("phone=13900000000&deliver_via=sms").status)
	heraldID = d.herald.activeChallenge("user-a")

	for form, want := range map[string]struct {
		status int
		body   string
	}{
		"phone=13900000000": {401, `{"code":401,"error":"User not found in allow list","reason":"authentication_failed"}`},
		"phone=13800000001": {400, `{"code":400,"error":"Verification code and challenge_id are required.","reason":"verification_required"}`},
		"phone=13800000001&challenge_id=ch_AAAAAAAAAAAAAAAAAAAAAA&verify_code=000000": {401, `{"code":401,"error":"Verification failed."}`},
		"phone=13800000001&challenge_id=" + heraldID + "&verify_code=000000":          {401, `{"code":401,"error":"Invalid verification code. 4 attempts remaining. Please check and try again.","reason":"invalid"}`},
		"phone=13800000001&use_otp=true&otp_code=000000":                              {400, `{"code":400,"error":"This account has not bound TOTP yet. Please sign in with verification code first, then bind TOTP in settings.","reason":"totp_not_enrolled"}`},
		"phone=13800000002&use_otp=true&otp_code=000000":                              {401, `{"code":401,"error":"Invalid OTP code."}`},
	} {
		reply := d.login(form)
		require.Equal(t, want.status, reply.status, form)
		require.Equal(t, want.body, reply.body, form)
	}
}

func withoutHeraldClient(t *testing.T) {
	t.Helper()
	previous := heraldClient
	heraldClient = nil
	t.Cleanup(func() { heraldClient = previous })
}

// A sweep over request shapes that the matrices above do not spell out:
// identifier combinations, delivery preferences, response formats and stray
// fields. Each shape is sent for every identity from the same starting state,
// and all answers to one shape must be equal.
func TestUniformResponsesAcrossRequestShapes(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(
		activeUser("user-plain", "13800000001", "plain@example.com"),
		warden.AllowListUser{UserID: "user-inactive", Phone: "13800000002", Mail: "inactive@example.com", Status: "inactive"},
		activeUser("user-totp", "13800000003", "totp@example.com"),
		activeUser("user-locked", "13800000004", "locked@example.com"),
		warden.AllowListUser{UserID: "user-dingtalk", Phone: "13800000005", Mail: "dingtalk@example.com", Status: "active", DingtalkUserID: "ding-1"},
		// Warden may return a record without a user id; Stargate derives one.
		activeUser("", "13800000006", "no-id@example.com"),
		// Not one of the identities below: the owner of the {OTHER} handle.
		activeUser("user-other", "13800000009", "other@example.com"),
	)
	d.herald.enrollTOTP("user-totp")
	d.herald.set(d.herald.createMode, "user-locked", "locked")
	identities := []struct{ name, phone, mail string }{
		{"unknown", "13900000000", "nobody@example.com"},
		{"inactive", "13800000002", "inactive@example.com"},
		{"listed", "13800000001", "plain@example.com"},
		{"listed with TOTP", "13800000003", "totp@example.com"},
		{"listed, locked at Herald", "13800000004", "locked@example.com"},
		{"listed with DingTalk", "13800000005", "dingtalk@example.com"},
		{"listed without a user id", "13800000006", "no-id@example.com"},
	}
	overlong := strings.Repeat("b", 600) + "@example.com"

	// {P} and {M} are the identity's phone and mail, {H} the handle of its own
	// send, {OTHER} a handle issued to another, listed identity, and {LONG} a
	// mail that Warden refuses for its length.
	sendShapes := []string{
		"phone={P}", "mail={M}", "phone={P}&mail={M}",
		"phone={P}&deliver_via=sms", "phone={P}&deliver_via=email", "phone={P}&deliver_via=dingtalk", "phone={P}&deliver_via=unknown",
		"mail={M}&deliver_via=dingtalk", "mail={M}&deliver_via=sms", "mail=%20{M}%20",
		"phone={P}&mail={LONG}", "phone={P}&mail=%20", "phone={P}&mail=someone-else@example.com", "phone=13911111111&mail={M}",
		"phone={P}&challenge_id=ch_AAAAAAAAAAAAAAAAAAAAAA&verify_code=000000",
		"phone={P}&challenge_id=not%20a%20handle", "phone={P}&challenge_id={LONG}", "phone={P}&challenge_id={OTHER}",
	}
	loginShapes := []string{
		"phone={P}", "mail={M}", "phone={P}&mail={M}",
		"phone={P}&challenge_id={H}&verify_code=000000", "mail={M}&challenge_id={H}&verify_code=000000",
		"phone={P}&mail={M}&challenge_id={H}&verify_code=000000",
		"phone={P}&challenge_id={OTHER}&verify_code=" + uniformTestCode,
		"phone={P}&challenge_id=live:0000&verify_code=" + uniformTestCode,
		"phone={P}&challenge_id={H}&verify_code=", "phone={P}&challenge_id=&verify_code=000000",
		"phone={P}&use_otp=true", "phone={P}&use_otp=true&otp_code=000000", "phone={P}&use_otp=true&otp_code=000000&challenge_id={H}",
		"phone={P}&use_otp=TRUE&otp_code=000000",
		"phone={P}&mail={LONG}&challenge_id={H}&verify_code=000000",
		"phone={P}&challenge_id={H}&verify_code=000000&callback=other.example",
		"phone={P}&challenge_id={H}&verify_code=000000&password=test123",
	}
	accepts := []string{"application/json", "application/xml", "text/plain", "text/html", "*/*", ""}

	compare := func(t *testing.T, request func(phone, mail string) uniformReply) {
		t.Helper()
		var reference uniformReply
		var referenceLookups int
		for index, who := range identities {
			// Every identity starts from empty stores.
			d.reset()
			before := d.warden.callCount()
			reply := request(who.phone, who.mail)
			lookups := d.warden.callCount() - before
			if index == 0 {
				reference, referenceLookups = reply, lookups
			}
			require.Equal(t, reference, reply, who.name)
			require.Equal(t, referenceLookups, lookups, "the directory is asked equally often for %s", who.name)
		}
	}
	for _, shape := range sendShapes {
		t.Run("send "+shape, func(t *testing.T) {
			compare(t, func(phone, mail string) uniformReply {
				other := d.sendHandle("phone=13800000009&deliver_via=sms")
				form := strings.NewReplacer("{P}", phone, "{M}", mail, "{LONG}", overlong, "{OTHER}", other).Replace(shape)
				reply, _ := maskHandle(t, d.post("/_send_verify_code", form, map[string]string{"Accept": "application/json", "Idempotency-Key": "client-chosen-key"}))
				return reply
			})
		})
	}
	for _, shape := range loginShapes {
		for _, accept := range accepts {
			t.Run("login "+shape+" as "+accept, func(t *testing.T) {
				compare(t, func(phone, mail string) uniformReply {
					other := d.sendHandle("phone=13800000009&deliver_via=sms")
					handle := d.sendHandle("phone=" + phone + "&deliver_via=sms")
					form := strings.NewReplacer("{P}", phone, "{M}", mail, "{H}", handle, "{OTHER}", other, "{LONG}", overlong).Replace(shape)
					return d.post("/_login", "auth_method=warden&"+form, map[string]string{"Accept": accept})
				})
			})
		}
	}
}

// challenge_id on a send request names the handle the client holds. It decides
// what a new handle is bound to when Herald creates no challenge, and it must
// never show in the answer: not for an unlisted identifier, and not for a
// listed one whose resend Herald serves or refuses.
func TestUniformSendResponseDoesNotDependOnThePresentedHandle(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(
		activeUser("user-served", "13800000001", "served@example.com"),
		warden.AllowListUser{UserID: "user-inactive", Phone: "13800000002", Mail: "inactive@example.com", Status: "inactive"},
		activeUser("user-refused", "13800000003", "refused@example.com"),
		activeUser("user-locked", "13800000004", "locked@example.com"),
		activeUser("user-other", "13800000009", "other@example.com"),
	)
	d.herald.set(d.herald.createMode, "user-refused", "resend-refused")
	d.herald.set(d.herald.createMode, "user-locked", "locked")
	identities := []struct{ name, phone string }{
		{"unknown", "13900000000"},
		{"inactive", "13800000002"},
		{"listed, resend served", "13800000001"},
		{"listed, resend refused", "13800000003"},
		{"listed, locked at Herald", "13800000004"},
	}
	// The value of challenge_id on the second send, given the handle of the
	// identity's own first send and a handle issued to another account.
	variations := []struct {
		name  string
		field func(own, other string) string
	}{
		{"nothing", func(string, string) string { return "" }},
		{"its own handle", func(own, _ string) string { return "&challenge_id=" + own }},
		{"the handle of another account", func(_, other string) string { return "&challenge_id=" + other }},
		{"an unknown handle", func(string, string) string { return "&challenge_id=ch_AAAAAAAAAAAAAAAAAAAAAA" }},
		{"something that is not a handle", func(string, string) string { return "&challenge_id=not%20a%20handle" }},
		{"an oversized value", func(string, string) string { return "&challenge_id=" + strings.Repeat("h", 200) }},
	}

	var reference []uniformReply
	for _, variation := range variations {
		for _, who := range identities {
			t.Run(who.name+" presents "+variation.name, func(t *testing.T) {
				d.reset()
				other := d.sendHandle("phone=13800000009&deliver_via=sms")
				form := "phone=" + who.phone + "&deliver_via=sms"
				first, own := maskHandle(t, d.send(form))
				require.NotEmpty(t, own)
				d.advance(61 * time.Second)
				second, handle := maskHandle(t, d.send(form+variation.field(own, other)))

				replies := []uniformReply{first, second}
				if reference == nil {
					reference = replies
				}
				require.Equal(t, reference, replies)
				require.Equal(t, uniformSendOK, second.body)

				// The field does matter on the server, in exactly one case:
				// Herald refused and the identity presented its own handle.
				_, bound := d.context(handle)
				switch who.name {
				case "listed, resend refused":
					require.Equal(t, variation.name == "its own handle", bound)
				case "listed, resend served":
					require.True(t, bound, "bound to the challenge this request created")
				default:
					require.False(t, bound)
				}
			})
		}
	}
}

// A request that supplies a phone number is handled by the phone number alone,
// as if no mail had been sent. The default mode falls back to the mail when
// the phone is not listed. With identical responses that fallback would still
// be visible to the owner of one listed mail address: a code would reach that
// mailbox, and sign-in with that mail would succeed, exactly when the phone
// next to it is not listed.
func TestUniformPhoneIsTheOnlyIdentifierWhenSupplied(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(
		activeUser("user-a", "13800000001", "a@example.com"),
		activeUser("user-b", "13800000002", "b@example.com"),
	)
	d.herald.enrollTOTP("user-a")
	const unlisted = "13900000000"
	overlong := strings.Repeat("a", 600) + "@example.com"
	byPhone := func(phone string, count int) []uniformWardenLookup {
		lookups := make([]uniformWardenLookup, count)
		for index := range lookups {
			lookups[index] = uniformWardenLookup{phone: phone}
		}
		return lookups
	}
	signedInAs := func(t *testing.T, reply uniformReply) any {
		t.Helper()
		require.Equal(t, fiber.StatusOK, reply.status, reply.body)
		match := regexp.MustCompile(`Set-Cookie: stargate_session_id=([^;]+)`).FindStringSubmatch(reply.headers)
		require.Len(t, match, 2, reply.headers)
		return reauthSessionValue(t, d.sessions, match[1], "user_id")
	}

	t.Run("no code is sent to a listed mail next to an unlisted phone", func(t *testing.T) {
		d.reset()
		alone, _ := maskHandle(t, d.send("phone="+unlisted+"&deliver_via=email"))
		d.reset()
		lookups, heraldBefore := d.warden.callCount(), d.herald.total()
		both, _ := maskHandle(t, d.send("phone="+unlisted+"&mail=a@example.com&deliver_via=email"))

		require.Equal(t, alone, both)
		require.Equal(t, uniformSendOK, both.body)
		require.Equal(t, heraldBefore, d.herald.total(), "no code may be sent to the mail")
		require.Equal(t, byPhone(unlisted, 1), d.warden.lookupsSince(lookups), "one lookup, by phone, without the mail")
	})

	t.Run("the code of a listed mail does not sign in next to an unlisted phone", func(t *testing.T) {
		d.reset()
		handle := d.sendHandle("mail=a@example.com&deliver_via=email")
		lookups, heraldBefore := d.warden.callCount(), d.herald.total()

		requireUniformLoginFailure(t, d.login("phone="+unlisted+"&mail=a@example.com&challenge_id="+handle+"&verify_code="+uniformTestCode))
		require.Equal(t, heraldBefore, d.herald.total(), "the code may not be verified")
		require.Equal(t, byPhone(unlisted, 1), d.warden.lookupsSince(lookups))

		// The handle and the code are valid, and still unused, for the mail alone.
		require.Equal(t, "user-a", signedInAs(t, d.login("mail=a@example.com&challenge_id="+handle+"&verify_code="+uniformTestCode)))
	})

	t.Run("the TOTP of a listed mail does not sign in next to an unlisted phone", func(t *testing.T) {
		d.reset()
		lookups, heraldBefore := d.warden.callCount(), d.herald.total()

		requireUniformLoginFailure(t, d.login("phone="+unlisted+"&mail=a@example.com&use_otp=true&otp_code="+uniformTestCode))
		require.Equal(t, heraldBefore, d.herald.total(), "the TOTP may not be verified")
		require.Equal(t, byPhone(unlisted, 1), d.warden.lookupsSince(lookups))

		// The TOTP is valid for the mail alone.
		require.Equal(t, "user-a", signedInAs(t, d.login("mail=a@example.com&use_otp=true&otp_code="+uniformTestCode)))
	})

	t.Run("a listed phone is handled as its own user whatever the mail", func(t *testing.T) {
		for _, mail := range []string{"b@example.com", "nobody@example.com", "not-an-address", overlong} {
			d.reset()
			lookups := d.warden.callCount()
			handle := d.sendHandle("phone=13800000001&mail=" + mail + "&deliver_via=sms")
			binding, found := d.context(handle)
			require.True(t, found, mail)
			require.Equal(t, "user-a", binding.UserID)
			require.Equal(t, "13800000001", binding.Destination)
			require.Empty(t, d.herald.activeChallenge("user-b"), "nothing is sent to the user of the mail")

			require.Equal(t, "user-a", signedInAs(t, d.login("phone=13800000001&mail="+mail+"&challenge_id="+handle+"&verify_code="+uniformTestCode)), mail)
			require.Equal(t, "user-a", signedInAs(t, d.login("phone=13800000001&mail="+mail+"&use_otp=true&otp_code="+uniformTestCode)), mail)
			require.Equal(t, byPhone("13800000001", 3), d.warden.lookupsSince(lookups), mail)
		}
	})

	// Warden refuses an identifier above its length limit. Such a mail next to
	// a phone number is never sent to Warden, so it cannot fail the request
	// for an unlisted phone only.
	t.Run("an over-long mail next to a phone is ignored", func(t *testing.T) {
		for _, phone := range []string{unlisted, "13800000001"} {
			exchange := func(mail string) []uniformReply {
				d.reset()
				sent, handle := maskHandle(t, d.send("phone="+phone+mail+"&deliver_via=sms"))
				return []uniformReply{
					sent,
					d.login("phone=" + phone + mail + "&challenge_id=" + handle + "&verify_code=000000"),
					d.login("phone=" + phone + mail + "&use_otp=true&otp_code=000000"),
				}
			}
			without := exchange("")
			lookups := d.warden.callCount()
			with := exchange("&mail=" + overlong)

			require.Equal(t, without, with, phone)
			require.Equal(t, uniformSendOK, with[0].body)
			requireUniformLoginFailure(t, with[1])
			requireUniformLoginFailure(t, with[2])
			require.Equal(t, byPhone(phone, 3), d.warden.lookupsSince(lookups), phone)
		}
	})

	// The fallback itself is unchanged in the default mode.
	t.Run("the default mode still falls back to the mail", func(t *testing.T) {
		d.reset()
		setConfigValue(t, &config.LoginUniformResponses, "false")
		lookups := d.warden.callCount()
		reply := d.send("phone=" + unlisted + "&mail=a@example.com&deliver_via=email")
		require.Equal(t, fiber.StatusOK, reply.status, reply.body)
		require.Equal(t, d.herald.activeChallenge("user-a"), jsonFields(t, reply.body)["challenge_id"])
		require.Equal(t, []uniformWardenLookup{{phone: unlisted}, {mail: "a@example.com"}}, d.warden.lookupsSince(lookups))
	})
}
