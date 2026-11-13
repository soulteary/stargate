package handlers

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"

	"github.com/soulteary/herald/pkg/herald"
	secure "github.com/soulteary/secure-kit/v2"
	"github.com/soulteary/stargate/src/internal/auditlog"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/stargate/src/internal/metrics"
	"github.com/soulteary/tracing-kit/v2"
	"github.com/soulteary/warden/pkg/warden"
)

// sendVerifyCodeUniform answers POST /_send_verify_code in uniform mode. It
// runs after the request-shape checks of SendVerifyCodeAPI.
//
// Every response is either decided before the directory lookup or does not
// depend on its result: a request that passes the configuration checks and
// the resend cooldown is answered 200 with a fresh handle whether or not a
// code was sent. What happened is recorded in the log and the audit log.
func sendVerifyCodeUniform(ctx fiber.Ctx, spanCtx context.Context, userPhone, userMail string) error {
	// A phone number, when supplied, is the only identifier. With the default
	// fallback the code would reach a listed mail only when the phone is not
	// listed, and so tell the owner of that mail which phones are listed.
	if userPhone != "" {
		userMail = ""
	}

	// Answers that depend only on configuration come before the lookup.
	deliverVia := ctx.FormValue("deliver_via")
	if (deliverVia == "sms" && !config.LoginSMSEnabled.ToBool()) ||
		(deliverVia == "email" && !config.LoginEmailEnabled.ToBool()) {
		return verificationSelectionError(ctx, deliverVia, "channel_disabled")
	}
	client := getHeraldClient()
	if client == nil {
		if config.HeraldTOTPEnabled.ToBool() {
			return sendVerifyCodeErrorJSON(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_use_otp"), "connection_failed")
		}
		return sendVerifyCodeErrorJSON(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_retry"), "connection_failed")
	}

	// The cooldown applies to the identifier as submitted. A refused request
	// reaches neither Warden nor Herald.
	if wait := uniformSendGate(ctx, userPhone, userMail); wait > 0 {
		ctx.Set(fiber.HeaderRetryAfter, strconv.Itoa(wait))
		return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"success": false, "message": i18n.T(ctx, "error.rate_limited_retry"), "reason": "rate_limited", "next_resend_in": wait,
		})
	}

	userInfo, err := uniformDirectoryLookup(spanCtx, userPhone, userMail)
	if err != nil {
		return sendVerifyCodeErrorJSON(ctx, fiber.StatusServiceUnavailable, i18n.T(ctx, "error.herald_unavailable_retry"), "service_unavailable")
	}

	handle, err := newUniformHandle()
	if err != nil {
		log.Error().Err(err).Msg("Failed to mint a challenge handle")
		return sendVerifyCodeErrorJSON(ctx, fiber.StatusInternalServerError, i18n.T(ctx, "error.send_verify_code_failed"), "unknown_error")
	}

	if userInfo == nil {
		log.Warn().Str("phone", secure.MaskPhone(userPhone)).Str("mail", secure.MaskEmail(userMail)).Msg("User not found in Warden or not active")
	} else {
		// challenge_id is the handle the client holds for this identifier, if
		// any. It never changes the response.
		deliverUniformChallenge(ctx, spanCtx, client, userInfo, deliverVia, handle, ctx.FormValue("challenge_id"))
	}

	ctx.Set("Content-Type", "application/json")
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":        true,
		"message":        i18n.T(ctx, "success.verify_code_sent_uniform"),
		"challenge_id":   handle,
		"next_resend_in": int(math.Ceil(uniformResendCooldown().Seconds())),
	})
}

// uniformSendGate applies the resend cooldown to the identifier of the request:
// the phone number, or the mail when no phone number is supplied. It returns
// the number of seconds to wait while the window of that identifier is open,
// or zero when the request may proceed.
//
// The outcome depends only on the identifier and on earlier requests. The
// store keeps a fixed window: it starts with the first request and a refused
// request does not extend it.
func uniformSendGate(ctx fiber.Ctx, phone, mail string) int {
	kind, value := "phone", phone
	if value == "" {
		kind, value = "mail", strings.ToLower(strings.TrimSpace(mail))
	}
	if value == "" {
		return 0
	}
	count, remaining, err := getRateLimitStore().Incr(ctx.Context(), "uniform-send:"+kind+":"+uniformDigest(kind, value), uniformResendCooldown())
	if err != nil && log != nil {
		// The store already fell back to process-local counting, so the
		// cooldown still applies; only cross-replica sharing is degraded.
		log.Warn().Err(err).Msg("Shared rate-limit store is unavailable, falling back to a per-replica resend cooldown")
	}
	if count <= 1 {
		return 0
	}
	seconds := int(math.Ceil(remaining.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	return seconds
}

// deliverUniformChallenge asks Herald to send a code to a listed user and
// binds handle to the resulting challenge. It never writes to the response:
// the caller answers the same way whatever happens here.
//
// A handle is bound either to the challenge its own request created or to
// the challenge of a handle the requester presented. When Herald does not
// create a challenge (cooldown, limit, lock, outage), presented names the
// handle the client already holds; see bindToPresentedChallenge.
func deliverUniformChallenge(ctx fiber.Ctx, spanCtx context.Context, client *herald.Client, userInfo *warden.AllowListUser, deliverVia, handle, presented string) {
	userID := userInfo.UserID
	if userID == "" {
		userID = generateUserID(userInfo.Phone, userInfo.Mail)
	}
	channel, destination, selectionReason := selectVerificationDestination(userInfo, deliverVia)
	if selectionReason != "" {
		log.Warn().Str("user_id", userID).Str("reason", selectionReason).Msg("Verification code not sent: no usable destination")
		return
	}
	binding := challengeContext{ChallengeID: handle, UserID: userID, Channel: channel, Destination: destination}

	heraldCtx, heraldSpan := tracing.StartSpan(spanCtx, "herald.create_challenge")
	defer heraldSpan.End()
	heraldSpan.SetAttributes(
		attribute.String("herald.user_id", userID),
		attribute.String("herald.channel", channel),
		attribute.String("herald.purpose", "login"),
	)
	// The client's Idempotency-Key is not forwarded: Herald's answer to a
	// reused key would depend on whether the identifier is listed.
	startTime := time.Now()
	createResp, _, err := client.CreateChallengeWithMetadata(heraldCtx, &herald.CreateChallengeRequest{
		UserID:      userID,
		Channel:     channel,
		Destination: destination,
		Purpose:     "login",
		Locale:      verificationLocale(ctx),
		ClientIP:    ctx.IP(),
		UA:          ctx.Get("User-Agent"),
	})
	if err != nil || createResp == nil {
		if err == nil {
			err = errors.New("empty challenge response")
		}
		tracing.RecordError(heraldSpan, err)
		log.Error().Err(err).Msg("Failed to create challenge")
		auditlog.LogVerifyCodeSend(ctx.Context(), userID, "", channel, destination, ctx.IP(), false, challengeCreationFailureReason(err))
		bindToPresentedChallenge(ctx.Context(), binding, presented)
		return
	}

	metrics.RecordHeraldCall("create_challenge", "success", time.Since(startTime))
	auditlog.LogVerifyCodeSend(ctx.Context(), userID, createResp.ChallengeID, channel, destination, ctx.IP(), true, "")
	heraldSpan.SetAttributes(
		attribute.String("herald.challenge_id", createResp.ChallengeID),
		attribute.Int("herald.expires_in", createResp.ExpiresIn),
		attribute.String("herald.result", "success"),
	)
	binding.HeraldChallengeID = createResp.ChallengeID
	binding.ExpiresAt = uniformNow().Add(time.Duration(createResp.ExpiresIn) * time.Second).Unix()
	if err := bindChallengeContext(ctx.Context(), binding, createResp.ExpiresIn); err != nil {
		log.Error().Err(err).Str("challenge_id", createResp.ChallengeID).Msg("Failed to persist challenge audit context")
	}
}

// maxPresentedHandleLength bounds the challenge_id a send request may present.
// A handle minted by newUniformHandle is 25 bytes long.
const maxPresentedHandleLength = 128

// bindToPresentedChallenge binds the handle of binding to the challenge behind
// presented, the handle the client sent along with its request. Holding that
// handle is the proof that the client was already given this challenge, so the
// new handle grants nothing the client did not have.
//
// The presented handle must name a login context that was created with
// uniform responses, whose challenge is still valid, and that belongs to the
// user, channel and destination this request resolved. Otherwise nothing is
// bound, and the new handle fails like any other unknown handle.
func bindToPresentedChallenge(ctx context.Context, binding challengeContext, presented string) {
	if presented == "" || len(presented) > maxPresentedHandleLength {
		return
	}
	held, found, err := getChallengeContextStore().Get(ctx, presented)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load the presented challenge context")
		return
	}
	remaining := int(held.ExpiresAt - uniformNow().Unix())
	if !found || held.Purpose != "" || held.HeraldChallengeID == "" || remaining < 1 ||
		held.UserID != binding.UserID || held.Channel != binding.Channel ||
		!sameVerificationDestination(binding.Channel, held.Destination, binding.Destination) {
		return
	}
	binding.HeraldChallengeID, binding.ExpiresAt = held.HeraldChallengeID, held.ExpiresAt
	if err := bindChallengeContext(ctx, binding, remaining); err != nil {
		log.Error().Err(err).Str("challenge_id", held.HeraldChallengeID).Msg("Failed to persist challenge audit context")
	}
}

// challengeCreationFailureReason classifies a failed challenge creation with
// the reasons handleChallengeCreationErrorWithMetadata writes to the audit log.
func challengeCreationFailureReason(err error) string {
	var heraldErr *herald.HeraldError
	if !errors.As(err, &heraldErr) {
		return "unknown_error"
	}
	switch {
	case heraldErr.StatusCode == 0 || heraldErr.Reason == "connection_failed":
		return "connection_failed"
	case heraldErr.StatusCode == http.StatusTooManyRequests:
		return "rate_limited"
	default:
		return "provider_error"
	}
}
