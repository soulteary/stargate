# Real Chromium acceptance

Dispatch `Real browser acceptance` with reviewed full 40-character
`fullstack_ref` and `gateway_ref` commits and an independently verified canonical
Herald image digest. Herald 1.5.0 and a gateway consuming its metadata SDK are
required for the known cooldown timing assertion. Do not substitute mutable tags.
The workflow source is the selected dispatch ref; results also bind both built
checkout identities. It builds a local gateway image, not a published artifact.

The reviewed full-stack configuration provisions a private SMTP test CA and a
Mailpit certificate with verified hostname; STARTTLS remains required. The CA is
scoped to Herald, with no TLS bypass. This uses the actual full-stack generator and Compose deployment, real Redis,
Warden, Herald, herald-totp, SMTP and Mailpit. Three service HMAC legs retain
separate credentials. No service test-code endpoint, limiter changes, Redis flush,
automatic retry of a consumptive POST, or injected authentication state is used.
A fresh task-owned stack is removed, including its volumes, in `finally`.

Chromium tests password and delivered email login, wrong email code, TOTP
enrollment, download/acknowledgement of the recovery set, authenticator login,
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
user. A real BFCache restoration checks cleared save controls; a reload is recorded
as not exercising BFCache. Logout must use same-origin POST; cross-origin POST and GET are rejected.
Authenticated pages must return `Cache-Control: no-store`.

Only a redacted outcome JSON and the allowlisted `browser-account.png` are uploaded.
The account screenshot is taken only after verifying no secret controls or synthetic email appear. No HAR, trace, video, secret-page screenshot,
storage state, service log, email body, QR, code, password or cookie is uploaded.
Recovery downloads stay in Playwright's temporary directory and disappear when
the browser context closes. Failure messages are generic, including subprocess
and Playwright failures. Fixed boolean signatures may be derived in memory from
bounded structured SMTP error fields; these are observed signatures, not a unique
root-cause determination. No input error text is returned. Do not add raw exception
bodies to the report.

The first profile is explicitly local HTTP on loopback with Chromium hostname
mapping. It does not prove Secure-cookie transmission, production TOTP TLS,
external email delivery, Safari, Firefox, iOS or Android behavior. HTTPS with an
isolated trusted test CA is a separate required follow-up profile; never use
`ignore_https_errors` or disable TLS verification to fill that gap.

Pure input checks: `python3 -m unittest discover -s .github/browser -p 'test_*.py'`.
Actual browser acceptance requires the workflow's clean Linux Docker runner.
