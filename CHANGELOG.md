# Changelog

This file records user-visible changes. For upgrade steps and configuration examples, see the [v1.0.0 migration guide](docs/enUS/MIGRATION_V1.md).

## [Unreleased]

### Added

- Add `HERALD_CLIENT_IP_HEADER`. When set, Stargate sends the end-user address
  to Herald in that header on challenge and verification requests, so Herald's
  `RATE_LIMIT_PER_IP` can be counted per end user. Herald must name the same
  header in `HERALD_TRUSTED_PROXY_HEADER` and list Stargate in
  `HERALD_TRUSTED_PROXIES`; it ignores the header otherwise. The default is
  empty: no header is sent and all users share one per-IP quota at Herald, as
  before.

## [1.3.8] - 2026-11-13

### Changed

- Update the Herald SDK to v1.6.0. The SDK now returns an error itself when
  Herald answers a TOTP revocation without confirming it. `/totp/revoke` keeps
  answering `502 Bad Gateway` with reason `revoke_not_confirmed` and records
  the same audit event. The SDK no longer quotes a TOTP response that is not
  valid JSON in its error message, so such a response no longer reaches
  Stargate's warning log. Stargate does not use the SDK's new `ClientIPHeader`
  option, and its requests to Herald are unchanged. Updating the SDK does not
  replace an independently deployed Herald service.

### Fixed

- Step-up confirmation and the start of TOTP enrollment now confirm that the
  session still exists after saving it. With Fiber v3.7.0 a save for a session
  that a logout had just ended returned no error, so these requests answered
  with a redirect or the enrollment page for a session that no longer existed.
  They now answer `500`.
- The login page no longer marks debug mode as active when `COOKIE_SECURE` is
  `true`. The debug code is not relayed in that configuration since v1.3.7, so
  the page showed an empty debug field.

### Documentation

- Document the client-IP header format that v1.3.7 requires. Every element of
  `X-Forwarded-For` (or the configured `PROXY_HEADER`) must be a bare IP
  address. Behind a trusted proxy that writes `IP:port` or `unknown`, all users
  share one rate limit and are recorded under the proxy address. Check the
  proxy before upgrading from v1.3.6 or earlier.
- Document that request trailers are not read as headers since v1.3.7.
- Document the `revoke_not_confirmed` reason and the TOTP audit events.
- `/log/level`: the forwarding-header check cannot detect a same-host proxy
  that adds no forwarding header. Keep the path out of every proxied route.
- Bring the `/log/level` and `debug_code` descriptions in the `zhCN` and `deDE`
  documents in line with v1.3.7.

### Upgrade and rollback

- This release adds and removes no configuration options. Redis records and
  stored sessions need no migration; do not clear Redis during rollout or
  rollback.
- A step-up confirmation or the start of a TOTP enrollment that overlaps a
  logout of the same session now answers `500`.
- When upgrading from v1.3.6 or earlier, first check the client-IP header that
  the trusted proxy writes, as described under Documentation.
- The Herald SDK update needs no change to a deployed Herald service.
- Rolling back to v1.3.7 restores the previous behaviour of the two fixes.
  Rebuild from the v1.3.7 source when rolling back a source build.

## [1.3.7] - 2026-11-13

### Changed

- Key login and verification rate limits on the /64 prefix for IPv6 clients.
  A client that rotates addresses inside its own prefix no longer receives a
  fresh quota for each address. IPv4 keys are unchanged.
- Update the Herald SDK to v1.5.2 and the Warden SDK to v1.7.0. The Warden
  client package used by Stargate is unchanged between v1.6.3 and v1.7.0.
  Updating these SDKs does not replace independently deployed Herald or Warden
  services.
- Update `audit-kit/v2` to v2.1.6, `challenge-kit` to v1.11.2, `cli-kit` to
  v1.9.4, `forwardauth-kit/v3` to v3.0.7, `health-kit/v4` to v4.0.7,
  `i18n-kit/v4` to v4.0.6, `logger-kit/v3` to v3.0.6, `metrics-kit/v3` to
  v3.0.6, `middleware-kit/v3` to v3.0.6, `redis-kit` to v1.8.2, `secure-kit/v2`
  to v2.1.5, `tracing-kit/v2` to v2.0.4, `version-kit/v4` to v4.0.5, the Fiber
  HTML template engine to v3.1.0, `gofiber/utils/v2` to v2.6.5, fasthttp to
  v1.78.0, and indirect dependencies.
- Update Fiber to v3.7.0 and `session-kit/v3` to v3.2.2, which keeps session
  storage keyed by the session ID under Fiber v3.7.0. Stored sessions need no
  migration. Fiber v3.7.0 no longer reports an error when a request saves a
  session that a logout in the same process has already ended, so Stargate now
  confirms that the session still exists after saving a directory refresh. A
  `/_auth` or factor-management request that overlaps a logout is refused with
  `500`, as before.

### Security

- Record TOTP enrollment and revocation in the audit log (`user_updated` with
  `action` `totp_enrolled` or `totp_revoked`), including refused revocations.
- `/totp/revoke` reports failure when Herald answers without confirming the
  revocation. It previously reported success.
- Reject `/log/level` requests that carry a forwarding header (`X-Forwarded-For`,
  `X-Forwarded-Host`, `X-Forwarded-Proto`, `Forwarded`, `X-Real-IP`) with
  `404 Not Found`, and apply the cross-site write check to its `PUT` and `POST`
  methods. A same-host reverse proxy connects from `127.0.0.1`, so the loopback
  allowlist alone let relayed clients change the log level. Direct local
  requests are unaffected.
- Match `STEP_UP_PATHS` case-insensitively and ignore `;` path parameters in
  each segment. With `STEP_UP_PATHS=/admin`, a backend that routes
  case-insensitively or drops path parameters served `/Admin` and `/admin;x`
  without step-up. Paths whose parameters hide a dot segment or an empty
  leading segment now require step-up.
- Refuse to start when `SESSION_EXCHANGE_SECRET` is the demo value published in
  `docker-compose.yml` (`local-development-session-secret-change-me`) and
  `COOKIE_SECURE` is `true`. The local HTTP demo, which sets
  `COOKIE_SECURE=false`, is unaffected. Log a startup warning when `PASSWORDS`
  uses the `plaintext` algorithm while `COOKIE_SECURE` is `true`.
- Send `Content-Security-Policy` on Stargate's own responses
  (`default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:;
  connect-src 'self'; base-uri 'none'; frame-ancestors 'none'`) and
  `Strict-Transport-Security: max-age=31536000` when `COOKIE_SECURE` is `true`.
  `/_auth` responses keep the previous header set.
- Stop relaying Herald's `debug_code` in the `POST /_send_verify_code` response
  when `COOKIE_SECURE` is `true`, even if `DEBUG=true`. The relay remains
  available for local HTTP testing with `COOKIE_SECURE=false`.
- Update Go to 1.27.4, which fixes GO-2026-6859 in `net/http`. The build image is
  now `golang:1.27.4-alpine3.24`.

### Upgrade and rollback

- This release adds and removes no configuration options. Redis records,
  stored sessions, pending bindings and absolute deadlines need no migration;
  do not clear Redis during rollout or rollback.
- Before upgrading an HTTPS deployment (`COOKIE_SECURE=true`), replace
  `SESSION_EXCHANGE_SECRET` if it still holds the demo value from
  `docker-compose.yml`; Stargate otherwise refuses to start.
- Send `/log/level` requests from the local host directly to Stargate. Requests
  relayed by a reverse proxy now receive `404 Not Found`.
- Review `STEP_UP_PATHS`: case variants and paths with `;` parameters now
  require step-up. IPv6 clients inside one /64 prefix now share one rate-limit
  quota.
- Stargate's own pages now carry a `Content-Security-Policy` that permits
  scripts and styles only from Stargate's `/assets` path. Check external
  template overrides that rely on inline scripts, inline styles or resources
  from other origins.
- Rolling back to v1.3.6 restores the previous behaviour, including the weaker
  checks described under Security. Rebuild from the v1.3.6 source when rolling
  back a source build, and do not independently downgrade Herald or Warden
  services to make a rollback pass.

## [1.3.6] - 2026-11-09

### Added

- Add opt-in `LOGIN_DINGTALK_REQUIRE_USERID`, defaulting to `false`. When enabled,
  explicit DingTalk delivery requires a single canonical Warden `dingtalk_userid`
  and refuses missing or invalid bindings before calling Herald. Preserve opaque
  userid case and apply the same constraint to pending login verification.
  Document the isolated IM-only profile in English and the six translated
  configuration guides; this does not verify or populate a real mapping.
- Add an authenticated `/_account` management page without generating enrollment
  secrets or backup codes on a read-only GET. Existing fresh-directory and
  sensitive-session authorization still apply.
- Add optional `LOGIN_SUPPORT_URL` for an HTTPS or single-mailbox support link,
  plus stable allowlisted authentication error reasons in JSON responses.
  Existing status codes, numeric error codes and non-JSON formats are retained.
- Add a scoped offline `doctor` command with explicit, bounded health checks and
  redacted human/JSON output. Health results do not prove service authentication
  or a working login; no active business probe is performed.

### Changed

- Unify login, authenticator enrollment/revocation and account-page feedback,
  localized messages, recovery actions and request state across supported UI
  languages. Keep authentication views non-cacheable and avoid automatic OTP
  resend or verification retries.
- Embed templates, static assets and the favicon for standalone binaries when
  external resource paths do not exist. Existing external overrides remain in
  use; inaccessible overrides are not silently replaced.
- Update the Herald SDK from v1.4.4 to v1.5.0 and forward validated optional
  challenge-creation retry timing to the login page. Missing or unusable timing
  remains unknown; a countdown is not guaranteed admission or a quota reset.
  Updating this SDK does not replace an independently deployed Herald service.
- Add complete mail-OTP and authenticator examples with explicit local versus
  production TLS profiles, private SMTP CA selection, protected build contexts
  and retained provider trust-state validation. Align local host/callback routing
  and startup guidance without treating the HTTP demo as a production profile.

### Fixed

- Keep page locale and management titles consistent across authentication
  requests, and restore backup-code controls after fresh confirmation.
- Preserve literal SMTP credential bytes and verify configured local SMTP TLS
  and retained example trust material instead of silently bypassing trust.

### Verification

- Add an always-run work-IM bridge CI job with a fixed reviewed Herald source
  and a loopback mock provider. Check canonical receiver binding, cross-user
  refusal, idempotency and failure behavior without real IM delivery.
- Add scoped Chromium acceptance for isolated full-stack HTTP and verified-HTTPS
  examples, protected application callbacks, authentication-page recovery and
  browser history. Record bounded navigation/observation failures and distinguish
  cookie privacy eviction from restoration.
- Observe product BFCache behavior without weakening authentication or changing
  cache policy. Synthetic and isolated browser coverage does not establish a
  production login, real mailbox delivery or 200-person capacity.

### Upgrade and rollback

- The strict DingTalk option remains off unless explicitly configured. Default
  behavior retains the existing canonical-phone fallback. For a separately
  approved userid-only profile, maintain verified directory mappings, disable
  alternate SMS/email channels as intended, and review pending phone-derived
  bindings before opting in; no live profile is enabled by this release.
- Preserve Redis records, pending bindings, replay/revocation state, credentials
  and absolute deadlines. This release performs no directory or application
  permission migration, session rewrite, credential rotation or provider rollout.
  Earlier v1.3.x migration requirements still apply to older deployments.
- Rolling back removes the new UI/diagnostic behavior and strict-userid option;
  phone fallback can return. Restore compatible source manifests when rebuilding,
  retain provider state, and do not independently downgrade services or clear
  persistent records to make a rollback pass.

## [1.3.5] - 2026-11-08

### Fixed

- Refresh directory authorization only when session authentication is selected.
  An unavailable or deleted identity in an existing cookie no longer blocks
  valid password-header or trusted-header authentication. Session fallback and
  session-bound step-up verification retain their existing checks.

### Verification

- Strengthen the manual compatibility harness to check absolute session expiry,
  stored identity and newly issued login sessions during upgrade and rollback.
  This changes verification coverage, not the session storage format.

### Upgrade and rollback

- Dependencies and configuration are unchanged from v1.3.4. Preserve Redis
  records, pending bindings, session namespaces and absolute deadlines.
- Selected session authentication still refreshes directory authorization and
  fails closed when it is unavailable. Invalid password headers retain the
  existing fallback to an authenticated session; service HMAC requests do not
  fall back to API-key authentication. Step-up verification remains session-bound.
- Rolling back to v1.3.4 restores the previous authentication ordering: an old
  directory session can again block otherwise valid explicit credentials. No
  schema migration, automatic OTP retry or session renewal is introduced.

## [1.3.4] - 2026-11-08

### Dependencies

- Update secure-kit/v2 to v2.1.4 for Unicode code-point masking and safe bounds
  handling. Non-ASCII masked log output can change; ASCII output is preserved.
- Update audit-kit/v2 to v2.1.4, challenge-kit to v1.11.1, Warden to v1.6.2 and
  Herald to v1.4.4, with herald-totp/v2 v2.0.3 in the resolved module graph.
  session-kit/v3 remains v3.2.1. SDK updates do not deploy separate services.

### Documentation

- Align authentication, session storage, browser-bound exchange and port guidance
  with the implemented interfaces across translated guides and current examples.
- Add manual compatibility verification for formal service binaries using shared
  Redis state, HMAC authentication and dependency-ordered upgrade/rollback.

### Upgrade and rollback

- Deploy the verified service releases before this gateway patch. Keep explicit
  HMAC key IDs and the selected request-authentication mode; HMAC requests do not
  fall back to an API key. Retain the v1.3.3 staged key-ID migration for older
  deployments.
- Preserve Redis records, pending challenge bindings, replay and revocation state,
  session namespaces and absolute deadlines. No new storage schema or credential
  migration is introduced by this dependency update. No OTP retry or session
  renewal is automatic; existing 503 and lock-contention 409 semantics remain.
- Rolling back to the verified v1.3.3 binary or image restores its compiled
  masking behavior and dependency versions. Source consumers must restore their
  pre-upgrade `go.mod` and `go.sum`, verify the effective `go list -m all` versions
  and rebuild; downgrading a parent module alone may retain newer dependencies
  under Go minimal version selection. Rollback does not rewrite prior logs,
  recover removed keys or undo state
  committed by a provider. Do not independently roll back provider services or
  discard shared state; verify the selected credentials and login flows first.

## [1.3.3] - 2026-11-07

### Security

- Configure the Herald SDK with HMAC credentials only when `HERALD_HMAC_SECRET` is set. Keep API-key-only deployments supported without sending the configured API key on signed requests.

### Dependencies and compatibility

- Pin the verified formal Herald v1.4.3 release. Warden v1.6.1,
  session-kit/v3 v3.2.1, herald-totp/v2 v2.0.2 and challenge-kit v1.11.0 remain
  unchanged. SDK updates do not deploy the separate Herald service.
- Before upgrading a Herald key-map verifier, set and verify the existing key
  ID through `HERALD_HMAC_KEY_ID` on every current Stargate caller. Then upgrade
  Herald to v1.4.3, followed by the Stargate v1.3.3 binary. Mapped-key verifiers
  require an explicit wire key ID.
  An empty HMAC secret selects API-key-only mode. A client configured with both
  credentials no longer sends its API key on signed requests and does not retry
  rejected HMAC requests with another credential.
- Preserve shared Redis data, pending challenge bindings, revocation markers,
  session namespaces and absolute deadlines. This patch changes no record schema,
  OTP consumption rule or successful login format, and does not renew sessions.
- Before rollback, review the authentication configuration. Rolling back to
  v1.3.2 restores the older dual-credential request behavior; it does not remove
  configured secrets or undo service-side key retirement. Restoring an older
  Herald service also restores its older key-selection rules. Do not delete
  replay or binding state to make a rollback pass, and keep the v1.3.0/v1.3.1
  migration requirements when upgrading from older versions.

## [1.3.2] - 2026-11-07

### Fixed

- Preserve retryable OTP verification failures as HTTP 503 for upstream server
  errors and HTTP 409 for published verifier lock contention. Retryable failure
  responses leave the gateway binding intact; no verification request is automatically
  retried. The existing Herald SDK does not expose upstream retry headers, so
  no `Retry-After` value is synthesized.

### Dependencies and compatibility

- Pin Herald v1.4.2, including herald-totp/v2 v2.0.2 transitively. Warden v1.6.1,
  session-kit/v3 v3.2.1 and challenge-kit v1.11.0 remain unchanged. Update the TOTP
  service, then Herald, before publishing the gateway upgrade; SDK pins do not
  deploy these independent services.
- This corrects an existing error-classification gap inherited by bound OTP
  verification. Request formats, success payloads, proof validation and one-time
  consumption rules are unchanged.
- Preserve Redis data, gateway challenge bindings, revocation markers and the
  current session namespace and deadlines. No schema or namespace migration or
  reauthentication is required when upgrading from v1.3.1. Keep the v1.3.0 and
  v1.3.1 migration requirements when upgrading from an older release.
- Clients should handle 503 and the specific retryable 409 rather than treating
  them as invalid codes. The gateway does not retry verification automatically.
  A failed response may follow a committed provider operation; preserve pending
  state and reconcile uncertain outcomes instead of assuming the code is unused.
- Rolling v1.3.1/v1.3.2 replicas share the same record formats but differ in these
  error statuses. Rollback restores the earlier 401 classification without
  converting records or extending session deadlines. Do not independently
  downgrade backend services as part of a gateway rollback.

## [1.3.1] - 2026-11-07

### Security

- Bind OTP login to a gateway-issued challenge for the current Warden subject,
  channel and directory contact. Use Herald's bound `/v2/otp/verifications`
  request with expected subject, `login` purpose and channel, then check the
  returned challenge identity. Missing or stale local bindings reject login;
  unrelated-purpose codes are not consumed by the bound provider check.
- Fail code creation closed when its trusted login binding cannot be persisted
  or an idempotent response conflicts with the stored binding. A provider may
  already have delivered the code; retain its original idempotency key.
- Preserve TOTP confirmation quota errors as fixed `429/rate_limited` responses
  instead of reporting an invalid code. Keep the enrollment binding for retry.
- Update Warden SDK diagnostics to omit raw transport errors, retry URLs and
  response bodies when custom logging is enabled. Returned errors are unchanged.

### Dependencies and upgrade

- Pin session-kit/v3 v3.2.1, Warden v1.6.1 and Herald v1.4.1, including
  herald-totp/v2 v2.0.1 transitively. challenge-kit remains v1.11.0. These
  SDK updates do not deploy independently running backend services.
- Use a shared challenge-context store across gateway replicas. Old pending
  challenges work only when their original binding survives and still matches
  the directory contact. Otherwise restart the login flow; there is no legacy
  verification fallback. Replace all gateway instances together, and preserve
  unresolved provider idempotency records through upgrade and rollback.
- The session clock tolerance only relaxes the future-deadline upper bound by
  the smaller of one second and 1% of the lifetime; absolute expiry and atomic
  revocation remain enforced. Rollback loses the new login binding checks and
  can reject sessions issued by a slightly ahead replica. See the current
  deployment guide before restoring older instances.


## [1.3.0] - 2026-11-07

### Security

- Prevent an in-flight authorization refresh from restoring an authenticated
  session under the same identifier after logout. Revocation and writes are
  atomic in memory and Redis, and identifiers have an absolute 24-hour deadline.
- Recheck Warden authorization before TOTP enrollment, confirmation, revocation
  and step-up, even when background refresh is disabled. Source failures return
  503 without performing the factor operation; definitive identity revocation
  returns 401.
- Bind cross-domain exchange tickets to a destination-issued host-only browser
  nonce, destination, scheme and configured landing path. Old tickets are not
  accepted. Missing state starts a new handshake; mismatched state is rejected.
- Use the registered method and route for authentication rate-limit buckets so
  case and trailing-slash variants share their endpoint quota.
- Add opt-in Warden source-age enforcement using negotiated snapshot metadata.
  Source failures cannot extend the stored authorization deadline. This does not
  make directory revocation instantaneous.
- Require a separate metrics bearer credential. Without `METRICS_API_KEY`, the
  metrics endpoint returns 404; upstream service credentials do not grant access.

### Added

- Return `X-Auth-Email-Verified` on successful ForwardAuth checks. A true value
  requires a server-bound Herald email challenge for the same explicit Warden
  subject and current email, with evidence no older than 24 hours. Other login
  methods do not create that evidence. This proves current email control, not
  historical ownership. See the [verified email guide](https://github.com/soulteary/stargate/blob/v1.3.0/docs/enUS/VERIFIED_EMAIL.md).

### Dependencies

- Pin session-kit/v3 v3.2.0, Warden v1.6.0 and Herald v1.4.0.
  HMAC-authenticated Warden requests omit redundant API-key and bearer headers.
  Updating these SDKs does not update deployed provider services.
- Include the README architecture animation and project banner updates.

### Upgrade and rollback

- Replace all gateway instances together. Sessions in the old storage namespace
  require a new login; do not copy records into the new revocable namespace.
  The absolute 24-hour lifetime is additional to the former idle timeout.
- Old SSO tickets require a restarted login. Keep production HTTPS and secure
  host-only nonce cookies, consistent exchange secrets, shared Redis replay
  storage and proxy routes for every allowed destination. Do not mix old and
  new exchange handlers in the serving pool.
- Configure a dedicated metrics credential and update scrape authorization.
  Source-age enforcement remains opt-in and needs a Warden server supporting
  metadata v1. Directory refresh frequency alone is not a freshness guarantee.
- When both Warden client credential types are configured, enable matching HMAC
  on Warden. For an API-key-only server, remove both client HMAC settings instead
  of relying on API-key fallback.
- Herald and herald-totp must be deployed as a compatible pair for upstream
  HMAC v2. Follow their versioned OTP namespace, replay-store and recovery
  instructions; do not downgrade independently deployed providers during a
  gateway rollback. See the [deployment guide](https://github.com/soulteary/stargate/blob/v1.3.0/docs/enUS/DEPLOYMENT.md#revocable-session-migration).
- A rollback loses these security fixes and does not translate new-format
  sessions or tickets. Retained records in the former namespace may become
  usable again when old code resumes. Revoke or retire that namespace, or wait
  for its records to expire, before returning traffic to an older gateway.
  Do not configure old code to read or write the new-format namespace.

## [1.2.7] - 2026-11-02

### Redis limiter changes

- Decode the unchanged Lua reply directly for native Redis Client, ClusterClient
  and Ring instances. Warm allocations fall from 16 to 10 and 487 to 368 B/op;
  cold allocations fall from 17 to 11 and 517 to 405 B/op. Custom adapters keep
  Script.Run and their Eval/EvalSha overrides. Both commands retain explicit
  cluster key routing.
- Reject malformed script reply lengths before indexing. An empty reply injected
  after a miniredis counter commit previously panicked; it now returns an error,
  retains the committed count and keeps the existing bounded local fallback.
  This synthetic reliability case is not an authentication-bypass finding.
- Add a scoped, reproducible Linux arm64 PR comparison with precompiled binaries,
  alternating order, raw samples, command statistics and separate allocation
  profiles. Functional checks and the 95% coverage requirement remain intact.

### Measurements and verification

- Go 1.27.3, Redis 8.2.1, 12 AB/BA rounds, 5,000 serial operations per warm/cold
  counter case. Script loading is outside timing; these are store operations,
  not end-to-end authentication requests.
- Independent Linux arm64 CI: four vCPUs, 16,324,144 KiB, kernel
  7.0.0-1017-azure. Warm medians 140,379 -> 135,891 ns/op, about
  7,124 -> 7,359 ops/s; paired ratio 0.971 (descriptive 95% bootstrap interval
  0.959–0.989). Cold medians 149,388 -> 146,372.5 ns/op, about
  6,694 -> 6,832 ops/s; ratio 0.977 (0.973–0.985). Both paths were faster in
  10/12 rounds. [Raw CI run](https://github.com/soulteary/stargate/actions/runs/41377728981).
- Local Linux arm64 VM: five vCPUs, 5,609,476 KiB, kernel 7.0.9-205.fc44.aarch64.
  Warm medians 94,740.5 -> 89,073.5 ns/op; paired ratio 0.963 (0.917–1.027).
  Cold medians 92,483 -> 92,419 ns/op; ratio 1.001 (0.942–1.032). Allocation
  reductions match CI. Local latency intervals include no change. An earlier
  contended prototype batch is retained without a latency claim. These results
  do not prove that the previously observed ~16% warm penalty is eliminated;
  the atomic Lua execution cost remains. Do not compare absolute times across
  the VM and CI. See the [performance guide](docs/enUS/PERFORMANCE.md).
- Linux race unit suite: 521 top-level passes, 9 skips, 0 failures;
  2138/2247 statements (95.1491%). Real Redis full suite: 528 passes,
  2 existing skips, 0 failures; 2139/2247 (95.1936%). Each also passes
  307 subtests. Seven unit skips require services; two existing empty-session-ID
  harness scenarios remain skipped in both suites, with helper behavior tested
  directly. Vet, golangci-lint 2.14.0, module integrity, repository contracts and
  Linux amd64/arm64 builds pass. PR Windows startup and packaging checks pass.
- Real Redis covers concurrent counts, final-millisecond expiry, ACL rejection,
  generic/native RESP2/RESP3 mixing and a committed write with an acknowledgement
  fault injected afterward. That fault disables client retries to inspect the
  application boundary; it is not an actual network-loss experiment. Two
  miniredis destinations verify cluster routing, not production cluster failover.

### Compatibility, upgrade and rollback

- Lua bytes and SHA-1 77b796d33596742046596a01e3b879c9d1b19715, integer counters,
  millisecond expiry, key prefixes, ACL commands, HTTP/configuration fields and
  metric names/labels are unchanged. Creation and expiry remain atomic. Only
  NOSCRIPT permits an application-level EVAL fallback; uncertain outcomes stay
  errors. Existing per-replica fallback and underlying client retries are unchanged.
- Native process hooks receive *redis.IntSliceCmd instead of *redis.Cmd for limiter
  scripts. Embedders with concrete-type assertions must update hooks to accept
  redis.Cmder or the typed command before rollout. Adapter overrides remain intact.
- No data migration, record removal, new persistent volume or Redis upgrade is
  needed. Mixed v1.2.6/v1.2.7 limiter instances are compatible. Rolling back to
  v1.2.6 is compatible with limiter data but loses this decoder/guard and the
  module-graph pin below; assess that separately. Do not downgrade separately
  deployed providers as part of this rollback.
- The v1.2.6 stable Warden-ID/session-review requirements are inherited, not new
  fixes here. Older published versions reintroduce those security/reliability
  defects; already-rebound historical sessions still require operator review and
  revocation/expiry under that guide.

### Earlier main-branch work first included in this release

- Include the already merged herald-totp v1.2.6 indirect module-graph pin (#225).
  Herald v1.3.13 and Warden v1.4.4 SDK imports/versions remain unchanged. This pin
  does not upgrade a separately deployed Herald/TOTP service.
- Publish the deployment-guide corrections from #224: current runtime descriptions,
  provider ports/data sources, matching templates/assets for standalone binaries,
  and corrected client-side examples. These document existing behavior. Retain
  existing provider data/volumes and use the versioned upstream deployment guides;
  no provider-data migration or deletion is performed.
- Compose validation examples discard resolved configuration stdout. Existing
  logs are not rewritten. If actual credentials were captured, restrict access,
  apply retention/cleanup to logs and exported copies, and rotate exposed
  credentials. English/Chinese guides are current; other partially updated guides
  remain explicitly historical and link to the English guide.

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

[1.3.1]: https://github.com/soulteary/stargate/compare/v1.3.0...v1.3.1

[1.3.2]: https://github.com/soulteary/stargate/compare/v1.3.1...v1.3.2

[1.3.3]: https://github.com/soulteary/stargate/compare/v1.3.2...v1.3.3

[1.3.4]: https://github.com/soulteary/stargate/compare/v1.3.3...v1.3.4
