package handlers

import (
	"strings"

	"github.com/soulteary/stargate/src/internal/auth"
)

func sameVerificationDestination(channel, left, right string) bool {
	left, right = strings.TrimSpace(left), strings.TrimSpace(right)
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	switch channel {
	case "email":
		a, validA := canonicalEvidenceEmail(left)
		b, validB := canonicalEvidenceEmail(right)
		return validA && validB && a == b
	case "sms":
		a, b := auth.NormalizePhone(left), auth.NormalizePhone(right)
		return auth.IsValidPhone(a) && auth.IsValidPhone(b) && a == b
	}
	return false
}

// Login challenges must have been issued by this gateway for the directory's
// current contact. Request-supplied identifiers are not trusted destinations.
func (identity *loginIdentity) matchesLoginChallenge(id string, binding challengeContext) bool {
	if id == "" || binding.ChallengeID != id || identity.userID == "" || binding.UserID != identity.userID {
		return false
	}
	channel, destination, reason := selectVerificationDestination(identity.wardenUserInfo, binding.Channel)
	return reason == "" && channel == binding.Channel &&
		sameVerificationDestination(channel, destination, binding.Destination)
}
