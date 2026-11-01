package handlers

import (
	"testing"

	forwardauth "github.com/soulteary/forwardauth-kit/v3"
	fafiber "github.com/soulteary/forwardauth-kit/v3/fiberadapter"
	"github.com/stretchr/testify/require"
)

func TestForwardAuthPasswordFailurePreservesRateLimitResponse(t *testing.T) {
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)
	withRateLimitConfig(t, "1", "1", "1m")
	for _, accept := range []string{"application/json", "text/html"} {
		resetRateLimitStateForTesting()
		for attempt := 0; attempt < 2; attempt++ {
			ctx, app := createTestContext("GET", "/_auth", map[string]string{"Accept": accept, "Stargate-Password": "wrong-password", "Host": "app.example.com"}, "")
			err := handleForwardAuthFailure(ctx, GetForwardAuthHandler(), fafiber.NewContext(ctx), forwardauth.ErrInvalidPassword)
			require.NoError(t, err)
			if attempt == 1 {
				require.Equal(t, 429, ctx.Response().StatusCode())
				require.NotEmpty(t, ctx.GetRespHeader("Retry-After"))
				require.Empty(t, ctx.GetRespHeader("Location"))
			}
			app.ReleaseCtx(ctx)
		}
	}
}
