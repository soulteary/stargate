package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/config"
)

const exchangeStateTTL = 10 * time.Minute
const exchangeStateParameter = "exchange_state"

var exchangeStateAAD = []byte("stargate:exchange-state:v1")
var exchangeTicketAAD = []byte("stargate:exchange-ticket:v2")

type exchangeStateClaims struct {
	Audience     string `json:"aud"`
	Challenge    string `json:"challenge"`
	RedirectPath string `json:"return"`
	Proto        string `json:"proto"`
	ExpiresAt    int64  `json:"exp"`
}

func exchangeChallenge(nonce string) string {
	if len(nonce) < 26 || len(nonce) > 128 {
		return ""
	}
	digest := sha256.Sum256([]byte(nonce))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func validExchangeChallenge(challenge string) bool {
	raw, err := base64.RawURLEncoding.Strict().DecodeString(challenge)
	return err == nil && len(raw) == sha256.Size
}

func sealExchangeValue(value any, aad []byte) (string, error) {
	aead, err := sessionExchangeAEAD()
	if err != nil {
		return "", err
	}
	plain, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(append(nonce, aead.Seal(nil, nonce, plain, aad)...)), nil
}

func openExchangeValue(token string, aad []byte, value any) error {
	if len(token) > 8192 {
		return errors.New("invalid exchange value")
	}
	aead, err := sessionExchangeAEAD()
	if err != nil {
		return err
	}
	payload, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(payload) <= aead.NonceSize() {
		return errors.New("invalid exchange value")
	}
	plain, err := aead.Open(nil, payload[:aead.NonceSize()], payload[aead.NonceSize():], aad)
	if err != nil {
		return errors.New("invalid exchange value")
	}
	if err := json.Unmarshal(plain, value); err != nil {
		return errors.New("invalid exchange value")
	}
	return nil
}

func validExchangeState(state exchangeStateClaims) bool {
	return normalizeExchangeAudience(state.Audience) == state.Audience && state.Audience != "" && validExchangeChallenge(state.Challenge) &&
		config.ValidateSessionExchangeRedirectPath(config.EnvVariable{Value: state.RedirectPath}) && (state.Proto == "http" || state.Proto == "https") && time.Now().Unix() < state.ExpiresAt
}

func exchangeCookieName(requestState bool) string {
	name := "stargate_exchange_nonce"
	if requestState {
		name = "stargate_exchange_request"
	}
	if config.CookieSecure.ToBool() {
		name = "__Host-" + name
	}
	return name
}

func setExchangeCookie(ctx fiber.Ctx, requestState bool, value string, clear bool) {
	cookie := &fiber.Cookie{Name: exchangeCookieName(requestState), Value: value, Path: "/", Secure: config.CookieSecure.ToBool(), HTTPOnly: true, SameSite: "Lax", Expires: time.Now().Add(exchangeStateTTL), MaxAge: int(exchangeStateTTL.Seconds())}
	if clear {
		cookie.Value = ""
		cookie.Expires = time.Unix(1, 0)
		cookie.MaxAge = -1
	}
	ctx.Cookie(cookie)
}

// startSessionExchange runs on the target origin. Query parameters never supply
// the browser nonce: a new host-only cookie and signed request are minted here.
func startSessionExchange(ctx fiber.Ctx) error {
	target, err := ValidateCallbackHost(GetForwardedHost(ctx))
	if err != nil {
		return SendErrorResponse(ctx, fiber.StatusBadRequest, "invalid exchange target")
	}
	proto := GetForwardedProto(ctx)
	if config.CookieSecure.ToBool() && proto != "https" {
		return SendErrorResponse(ctx, fiber.StatusBadRequest, "secure session exchange requires HTTPS")
	}
	state := exchangeStateClaims{Audience: normalizeExchangeAudience(target), RedirectPath: config.SessionExchangeRedirectPath.String(), Proto: proto, ExpiresAt: time.Now().Add(exchangeStateTTL).Unix()}
	// Generate the cookie separately; only its hash is carried through the auth host.
	nonce := rand.Text()
	state.Challenge = exchangeChallenge(nonce)
	if !validExchangeState(state) {
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, "invalid session exchange configuration")
	}
	token, err := sealExchangeValue(state, exchangeStateAAD)
	if err != nil {
		return SendErrorResponse(ctx, fiber.StatusInternalServerError, "session exchange is not configured")
	}
	setExchangeCookie(ctx, false, nonce, false)
	destination := url.URL{Scheme: proto, Host: config.AuthHost.String(), Path: "/_login"}
	query := destination.Query()
	query.Set("callback", target)
	query.Set(exchangeStateParameter, token)
	destination.RawQuery = query.Encode()
	return ctx.Redirect().Status(fiber.StatusFound).To(destination.String())
}

func bootstrapExchangeURL(ctx fiber.Ctx, target string) string {
	destination := url.URL{Scheme: GetForwardedProto(ctx), Host: target, Path: "/_session_exchange"}
	query := destination.Query()
	query.Set("start", "1")
	destination.RawQuery = query.Encode()
	return destination.String()
}

// requestedExchangeState accepts only sealed target-issued requests. The target
// cookie match, not the auth host's receipt of this value, proves browser ownership.
func requestedExchangeState(ctx fiber.Ctx, target string) (exchangeStateClaims, string, error) {
	raw := ctx.FormValue(exchangeStateParameter)
	if raw == "" {
		raw = ctx.Query(exchangeStateParameter)
	}
	explicit := raw != ""
	if raw == "" {
		raw = ctx.Cookies(exchangeCookieName(true))
	}
	if raw == "" {
		return exchangeStateClaims{}, "", nil
	}
	var state exchangeStateClaims
	if err := openExchangeValue(raw, exchangeStateAAD, &state); err != nil {
		if !explicit {
			return exchangeStateClaims{}, "", nil
		}
		return exchangeStateClaims{}, "", err
	}
	if state.Audience != normalizeExchangeAudience(target) || !validExchangeState(state) {
		if !explicit || time.Now().Unix() >= state.ExpiresAt {
			return exchangeStateClaims{}, "", nil
		}
		return exchangeStateClaims{}, "", errors.New("invalid session exchange request")
	}
	if explicit {
		setExchangeCookie(ctx, true, raw, false)
	}
	return state, raw, nil
}

func sessionExchangeDestination(ctx fiber.Ctx, sessionID, target string) (string, error) {
	state, raw, err := requestedExchangeState(ctx, target)
	if err != nil {
		return "", err
	}
	if raw == "" {
		return bootstrapExchangeURL(ctx, target), nil
	}
	ticket, err := createSessionExchangeTicket(sessionID, state)
	if err != nil {
		return "", err
	}
	setExchangeCookie(ctx, true, "", true)
	destination := url.URL{Scheme: state.Proto, Host: state.Audience, Path: "/_session_exchange"}
	query := destination.Query()
	query.Set("ticket", ticket)
	destination.RawQuery = query.Encode()
	return destination.String(), nil
}

func exchangeNonceCookie(ctx fiber.Ctx) string {
	return ctx.Cookies(exchangeCookieName(false))
}
