# Fixed released-version CI qualification

`released-work-im-native-three.yml` runs on manual dispatch or pull requests that
change this fixture/workflow only. It does not replace the always-run Work IM
bridge or any current-main gate. No push event or unrelated application source
path triggers this expensive released-version suite.

The eight original files are copied byte for byte from source-bound candidate
56ed56c5e14c0567a69c4c546bc28be9be40a4e8c346967900913b9bc409bb2c.
Its independent source peer is b3a3da9a2db55c36688f342f6463e2d56bc4f69fe25484244426086e6ecaaa32.
Root subsequently ran that exact candidate: all three builds and runtimes exited
0 and all seven required subtests passed; independent actual peer
8660052e6537a19782ef0367a2063a7902926f2c3489a39b44512de1ff9072ed.
Those receipts are prior local evidence, not evidence that this new CI has run.
The inherited README's source-candidate evidence paragraph describes the original
author stage, not the later Root result. No private Root receipts are published.

CI checks out exact released SG6f/Herald a4/adapter47c with checkout credentials
not persisted. Go 1.27.3 downloads and verifies their existing public go.mod/go.sum
graphs; no new dependencies are added. Only this dependency preparation requires
module-network access. The runner then forces GOPROXY=off/GOSUMDB=off/GOTOOLCHAIN=local,
uses a clean synthetic environment and exercises only owned loopback services.
The adapter's closed RoundTripper has no external transport fallback. This is not
an OS-level network namespace guarantee; it is the reviewed fixed test transport.

The archive SHA, size, all 279 selected source/test blob hashes and three overlay
hashes must match before compilation. Git archive is expected to reproduce the
same fixed public bytes on Ubuntu; a Git/tar/attributes difference fails closed.
Do not normalize archives, refresh expected hashes or silently substitute a source
if public CI reveals a difference. Investigate it as a separate source successor.
The original source-side `test_controls.py` is retained unchanged and all nine
methods run before the four portable `test_public_ci.py` controls. Its actual
adapter overlay-tamper materialization test uses a fixed sibling repository name;
CI creates that test-only alias to the already checked-out exact released adapter.
No archive, provider or dependency is downloaded through the alias. These controls
need no listeners or component execution.

The only uploaded files are projected `result.json`, `native-results.json`,
`source-inputs.json`, or fixed `failure.json`. `public_results.py` reads only three
bounded fixed-schema metadata inputs. It never opens raw stdout/stderr, readiness,
PID/argv records, OTP or inbox data. Raw fixture logs remain in runner-private temp
storage and expire with the ephemeral CI runner. Projection failure fails its CI
step; the always-run artifact step cannot convert native failure into success.

The suite qualifies these fixed released versions through native adapter Setup,
Herald router, SG send/login/session helpers and a synthetic upstream only. It
does not qualify current-HEAD whole SG routing, TLS, real Warden, physical Redis,
upgrade/session migration, verified live mappings, real IM delivery, restarts,
durable exactly-once behavior, capacity or deployment. Native comma-recipient
semantics are unchanged. All application and production dependencies remain intact.
