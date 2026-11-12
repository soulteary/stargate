# Local work-IM integration regression

This opt-in regression connects actual Stargate handlers and its Herald SDK to
a separately compiled Herald handler fixture, including its challenge manager,
Redis test backend, idempotency store and HTTP provider. The last provider hop
is a controlled loopback inbox. It does not use DingTalk credentials, send work
notifications, load deployment configuration or test a public gateway.

Use explicitly reviewed source checkouts. Herald must contain
`TestWorkIMPilotServeLocalBridge`; the companion recipient-preservation change
adds this helper. Pin the Herald commit before running. Do not use an arbitrary
test executable or a running application as the fixture.

From the Stargate checkout:

```sh
python3 -I -B .github/integration/run_work_im.py \
  --stargate . \
  --herald /path/to/reviewed/herald-checkout \
  --go /path/to/go1.27.4/bin/go \
  --output /path/to/fresh-result-directory
```

An existing module cache is required; module downloads are disabled. Set
`GOROOT` and `GOMODCACHE` explicitly if the shell selects another Go installation.
`--go-cache` optionally reuses an existing build cache. The output directory must
not already exist. The runner records source hashes, exact Go arguments, logs,
test outcomes and the compiled fixture hash. Source hashes are checked before
and after each step. Failure logs are retained, without automatic retries.

The runner first executes Herald component tests, builds its local fixture,
then runs `TestWorkIMCurrentPair`. The fixture creates only ephemeral loopback
listeners and uses synthetic identities. Its test-only inbox contains synthetic
codes, and its shutdown endpoint exists only in this test binary. The caller
requests shutdown and requires the child to exit normally.

The paired test verifies opaque receiver preservation, matching gateway challenge
binding, replay without a second provider send, refusal of another user, valid
owner login after that refusal, and refusal after consumption. Existing unit
tests cover the additional source, purpose, channel and response binding rules.

Without `WORK_IM_HERALD_TEST_BINARY`, the paired Go test explicitly skips.
The runner supplies its own freshly built fixture and rejects a skipped paired
test. Existing Herald tests may explicitly skip separate real-Redis integrations;
those skips remain visible and do not become real-Redis evidence.

This fixture does not prove real enterprise recipient ownership, actual work-IM
delivery, production service authentication, TLS, browser behavior, capacity or
revocation timing. Validate those separately before admitting ordinary users.
