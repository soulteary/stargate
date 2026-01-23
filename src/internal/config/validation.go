package config

import (
	"strings"

	"github.com/soulteary/cli-kit/env"
	"github.com/soulteary/cli-kit/validator"
	"github.com/soulteary/stargate/src/internal/i18n"
	"github.com/soulteary/stargate/src/internal/secure"
)

type EnvVariable struct {
	Name           string
	Required       bool
	DefaultValue   string
	Value          string
	PossibleValues []string
	Validator      func(v EnvVariable) bool
	Trimmed        bool // If true, use env.GetTrimmed instead of env.Get
}

func (v *EnvVariable) String() string {
	return v.Value
}

func (v *EnvVariable) ToBool() bool {
	return strings.ToLower(v.Value) == "true"
}

func (v *EnvVariable) Validate() error {
	if v.Trimmed {
		v.Value = env.GetTrimmed(v.Name, v.DefaultValue)
	} else {
		v.Value = env.Get(v.Name, v.DefaultValue)
	}

	if v.Required && v.Value == "" {
		return NewValidationError(v.Name, i18n.T("error.config_required_not_set"), v.PossibleValues)
	}

	if !v.Validator(*v) {
		return NewValidationError(v.Name, v.Value, v.PossibleValues)
	}

	return nil
}

var (
	SupportedAlgorithms = map[string]secure.HashResolver{
		"plaintext": &secure.PlaintextResolver{},
		"bcrypt":    &secure.BcryptResolver{},
		"md5":       &secure.MD5Resolver{},
		"sha512":    &secure.SHA512Resolver{},
	}

	ValidateNotEmptyString = func(v EnvVariable) bool {
		return v.Value != ""
	}
	ValidateAny = func(v EnvVariable) bool {
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
		// Schema: "algorithm:pass1|pass2|pass3"
		passwordsRaw := v.Value
		if passwordsRaw == "" {
			return false
		}
		parts := strings.Split(passwordsRaw, ":")
		if len(parts) < 2 {
			return false
		}
		algorithm := parts[0]
		passwords := strings.Split(parts[1], "|")

		algoSupported := false
		for possibleValue := range SupportedAlgorithms {
			if algorithm == possibleValue {
				algoSupported = true
				break
			}
		}
		if !algoSupported {
			return false
		}

		for _, password := range passwords {
			if password == "" {
				return false
			}
		}

		return true
	}
)

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
		return i18n.Tf("error.config_invalid_values", e.KeyName, e.ProvidedValue, e.AcceptedValues)
	}
	return i18n.Tf("error.config_invalid", e.KeyName, e.ProvidedValue)
}
