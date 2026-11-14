package auth

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"

	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/testsupport"
	"github.com/soulteary/warden/pkg/warden"
)

// globalState is the package-level state that tests change and that t.Setenv
// does not put back: config.Initialize copies the environment into the config
// package's variables, and this package keeps a logger and a Warden client
// that InitWardenClient builds from them once.
type globalState struct {
	config              testsupport.ConfigState
	logger              *logger.Logger
	wardenClient        *warden.Client
	wardenClientInitErr error
}

func captureGlobalState() globalState {
	return globalState{
		config:              testsupport.CaptureConfig(),
		logger:              log,
		wardenClient:        wardenClient,
		wardenClientInitErr: wardenClientInitErr,
	}
}

func (state globalState) restore() {
	state.config.Restore()
	log = state.logger
	wardenClient = state.wardenClient
	wardenClientInitErr = state.wardenClientInitErr
	wardenClientInit = sync.Once{}
	if state.wardenClient != nil || state.wardenClientInitErr != nil {
		// A sync.Once cannot be read, so a completed initialisation is
		// recognised by its result.
		wardenClientInit.Do(func() {})
	}
}

// isolateGlobalState puts the package-level state back when the test ends.
// Call it before the first t.Setenv, config.Initialize, InitWardenClient or
// assignment to that state. Cleanups run last in, first out, so the state
// comes back after the cleanups the test registers later have run.
func isolateGlobalState(t testing.TB) {
	t.Helper()
	state := captureGlobalState()
	t.Cleanup(state.restore)
}

// globalStateChanges names what differs between two states, in a fixed
// order. The logger is restored by isolateGlobalState but not compared:
// replacing it with an equal one is harmless and cannot be told apart from a
// different one.
func globalStateChanges(before, after globalState) []string {
	changes := testsupport.ConfigChanges(before.config, after.config)
	if before.wardenClient != after.wardenClient {
		changes = append(changes, "Warden client")
	}
	if fmt.Sprint(before.wardenClientInitErr) != fmt.Sprint(after.wardenClientInitErr) {
		changes = append(changes, "Warden client error")
	}
	sort.Strings(changes)
	return changes
}

func TestMain(m *testing.M) {
	_ = os.Setenv("WARDEN_URL", "http://warden.test")
	// Functions of this package log through the package logger, which the
	// application sets in InitWardenClient before it serves requests.
	log = testLogger()

	initial := captureGlobalState()
	code := m.Run()
	// A test that leaves this state changed makes the tests after it depend on
	// the order they run in.
	if changes := globalStateChanges(initial, captureGlobalState()); code == 0 && len(changes) > 0 {
		fmt.Fprintf(os.Stderr, "FAIL: package-level state was not restored when the tests ended: %s\nCall isolateGlobalState(t) at the start of the test that changes it.\n", strings.Join(changes, ", "))
		code = 1
	}
	os.Exit(code)
}
