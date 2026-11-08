# Formal release compatibility verification

The **Release compatibility** workflow is manual and uses public Linux/amd64
release binaries for Warden, Herald, herald-totp and Stargate. It builds no
application and uses no production credentials. A real Redis service retains the
same state throughout the run. The separate harness-test workflow gates changes
to the manifest parser and download boundaries on pull requests.

An empty `manifest_json` input selects `manifests/baseline.json`. For a full
rollout, supply the same schema with `"mode":"upgrade"`, `old` and `new` tuples.
Each tuple must contain exactly `warden`, `herald`, `totp`, and `gateway`. Each
entry supplies a stable `version` without `v`, full `source` commit, expected
Linux/amd64 asset `sha256`, and the service's exact GHCR `image@sha256:...`.
The input is parsed as JSON, never interpolated into a shell command. The
harness generates download URLs only for the four fixed soulteary repositories.

Before submitting a new manifest, independently verify release signatures and
provenance under the publication procedure. The harness checks the stable
GitHub release, tag target, pinned asset hash and published checksum manifest.
It records Go platform/build metadata and checks version/source fields when
present. Verified Herald v1.4.3 and v1.4.4 binaries omit those fields; their binding is the
previously verified asset hash plus formal tag, not invented embedded metadata.
OCI references are recorded only: native binary execution is not image-entrypoint
or image-signature verification. The workflow source commit is recorded
separately from all product source commits.

The baseline starts the four real services, sends and verifies an OTP through
Stargate and Herald, enrolls and verifies TOTP, and checks single consumption.
Upgrade mode replaces TOTP, Warden, Herald, then Stargate; rollback reverses that
order. Shared Redis, credentials, pending OTP bindings, enrolled credentials,
consumed challenges, backup-code consumption and session absolute deadlines are retained.
Backup checks use the actual persistent Redis array, preserve ordered hashes and
used timestamps, reject a used code with a fresh challenge, then consume an
unused code after rollback. Only counts, used timestamps and a whole-record
fingerprint are recorded; raw backup codes and individual code hashes are not
written to artifacts. Five-minute challenge replay markers are checked only
while their actual TTL is positive. The existing normal quota profile remains
unchanged; no counters or cooldowns are cleared. Fresh HMAC
nonces are accepted before each adjacent transition and re-signed with current
timestamps afterwards; positive Redis marker TTLs and replay-specific evidence
are checked. A separate configuration phase exercises three-leg key overlap,
caller switching, retirement and explicit configuration rollback on the complete
new tuple before reverse code rollback (or the old tuple in baseline mode). Replacing code
alone is never described as restoring retired secrets.

All HTTP requests use loopback on the ephemeral GitHub runner. Services may use
their default wildcard listener; only synthetic state is involved. Email delivery
is deliberately disabled under the test soft policy. Codes are read through
Herald's separate loopback test listener using a synthetic test key. This proves
real challenge/service contracts, not external email delivery or production
acceptance. Redis is an actual workflow service, not miniredis.

An 18-minute whole-run deadline and bounded HTTP/process operations precede
child-process cleanup. The workflow has a 25-minute job limit. Structured results,
build metadata and process logs are uploaded even on failure. GitHub Actions
removes the Redis service container and its network after the job. Failed
verification must remain failed; do not replace formal artifacts with local
application builds or weaken authentication assertions to obtain a pass.
