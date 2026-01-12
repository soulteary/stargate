package secure

import (
	"crypto/sha512"
	"encoding/hex"
	"strings"
)

type SHA512Resolver struct{}

func (s *SHA512Resolver) Check(h string, password string) bool {
	// Compare case-insensitively since GetValidPasswords converts hashes to uppercase
	// but GetSHA512Hash returns lowercase
	return strings.EqualFold(h, GetSHA512Hash(password))
}

func GetSHA512Hash(text string) string {
	hasher := sha512.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
