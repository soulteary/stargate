# Real Chromium acceptance

The `Real browser acceptance` workflow runs on a pull request and on manual
dispatch. It builds a local gateway image from source, not a published artifact,
and the report records the harness commit and both product commits it built.

On a pull request the product under test is the pull request itself:
`fullstack_ref` and `gateway_ref` default to the commit the run is for
(`github.sha`, the merge commit of the pull request with its base, the same
commit the harness is checked out from). The workflow holds no fixed product
commit. It runs when a pull request changes

- the harness or the workflow: `.github/browser/**`,
  `.github/workflows/browser-acceptance.yml`;
- what the gateway image is built from: `src/**` except `*_test.go`, `go.mod`,
  `go.sum`, `docker/Dockerfile`, `.dockerignore`;
- the deployment the scenario starts: `examples/full-stack/**` except its
  Markdown files.

A pull request that changes none of these does not run it. A newer push to the
same pull request cancels the run in progress.

On manual dispatch both refs are optional and default to the dispatched commit.
To test other sources, give full 40-character commits; `validate_inputs.py`
refuses anything else before a checkout or a shell command uses the value. The
harness always comes from the dispatched ref. The Herald image stays pinned: give
the independently verified canonical digest of Herald 1.8.0; the HTTPS profile
accepts only that digest. Retry metadata (Herald 1.5.0 or later) and a gateway
consuming its metadata SDK are required for the known cooldown timing assertion.
Do not substitute mutable tags. When `examples/full-stack` moves to another
Herald release, the digest in the workflow and in `production_tls.py` must move
with it.

The workflow uses the `pull_request` event with `contents: read` and no secrets,
so a pull request from a fork runs it with a read-only token. It needs a
GitHub-hosted Linux runner with Docker and `sudo` (loopback host names, a private
test CA in the system and Chromium trust stores). The three jobs run one after
another and take about nine minutes together.

The reviewed full-stack configuration provisions a private SMTP test CA and a
Mailpit certificate with verified hostname; STARTTLS remains required. The CA is
scoped to Herald, with no TLS bypass. This uses the actual full-stack generator and Compose deployment, real Redis,
Warden, Herald, herald-totp, SMTP and Mailpit. Three service HMAC legs retain
separate credentials. No service test-code endpoint, limiter changes, Redis flush,
explicit retry of a consumptive POST, or injected authentication state is used.
The HTTP profile never navigates Back to the enrollment POST document.
A fresh task-owned stack is removed, including its volumes, in `finally`.

Chromium tests password and delivered email login, wrong email code, TOTP
enrollment with its proof, download/acknowledgement of the recovery set, authenticator login,
backup recovery and refusal to reuse a backup. The mail path starts at the protected
app, retains its callback, and must return to the actual hello page with200 and
the published response contract; ForwardAuth must identify the stable account.
After logout the protected app must require login again. Enrollment's consumed TOTP step
expires naturally before the next login. A second real send encounters normal
cooldown. Optional timing is stripped from another **real 429** response only to
exercise old-server UI compatibility; a separate intercepted disconnection
checks ambiguous-result handling and absence of POST retry. These two browser
transport fixtures are identified separately from actual backend success checks.
A natural 61-second quiet interval precedes consumed-code rejection, which must
return 401; an unused-code positive control must then authenticate the expected
user. Logout must use same-origin POST; cross-origin POST and GET are rejected.
Authenticated pages must return `Cache-Control: no-store`.

`POST /totp/enroll` needs the user's proof in the same request. The scenario's
session is created by a delivered email code and the user has no authenticator,
so the proof is a new code sent from the enrollment page to the address
registered in Warden. Both profiles that enroll first send the start request
with the session cookie and no proof and require 401 without an `otpauth://`
URI, then request the code, read it from the captured mail and start the
enrollment with it; the start must answer 200. Herald applies the resend
cooldown of the sign-in code to the same address, so the first request for the
proof code may be refused with 429. The harness then waits the reported time,
at most 120 seconds, and asks once more. The refused request delivers nothing,
so this is not a retry of a consumptive POST. The gateway source must contain
the proof requirement (Stargate pull requests #295 and #334); with an older
`gateway_ref` the enrollment page has no proof form and the run fails.

Only a redacted outcome JSON and the allowlisted `browser-account.png` are uploaded.
The account screenshot is taken only after verifying no secret controls or synthetic email appear. No HAR, trace, video, secret-page screenshot,
storage state, service log, email body, QR, code, password or cookie is uploaded.
Recovery downloads stay in Playwright's temporary directory and disappear when
the browser context closes. Failure messages are generic, including subprocess
and Playwright failures. Fixed boolean signatures may be derived in memory from
bounded structured SMTP error fields; these are observed signatures, not a unique
root-cause determination. No input error text is returned. Do not add raw exception
bodies to the report.

The first profile is local HTTP on loopback. The separate `production_tls` job
runs only after that profile passes, on a new GitHub-hosted Linux runner. It uses
the production Compose and TOTP TLS extension with an immutable local image ID
built from the same gateway source. A private test CA is explicitly trusted by
the ephemeral runner and Chromium NSS store, then removed. SMTP uses a separate
CA; Herald's merged trust bundle must verify both SMTP and TOTP. Incorrect SMTP
credentials and invalid service HMAC are negative controls. The browser must
complete HTTPS mail login, reach the actual protected app, retain Secure HttpOnly
host-only cookies on both hosts, and require login again after POST logout.
No certificate bypass is used. The signed status request is an active probe that
may write replay and audit records in the owned test Redis; it is not a read-only
health check. Each profile cleans up only its own containers/networks/volumes.

The separate `bfcache` job runs after HTTPS acceptance with the same immutable
product inputs and a new owned stack. It uses full Chromium with only Playwright's
BFCache-disabling and repost-prompt-disabling defaults removed. Product no-store,
TLS verification, authentication and cookie behavior remain unchanged. A GET
login restoration must simultaneously preserve the per-document random identity,
produce a trusted `pageshow.persisted`, and issue zero new main-document requests.
The authenticated account must either meet that same proof or report only the
explicit `CacheControlNoStoreHTTPOnlyCookieModified` privacy exclusion, with no
unknown or additional reasons. The latter is not account restoration coverage;
normal account navigation and real ForwardAuth identity must still succeed.

Actual enrollment and confirmation then display ten recovery entries. A pagehide
observer registered after product initialization verifies rendered codes and QR
are cleared and save controls disabled when leaving. Chromium excludes POST main
documents through `HTTPMethodNotGET`; this is a source-based limitation, not an
observed restoration. The job never navigates Back to the secret POST page.
Exact CDP response-body unavailability is reported separately: login POST200 plus
actual protected-app response and ForwardAuth identity are required; unread JSON
is never described as `success=true`. Other response errors fail.

This is controlled acceptance of the flows described above, not regression
coverage of every UI change: it covers what the scenario drives and nothing
else. A product change is exercised when its pull request touches one of the
paths listed at the top; the report names the commits that were built.
These profiles do not prove external email delivery, Safari, Firefox, iOS or
Android behavior.

Pure input checks: `python3 -m unittest discover -s .github/browser -p 'test_*.py'`.
Actual browser acceptance requires the workflow's clean Linux Docker runner.
