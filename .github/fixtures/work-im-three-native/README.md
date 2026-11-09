# Fixed three-service native work-IM fixture (source candidate)

This is a local synthetic integration harness, not deployment or a production
adapter. The three exact public sources are Stargate 6f04912, Herald a4e611f and
herald-dingtalk 47c2d02. The runner reads their fixed Git objects and archives each
into a new owned directory, adding exactly one test file to each independent Go
module. It never edits the source clones, go.mod, handlers, middleware or clients.

The adapter test installs a closed `http.DefaultTransport` before unchanged
`router.Setup`. The exact adapter source contains no init functions; its config
package variables read environment and the NewClient constructors only construct
a client with nil Transport and ErrUseLastResponse. No HTTP is sent until a send
operation. The closed transport handles only HTTPS oapi.dingtalk.com GET gettoken
and POST asyncsend_v2 with fixed synthetic credentials and bounded JSON. It has no
fallback or network implementation; unexpected endpoints fail. Mobile lookup is
not enabled, and general native comma-recipient behavior is not changed. This
fixture intentionally tests individual server-derived userid mappings only.

Herald uses its actual complete `NewRouterWithClientAndHandlersE` router, API-key
middleware, native OTP handlers, rate limits, provider registry/HTTPProvider and
native audit shutdown. Redis is explicitly miniredis, not a physical Redis upgrade
or persistence test. Stargate invokes its actual send/login handlers, SDK and
challenge binding store, then reads the authenticated session using actual auth
session helpers. Warden /user is explicitly synthetic; account freshness,
restarts, Warden authorization and live administrator mappings remain unproved.
Stargate whole production router/proxy/TLS is also outside this handler slice.

Seven required subtests cover opaque server mapping/client destination injection,
replay without second delivery, wrong-user refusal without consuming the owner
challenge, owner session and consumed replay, unmapped refusal, adapter API-key
and native idempotency/fingerprint conflict, and provider failure propagation.
Codes are obtained only from the adapter's closed native upstream request, never
from Herald debug-code endpoints. Failure mode is an actual native 503 response;
existing retry behavior is preserved, so its assertion requires an upstream
attempt and refusal rather than incorrectly claiming one attempt/exactly once.
Adapter idempotency is process-local; failed retries/restarts are not covered as
durable guarantees. A successful send task is not a real IM delivery guarantee.

## Root-only future local invocation (not run by the author)

Use existing Go 1.27.3 and a populated trusted module cache; downloads are disabled.
Choose a new absolute output path whose parent exists and is canonical:

```
python3 -I -B run_three.py \
  --sg-source /absolute/stargate-repository \
  --herald-source /absolute/herald-repository \
  --adapter-source /absolute/herald-dingtalk-repository \
  --go /absolute/go1.27.3/bin/go \
  --go-mod-cache /absolute/populated/pkg/mod \
  --output /absolute/new-owned-output
```

The process environment is rebuilt from a small allowlist; no inherited service
configuration, proxy, key, GOFLAGS or workspace. Package initialization still
runs and reads these synthetic/default env values. Compilation occurs once per
component with -mod=readonly. Each component is a separate process: no Go internal
package import crosses modules and the adapter global transport is process-local.
No DingTalk network requests occur. Owned loopback listeners are necessary for
the future integration run; no component/listener has been started by this author.

Normal completion asks only the two owned fixture HTTP endpoints to stop, then
requires terminal exit 0 and each exact expected Go test/subtest PASS exactly
once, no skip/fail. On timeout, missing readiness, unexpected result or exception,
logs and PID records are retained; the runner does not signal, kill, restart or
remove processes/directories. Fixtures have their own 90/100 second test waits,
which may later terminate naturally; a runner unknown record is not retroactively
converted to success. A failed known SG test still receives orderly fixture
shutdown; compile/start/unknown failures may leave fixtures for Root to inspect.
Logs and readiness are private 0600 under owned 0700 output. Public result contains
only source/hash/terminal metadata, no OTP, body, recipient or endpoint.

Current evidence is source review preparation, gofmt parsing and four pure Python
controls. No Go compilation, components, HTTP listeners, real Redis/Warden,
DingTalk requests, network, Docker, SSH, deployment, migrations or capacity tests
have run here. An independent peer and Root's future actual invocation are required.

`enterprise-inputs.TEMPLATE.json` separately identifies the three exact enterprise
configuration keys at adapter47c and a verified canary `dingtalk_userid` mapping.
The userid is not a native adapter ENV key. `API_KEY` is an independent internal
service secret, not an enterprise DingTalk credential. No template value enables
or authorizes delivery; real values must remain outside this fixture bundle.

## Source-bound successor

The original 806ca source candidate is preserved and blocked by peer 907d104c:
its run records were self-reported rather than compared to frozen source inputs.
This successor pins the exact source-inputs.json SHA inside the runner, verifies
fixed commits/trees, archive size/SHA and all 279 selected source/test blobs before
compilation, and separately pins the three overlay byte hashes. Git replace
objects are disabled and the global attributes file is /dev/null. Any repository
info-attributes/config behavior that changes archive bytes fails the exact archive
check; it is not normalized or trusted. The manifest does not contain the runner
or freeze hash, so there is no self-hash cycle. Controls include actual temporary
Git replacement trees and replacement-archive refusal, manifest/archive/blob and
overlay tampering. These are local source/file controls, not component execution.
