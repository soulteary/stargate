package secure

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

type MD5Resolver struct{}

func (m *MD5Resolver) Check(h string, password string) bool {
	// Compare case-insensitively since GetValidPasswords converts hashes to uppercase
	// but GetMD5Hash returns lowercase
	return strings.EqualFold(h, GetMD5Hash(password))
}

func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
