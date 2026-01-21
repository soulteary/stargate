package utils

import (
	"strings"
)

// MaskPhone masks a phone number, showing only the first 3 and last 4 digits.
// Example: "13812345678" -> "138****5678"
// If the phone number is shorter than 7 characters, it returns "****"
func MaskPhone(phone string) string {
	if phone == "" {
		return ""
	}

	phone = strings.TrimSpace(phone)
	length := len(phone)

	// If phone is too short, mask everything
	if length < 7 {
		return "****"
	}

	// Show first 3 and last 4 digits
	if length <= 7 {
		return phone[:3] + "****"
	}

	return phone[:3] + strings.Repeat("*", length-7) + phone[length-4:]
}

// MaskEmail masks an email address, showing only the first character of the local part
// and the domain.
// Example: "user@example.com" -> "u***@example.com"
// Example: "test.user@example.com" -> "t***@example.com"
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}

	email = strings.TrimSpace(email)
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		// Invalid email format, mask everything
		return "***@***"
	}

	localPart := parts[0]
	domain := parts[1]

	// If local part is empty, mask it
	if localPart == "" {
		return "***@" + domain
	}

	// Show first character and mask the rest
	if len(localPart) == 1 {
		return localPart + "***@" + domain
	}

	return string(localPart[0]) + strings.Repeat("*", len(localPart)-1) + "@" + domain
}
