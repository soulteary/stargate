package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty phone",
			input:    "",
			expected: "",
		},
		{
			name:     "Short phone",
			input:    "123",
			expected: "****",
		},
		{
			name:     "Length 7 phone",
			input:    "1234567",
			expected: "123****",
		},
		{
			name:     "Normal phone",
			input:    "13812345678",
			expected: "138****5678",
		},
		{
			name:     "Long phone",
			input:    "138123456789",
			expected: "138*****6789",
		},
		{
			name:     "Phone with spaces",
			input:    " 13812345678 ",
			expected: "138****5678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskPhone(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty email",
			input:    "",
			expected: "",
		},
		{
			name:     "Invalid email (no @)",
			input:    "invalid",
			expected: "***@***",
		},
		{
			name:     "Invalid email (multiple @)",
			input:    "user@example.com@extra",
			expected: "***@***",
		},
		{
			name:     "Empty local part",
			input:    "@example.com",
			expected: "***@example.com",
		},
		{
			name:     "Single char local part",
			input:    "u@example.com",
			expected: "u***@example.com",
		},
		{
			name:     "Normal email",
			input:    "user@example.com",
			expected: "u***@example.com",
		},
		{
			name:     "Long local part",
			input:    "longuser@example.com",
			expected: "l*******@example.com",
		},
		{
			name:     "Email with spaces",
			input:    " user@example.com ",
			expected: "u***@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskEmail(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
