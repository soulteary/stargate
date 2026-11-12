package handlers

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v3"
)

func TestRateLimitClientKey(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want string
	}{
		{name: "ipv4 is unchanged", ip: "203.0.113.7", want: "203.0.113.7"},
		{name: "ipv4-mapped ipv6 becomes ipv4", ip: "::ffff:203.0.113.7", want: "203.0.113.7"},
		{name: "ipv6 is reduced to its /64", ip: "2001:db8:1:2:aaaa:bbbb:cccc:dddd", want: "2001:db8:1:2::/64"},
		{name: "another host in the same /64", ip: "2001:db8:1:2::1", want: "2001:db8:1:2::/64"},
		{name: "a different /64", ip: "2001:db8:1:3::1", want: "2001:db8:1:3::/64"},
		{name: "zone is ignored", ip: "fe80::1%eth0", want: "fe80::/64"},
		{name: "loopback", ip: "::1", want: "::/64"},
		{name: "unparsable value is unchanged", ip: "not-an-ip", want: "not-an-ip"},
		{name: "empty", ip: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testza.AssertEqual(t, tt.want, rateLimitClientKey(tt.ip))
		})
	}
}

// A client that rotates through addresses in its own IPv6 /64 must stay inside
// one login quota, while a client in another /64 keeps a separate one.
func TestLoginRateLimitGroupsIPv6ByPrefix(t *testing.T) {
	resetRateLimitStateForTesting()
	t.Cleanup(resetRateLimitStateForTesting)

	app := fiber.New(fiber.Config{
		ProxyHeader:      fiber.HeaderXForwardedFor,
		TrustProxy:       true,
		TrustProxyConfig: fiber.TrustProxyConfig{Proxies: []string{"0.0.0.0"}},
	})
	app.Post("/_login", LoginRateLimit(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	post := func(clientIP string) int {
		t.Helper()
		req := newRequestProtectionTestRequest(fiber.MethodPost, "/_login")
		req.Header.Set(fiber.HeaderXForwardedFor, clientIP)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("execute request from %s: %v", clientIP, err)
		}
		return resp.StatusCode
	}

	rotating := []string{
		"2001:db8:1:2::1", "2001:db8:1:2::2", "2001:db8:1:2::3", "2001:db8:1:2::4", "2001:db8:1:2::5",
		"2001:db8:1:2::6", "2001:db8:1:2::7", "2001:db8:1:2::8", "2001:db8:1:2::9", "2001:db8:1:2::a",
	}
	for _, ip := range rotating {
		testza.AssertEqual(t, fiber.StatusNoContent, post(ip))
	}
	testza.AssertEqual(t, fiber.StatusTooManyRequests, post("2001:db8:1:2:ffff:ffff:ffff:ffff"))
	testza.AssertEqual(t, fiber.StatusNoContent, post("2001:db8:1:3::1"))
}
