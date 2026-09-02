package requestcontext

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeProvider struct {
	call func(context.Context) error
}

func (p fakeProvider) Call(ctx context.Context) error {
	return p.call(ctx)
}

func TestMiddlewareProvidersObserveDeadlineAndKeepResults(t *testing.T) {
	providerErr := errors.New("provider error")
	for _, test := range []struct {
		name string
		err  error
		want int
	}{
		{name: "warden success", want: fiber.StatusNoContent},
		{name: "warden error", err: providerErr, want: fiber.StatusBadGateway},
		{name: "herald success", want: fiber.StatusNoContent},
		{name: "herald error", err: providerErr, want: fiber.StatusBadGateway},
	} {
		t.Run(test.name, func(t *testing.T) {
			var captured context.Context
			provider := fakeProvider{call: func(ctx context.Context) error {
				captured = ctx
				deadline, ok := ctx.Deadline()
				require.True(t, ok, "provider must receive a context deadline")
				assert.WithinDuration(t, time.Now().Add(time.Second), deadline, 200*time.Millisecond)
				return test.err
			}}

			app := fiber.New()
			app.Use(Middleware(time.Second))
			app.Get("/provider", func(c fiber.Ctx) error {
				if err := provider.Call(c.Context()); err != nil {
					return c.SendStatus(fiber.StatusBadGateway)
				}
				return c.SendStatus(fiber.StatusNoContent)
			})

			resp, err := app.Test(httptest.NewRequest("GET", "/provider", nil))
			require.NoError(t, err)
			assert.Equal(t, test.want, resp.StatusCode)
			require.NotNil(t, captured)
			select {
			case <-captured.Done():
				assert.ErrorIs(t, captured.Err(), context.Canceled)
			case <-time.After(time.Second):
				t.Fatal("request context was not canceled after the handler returned")
			}
		})
	}
}

func TestMiddlewareTimeoutCancelsBlockedProvider(t *testing.T) {
	const timeout = 25 * time.Millisecond
	provider := fakeProvider{call: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}

	app := fiber.New()
	app.Use(Middleware(timeout))
	app.Get("/provider", func(c fiber.Ctx) error {
		if err := provider.Call(c.Context()); !errors.Is(err, context.DeadlineExceeded) {
			return c.Status(fiber.StatusInternalServerError).SendString("unexpected provider result")
		}
		return c.SendStatus(fiber.StatusGatewayTimeout)
	})

	started := time.Now()
	resp, err := app.Test(httptest.NewRequest("GET", "/provider", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusGatewayTimeout, resp.StatusCode)
	assert.GreaterOrEqual(t, time.Since(started), timeout)
	assert.Less(t, time.Since(started), time.Second)
}
