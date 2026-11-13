package main

import "github.com/gofiber/fiber/v3"

// requestLogClientIP adds the client address to each request log line, resolved
// the way audit records and rate limits resolve it: through TRUSTED_PROXIES and
// PROXY_HEADER. The line's own "ip" field stays the direct peer, which behind a
// reverse proxy is the proxy.
//
// The logger middleware has a trusted-proxy option of its own, but it takes the
// leftmost X-Forwarded-For element and ignores PROXY_HEADER. Behind a proxy
// that appends to the header, the leftmost element is whatever the client
// sent, so that option would both disagree with the audit log and let a client
// choose the address written to the request log.
func requestLogClientIP(c fiber.Ctx) map[string]interface{} {
	return map[string]interface{}{"client_ip": c.IP()}
}
