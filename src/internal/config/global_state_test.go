package config

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// configVariables is every setting of this package. The list is kept in the
// testsupport package, which imports this one, so a test file of this package
// cannot import it. global_state_list_test.go, in the external test package,
// hands the list over before TestMain runs.
var configVariables []*EnvVariable

// SetConfigVariablesForTest is called by the external test package.
func SetConfigVariablesForTest(variables []*EnvVariable) {
	configVariables = variables
}

// globalState is the package-level state that Initialize and the tests
// change. t.Setenv puts the environment back when a test ends, but not the
// values that Initialize copied from it.
type globalState struct {
	values        []string
	language      i18n.Language
	logger        *logger.Logger
	stepUpMatcher *StepUpMatcher
}

func captureGlobalState() globalState {
	state := globalState{
		values:        make([]string, len(configVariables)),
		language:      i18n.GetLanguage(),
		logger:        log,
		stepUpMatcher: stepUpMatcher,
	}
	for i, variable := range configVariables {
		state.values[i] = variable.Value
	}
	return state
}

func (state globalState) restore() {
	for i, variable := range configVariables {
		variable.Value = state.values[i]
	}
	i18n.SetLanguage(state.language)
	log = state.logger
	stepUpMatcher = state.stepUpMatcher
}

// isolateGlobalState puts the package-level state back when the test ends.
// Call it before the first t.Setenv, Initialize or assignment to that state.
// Cleanups run last in, first out, so the state comes back after the cleanups
// the test registers later have run.
func isolateGlobalState(t testing.TB) {
	t.Helper()
	state := captureGlobalState()
	t.Cleanup(state.restore)
}

// globalStateChanges names what differs between two states, in a fixed
// order. The logger is restored by isolateGlobalState but not compared: no
// test depends on which logger an earlier Initialize was given.
func globalStateChanges(before, after globalState) []string {
	var changes []string
	for i, variable := range configVariables {
		if before.values[i] != after.values[i] {
			changes = append(changes, variable.Name)
		}
	}
	if before.language != after.language {
		changes = append(changes, "i18n language")
	}
	if before.stepUpMatcher != after.stepUpMatcher {
		changes = append(changes, "step-up matcher")
	}
	sort.Strings(changes)
	return changes
}

func TestMain(m *testing.M) {
	if len(configVariables) == 0 {
		fmt.Fprintln(os.Stderr, "FAIL: the list of settings was not handed to this package; see global_state_list_test.go")
		os.Exit(1)
	}
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
