# Redis rate-limit measurements

## Scope and safety

This follow-up measures the Redis fixed-window limiter after v1.2.6. The native
client path decodes the script's two integers directly instead of allocating a
generic reply, boxed integers and a converted integer slice. Native `Client`,
`ClusterClient` and `Ring` use this path. Custom adapters retain `Script.Run`,
including overridden `Eval`/`EvalSha` methods and deferred clients. Native process
hooks now receive `*redis.IntSliceCmd` for these scripts rather than `*redis.Cmd`;
custom hooks must use the `redis.Cmder` interface or handle the typed command.
Update hooks that assert the old concrete type before deployment. HTTP fields,
configuration names, metric names and labels are unchanged.

The Lua source and SHA-1 (`77b796d33596742046596a01e3b879c9d1b19715`) are unchanged.
Counter creation and expiry still commit together. The millisecond window,
legacy counters without expiry, Redis key names, ACL commands and existing
records stay compatible. Both script commands declare their first key position
for cluster routing. Only `NOSCRIPT`, which means no execution occurred, permits
an application-level retry using `EVAL`. Other failures remain observable; the
existing local fallback and underlying Redis client's retry policy are unchanged.
No counter deletion, new key, relaxed quota or ignored write error is introduced.

The decoder rejects replies that do not contain exactly two integers. A miniredis test injects an
empty reply after committing its counter and panics in v1.2.6; the regression now returns
an error, retains the committed Redis count and uses the bounded local fallback.
This is a malformed-reply reliability test, not an authentication-bypass finding.

## Local investigation on 2026-11-02

The VM matches the earlier investigation: Linux arm64, kernel
`7.0.9-205.fc44.aarch64`, Go 1.27.3, Redis 8.2.1, five vCPUs and 5,609,476 KiB
reported memory. Both binaries are compiled before timing. They use the same
benchmark, dependencies, Redis and loopback TCP settings. Historical comparisons
replace only `ratelimit.go` with the pre-atomic implementation from `a55659a`;
that implementation is a diagnostic control, never a production rollback target.

Twelve AB/BA rounds of 5,000 serial operations per warm/cold case reproduced a
warm-path cost, but not a fixed 16%: historical/current warm median times were
83,181/91,587.5 ns/op, with a paired latency-ratio median of 1.085 and the current
path slower in 9/12 rounds. Cold median times were 170,773.5/94,940 ns/op.
These serial times imply about 12,022/10,919 warm operations per second.

Separate diagnostics compare the historical `TTL` pipeline, a `PTTL` pipeline,
the current script, raw generic script replies and `PING`. Redis `INFO
commandstats` reports roughly 13–14 microseconds per `EVALSHA` versus 3–4
microseconds for `INCR` plus `TTL`/`PTTL`. These are server execution times,
including scheduling effects, not an OS CPU profile. The Lua total already
includes its nested commands; do not add their times again. `PING` itself varied
around 88 microseconds, demonstrating substantial transport/scheduling cost.
The atomic script contributes server overhead; returning to the historical
pipeline would reintroduce the expiry and uncertain-write defects.

The first optimization comparison encountered VM load averages around 15 and
above on five vCPUs, with concurrent compiler/package work. Its latency samples
are retained, but do not establish an improvement or regression. Across those
samples the native warm path consistently reduced allocations from 16 to 12 and
about 487 to 400 B/op; cold allocations changed from 17 to 13 and about 518 to
438 B/op. Allocation reduction alone does not prove lower end-to-end latency.

## Repeatable Linux comparison

Use an isolated Redis 8.2.1 instance and Go 1.27.3 on Linux. Never point benchmarks
at a production Redis. Do not compile, run unrelated tests or profile during the
timed comparison. The script compiles both revisions first and uses the same
Redis for alternating rounds. Record concurrent VM load and retain every sample.

```bash
STARGATE_TEST_REDIS_ADDR=127.0.0.1:6379 \
  python3 .github/scripts/benchmark-redis.py \
  --baseline 7f7c0b101baa1b58da98eda98c9a5c8a054100b0 \
  --output /tmp/stargate-redis-performance --rounds 12 --operations 5000
```

The JSON records revisions, dirty state, environment, times, serial throughput,
bytes and allocations, load averages and command statistics. Server statistics
cover both cases plus setup/calibration in each invocation. The paired bootstrap
interval is descriptive; it cannot eliminate shared-VM interference. Allocation
profiles run separately after timing. The PR CI repeats this on an independent
Linux arm64 runner and uploads raw logs, profiles and JSON for 30 days when
rate-limit code, its benchmark script or the PR workflow changes. Benchmark
errors or skipped cases fail the job; latency noise is not a reason to weaken
functional tests or the 95% coverage requirement.

For a same-binary decoder comparison:

```bash
STARGATE_TEST_REDIS_ADDR=127.0.0.1:6379 \
  go test ./src/internal/handlers -run '^$' \
  -bench '^BenchmarkRedisRateLimitReplyDecoder$' -benchmem -count=12
```

## Compatibility and validation limits

No data migration, new persistent volume, Redis version upgrade or traffic pause
is required for this optimization. v1.2.6 and the optimized version share the same
script and records and can run together. Rolling back to v1.2.6 restores generic
decoding without changing stored data. Earlier published versions lack the
v1.2.6 security/reliability fixes and remain unsuitable rollback targets.

Regression coverage includes normal counts, final-millisecond expiry, legacy
no-expiry counters, script cache flushes, raw/normalized `NOSCRIPT`, native and
adapter mixing, RESP2/RESP3, ACL rejection, concurrent requests and a committed
write with an injected lost acknowledgement. The last fault is injected after
an actual Redis write with client retries disabled; it is not a production
network-failure experiment. Two miniredis destinations verify cluster key routing;
this does not claim production cluster failover validation. Production load,
Redis persistence loss, replication failover and real network loss remain
separate operational concerns.
