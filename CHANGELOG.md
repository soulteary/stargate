# Changelog

## Unreleased

- Shared endpoint rate-limit state through the configured session Redis so a
  multi-replica deployment enforces one quota instead of one quota per replica.
  A Redis outage degrades to per-replica counting instead of removing the limit
  or rejecting every request.
- Made the login and verification quotas configurable through
  `RATE_LIMIT_LOGIN_MAX`, `RATE_LIMIT_VERIFICATION_MAX` and `RATE_LIMIT_WINDOW`,
  and applied the login quota to repeated `Stargate-Password` header failures.
- Warned at startup when `TRUSTED_PROXIES` is empty, and again on the first
  forwarded headers received from an untrusted peer, because that is the
  configuration in which every client behind a reverse proxy is attributed to
  the proxy address and shares a single rate-limit quota.
- Replaced repository-local CI and release Bash programs with the pinned,
  tested `soulteary/ci-recipes` Go tool.
- Preserve authenticated Warden sessions when an authorization refresh is
  canceled or reaches the request deadline; the current request still fails
  closed without turning a transient provider timeout into a logout.
- Added a 10-second, strictly validated request-context deadline before tracing
  so Warden and Herald calls receive a real `Done` signal and are canceled on
  timeout or handler completion. Fiber/fasthttp client-disconnect limitations
  are now documented explicitly.
- Made tag releases independently queueable and immutable, with fail-fast
  release-note validation and SemVer high-water reconciliation for mutable
  container aliases.
- Bound verification audit events to server-side challenge context. The context
  uses process memory for standalone deployments and the configured session
  Redis for multi-instance deployments. Idempotent retries preserve the original
  attribution, failed attempts retain it through a short audit grace period,
  and successful verification consumes it.
- Upgraded Herald to v1.3.0 and Warden to v1.4.0, and every kit to its current
  major: audit-kit v2.1.0, forwardauth-kit v3.0.0, health-kit v4.0.0, i18n-kit
  v4.0.1, logger-kit v3.0.0, metrics-kit v3.0.0, middleware-kit v3.0.0,
  redis-kit v1.7.0, secure-kit v2.1.0, session-kit v3.1.0, tracing-kit v2.0.0
  and version-kit v4.0.0. These releases move their framework-specific entry
  points into dedicated subpackages, so Stargate now reaches Fiber handlers and
  middleware through each kit's `fiberadapter`, Redis session storage through
  session-kit's `redisstore`, Redis health probing through health-kit's
  `redisprobe`, and OTLP tracer setup through tracing-kit's `otlp`. Responses
  are unchanged: session cookie attributes, Redis session key prefixes,
  ForwardAuth decisions, translated messages, security headers, health, metrics
  and log-level access control all behave exactly as before.
- Rotated the session identifier on login. session-kit v3 regenerates the
  identifier inside `Authenticate`, so the identifier a client holds before
  signing in can no longer be carried into an authenticated session.
- Build stamping now targets `github.com/soulteary/version-kit/v4`. A build
  still using the old `/v2` path succeeds but silently reports version `dev`,
  so out-of-tree build scripts need the same change.

- Fixed a crash on every rate-limited endpoint when Redis session storage is
  disabled, which is the default. Login, verification-code sending, TOTP
  enrolment and revocation, and step-up all answered `500` because the shared
  rate-limit, challenge-context and session-exchange replay stores mistook a
  nil `*redis.Client` held in a `redis.Cmdable` interface for a live client
  instead of falling back to their in-memory implementations.

This file records user-visible changes. For upgrade steps and configuration examples, see the [v1.0.0 migration guide](docs/enUS/MIGRATION_V1.md).

## [1.0.0] - 2026-08-27

### Breaking changes

- Go 1.27 or later is required to build Stargate.
- The official container now listens on port `8080` instead of `80`; update container port mappings, reverse-proxy targets, and health probes accordingly.
- `Stargate-Password` request-header authentication is disabled by default; trusted legacy integrations must explicitly set `PASSWORD_HEADER_AUTH_ENABLED=true`.
- Logout and account-state changes use POST requests with same-origin validation.
- Cross-domain session exchange uses short-lived, signed, single-use tickets instead of raw session IDs and requires `SESSION_EXCHANGE_SECRET`.
- Forwarded host, protocol, URI, and client-IP headers are ignored unless the immediate proxy is listed in `TRUSTED_PROXIES`.
- Invalid or incomplete security-sensitive configuration now stops startup instead of silently degrading.
- The removed `WARDEN_OTP_ENABLED` and `WARDEN_OTP_SECRET_KEY` settings are rejected; TOTP is provided through Herald.

### Authentication and authorization

- Added Warden authorization refresh with revocation handling.
- Added trusted-header authentication with a shared proxy secret.
- Added per-user Herald TOTP enrollment, confirmation, revocation, and backup-code flows.
- Added configurable password re-verification for sensitive paths through `STEP_UP_ENABLED` and `STEP_UP_PATHS`.
- Session state is reset after login so prior authorization, Step-up, refresh, or enrollment state cannot cross an authentication boundary.

### Security and operations

- Added strict callback-host, cookie-domain, HTTP-header, service-URL, secret-length, and TLS-pair validation.
- Added Redis-backed shared session and session-ticket replay state for multi-instance deployments.
- Split liveness (`/healthz`) from dependency readiness (`/readyz`) and added TLS-aware Warden and Herald checks.
- Added structured audit events, configurable log levels, container hardening, SBOMs, artifact attestations, checksums, Cosign signatures, and multi-architecture image scanning.
- User-visible verification errors no longer expose upstream provider details.

### Compatibility and documentation

- Added Linux, macOS, and Windows builds for amd64 and arm64.
- Added a seven-language API and deployment contract checker.
- Added v1.0.0 migration guides in all seven supported languages and corrected executable Docker, Compose, and Traefik examples.

### Release verification

Before publishing the first release candidate, replace `Unreleased` with the intended release date and freeze these notes:

1. Finalize this dated changelog entry and its migration-guide links.
2. Run the Nightly compatibility workflow successfully on the intended commit.
3. Publish `v1.0.0-rc.1` from that commit and verify binaries, checksums, SBOM, attestations, signatures, and the multi-architecture image.
4. Publish `v1.0.0` from the same verified source commit. If the source changes, publish and verify a new release candidate first.

[1.0.0]: https://github.com/soulteary/stargate/compare/v0.12.0...v1.0.0
