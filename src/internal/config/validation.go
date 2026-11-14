package config

import (
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/soulteary/cli-kit/env"
	"github.com/soulteary/cli-kit/validator"
	secure "github.com/soulteary/secure-kit/v2"
	"github.com/soulteary/secure-kit/v2/passwd"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// ValidateSessionExchangeRedirectPath keeps the configured landing page on the
// callback origin. Request-supplied redirect parameters are never used.
func ValidateSessionExchangeRedirectPath(v EnvVariable) bool {
	p := v.Value
	if !strings.HasPrefix(p, "/") || strings.Contains(p, "//") ||
		strings.ContainsAny(p, "\\%?#") || strings.IndexFunc(p, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) >= 0 {
		return false
	}
	// Do not send a consumed ticket back into the exchange or login endpoints.
	for _, segment := range strings.Split(p, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return p != "/_session_exchange" && p != "/_session_exchange/" &&
		p != "/_login" && p != "/_login/"
}

// ValidateUniformResendCooldown accepts a Go duration of at least one second.
// The handlers report the window to clients in whole seconds.
func ValidateUniformResendCooldown(v EnvVariable) bool {
	cooldown, err := time.ParseDuration(v.Value)
	return err == nil && cooldown >= time.Second
}

type EnvVariable struct {
	Name           string
	Required       bool
	DefaultValue   string
	Value          string
	PossibleValues []string
	Validator      func(v EnvVariable) bool
	Trimmed        bool // If true, use env.GetTrimmed instead of env.Get
	Sensitive      bool // If true, never expose the value in logs or validation errors
}

const redactedValue = "[REDACTED]"

// SafeValue returns a representation suitable for logs and errors.
func (v EnvVariable) SafeValue() string {
	if v.Sensitive && v.Value != "" {
		return redactedValue
	}
	return v.Value
}

func (v *EnvVariable) String() string {
	return v.Value
}

func (v *EnvVariable) ToBool() bool {
	return strings.ToLower(v.Value) == "true"
}

// ToDuration parses the value as a duration string (e.g., "5m", "1h", "30s")
// Returns the parsed duration, or 0 if parsing fails
func (v *EnvVariable) ToDuration() time.Duration {
	if v.Value == "" {
		return 0
	}
	duration, err := time.ParseDuration(v.Value)
	if err != nil {
		return 0
	}
	return duration
}

// ToInt parses the value as a base-10 integer.
// Returns fallback when the value is empty or cannot be parsed.
func (v *EnvVariable) ToInt(fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(v.Value))
	if err != nil {
		return fallback
	}
	return parsed
}

func (v *EnvVariable) Validate() error {
	if v.Trimmed {
		v.Value = env.GetTrimmed(v.Name, v.DefaultValue)
	} else {
		v.Value = env.Get(v.Name, v.DefaultValue)
	}

	if v.Required && v.Value == "" {
		return NewRequiredError(v.Name, v.PossibleValues)
	}

	if !v.Validator(*v) {
		return NewValidationError(v.Name, v.SafeValue(), v.PossibleValues)
	}

	return nil
}

var (
	SupportedAlgorithms = map[string]secure.HashResolver{
		"plaintext": &secure.PlaintextResolver{},
		"bcrypt":    &passwd.BcryptResolver{},
	}

	ValidateNotEmptyString = func(v EnvVariable) bool {
		return v.Value != ""
	}
	ValidateAny = func(v EnvVariable) bool {
		return true
	}
	ValidateIPOrCIDRList = func(v EnvVariable) bool {
		if strings.TrimSpace(v.Value) == "" {
			return true
		}
		for _, value := range strings.Split(v.Value, ",") {
			value = strings.TrimSpace(value)
			if value == "" {
				return false
			}
			if net.ParseIP(value) != nil {
				continue
			}
			if _, _, err := net.ParseCIDR(value); err != nil {
				return false
			}
		}
		return true
	}
	ValidateCaseInsensitivePossibleValues = func(v EnvVariable) bool {
		// Use cli-kit validator for case-insensitive enum validation
		// Skip validation if PossibleValues contains "*" (any value allowed)
		if len(v.PossibleValues) > 0 && v.PossibleValues[0] == "*" {
			return true
		}
		if err := validator.ValidateEnum(v.Value, v.PossibleValues, false); err != nil {
			return false
		}
		return true
	}

	ValidatePasswords = func(v EnvVariable) bool {
		_, _, ok := ParsePasswords(v.Value)
		return ok
	}

	// ValidatePasswordsOrEmpty allows empty value (for pure Warden deployment); otherwise same as ValidatePasswords.
	ValidatePasswordsOrEmpty = func(v EnvVariable) bool {
		if v.Value == "" {
			return true
		}
		return ValidatePasswords(v)
	}
)

// ParsePasswords is the single parser for the PASSWORDS contract used by both
// startup validation and authentication. Password values are opaque and may
// contain additional colons, so only the first colon separates the algorithm.
func ParsePasswords(raw string) (string, []string, bool) {
	if raw == "" {
		return "", nil, false
	}
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return "", nil, false
	}
	algorithm := strings.ToLower(strings.TrimSpace(parts[0]))
	if _, supported := SupportedAlgorithms[algorithm]; !supported {
		return "", nil, false
	}
	passwords := strings.Split(parts[1], "|")
	if len(passwords) == 0 {
		return "", nil, false
	}
	for _, password := range passwords {
		if password == "" {
			return "", nil, false
		}
	}
	return algorithm, passwords, true
}

// ValidationError reports one rejected setting. It describes exactly one of
// three cases: a value that is not accepted (ProvidedValue), a required
// setting that is missing (Missing), or a rule the setting breaks (Reason),
// such as a dependency on another setting.
type ValidationError struct {
	KeyName        string
	AcceptedValues []string
	ProvidedValue  string
	// Reason states the rule that is broken, as a predicate of the setting
	// ("requires WARDEN_ENABLED=true"). It is never a configured value.
	Reason string
	// Missing reports a required setting that is not set.
	Missing bool
}

// NewValidationError reports a provided value that is not accepted.
func NewValidationError(keyName, providedValue string, acceptedValues []string) *ValidationError {
	return &ValidationError{
		KeyName:        keyName,
		AcceptedValues: acceptedValues,
		ProvidedValue:  providedValue,
	}
}

// NewRuleError reports a setting that breaks a rule, for example one that
// needs another setting or a minimum length. The reason is rendered as the
// explanation, not as the value the operator provided.
func NewRuleError(keyName, reason string, acceptedValues []string) *ValidationError {
	return &ValidationError{
		KeyName:        keyName,
		AcceptedValues: acceptedValues,
		Reason:         reason,
	}
}

// NewRequiredError reports a required setting that is not set.
func NewRequiredError(keyName string, acceptedValues []string) *ValidationError {
	return &ValidationError{
		KeyName:        keyName,
		AcceptedValues: acceptedValues,
		Missing:        true,
	}
}

func (e ValidationError) Error() string {
	return e.String()
}

func (e ValidationError) String() string {
	if e.Missing {
		return i18n.TfStatic("error.config_required", e.KeyName)
	}
	if e.Reason != "" {
		// The accepted values describe the setting's own format. They say
		// nothing about a rule that involves another setting, so they are
		// not listed here.
		return i18n.TfStatic("error.config_rule", e.KeyName, e.Reason)
	}
	if len(e.AcceptedValues) > 0 && e.AcceptedValues[0] != "*" {
		return i18n.TfStatic("error.config_invalid_values", e.KeyName, e.ProvidedValue, e.AcceptedValues)
	}
	return i18n.TfStatic("error.config_invalid", e.KeyName, e.ProvidedValue)
}
