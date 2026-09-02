# Changelog

## Unreleased

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

This file records user-visible changes. For upgrade steps and configuration examples, see the [v1.0.0 migration guide](docs/enUS/MIGRATION_V1.md).

## [1.0.0] - 2026-08-27

### Breaking changes

- Go 1.27 or later is required to build Stargate.
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
