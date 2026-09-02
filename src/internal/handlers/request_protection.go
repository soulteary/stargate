package handlers

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

type requestOrigin struct {
	scheme   string
	hostname string
	port     string
}

func canonicalRequestOrigin(raw string) (requestOrigin, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
		return requestOrigin{}, false
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return requestOrigin{}, false
	}
	hostname := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if hostname == "" {
		return requestOrigin{}, false
	}
	port := parsed.Port()
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return requestOrigin{scheme: scheme, hostname: hostname, port: port}, true
}

// RequireSameOrigin blocks browser cross-site writes while preserving clients
// such as curl that do not send browser Origin or Fetch Metadata headers.
func RequireSameOrigin() fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if ctx.Method() == fiber.MethodGet || ctx.Method() == fiber.MethodHead || ctx.Method() == fiber.MethodOptions {
			return ctx.Next()
		}
		if strings.EqualFold(strings.TrimSpace(ctx.Get("Sec-Fetch-Site")), "cross-site") {
			return SendErrorResponse(ctx, fiber.StatusForbidden, "cross-site request rejected")
		}
		origin := strings.TrimSpace(ctx.Get("Origin"))
		if origin == "" {
			return ctx.Next()
		}
		suppliedOrigin, suppliedOK := canonicalRequestOrigin(origin)
		requestOrigin, requestOK := canonicalRequestOrigin(GetForwardedProto(ctx) + "://" + GetForwardedHost(ctx))
		if !suppliedOK || !requestOK || suppliedOrigin != requestOrigin {
			return SendErrorResponse(ctx, fiber.StatusForbidden, "cross-site request rejected")
		}
		return ctx.Next()
	}
}

func endpointRateLimit(max int, expiration time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: expiration,
		KeyGenerator: func(ctx fiber.Ctx) string {
			return ctx.IP() + ":" + ctx.Path()
		},
		LimitReached: func(ctx fiber.Ctx) error {
			ctx.Set(fiber.HeaderRetryAfter, "60")
			return SendErrorResponse(ctx, fiber.StatusTooManyRequests, "too many requests")
		},
	})
}

func LoginRateLimit() fiber.Handler {
	return endpointRateLimit(10, time.Minute)
}

func VerificationRateLimit() fiber.Handler {
	return endpointRateLimit(5, time.Minute)
}

type passwordFailureBucket struct {
	count  int
	resets time.Time
}

var (
	passwordFailureMu          sync.Mutex
	passwordFailureBuckets     = make(map[string]passwordFailureBucket)
	passwordFailureLastCleanup time.Time
)

//nolint:unused // Test helper; test packages are compiled and run separately.
func resetPasswordFailureBucketsForTesting() {
	passwordFailureMu.Lock()
	defer passwordFailureMu.Unlock()
	passwordFailureBuckets = make(map[string]passwordFailureBucket)
	passwordFailureLastCleanup = time.Time{}
}

func rateLimitPasswordHeaderFailure(ctx fiber.Ctx) error {
	if strings.TrimSpace(ctx.Get("Stargate-Password")) == "" {
		return nil
	}

	now := time.Now()
	key := ctx.IP()
	passwordFailureMu.Lock()
	if passwordFailureLastCleanup.IsZero() || now.Sub(passwordFailureLastCleanup) >= time.Minute {
		for bucketKey, candidate := range passwordFailureBuckets {
			if !now.Before(candidate.resets) {
				delete(passwordFailureBuckets, bucketKey)
			}
		}
		passwordFailureLastCleanup = now
	}
	bucket := passwordFailureBuckets[key]
	if bucket.resets.IsZero() || !now.Before(bucket.resets) {
		bucket = passwordFailureBucket{resets: now.Add(time.Minute)}
	}
	bucket.count++
	passwordFailureBuckets[key] = bucket
	passwordFailureMu.Unlock()

	if bucket.count <= 10 {
		return nil
	}
	retryAfter := int(time.Until(bucket.resets).Seconds())
	if retryAfter < 1 {
		retryAfter = 1
	}
	ctx.Set(fiber.HeaderRetryAfter, fmt.Sprintf("%d", retryAfter))
	return SendErrorResponse(ctx, fiber.StatusTooManyRequests, "too many failed password attempts")
}
