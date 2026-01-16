package oidc

import (
	"testing"
	"github.com/MarvinJWendt/testza"
)

func TestNewProviderInvalidURL(t *testing.T) {
	_, err := NewProvider("invalid-url", "client-id", "client-secret", "")
	testza.AssertNotNil(t, err)
}

func TestNewProviderMissingFields(t *testing.T) {
	_, err := NewProvider("", "client-id", "client-secret", "")
	testza.AssertNotNil(t, err)
}
