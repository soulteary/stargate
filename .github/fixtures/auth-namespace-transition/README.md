# Native session / OTP namespace transition regression

This new local fixture addresses the existing compatibility review axes
`Herald challenge state migration`, `Stargate sessions/exchange` and
`Redis/auth isolation`. It does not invent an upgrade-completion contract or
reuse a remote executor. All production repositories remain unchanged.

## Fixed sources and observed starting facts

- Old Stargate 1.2.4: 791b6ccaf506f3197b19f7b406f6e56be668b116.
- New Stargate 1.3.6: 6f04912db1dafa953d244e3f7350bd9ead94cc80.
- Old Herald 1.3.2: 0f90a5ec6d4cb557cb16e0ec70feedb9d5c7bf90.
- New Herald 1.5.2: a4e611f3763d670adec310d222cefed33930765e.

The existing Root safe old-config observation records Redis session storage
(enabled, DB 1), compiled OTP expiry 300 seconds and TOTP false. The fixture uses
its own synthetic prefixes and one isolated DB 0; it does not reuse real Redis,
actual prefixes, credentials, directory records or configuration. Those observed
facts do not prove current traffic has drained or that no old replica exists.

Old Stargate server.go constructs ordinary session storage with the configured
prefix and `sessionfiber.SessionConfig` (session-kit 3.1.3). New server.go normalizes
the base prefix, appends `revocable:v1:` and uses `RevocableSessionConfig`
(session-kit 3.2.1). Existing ordinary cookies/exchange tickets require fresh login.
New logout tombstones do not protect an old binary that reads a different ordinary
record: restoring the old prefix can still authenticate the original old cookie.
This unsafe rollback witness MUST remain in the test and is not a successful
migration outcome. The fixture's safe rollback explicitly chooses a fresh,
disjoint ordinary prefix; no records are copied. Restoring production with its
old prefix is not authorized by a passing fixture.

Old Herald's actual constructor chooses `otp:ch:`; new constructor chooses
`otp2:ch:`, `otp2:vlock:` and `otp2:active:` with strict activation. Both retain
`otp:lock:`. The old and new exact dependency versions are challenge-kit 1.10.2
and 1.11.0. Their actual VerifyWithOptions returns a non-nil error plus
ReasonExpired for a missing/consumed/expired record. The fixture requires that
precise negative result, not any backend failure. No provider is registered or
called. Creating a challenge directly through the native manager explicitly does
not prove delivery, auth middleware, API compatibility or a real login.

## What the executable actually tests (after review)

Each exact repository is archived into a new owned output copy. Frozen manifest,
commit/tree, exact archive size/hash, 380 source/test blob hashes and all overlay
hashes must match before any Go compilation. Each future build is preceded by explicit offline `go mod verify`; this step has NOT run and does not assert that current unpacked cache bytes are already verified. Git replacement objects and ambient
global attributes are disabled. Existing modules are not rewritten; network
module resolution is off. Separate compiled test processes avoid mixing old and
new module versions in one Go import graph.

The session overlay calls each source's actual `setupSessionStore` and uses native
session-kit/Fiber auth helpers to seed only a synthetic authenticated identity.
It reads sessions through the actual store/auth path and invokes actual logout.
Credential proof, Warden, whole production router, browser-bound exchange and
TLS are excluded. The sequence preserves an old ordinary record, rejects it in
new SG, creates/logs out a new session, proves the unsafe old-prefix revival,
rejects both cookies in a fresh rollback namespace, then confirms the new binary
still rejects the logged-out new cookie and the rollback cookie.

The OTP overlay calls each source's actual `NewHandlersWithError`, then that
constructor-selected native challenge manager. It proves the new manager cannot
consume old challenges; an old verifier still can (unsafe mixed-pool witness).
One separate old challenge remains until miniredis advances its TTL by 301 seconds;
the old verifier then requires ReasonExpired. A new strictly activated challenge
is rejected by the old verifier, accepted once by the new verifier, and rejected
on replay. The unrelated lockout key and an unrelated sentinel remain present.
No FLUSHALL/FLUSHDB or wholesale key deletion is implemented. Native consumption
or logout may delete/revoke only their own test records.

Miniredis speaks Redis protocol and executes supported scripts, but it is a test
collaborator. FastForward is a simulated expiry clock, NOT an actual 300-second
wait, physical Redis persistence, failover, real script compatibility or evidence
that production requests/replicas have drained. Closing the owned in-memory test
server at normal end is fixture teardown, not a production Redis cleanup. On any
failed/unknown phase the runner retains evidence, does not kill/restart/clear the
server and does not report passed. It may later reach its own 120-second fixture
wait. No remote system is contacted.

## Root's future local command (not yet executed)

Use already available Go 1.27.3 and a prepopulated trusted dependency cache. No
installation/download happens in this runner. Output must be a new absolute path
whose existing parent is canonical:

```
python3 -I -B run_transition.py \
  --sg-old-source /absolute/stargate-git-repository \
  --sg-new-source /absolute/stargate-git-repository \
  --herald-old-source /absolute/herald-git-repository-with-old-object \
  --herald-new-source /absolute/herald-git-repository-with-new-object \
  --go /absolute/go1.27.3/bin/go \
  --go-mod-cache /absolute/existing/pkg/mod \
  --output /absolute/new-owned-output
```

This starts only local synthetic test processes and loopback miniredis/control
listeners after independent source review. Ready files, session IDs, codes, logs
and PID records stay private under the owned output. The safe result contains only
fixed source/phase labels, terminal statuses, hashes and explicit limitation flags.
Nothing is uploaded. Five pure offline controls and gofmt parsing are current
author evidence; native Go compilation and the 13-phase sequence remain unrun.

## Mandatory real upgrade gates still open

Pause OTP creation and drain all in-flight requests; establish the final old
issuance time and wait at least its real maximum 300-second lifetime plus bounded
clock/queue margin. Replace every old verifier before resuming. Do not use the
fixture's simulated clock or namespace rejection as proof of these facts.
Preserve idempotency derivation secrets/unresolved sends, nonce state, audit and
lockouts; do not clear Redis. The fixture does not reconcile delivery_unknown.
Deploy all SG replicas coherently, require fresh login for old cookies/tickets,
and bind rollback to a separately approved fresh namespace rather than revive old
ordinary records. Warden freshness, external session exchange, old TOTP recovery,
real Redis permissions/scripts, persistence and rollback operational coordination
remain separate gates. Actual TOTP is currently false; no enabling or migration
is performed here. Optional features remain off and no production factory, permit,
profile, deployment authority or success of the actual upgrade is created.

## Atomic readiness successor

Original fd572 is retained as blocked before execution: it created the final
readiness filename before writing, so polling could observe empty/partial JSON.
This successor writes a distinct O_EXCL 0600 pending file, checks full write and
Close errors, then publishes with os.Rename. Only the complete final name is
polled. Existing pending/final names are refused; no retries or overwrite adoption.
The new native file-only publication test runs before the Redis fixture in a
future reviewed invocation, checking complete bytes and spent-name refusal.
It has not run yet. Six pure Python offline methods plus gofmt/AST are current
evidence. No native component or real TTL/storage result is claimed.
