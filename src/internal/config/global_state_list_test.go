package config_test

import (
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/testsupport"
)

// The internal tests of the config package restore every setting after each
// test and need the list of settings for that. testsupport keeps the list and
// checks that it is complete.
func init() {
	config.SetConfigVariablesForTest(testsupport.ConfigVariables())
}
