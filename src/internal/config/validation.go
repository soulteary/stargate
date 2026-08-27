package config

import (
	"net"
	"strings"
	"time"

	"github.com/soulteary/cli-kit/env"
	"github.com/soulteary/cli-kit/validator"
	secure "github.com/soulteary/secure-kit"
	"github.com/soulteary/stargate/src/internal/i18n"
)

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

func (v *EnvVariable) Validate() error {
	if v.Trimmed {
		v.Value = env.GetTrimmed(v.Name, v.DefaultValue)
	} else {
		v.Value = env.Get(v.Name, v.DefaultValue)
	}

	if v.Required && v.Value == "" {
		return NewValidationError(v.Name, i18n.TStatic("error.config_required_not_set"), v.PossibleValues)
	}

	if !v.Validator(*v) {
		return NewValidationError(v.Name, v.SafeValue(), v.PossibleValues)
	}

	return nil
}

var (
	SupportedAlgorithms = map[string]secure.HashResolver{
		"plaintext": &secure.PlaintextResolver{},
		"bcrypt":    &secure.BcryptResolver{},
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
	ValidateStrictPossibleValues = func(v EnvVariable) bool {
		// Use cli-kit validator for case-sensitive enum validation
		// Skip validation if PossibleValues contains "*" (any value allowed)
		if len(v.PossibleValues) > 0 && v.PossibleValues[0] == "*" {
			return true
		}
		if err := validator.ValidateEnum(v.Value, v.PossibleValues, true); err != nil {
			return false
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

type ValidationError struct {
	KeyName        string
	AcceptedValues []string
	ProvidedValue  string
}

func NewValidationError(keyName, providedValue string, acceptedValues []string) *ValidationError {
	return &ValidationError{
		KeyName:        keyName,
		AcceptedValues: acceptedValues,
		ProvidedValue:  providedValue,
	}
}

func (e ValidationError) Error() string {
	return e.String()
}

func (e ValidationError) String() string {
	if len(e.AcceptedValues) > 0 && e.AcceptedValues[0] != "*" {
		return i18n.TfStatic("error.config_invalid_values", e.KeyName, e.ProvidedValue, e.AcceptedValues)
	}
	return i18n.TfStatic("error.config_invalid", e.KeyName, e.ProvidedValue)
}
