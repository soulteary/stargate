# Fixed released native namespace qualification

The path-scoped/manual `released-auth-namespace-transition.yml` qualifies only
these released sources: Stargate 1.2.4 to 1.3.6 and Herald 1.3.2 to 1.5.2. It does
not replace existing current-main CI, Work IM suites or deployment gates. There
is no push trigger or unrelated application-path trigger.

The eight original files are byte-for-byte copies of reviewed fixture freeze
35bca125daf37625a7f2b3634fa6ad12f1eba5a1a4d8ed3c2634db00fb148d84.
Independent source peer:
01f9f228a3193128adaef60b01f6c2846d9a9b0bda97d8f4efa74e249f90bf71.
Root subsequently ran that exact fixture; independent actual peer
78942d7ca2c4ce629e015ab4a674f640dc503777382b8a86fcedd9ec21664d8e
confirmed 22 known-zero children and 13 ordered phases, including one simulated
clock step. The inherited README's source-stage unrun statement is historical;
these prior local results do not imply this newly packaged CI has run. Private
Root receipts are not included or uploaded.

CI checks out four exact released source commits with no persisted credentials.
Existing Go 1.27.3 downloads and verifies their unmodified go.mod/go.sum graphs;
no dependencies are added. The runner then disables module downloads and verifies
each local graph before building. Its fixed manifest/commit/tree/archive-size/SHA,
380 source/test blob and overlay checks run before compilation. An Ubuntu Git/tar
or attributes difference fails closed: never silently refresh/normalize hashes or
substitute current source for an old version. All six source/file controls and
seven additional portable projection/CI methods are CI-gated before execution.

Native SG setupSessionStore and auth/logout helpers exercise real released
session storage APIs; synthetic authentication seeding is not proof of credentials
or a whole login/router. Each Herald native constructor chooses its own actual
challenge manager and namespaces; direct manager calls intentionally perform no
provider delivery. All four versions use one owned miniredis test server, not a
physical Redis instance. Its FastForward is not a real 300-second wait or proof
of a drained production pool. TOTP, Warden, real IM, physical persistence, browser
exchange, TLS, capacity and actual deployment/rollback remain outside acceptance.

The unsafe old-prefix session revival and old-verifier OTP consumption witnesses
are required evidence, not a success criterion for operating that unsafe topology.
The safe test route requires a fresh disjoint rollback session prefix; all real
replicas, in-flight requests and maximum OTP TTL must still be coordinated before
an actual upgrade. Redis is never flushed; unrelated sentinel/lockout state is
preserved. Native consumption or logout may revoke its own fixture records.

## Public artifact boundary

`public_results.py` has its own namespace schema; it does not reuse the Work IM
three-service schema. It accepts only exact four-source metadata and all 22
fixed terminal files: four module verifies, four builds, twelve native phases,
one file-publication test and one Redis fixture. The 12 native phase summaries
must match their terminal records and the exact 13-step order; the clock entry
must say simulated. Native terminal PASS is mandatory, build/module PASS markers
must remain null, all exits must be integer zero, and stdout/stderr references
must be SHA-256 values. Unknown/missing/extra/wrong-source/false witness or true
real-world acceptance fields refuse projection.

Only `result.json`, `native-results.json`, `source-inputs.json`, or a fixed
`failure.json` can be uploaded, using four explicit paths and no globs. The
projector never opens readiness, token/code records, raw stdout/stderr, PID/start
records, copied sources, test binaries or caches. Errors produce only a fixed safe
failure statement and fail the step; artifact upload cannot hide a native failure.
Raw private files stay solely in ephemeral runner storage. No secrets, live
configuration, mappings, IM credentials or new deployment authority are supplied.
