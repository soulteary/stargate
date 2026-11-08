package config

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var supportMailbox = regexp.MustCompile(`^[A-Za-z0-9._+\-]+@[A-Za-z0-9](?:[A-Za-z0-9.\-]*[A-Za-z0-9])?$`)

// ValidateLoginSupportURL accepts an optional HTTPS link or a single mailto
// mailbox without headers. It never contacts the configured destination.
func ValidateLoginSupportURL(v EnvVariable) bool {
	raw := v.Value
	if raw == "" {
		return true
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil || strings.IndexFunc(decoded, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return false
	}
	switch u.Scheme {
	case "https":
		return u.Hostname() != "" && u.Opaque == ""
	case "mailto":
		return u.Host == "" && u.RawQuery == "" && u.Fragment == "" && supportMailbox.MatchString(u.Opaque) && !strings.Contains(u.Opaque, "%")
	default:
		return false
	}
}
