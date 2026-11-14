package handlers

import (
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"testing"
	"time"

	forwardauth "github.com/soulteary/forwardauth-kit/v3"
	"github.com/soulteary/herald/pkg/herald"
	logger "github.com/soulteary/logger-kit/v3"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/testsupport"
	"github.com/soulteary/warden/pkg/warden"
)

type lookupUserFunc = func(ctx context.Context, phone, mail string) (*warden.AllowListUser, time.Time, error)

// globalState is the package-level state that tests change and that t.Setenv
// does not put back: config.Initialize copies the environment into the config
// package's variables, and the handlers keep clients, stores and a
// ForwardAuth handler that were built from them.
type globalState struct {
	config                  testsupport.ConfigState
	logger                  *logger.Logger
	forwardAuthHandler      *forwardauth.Handler
	heraldClient            *herald.Client
	heraldClientInitErr     error
	rateLimitStore          RateLimitStore
	challengeContexts       ChallengeContextStore
	uniformDigestKey        []byte
	lookupRefreshUser       lookupUserFunc
	lookupTrustedHeaderUser lookupUserFunc
	authorizationNow        func() time.Time
	uniformNow              func() time.Time
	uniformEntropy          io.Reader
}

func captureGlobalState() globalState {
	state := globalState{
		config:                  testsupport.CaptureConfig(),
		logger:                  log,
		forwardAuthHandler:      forwardAuthHandler,
		heraldClient:            heraldClient,
		heraldClientInitErr:     heraldClientInitErr,
		rateLimitStore:          getRateLimitStore(),
		challengeContexts:       getChallengeContextStore(),
		lookupRefreshUser:       lookupRefreshUser,
		lookupTrustedHeaderUser: lookupTrustedHeaderUser,
		authorizationNow:        authorizationNow,
		uniformNow:              uniformNow,
		uniformEntropy:          uniformEntropy,
	}
	uniformDigestKeyState.Lock()
	state.uniformDigestKey = uniformDigestKeyState.key
	uniformDigestKeyState.Unlock()
	return state
}

func (state globalState) restore() {
	state.config.Restore()
	log = state.logger
	forwardAuthHandler = state.forwardAuthHandler

	heraldClient = state.heraldClient
	heraldClientInitErr = state.heraldClientInitErr
	heraldClientInit = sync.Once{}
	if state.heraldClient != nil || state.heraldClientInitErr != nil {
		// A sync.Once cannot be read, so a completed initialisation is
		// recognised by its result.
		heraldClientInit.Do(func() {})
	}

	resetRateLimitStateForTesting()
	SetRateLimitStore(state.rateLimitStore)
	SetChallengeContextStore(state.challengeContexts)
	uniformDigestKeyState.Lock()
	uniformDigestKeyState.key = state.uniformDigestKey
	uniformDigestKeyState.Unlock()
	lookupRefreshUser = state.lookupRefreshUser
	lookupTrustedHeaderUser = state.lookupTrustedHeaderUser
	authorizationNow = state.authorizationNow
	uniformNow = state.uniformNow
	uniformEntropy = state.uniformEntropy
}

// isolateGlobalState gives the test its own rate-limit and challenge-context
// stores and one-shot warnings, and puts the package-level state back when
// the test ends. Call it before the first t.Setenv, config.Initialize or
// assignment to that state. Cleanups run last in, first out, so the state
// comes back after the cleanups the test registers later have run.
//
// The Warden client of the auth package cannot be read from here. It is
// discarded when the test ends, which is the state TestMain starts from.
func isolateGlobalState(t testing.TB) {
	t.Helper()
	state := captureGlobalState()
	t.Cleanup(func() {
		state.restore()
		auth.ResetWardenClientForTesting()
	})
	resetRateLimitStateForTesting()
	SetChallengeContextStore(newMemoryChallengeContextStore())
}

// describe lists the part of the state, other than the configuration, that
// later tests depend on. The
// logger, the clocks and the lookup functions are restored by
// isolateGlobalState but not compared: replacing them with an equal value is
// harmless and cannot be told apart from a different one.
func (state globalState) describe() map[string]string {
	return map[string]string{
		"ForwardAuth handler":     fmt.Sprintf("%p", state.forwardAuthHandler),
		"Herald client":           fmt.Sprintf("%p", state.heraldClient),
		"Herald client error":     fmt.Sprint(state.heraldClientInitErr),
		"rate-limit store":        describeStore(state.rateLimitStore),
		"challenge-context store": describeStore(state.challengeContexts),
		"uniform digest key":      fmt.Sprintf("%x", state.uniformDigestKey),
	}
}

// describeStore tells a replaced process-local store from the first one only
// by what it holds: an empty one behaves like any other empty one.
func describeStore(store any) string {
	switch memory := store.(type) {
	case *memoryRateLimitStore:
		memory.mu.Lock()
		defer memory.mu.Unlock()
		return fmt.Sprintf("process-local with %d entries", len(memory.buckets))
	case *memoryChallengeContextStore:
		memory.mu.Lock()
		defer memory.mu.Unlock()
		return fmt.Sprintf("process-local with %d entries", len(memory.entries))
	}
	return fmt.Sprintf("%T %p", store, store)
}

// globalStateChanges names what differs between two states, in a fixed order.
func globalStateChanges(before, after globalState) []string {
	was, is := before.describe(), after.describe()
	changes := testsupport.ConfigChanges(before.config, after.config)
	for name, value := range is {
		if was[name] != value {
			changes = append(changes, name)
		}
	}
	sort.Strings(changes)
	return changes
}
