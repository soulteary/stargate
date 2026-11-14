package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/handlers"
	"github.com/soulteary/stargate/src/internal/testsupport"
)

// globalState is the package-level state that tests change and that t.Setenv
// does not put back: config.Initialize copies the environment into the config
// package's variables, and initLogger replaces the logger of this package.
type globalState struct {
	config testsupport.ConfigState
	logger *logger.Logger
}

func captureGlobalState() globalState {
	return globalState{config: testsupport.CaptureConfig(), logger: log}
}

// isolateGlobalState puts the package-level state back when the test ends.
// Call it before the first t.Setenv, initLogger, initConfig,
// config.Initialize, createApp or assignment to that state. Cleanups run last
// in, first out, so the state comes back after the cleanups the test
// registers later have run.
//
// The Warden client, the Herald client and the rate-limit and
// challenge-context stores belong to other packages and cannot be read from
// here. They are discarded when the test ends, which is the state TestMain
// starts from. The ForwardAuth handler cannot be discarded from here; every
// test that serves a request builds its own with setupRoutes or createApp.
func isolateGlobalState(t testing.TB) {
	t.Helper()
	state := captureGlobalState()
	t.Cleanup(func() {
		state.config.Restore()
		log = state.logger
		auth.ResetWardenClientForTesting()
		handlers.ResetHeraldClientForTest()
		handlers.SetRateLimitStore(nil)
		handlers.SetChallengeContextStore(nil)
	})
}

func TestMain(m *testing.M) {
	// main sets the logger in initLogger before anything else runs. A test
	// that builds part of the application without calling initLogger gets
	// this one.
	log = testLoggerMain()

	initial := captureGlobalState()
	code := m.Run()
	// A test that leaves this state changed makes the tests after it depend on
	// the order they run in. The logger is restored by isolateGlobalState but
	// not compared: replacing it with an equal one is harmless and cannot be
	// told apart from a different one.
	if changes := testsupport.ConfigChanges(initial.config, testsupport.CaptureConfig()); code == 0 && len(changes) > 0 {
		fmt.Fprintf(os.Stderr, "FAIL: package-level state was not restored when the tests ended: %s\nCall isolateGlobalState(t) at the start of the test that changes it.\n", strings.Join(changes, ", "))
		code = 1
	}
	os.Exit(code)
}
