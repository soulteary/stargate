# Changelog

This file records user-visible changes. For upgrade steps and configuration examples, see the [v1.0.0 migration guide](docs/enUS/MIGRATION_V1.md).

## [1.2.6] - 2026-11-02

### Fixed in this maintenance review

- Bind Warden authorization refresh to a nonempty, matching stable user ID.
  Previously, a holder of a valid old session could inherit another account's
  ID, role and scopes after its stored phone or email was reassigned. This
  requires Warden, authorization refresh, a session due for refresh, and a
  directory record resolving the old contact to a different active account.
  The HTTP regression returns `200` before the fix and `401` afterward. This
  does not demonstrate anonymous authentication bypass.
- Keep Redis rate-limit windows fixed in their final second. A request with
  150ms remaining previously rearmed a one-minute window. Counter creation and
  expiry now commit in one Lua operation; legacy counters without expiry keep
  their counts and receive the configured expiry.
- Preserve `429` and `Retry-After` after password-header failures exceed the
  quota. The response was previously overwritten by `401` or a login redirect.
  Invalid passwords remain rejected; this does not avoid password verification
  work or change successful-password behavior.
- Repair 56 documentation heading links across translations. Configuration
  translations other than English and Chinese are explicitly marked historical.

- Match startup failure-test listeners to Fiber's IPv4 network and assert the
  typed listen operation and requested port across operating systems. Add a
  bounded Windows startup check to PR CI; retain every existing release check.

### Validation and coverage

- Enforce at least 95% unit/main/release coverage. Compare statement counts
  directly so 94.999% cannot pass after rounding. No production files are
  excluded from coverage.
- Linux arm64, Go 1.27.3, Redis 8.2.1: 516 top-level unit tests passed, 7 skipped;
  coverage 2121/2229 statements (95.1548%). With real Redis: 521 passed,
  2 skipped; coverage 2122/2229 (95.1996%). Race detection and
  golangci-lint 2.14.0 passed. Linux amd64/arm64 builds passed.
  Each run also passed 304 subtests (820/825 passing results when included).
- Unit skips comprise five service/integration cases and two existing
  empty-session-ID scenarios that the end-to-end harness cannot construct.
  The latter fallback behavior is tested directly with the session helper.
  The full run retains only those two existing skips.
- Regressions fail before each fix and pass afterward. Real Redis tests cover
  concurrent counting, final-millisecond expiry and a committed write whose
  acknowledgement is lost. That acknowledgement fault is injected after the
  actual Redis write; it is not a production network-failure experiment.
- Documentation/version/release contracts and launcher checks pass. All 626
  local file links and 301 local heading links resolve.

### Compatibility and upgrade

- Configuration names, HTTP fields, key prefixes, integer counters and session
  storage formats stay compatible. No new persistent volume or automatic data
  migration is introduced. Authorization refresh updates roles/scopes only for
  the same stable identity; missing or mismatching IDs require fresh login.
  Legacy sessions without `auth_method` remain supported when their nonempty
  `user_id` matches Warden. Warden records used for refresh need stable IDs.
- Redis ACLs must permit `EVAL`, `EVALSHA`, `PTTL`, `SET`, `PEXPIRE` and `INCR`
  on rate-limit keys. Script cache loss is handled automatically. The existing
  per-replica fallback during Redis failure remains unchanged.
- Pause protected traffic for a coordinated replacement of every instance.
  Old instances retain the identity-rebinding and window-extension behavior
  during mixed-version operation. For consistent fresh limit windows, wait one
  configured window after replacement before resuming traffic. Do not delete
  counters to bypass active quotas.
- Already-rebound sessions cannot be identified or repaired automatically.
  Review directory reassignment history, restrict access to retained audit and
  session evidence, and revoke or expire affected sessions before restoring
  access. If integrity cannot be established, require fresh Warden authentication
  for existing sessions. Existing stored records are not rewritten by this fix.
- Rollback needs no data-format conversion but restores the vulnerable identity
  refresh and prior rate-limit behavior. Do not resume affected traffic on an
  old version. Existing Redis persistence and session-exchange replay-storage
  requirements continue to apply; they are not new in this release.

### Performance measurements

On the same Linux arm64 VM (five vCPUs, 5.35GiB reported total RAM), Go 1.27.3
and Redis 8.2.1, three alternating precompiled-binary runs used 5000 operations
per case. Cold-counter median latency changed from 324933 to 194500 ns/op
(about 3078 to 5141 operations/s); allocation changed from 886 B/25 allocations
to approximately 517 B/17 allocations. Warm-counter allocation fell from
616 B/18 allocations to 487 B/16 allocations, while median latency increased
from 169126 to 196305 ns/op (about 5913 to 5094 operations/s, a 16% latency
increase). These measurements include VM scheduling noise and do not establish
an across-the-board latency improvement.

### Earlier merged maintenance included in this version

The release also includes the dependency refresh merged before this review:
Herald v1.3.13, Warden v1.4.4, compatible Kit maintenance patches, and compatible
template/schema, Prometheus, ABNF and YAML updates. The earlier code extraction
and language-aware quality-report maintenance are retained. These are separate
from the three newly reproduced fixes above; existing security and deployment
requirements are not presented as new changes.

## [1.2.5] - 2026-11-02

The signed tag was created, but the release was not published. Windows
compatibility validation timed out in a newly added startup failure test: its
reserved socket did not prevent Fiber's IPv4 server from starting. The immutable
tag is retained. Version 1.2.6 carries the maintenance changes above and the
portable test-fixture correction; no check was skipped to publish it.

## [1.2.4] - 2026-11-01

### Changed

- Package the already-built Linux release binaries directly into container
  images, avoiding repeated Go compilation under emulation. Container binaries
  now match the downloadable release binaries byte for byte; release images no
  longer apply a separate UPX compression step.
- Add native AMD64 and ARM64 packaging checks to pull requests, and verify
  binary identity, non-root execution, login templates and isolated runtime
  health before each release image vulnerability scan. Existing compatibility,
  coverage, dependency scans, both image scans and signing gates are retained.

### Fixed

- Derive container revision labels from the checked-out release source during
  manual publication. Validate that only the immutable version tag is emitted;
  mutable aliases remain owned by the release reconciliation job.

## [1.2.3] - 2026-11-01

### Fixed

- Show rejected verification-code errors in the browser with accessible alert
  semantics across all login modes.
- Support a strictly validated `SESSION_EXCHANGE_REDIRECT_PATH` so applications
  can complete native login immediately after session exchange. The default
  remains `/`, and request parameters cannot override the destination.

## [1.2.2] - 2026-10-28

### Changed

- Update the released authentication and health libraries: forwardauth-kit
  v3.0.4, health-kit v4.0.4, middleware-kit v3.0.3 and session-kit v3.1.3.
- Update compatible runtime dependencies, including redis-kit v1.8.0,
  http-kit/v2 v2.1.0, fasthttp v1.77.0, x/net v0.62.0, gRPC v1.85.0
  and Prometheus support modules; refresh module checksums.
- Update govulncheck to v1.9.0 and refresh CI artifact, report and container
  login Actions. Go 1.27.3 and application configuration remain unchanged.

## [1.2.1] - 2026-10-22

### Changed

- Require Go 1.27.3 and align build images and CI toolchains.
- Update compatible Go dependencies and published Kit versions; refresh module checksums.
- Preserve the project toolchain after quality report generation.
- Align Warden with v1.4.2 and pin the validated Go 1.27.3-compatible ci-recipes revision.
- Align the Herald client with the released v1.3.2 service and TOTP v1.2.5 API module.

## [1.2.0] - 2026-09-22

- Fixed a crash on every rate-limited endpoint when Redis session storage is
  disabled, which is the default. Login, verification-code sending, TOTP
  enrolment and revocation, and step-up all answered `500` because the shared
  rate-limit, challenge-context and session-exchange replay stores mistook a
  nil `*redis.Client` held in a `redis.Cmdable` interface for a live client
  instead of falling back to their in-memory implementations.
- Rotated the session identifier on login. session-kit v3 regenerates the
  identifier inside `Authenticate`, so the identifier a client holds before
  signing in can no longer be carried into an authenticated session.
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
- Build stamping now targets `github.com/soulteary/version-kit/v4`. A build
  still using the old `/v2` path succeeds but silently reports version `dev`,
  so out-of-tree build scripts need the same change.

## [1.1.0] - 2026-09-14

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

[1.2.0]: https://github.com/soulteary/stargate/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/soulteary/stargate/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/soulteary/stargate/compare/v0.12.0...v1.0.0
