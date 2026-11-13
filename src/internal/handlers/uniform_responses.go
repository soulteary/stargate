package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
)

// Uniform responses (LOGIN_UNIFORM_RESPONSES=true) make POST /_send_verify_code
// and Warden sign-in through POST /_login answer with the same status, headers
// and body whether or not the identifier is on the allow list.
//
// The mode unifies response content only. A request for a listed user still
// calls Herald, so response time is not equalized.

// uniformResponses reports whether the mode is on.
func uniformResponses() bool {
	return config.LoginUniformResponses.ToBool()
}

// uniformResendCooldown returns the resend window. Startup validation rejects
// a value below one second, so the fallback only covers callers that run
// without Initialize, such as unit tests.
func uniformResendCooldown() time.Duration {
	if cooldown := config.LoginUniformResendCooldown.ToDuration(); cooldown >= time.Second {
		return cooldown
	}
	return config.DefaultLoginUniformResendCooldown
}

// uniformNow is the clock for the expiry of Herald challenges. Tests replace it.
var uniformNow = time.Now

// uniformEntropy is the random source for challenge handles. Tests replace it.
var uniformEntropy io.Reader = rand.Reader

// uniformDigestContext separates the digest key from every other use of the
// secret it is derived from.
const uniformDigestContext = "stargate/uniform-identifier"

var uniformDigestKeyState struct {
	sync.Mutex
	key []byte
}

// uniformDigestKey returns the key for uniformDigest, derived once per
// process from the first configured service secret. Replicas that share that
// secret compute the same digests, which the shared rate-limit store relies
// on. Without any secret the key is random and private to this process, so
// the resend cooldown is then kept per replica.
func uniformDigestKey() []byte {
	uniformDigestKeyState.Lock()
	defer uniformDigestKeyState.Unlock()
	if uniformDigestKeyState.key != nil {
		return uniformDigestKeyState.key
	}
	for _, secret := range []string{
		config.SessionExchangeSecret.Value,
		config.HeraldHMACSecret.Value,
		config.HeraldAPIKey.Value,
		config.WardenHMACSecret.Value,
		config.WardenAPIKey.Value,
	} {
		if secret == "" {
			continue
		}
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(uniformDigestContext))
		uniformDigestKeyState.key = mac.Sum(nil)
		return uniformDigestKeyState.key
	}
	key := make([]byte, sha256.Size)
	_, _ = rand.Read(key) // crypto/rand.Read does not return an error
	uniformDigestKeyState.key = key
	return key
}

// resetUniformDigestKeyForTesting discards the derived key so that the next
// digest derives it again from the current configuration.
func resetUniformDigestKeyForTesting() {
	uniformDigestKeyState.Lock()
	defer uniformDigestKeyState.Unlock()
	uniformDigestKeyState.key = nil
}

// uniformDigest returns a keyed digest of parts for use in store keys, so that
// no identifier is written there. Each part is prefixed with its length,
// which keeps different part lists apart.
func uniformDigest(parts ...string) string {
	mac := hmac.New(sha256.New, uniformDigestKey())
	var size [8]byte
	for _, part := range parts {
		binary.BigEndian.PutUint64(size[:], uint64(len(part)))
		mac.Write(size[:])
		mac.Write([]byte(part))
	}
	return hex.EncodeToString(mac.Sum(nil))
}

// newUniformHandle mints the challenge_id a client receives in uniform mode.
// It has the shape of a Herald challenge id and is issued for every request,
// so it says nothing about the identifier. The Herald id, when there is one,
// stays in the challenge context stored under the handle.
func newUniformHandle() (string, error) {
	var raw [16]byte
	if _, err := io.ReadFull(uniformEntropy, raw[:]); err != nil {
		return "", err
	}
	return "ch_" + base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

// uniformDirectoryLookup returns the listed, active user for the identifier,
// or nil when there is none. An error means the lookup itself failed; callers
// answer that the same way for every identifier.
//
// Callers pass one identifier: a phone number, or a mail when no phone number
// was supplied. The directory is therefore asked exactly once per request.
func uniformDirectoryLookup(ctx context.Context, phone, mail string) (*warden.AllowListUser, error) {
	user, _, err := auth.LookupUserInfo(ctx, phone, mail)
	if err != nil && !errors.Is(err, auth.ErrUserUnavailable) {
		return nil, err
	}
	return user, nil
}
