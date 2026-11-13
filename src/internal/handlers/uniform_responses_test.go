package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/herald/pkg/herald"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/stretchr/testify/require"
)

const uniformLoginFailureJSON = `{"code":401,"error":"` + uniformLoginFailure + `","reason":"authentication_failed"}`

// requireUniformLoginFailure checks the one failure answer of uniform sign-in.
func requireUniformLoginFailure(t *testing.T, reply uniformReply) {
	t.Helper()
	require.Equal(t, fiber.StatusUnauthorized, reply.status, reply.body)
	require.Equal(t, uniformLoginFailureJSON, reply.body)
	require.NotContains(t, reply.headers, "Set-Cookie")
}

// A handle stands in for the Herald challenge towards the client: it signs in
// once, and the id Herald issued is neither returned nor accepted.
func TestUniformHandleSignsInOnceAndHidesTheHeraldChallenge(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))

	reply := d.send("phone=13800000001&deliver_via=sms")
	require.Equal(t, fiber.StatusOK, reply.status)
	_, handle := maskHandle(t, reply)
	heraldID := d.herald.activeChallenge("user-a")
	require.NotEmpty(t, heraldID)
	require.NotEqual(t, heraldID, handle)
	require.NotContains(t, reply.body, heraldID, "the Herald challenge id must not reach the client")
	fields := jsonFields(t, reply.body)
	require.Len(t, fields, 4, "challenge_id, message, next_resend_in and success only")
	require.NotContains(t, fields, "expires_in")
	require.NotContains(t, fields, "debug_code")

	binding, found := d.context(handle)
	require.True(t, found)
	require.Equal(t, challengeContext{ChallengeID: handle, HeraldChallengeID: heraldID, UserID: "user-a", Channel: "sms", Destination: "13800000001", ExpiresAt: d.now().Unix() + 300}, binding)
	require.Equal(t, 10*time.Minute, d.contextTTL(handle), "kept for the lifetime of the challenge and the audit grace")

	// The id Herald issued has no context of its own, so it is refused
	// without asking Herald.
	requireUniformLoginFailure(t, d.login("phone=13800000001&challenge_id="+heraldID+"&verify_code="+uniformTestCode))
	require.Empty(t, d.herald.callsTo("/v2/otp/verifications", ""))

	requireUniformLoginFailure(t, d.login("phone=13800000001&challenge_id="+handle+"&verify_code=000000"))
	signedIn := d.login("phone=13800000001&challenge_id=" + handle + "&verify_code=" + uniformTestCode)
	require.Equal(t, fiber.StatusOK, signedIn.status, signedIn.body)
	require.Equal(t, `{"message":"Login successful","success":true}`, signedIn.body)
	require.Contains(t, signedIn.headers, "Set-Cookie: stargate_session_id=")
	verifications := d.herald.callsTo("/v2/otp/verifications", "user-a")
	require.Len(t, verifications, 2)
	for _, call := range verifications {
		require.Equal(t, heraldID, call.challengeID, "Herald is addressed by the id it issued")
	}

	// Redeemed: the handle is gone.
	_, found = d.context(handle)
	require.False(t, found)
	requireUniformLoginFailure(t, d.login("phone=13800000001&challenge_id="+handle+"&verify_code="+uniformTestCode))
	require.Len(t, d.herald.callsTo("/v2/otp/verifications", "user-a"), 2)
}

// A handle is verified only for the account it was issued to. For any other
// identifier Herald is not asked, whether that identifier is listed or not.
// If it were asked for a listed one, wrong guesses would use up the attempts
// of the holder's own challenge only then, and the holder could tell from its
// code no longer working.
func TestUniformHandleOfAnotherAccountNeverReachesHerald(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(
		activeUser("user-a", "13800000001", "a@example.com"),
		activeUser("user-b", "13800000002", "b@example.com"),
		// Another account with the mail the handle was issued for.
		activeUser("user-c", "13800000003", "a@example.com"),
	)
	handle := d.sendHandle("mail=a@example.com&deliver_via=email")
	binding, found := d.context(handle)
	require.True(t, found)
	require.Equal(t, "user-a", binding.UserID)
	heraldBefore := d.herald.total()

	for _, identifier := range []string{"phone=13900000000", "phone=13800000002", "phone=13800000003", "mail=nobody@example.com", "mail=b@example.com"} {
		for _, code := range []string{"000000", uniformTestCode} {
			requireUniformLoginFailure(t, d.login(identifier+"&challenge_id="+handle+"&verify_code="+code))
		}
		require.Equal(t, heraldBefore, d.herald.total(), identifier)
	}

	// The holder's challenge is untouched.
	reply := d.login("mail=a@example.com&challenge_id=" + handle + "&verify_code=" + uniformTestCode)
	require.Equal(t, fiber.StatusOK, reply.status, reply.body)
}

// The challenge context resolves the Herald id in either mode, so a challenge
// issued before the flag is changed can still be redeemed afterwards.
func TestChallengeContextsSurviveTogglingUniformResponses(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))

	config.LoginUniformResponses.Value = "false"
	issued := jsonFields(t, d.send("phone=13800000001&deliver_via=sms").body)["challenge_id"].(string)
	require.Equal(t, d.herald.activeChallenge("user-a"), issued)
	config.LoginUniformResponses.Value = "true"
	reply := d.login("phone=13800000001&challenge_id=" + issued + "&verify_code=" + uniformTestCode)
	require.Equal(t, fiber.StatusOK, reply.status, reply.body)

	d.reset()
	handle := d.sendHandle("phone=13800000001&deliver_via=sms")
	config.LoginUniformResponses.Value = "false"
	reply = d.login("phone=13800000001&challenge_id=" + handle + "&verify_code=" + uniformTestCode)
	require.Equal(t, fiber.StatusOK, reply.status, reply.body)
	calls := d.herald.callsTo("/v2/otp/verifications", "user-a")
	require.NotEqual(t, handle, calls[len(calls)-1].challengeID)
}

// A login handle and a re-authentication challenge are not interchangeable in
// uniform mode either, and re-authentication itself is unchanged.
func TestUniformHandleAndReauthChallengeAreNotInterchangeable(t *testing.T) {
	d := newUniformDeployment(t)
	setConfigValue(t, &config.AuthRefreshEnabled, "false")
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	sessionID := newReauthSession(t, d.sessions, map[string]any{"auth_method": "warden", "user_id": "user-a", "user_mail": "a@example.com"})
	stepUp := func(code string) int {
		return reauthRequest(t, sessionID, "POST", "/_step_up", "code="+code, StepUpAPI(d.sessions)).Response().StatusCode()
	}

	// A login handle, or the Herald challenge behind it, named by the session
	// as its re-authentication challenge is refused without asking Herald.
	handle := d.sendHandle("mail=a@example.com&deliver_via=email")
	loginChallenge := d.herald.activeChallenge("user-a")
	for _, challengeID := range []string{handle, loginChallenge} {
		setReauthSessionValue(t, d.sessions, sessionID, reauthChallengeIDKey, challengeID)
		require.Equal(t, fiber.StatusUnauthorized, stepUp(uniformTestCode))
	}
	require.Empty(t, d.herald.callsTo("/v2/otp/verifications", ""))
	require.Nil(t, reauthSessionValue(t, d.sessions, sessionID, "step_up_verified_at"))

	// A re-authentication challenge is refused as a login challenge_id. It
	// still carries the id Herald issued, and it still passes step-up.
	reauthChallenge := sendReauthCode(t, d.sessions, sessionID, "deliver_via=email")
	require.Equal(t, d.herald.activeChallenge("user-a"), reauthChallenge)
	requireUniformLoginFailure(t, d.login("mail=a@example.com&challenge_id="+reauthChallenge+"&verify_code="+uniformTestCode))
	require.Empty(t, d.herald.callsTo("/v2/otp/verifications", ""))
	require.Equal(t, fiber.StatusFound, stepUp(uniformTestCode))
	require.NotNil(t, reauthSessionValue(t, d.sessions, sessionID, "step_up_verified_at"))
}

// heraldRefusals are the ways in which Herald does not create a challenge.
var heraldRefusals = []string{"cooldown", "locked", "error", "unreachable"}

// uniformResend is the state after a send that Herald served and a later one
// that Herald refused.
type uniformResend struct {
	first, second string // the handles of the two sends
	issued        string // the Herald challenge of the first send
}

// refusedResend sends a code for user-a, waits out Stargate's window, makes
// Herald refuse the next creation and sends again. present builds the extra
// form fields of the second request from the first handle.
func refusedResend(t *testing.T, d *uniformDeployment, mode string, present func(first string) string) uniformResend {
	t.Helper()
	const form = "phone=13800000001&deliver_via=sms"
	first := d.sendHandle(form)
	issued := d.herald.activeChallenge("user-a")
	require.NotEmpty(t, issued)

	d.advance(61 * time.Second)
	d.herald.set(d.herald.createMode, "user-a", mode)
	reply, second := maskHandle(t, d.send(form+present(first)))
	require.Equal(t, fiber.StatusOK, reply.status)
	require.Equal(t, uniformSendOK, reply.body)
	require.NotContains(t, reply.headers, "Retry-After")
	require.NotEqual(t, first, second)
	require.GreaterOrEqual(t, len(d.herald.callsTo("/v1/otp/challenges", "user-a")), 2)
	require.Equal(t, issued, d.herald.activeChallenge("user-a"), "Herald created no new challenge")
	return uniformResend{first: first, second: second, issued: issued}
}

// Herald refused the resend and the client presented the handle it holds. The
// new handle is bound to the challenge of that handle, so the code the user
// already received signs in with it.
func TestUniformRefusedResendIsBoundToThePresentedHandle(t *testing.T) {
	for _, mode := range heraldRefusals {
		t.Run(mode, func(t *testing.T) {
			d := newUniformDeployment(t)
			d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
			resend := refusedResend(t, d, mode, func(first string) string { return "&challenge_id=" + first })

			held, found := d.context(resend.first)
			require.True(t, found)
			bound, found := d.context(resend.second)
			require.True(t, found)
			require.Equal(t, challengeContext{ChallengeID: resend.second, HeraldChallengeID: resend.issued, UserID: "user-a", Channel: "sms", Destination: "13800000001", ExpiresAt: held.ExpiresAt}, bound)

			signedIn := d.login("phone=13800000001&challenge_id=" + resend.second + "&verify_code=" + uniformTestCode)
			require.Equal(t, fiber.StatusOK, signedIn.status, signedIn.body)
			calls := d.herald.callsTo("/v2/otp/verifications", "user-a")
			require.Len(t, calls, 1)
			require.Equal(t, resend.issued, calls[0].challengeID)
		})
	}
}

// The invariant of handles: a handle is bound to the challenge its own request
// created, or to the challenge of a handle the requester presented. A request
// that presents nothing gets a handle that is bound to nothing, also while a
// code for the same user, channel and destination is still valid. Otherwise
// anyone who asks for a code would receive a handle for a challenge that
// another client holds. The earlier handle keeps working for its holder.
func TestUniformRefusedResendWithoutAHandleIsNotBound(t *testing.T) {
	for _, mode := range heraldRefusals {
		t.Run(mode, func(t *testing.T) {
			d := newUniformDeployment(t)
			d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
			resend := refusedResend(t, d, mode, func(string) string { return "" })

			_, found := d.context(resend.second)
			require.False(t, found, "a handle must not be bound to a challenge its request neither created nor proved")
			requireUniformLoginFailure(t, d.login("phone=13800000001&challenge_id="+resend.second+"&verify_code="+uniformTestCode))
			require.Empty(t, d.herald.callsTo("/v2/otp/verifications", ""))

			signedIn := d.login("phone=13800000001&challenge_id=" + resend.first + "&verify_code=" + uniformTestCode)
			require.Equal(t, fiber.StatusOK, signedIn.status, signedIn.body)
		})
	}
}

// Binding a second handle does not take the first one away, and the challenge
// is still redeemed only once.
func TestUniformResendRefusedByHeraldLeavesTheFirstHandleUsable(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	resend := refusedResend(t, d, "cooldown", func(first string) string { return "&challenge_id=" + first })

	reply := d.login("phone=13800000001&challenge_id=" + resend.first + "&verify_code=" + uniformTestCode)
	require.Equal(t, fiber.StatusOK, reply.status, reply.body)
	requireUniformLoginFailure(t, d.login("phone=13800000001&challenge_id="+resend.second+"&verify_code="+uniformTestCode))
}

// The presented handle is accepted only as proof of the challenge this request
// would have created: a login context from uniform mode, still valid, for the
// same user, channel and destination. Each case below fails exactly one of
// these conditions; the control cases satisfy all of them.
func TestUniformPresentedHandleMustProveTheSameChallenge(t *testing.T) {
	type refused struct {
		form, presented, userID string
		bound                   bool
	}
	const smsForm = "phone=13800000001&deliver_via=sms"
	// crafted stores a context that satisfies every condition for a refused
	// SMS resend of user-a, after applying change.
	crafted := func(d *uniformDeployment, id string, change func(*challengeContext)) refused {
		binding := challengeContext{ChallengeID: id, HeraldChallengeID: "ch_crafted", UserID: "user-a", Channel: "sms", Destination: "13800000001", ExpiresAt: d.now().Unix() + 300}
		change(&binding)
		stored, err := d.contexts.PutIfAbsent(context.Background(), binding, 10*time.Minute)
		require.NoError(d.t, err)
		require.True(d.t, stored)
		return refused{form: smsForm, presented: id, userID: "user-a"}
	}
	cases := map[string]func(t *testing.T, d *uniformDeployment) refused{
		"control: a context that satisfies every condition": func(_ *testing.T, d *uniformDeployment) refused {
			request := crafted(d, "ch_control", func(*challengeContext) {})
			request.bound = true
			return request
		},
		"control: a value of 128 bytes": func(_ *testing.T, d *uniformDeployment) refused {
			request := crafted(d, strings.Repeat("h", 128), func(*challengeContext) {})
			request.bound = true
			return request
		},
		"a value of 129 bytes": func(_ *testing.T, d *uniformDeployment) refused {
			return crafted(d, strings.Repeat("h", 129), func(*challengeContext) {})
		},
		"an unknown value": func(*testing.T, *uniformDeployment) refused {
			return refused{form: smsForm, presented: "ch_AAAAAAAAAAAAAAAAAAAAAA", userID: "user-a"}
		},
		"a context with a purpose": func(_ *testing.T, d *uniformDeployment) refused {
			return crafted(d, "ch_purpose", func(binding *challengeContext) { binding.Purpose = reauthChallengePurpose })
		},
		"a context without a Herald challenge": func(_ *testing.T, d *uniformDeployment) refused {
			return crafted(d, "ch_no_herald_id", func(binding *challengeContext) { binding.HeraldChallengeID = "" })
		},
		"a context of another user": func(_ *testing.T, d *uniformDeployment) refused {
			return crafted(d, "ch_other_user", func(binding *challengeContext) { binding.UserID = "user-b" })
		},
		"a context for another channel": func(_ *testing.T, d *uniformDeployment) refused {
			return crafted(d, "ch_other_channel", func(binding *challengeContext) { binding.Channel = "dingtalk" })
		},
		"a context for another destination": func(_ *testing.T, d *uniformDeployment) refused {
			return crafted(d, "ch_other_destination", func(binding *challengeContext) { binding.Destination = "13800000002" })
		},
		"a context whose challenge expires now": func(_ *testing.T, d *uniformDeployment) refused {
			// The driver below lets 61 seconds pass before the refused send.
			return crafted(d, "ch_expiring", func(binding *challengeContext) { binding.ExpiresAt = d.now().Unix() + 61 })
		},
		"control: a context whose challenge expires one second later": func(_ *testing.T, d *uniformDeployment) refused {
			request := crafted(d, "ch_one_second", func(binding *challengeContext) { binding.ExpiresAt = d.now().Unix() + 62 })
			request.bound = true
			return request
		},
		// The same conditions with contexts that requests created.
		"a handle of another user with the same destination": func(_ *testing.T, d *uniformDeployment) refused {
			d.warden.add(activeUser("user-b", "13800000002", "a@example.com"))
			return refused{form: "phone=13800000002&deliver_via=email", presented: d.sendHandle("phone=13800000001&deliver_via=email"), userID: "user-b"}
		},
		"a handle for another channel with the same destination": func(_ *testing.T, d *uniformDeployment) refused {
			// Without a DingTalk binding the DingTalk destination is the
			// phone number, which is also the SMS destination.
			return refused{form: "phone=13800000001&deliver_via=dingtalk", presented: d.sendHandle(smsForm), userID: "user-a"}
		},
		"a handle for another destination on the same channel": func(_ *testing.T, d *uniformDeployment) refused {
			held := d.sendHandle("phone=13800000001&deliver_via=email")
			d.warden.replace(activeUser("user-a", "13800000001", "moved@example.com"))
			return refused{form: "phone=13800000001&deliver_via=email", presented: held, userID: "user-a"}
		},
		"a handle for another channel and destination": func(_ *testing.T, d *uniformDeployment) refused {
			return refused{form: "phone=13800000001&deliver_via=email", presented: d.sendHandle(smsForm), userID: "user-a"}
		},
		"a handle whose challenge is past its lifetime": func(_ *testing.T, d *uniformDeployment) refused {
			held := d.sendHandle(smsForm)
			d.advance(239 * time.Second) // 300 seconds with the driver's 61
			_, found := d.context(held)
			require.True(d.t, found, "the context outlives the challenge by the audit grace")
			return refused{form: smsForm, presented: held, userID: "user-a"}
		},
		"a handle that was redeemed": func(_ *testing.T, d *uniformDeployment) refused {
			held := d.sendHandle(smsForm)
			reply := d.login("phone=13800000001&challenge_id=" + held + "&verify_code=" + uniformTestCode)
			require.Equal(d.t, fiber.StatusOK, reply.status, reply.body)
			return refused{form: smsForm, presented: held, userID: "user-a"}
		},
		"a challenge issued in the default mode": func(t *testing.T, d *uniformDeployment) refused {
			config.LoginUniformResponses.Value = "false"
			issued, _ := jsonFields(t, d.send(smsForm).body)["challenge_id"].(string)
			config.LoginUniformResponses.Value = "true"
			require.Equal(t, d.herald.activeChallenge("user-a"), issued)
			return refused{form: smsForm, presented: issued, userID: "user-a"}
		},
		"a re-authentication challenge": func(t *testing.T, d *uniformDeployment) refused {
			setConfigValue(t, &config.AuthRefreshEnabled, "false")
			sessionID := newReauthSession(t, d.sessions, map[string]any{"auth_method": "warden", "user_id": "user-a", "user_mail": "a@example.com"})
			return refused{form: "mail=a@example.com&deliver_via=email", presented: sendReauthCode(t, d.sessions, sessionID, "deliver_via=email"), userID: "user-a"}
		},
	}
	for name, prepare := range cases {
		t.Run(name, func(t *testing.T) {
			d := newUniformDeployment(t)
			d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
			request := prepare(t, d)

			d.advance(61 * time.Second)
			d.herald.set(d.herald.createMode, request.userID, "cooldown")
			createsBefore := len(d.herald.callsTo("/v1/otp/challenges", request.userID))
			verificationsBefore := len(d.herald.callsTo("/v2/otp/verifications", ""))
			reply, handle := maskHandle(t, d.send(request.form+"&challenge_id="+request.presented))
			require.Equal(t, uniformSendOK, reply.body)
			require.Len(t, d.herald.callsTo("/v1/otp/challenges", request.userID), createsBefore+1, "Herald was asked and refused")

			bound, found := d.context(handle)
			if request.bound {
				require.True(t, found)
				held, _ := d.context(request.presented)
				require.Equal(t, held.HeraldChallengeID, bound.HeraldChallengeID)
				require.Equal(t, held.ExpiresAt, bound.ExpiresAt)
				return
			}
			require.False(t, found, "the new handle must not be bound: %+v", bound)
			requireUniformLoginFailure(t, d.login(strings.Split(request.form, "&")[0]+"&challenge_id="+handle+"&verify_code="+uniformTestCode))
			require.Len(t, d.herald.callsTo("/v2/otp/verifications", ""), verificationsBefore)
		})
	}
}

// A client that keeps presenting its handle across refused resends keeps a
// working handle, and nothing is extended: every handle carries the expiry of
// the original challenge and leaves the store at the same moment.
func TestUniformRefusedResendsKeepTheOriginalExpiry(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	const form = "phone=13800000001&deliver_via=sms"

	handles := []string{d.sendHandle(form)}
	issued := d.herald.activeChallenge("user-a")
	expiresAt := d.now().Unix() + 300
	d.herald.set(d.herald.createMode, "user-a", "cooldown")
	for range 2 {
		d.advance(61 * time.Second)
		handles = append(handles, d.sendHandle(form+"&challenge_id="+handles[len(handles)-1]))
	}

	require.Len(t, handles, 3)
	for _, handle := range handles {
		binding, found := d.context(handle)
		require.True(t, found)
		require.Equal(t, issued, binding.HeraldChallengeID)
		require.Equal(t, expiresAt, binding.ExpiresAt, "the expiry of the challenge is not extended")
		// Ten minutes from the first send (lifetime plus audit grace), of
		// which 122 seconds have passed.
		require.Equal(t, 478*time.Second, d.contextTTL(handle))
	}
	reply := d.login("phone=13800000001&challenge_id=" + handles[2] + "&verify_code=" + uniformTestCode)
	require.Equal(t, fiber.StatusOK, reply.status, reply.body)
	calls := d.herald.callsTo("/v2/otp/verifications", "user-a")
	require.Equal(t, issued, calls[len(calls)-1].challengeID)
}

// When Herald creates a challenge, the presented handle plays no part: the new
// handle is bound to the new challenge and its own expiry.
func TestUniformAcceptedResendIgnoresThePresentedHandle(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	const form = "phone=13800000001&deliver_via=sms"

	first := d.sendHandle(form)
	earlier := d.herald.activeChallenge("user-a")
	d.advance(61 * time.Second)
	second := d.sendHandle(form + "&challenge_id=" + first)
	current := d.herald.activeChallenge("user-a")
	require.NotEqual(t, earlier, current)

	binding, found := d.context(second)
	require.True(t, found)
	require.Equal(t, challengeContext{ChallengeID: second, HeraldChallengeID: current, UserID: "user-a", Channel: "sms", Destination: "13800000001", ExpiresAt: d.now().Unix() + 300}, binding)
	reply := d.login("phone=13800000001&challenge_id=" + second + "&verify_code=" + uniformTestCode)
	require.Equal(t, fiber.StatusOK, reply.status, reply.body)
}

// A code was sent, but its context could not be stored. The reply is the
// usual one and the handle fails like any other.
func TestUniformContextWriteFailureIsFlattened(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	d.contexts.setFailWriteFor("user-a")

	reply, handle := maskHandle(t, d.send("phone=13800000001&deliver_via=sms"))
	require.Equal(t, fiber.StatusOK, reply.status)
	require.Equal(t, uniformSendOK, reply.body)
	require.NotEmpty(t, d.herald.activeChallenge("user-a"))
	requireUniformLoginFailure(t, d.login("phone=13800000001&challenge_id="+handle+"&verify_code="+uniformTestCode))
	require.Empty(t, d.herald.callsTo("/v2/otp/verifications", ""))
}

func TestUniformSendGateUsesAFixedWindow(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	const form = "phone=13800000001&deliver_via=sms"
	refused := func(wait string) {
		t.Helper()
		reply := d.send(form)
		require.Equal(t, fiber.StatusTooManyRequests, reply.status, reply.body)
		require.Equal(t, `{"message":"Too many requests. Please try again later.","next_resend_in":`+wait+`,"reason":"rate_limited","success":false}`, reply.body)
		require.Contains(t, reply.headers, "Retry-After: "+wait+"\n")
	}

	require.Equal(t, fiber.StatusOK, d.send(form).status)
	d.advance(40 * time.Second)
	refused("20")
	d.advance(19 * time.Second)
	refused("1")
	// A wait below one second is reported as one second.
	d.advance(500 * time.Millisecond)
	refused("1")
	// Sixty seconds after the first request. The window would still be open
	// had one of the refused requests extended it.
	d.advance(500 * time.Millisecond)
	require.Equal(t, fiber.StatusOK, d.send(form).status)
	refused("60")
}

func TestUniformSendGateFollowsTheConfiguredCooldown(t *testing.T) {
	d := newUniformDeployment(t)
	setConfigValue(t, &config.LoginUniformResendCooldown, "1500ms")

	reply := d.send("phone=13900000000")
	require.Equal(t, fiber.StatusOK, reply.status)
	require.Equal(t, float64(2), jsonFields(t, reply.body)["next_resend_in"], "the window is reported in whole seconds, rounded up")
	require.Equal(t, float64(2), jsonFields(t, d.send("phone=13900000000").body)["next_resend_in"])
	d.advance(1500 * time.Millisecond)
	require.Equal(t, fiber.StatusOK, d.send("phone=13900000000").status)
}

// A request is counted against one identifier: its phone number, or its mail
// when it supplies no phone number. A mail next to a phone number is neither
// counted nor consulted.
func TestUniformSendGateCountsOneIdentifierPerRequest(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	status := func(form string) int { return d.send(form).status }

	t.Run("the phone and the mail of one user have separate windows", func(t *testing.T) {
		d.reset()
		require.Equal(t, fiber.StatusOK, status("phone=13800000001"))
		require.Equal(t, fiber.StatusOK, status("mail=a@example.com"))
		require.Equal(t, fiber.StatusTooManyRequests, status("phone=13800000001"))
		require.Equal(t, fiber.StatusTooManyRequests, status("mail=a@example.com"))
	})

	t.Run("a request with both opens the window of the phone only", func(t *testing.T) {
		d.reset()
		require.Equal(t, fiber.StatusOK, status("phone=13800000001&mail=a@example.com"))
		require.Equal(t, fiber.StatusOK, status("mail=a@example.com"), "the mail next to a phone number was not counted")
		require.Equal(t, fiber.StatusTooManyRequests, status("phone=13800000001"))
	})

	t.Run("an open window of the mail does not refuse a request with a phone", func(t *testing.T) {
		d.reset()
		require.Equal(t, fiber.StatusOK, status("mail=a@example.com"))
		d.advance(30 * time.Second)
		require.Equal(t, fiber.StatusOK, status("phone=13800000009&mail=a@example.com"))
		d.advance(20 * time.Second)
		refused := d.send("phone=13800000009&mail=a@example.com")
		require.Equal(t, fiber.StatusTooManyRequests, refused.status)
		require.Equal(t, float64(40), jsonFields(t, refused.body)["next_resend_in"], "the wait is that of the phone")
	})

	t.Run("spellings that name the same identifier share one window", func(t *testing.T) {
		d.reset()
		require.Equal(t, fiber.StatusOK, status("mail=a@example.com"))
		require.Equal(t, fiber.StatusTooManyRequests, status("mail=%20A%40Example.COM%20"))
		require.Equal(t, fiber.StatusOK, status("phone=13800000001"))
		require.Equal(t, fiber.StatusTooManyRequests, status("phone=138-0000-0001"))
		require.Equal(t, fiber.StatusTooManyRequests, status("phone=138%200000%200001"))
	})
}

// Store keys carry keyed digests. Neither the identifier that was submitted
// nor the user it resolves to can be read from them.
func TestUniformStoreKeysContainNoIdentifier(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	for _, form := range []string{"phone=13800000001&deliver_via=sms", "mail=a@example.com&deliver_via=email", "phone=13900000000", "mail=nobody@example.com"} {
		d.sendHandle(form)
	}

	patterns := map[string]*regexp.Regexp{
		"phone gate":        regexp.MustCompile(`^stargate:session:ratelimit:uniform-send:phone:[0-9a-f]{64}$`),
		"mail gate":         regexp.MustCompile(`^stargate:session:ratelimit:uniform-send:mail:[0-9a-f]{64}$`),
		"per-client limit":  regexp.MustCompile(`^stargate:session:ratelimit:0\.0\.0\.0:POST:/_send_verify_code$`),
		"challenge context": regexp.MustCompile(`^stargate:session:challenge:context:ch_[A-Za-z0-9_-]{22}$`),
	}
	seen := map[string]int{}
	for _, key := range d.redis.Keys() {
		for _, private := range []string{"13800000001", "13900000000", "a@example.com", "nobody@example.com", "user-a", "example"} {
			require.NotContains(t, key, private)
		}
		matched := false
		for name, pattern := range patterns {
			if pattern.MatchString(key) {
				seen[name]++
				matched = true
			}
		}
		require.True(t, matched, "unexpected key %q", key)
	}
	require.Equal(t, map[string]int{"phone gate": 2, "mail gate": 2, "per-client limit": 1, "challenge context": 2}, seen)
}

// The cooldown also works without Redis, and survives losing it.
func TestUniformSendGateWithoutSharedStore(t *testing.T) {
	t.Run("process-local store", func(t *testing.T) {
		d := newUniformDeployment(t)
		SetRateLimitStore(nil)
		require.Equal(t, fiber.StatusOK, d.send("phone=13900000000").status)
		reply := d.send("phone=13900000000")
		require.Equal(t, fiber.StatusTooManyRequests, reply.status)
		require.InDelta(t, 60, jsonFields(t, reply.body)["next_resend_in"], 1)
	})
	t.Run("Redis outage", func(t *testing.T) {
		d := newUniformDeployment(t)
		d.redis.Close()
		require.Equal(t, fiber.StatusOK, d.send("phone=13900000000").status)
		require.Equal(t, fiber.StatusTooManyRequests, d.send("phone=13900000000").status)
	})
}

func TestUniformDigestKeyDerivation(t *testing.T) {
	secrets := []*config.EnvVariable{
		&config.SessionExchangeSecret, &config.HeraldHMACSecret, &config.HeraldAPIKey,
		&config.WardenHMACSecret, &config.WardenAPIKey,
	}
	for _, variable := range secrets {
		setConfigValue(t, variable, "")
	}
	t.Cleanup(resetUniformDigestKeyForTesting)
	derived := func(secret string) []byte {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte("stargate/uniform-identifier"))
		return mac.Sum(nil)
	}

	// Set the secrets from the last to the first: each one takes over from
	// those after it.
	for index := len(secrets) - 1; index >= 0; index-- {
		secrets[index].Value = "secret-" + strconv.Itoa(index)
		resetUniformDigestKeyForTesting()
		require.Equal(t, derived("secret-"+strconv.Itoa(index)), uniformDigestKey(), secrets[index].Name)
	}
	require.NotEqual(t, []byte("secret-0"), uniformDigestKey(), "the secret itself is not the key")

	// Derived once: a changed setting is not picked up by a running process.
	config.SessionExchangeSecret.Value = "changed"
	require.Equal(t, derived("secret-0"), uniformDigestKey())

	// Without any secret the key is random and private to the process.
	for _, variable := range secrets {
		variable.Value = ""
	}
	resetUniformDigestKeyForTesting()
	first := append([]byte(nil), uniformDigestKey()...)
	require.Len(t, first, sha256.Size)
	require.NotEqual(t, make([]byte, sha256.Size), first)
	require.Equal(t, first, uniformDigestKey())
	resetUniformDigestKeyForTesting()
	require.NotEqual(t, first, uniformDigestKey())

	require.Regexp(t, `^[0-9a-f]{64}$`, uniformDigest("phone", "13800000001"))
	require.Equal(t, uniformDigest("phone", "13800000001"), uniformDigest("phone", "13800000001"))
	require.NotEqual(t, uniformDigest("phone", "13800000001"), uniformDigest("mail", "13800000001"))
	require.NotEqual(t, uniformDigest("ab", "c"), uniformDigest("a", "bc"))
	previous := uniformDigest("phone", "13800000001")
	resetUniformDigestKeyForTesting()
	require.NotEqual(t, previous, uniformDigest("phone", "13800000001"), "the digest depends on the key")
}

type failingEntropy struct{}

func (failingEntropy) Read([]byte) (int, error) { return 0, errors.New("entropy unavailable") }

// Without entropy no handle can be minted. That is decided before Herald is
// asked, so the answer is the same for every identifier.
func TestUniformHandleEntropyFailureAnswersTheSameForEveryIdentifier(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	previous := uniformEntropy
	uniformEntropy = failingEntropy{}
	t.Cleanup(func() { uniformEntropy = previous })

	var replies []uniformReply
	for _, phone := range []string{"13900000000", "13800000001"} {
		d.reset()
		replies = append(replies, d.send("phone="+phone+"&deliver_via=sms"))
	}
	require.Equal(t, replies[0], replies[1])
	require.Equal(t, fiber.StatusInternalServerError, replies[0].status)
	require.Equal(t, `{"message":"Failed to send verification code","reason":"unknown_error","success":false}`, replies[0].body)
	require.Zero(t, d.herald.total())
}

// The debug code is a credential for the listed user. Returning it would mark
// the identifier as listed, so uniform mode never relays it.
func TestUniformSendNeverRelaysTheDebugCode(t *testing.T) {
	d := newUniformDeployment(t)
	setConfigValue(t, &config.Debug, "true")
	setConfigValue(t, &config.CookieSecure, "false")
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))

	require.NotContains(t, d.send("phone=13800000001&deliver_via=sms").body, "debug_code")

	d.reset()
	config.LoginUniformResponses.Value = "false"
	require.Equal(t, uniformTestCode, jsonFields(t, d.send("phone=13800000001&deliver_via=sms").body)["debug_code"], "the default mode relays it under the same settings")
}

// In uniform mode challenge_id is a Stargate handle, which Herald's TOTP
// verification does not know.
func TestUniformTOTPDoesNotForwardTheHandle(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	d.herald.enrollTOTP("user-a")
	handle := d.sendHandle("phone=13800000001&deliver_via=sms")
	form := "phone=13800000001&use_otp=true&otp_code=" + uniformTestCode + "&challenge_id=" + handle

	reply := d.login(form)
	require.Equal(t, fiber.StatusOK, reply.status, reply.body)
	calls := d.herald.callsTo("/v1/totp/verify", "user-a")
	require.Len(t, calls, 1)
	require.Empty(t, calls[0].challengeID)

	config.LoginUniformResponses.Value = "false"
	reply = d.login(form)
	require.Equal(t, fiber.StatusOK, reply.status, reply.body)
	calls = d.herald.callsTo("/v1/totp/verify", "user-a")
	require.Len(t, calls, 2)
	require.Equal(t, handle, calls[1].challengeID, "the default mode forwards challenge_id as before")
}

// The audit reason of a failed send in uniform mode is the one the default
// handler records for the same error.
func TestChallengeCreationFailureReasonMatchesTheDefaultHandler(t *testing.T) {
	for _, failure := range []error{
		&herald.HeraldError{StatusCode: 0, Reason: "connection_failed"},
		&herald.HeraldError{StatusCode: 502, Reason: "connection_failed"},
		&herald.HeraldError{StatusCode: 429, Reason: "resend_cooldown"},
		&herald.HeraldError{StatusCode: 403, Reason: "user_locked"},
		&herald.HeraldError{StatusCode: 409, Reason: "idempotency_conflict"},
		&herald.HeraldError{StatusCode: 500, Reason: "internal_error"},
		errors.New("failed to decode response"),
	} {
		ctx, app := createTestContext("POST", "/_send_verify_code", nil, "")
		require.NoError(t, handleChallengeCreationError(ctx, failure, "user-1", "sms", "13800000001"))
		require.Equal(t, jsonFields(t, string(ctx.Response().Body()))["reason"], challengeCreationFailureReason(failure), failure.Error())
		app.ReleaseCtx(ctx)
	}
}

// The two uniform messages follow the language of the request, and stay the
// same for a listed and an unlisted identifier in each language.
func TestUniformMessagesFollowTheRequestLanguage(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	for _, lang := range []i18n.Language{i18n.LangEN, i18n.LangZH, i18n.LangFR, i18n.LangDE, i18n.LangIT, i18n.LangJA, i18n.LangKO} {
		t.Run(string(lang), func(t *testing.T) {
			var sends, logins []uniformReply
			for _, phone := range []string{"13900000000", "13800000001"} {
				d.reset()
				sent, handle := maskHandle(t, d.post("/_send_verify_code?lang="+string(lang), "phone="+phone+"&deliver_via=sms", acceptJSON))
				sends = append(sends, sent)
				logins = append(logins, d.post("/_login?lang="+string(lang), "auth_method=warden&phone="+phone+"&challenge_id="+handle+"&verify_code=000000", acceptJSON))
			}
			require.Equal(t, sends[0], sends[1])
			require.Equal(t, logins[0], logins[1])
			require.Equal(t, i18n.TWithLang(lang, "success.verify_code_sent_uniform"), jsonFields(t, sends[0].body)["message"])
			require.Equal(t, i18n.TWithLang(lang, "error.login_failed_uniform"), jsonFields(t, logins[0].body)["error"])
		})
	}
}

// Password sign-in carries no identifier and is not part of uniform mode.
func TestUniformModeLeavesPasswordSignInUnchanged(t *testing.T) {
	d := newUniformDeployment(t)
	setConfigValue(t, &config.Passwords, "plaintext:test123")

	wrong := d.post("/_login", "auth_method=password&password=nope&phone=13900000000", acceptJSON)
	require.Equal(t, fiber.StatusUnauthorized, wrong.status)
	require.Equal(t, `{"code":401,"error":"Invalid password","reason":"authentication_failed"}`, wrong.body)

	right := d.post("/_login", "auth_method=password&password=test123", acceptJSON)
	require.Equal(t, fiber.StatusOK, right.status, right.body)
	require.Contains(t, right.headers, "Set-Cookie: stargate_session_id=")
	require.Zero(t, d.warden.callCount())
	require.Zero(t, d.herald.total())
}

// The verified-email declaration is bound to the challenge the user redeemed.
// It must survive the handle standing in for the Herald challenge id.
func TestUniformHandleKeepsVerifiedEmailEvidence(t *testing.T) {
	d := newUniformDeployment(t)
	d.warden.add(activeUser("user-a", "13800000001", "a@example.com"))
	handle := d.sendHandle("mail=a@example.com&deliver_via=email")

	reply := d.login("mail=a@example.com&challenge_id=" + handle + "&verify_code=" + uniformTestCode)
	require.Equal(t, fiber.StatusOK, reply.status, reply.body)
	match := regexp.MustCompile(`Set-Cookie: stargate_session_id=([^;]+)`).FindStringSubmatch(reply.headers)
	require.Len(t, match, 2, reply.headers)
	require.Equal(t, "a@example.com", reauthSessionValue(t, d.sessions, match[1], "verified_email_address"))
	require.Equal(t, "user-a", reauthSessionValue(t, d.sessions, match[1], "verified_email_subject"))
}
