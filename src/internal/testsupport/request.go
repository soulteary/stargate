package testsupport

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

// RequestAllowance is the fiber.TestConfig every in-process test request
// passes to app.Test. Without it Fiber gives a request one second and then
// returns an error while the handler keeps running, which on a loaded machine
// fails the test and lets the abandoned handler write package-level state
// while the next test runs. The allowance only bounds a handler that hangs:
// app.Test returns as soon as the handler has answered. When it expires,
// app.Test returns os.ErrDeadlineExceeded and no response, so the test fails.
var RequestAllowance = fiber.TestConfig{Timeout: time.Minute, FailOnTimeout: true}
