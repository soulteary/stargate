# Deployment diagnostics

This command is introduced on the current source branch and is not present in
the published v1.3.5 binary. Use a build containing this change. Older binaries
may ignore an unknown subcommand and start the server; do not invoke `doctor`
on them as a capability check. Inspect their release/build metadata first.

`stargate doctor` inspects a scoped configuration snapshot. It does not start the
application, audit writer, tracing exporter, SDK clients or Redis connections.
Running Stargate without arguments keeps its normal startup behavior.

```sh
stargate doctor
stargate doctor --config doctor.json --json
stargate doctor --config doctor.json --health --json
```

Without `--config`, only documented gateway environment fields are read: host,
cookie, callback, session-exchange, enabled-service URL/authentication/TLS fields,
TOTP enablement and session Redis enablement/address. The process does not load
`.env` files. With `--config`, the JSON file is the complete diagnostic snapshot;
it is not merged with environment values or interpreted as shell code.

A manifest has a required `gateway` object and optional `warden`, `herald` and
`totp` objects. Values inside each service object are literal strings using the
service's environment variable names. Unused fields are not inspected. Optional
`gateway_url` supplies the gateway's health root; `gateway_tls_ca_file` supplies
its private CA. An explicit HTTP public URL with Secure cookies is flagged;
localhost browser exceptions remain incomplete. `--gateway-url` and `--gateway-ca-file` override those two fields.
The gateway URL authority must match `AUTH_HOST`; its scheme is never inferred
from the cookie's Secure attribute.

```json
{
  "gateway_url": "https://auth.example.com",
  "gateway": {
    "AUTH_HOST": "auth.example.com",
    "COOKIE_SECURE": "true",
    "WARDEN_ENABLED": "true",
    "WARDEN_URL": "https://directory.internal.example.com",
    "WARDEN_HMAC_KEY_ID": "gateway",
    "WARDEN_HMAC_SECRET": "<copy the configured credential locally>"
  },
  "warden": {
    "WARDEN_HMAC_KEYS": "{\"gateway\":\"<copy the same configured credential locally>\"}"
  }
}
```

The placeholders are not generated secrets or environment substitutions. Use a
private file populated locally from the operator's existing configuration. On
Unix, credential-bearing files must have no group/other permissions (`chmod 600
doctor.json`). Do not pass secrets as command-line arguments, paste this file into
an issue, or publish it with diagnostic output. The tool never prints input
values, hashes, cookies, file paths, raw errors or response bodies. Keep the input
file under your normal secret-management policy.

## What is checked

Offline checks reuse the startup host/cookie-domain, callback-list syntax,
service-root URL and session-exchange-path validators. Other checks inspect
boolean fields, the startup requirement for a 32-byte session-exchange secret
when COOKIE_DOMAIN or CALLBACK_ALLOWED_HOSTS is set, session Redis address syntax, and locally supplied credential pairs.
Password syntax uses the shared startup parser, but no password is authenticated.
This is not full startup validation: UI settings (including
`LOGIN_SUPPORT_URL`), all enablement combinations and service action policies
are outside its scope. A missing server snapshot produces `unknown`, not a
successful authentication claim.

The three credential pairs are gateway→Warden, gateway→Herald and Herald→TOTP.
HMAC takes precedence over API-key emission in SDK requests. Herald configured
for API keys cannot receive an HMAC-selected client. Mapped Herald keys require
an explicit ID; single-secret Herald accepts its default identity. The published
TOTP singleton-map implicit-ID compatibility is retained. An mTLS configuration
is reported as incomplete because acceptance and authorization cannot be proven
by comparing credentials. Secret equality alone does not prove key strength,
protocol support, service/action policy, runtime configuration or a working login.

## Explicit online health mode

`--health` sends only GET requests to fixed configured targets:

- Gateway `/readyz` (its `/healthz` is liveness only).
- Warden `/health`, including its configured source/read policy and health access policy.
- Herald `/readyz`, which checks Redis, not TOTP proxy authentication or delivery.
- TOTP `/healthz`, which checks Redis, not a particular account or encryption key.

Each request is bounded to two seconds and 64 KiB, with a ten-second overall
context. HTTPS verifies the system or explicitly configured CA and server name;
configured service mTLS material is reused. Redirects and environment proxies are
disabled. No API key, HMAC signature or browser cookie is sent. A proxy login page
returning HTTP 200 is not accepted as JSON health. Unrecognized health JSON is
`unknown`. Health requests may create access logs and metrics, but do not call
business or session endpoints.

Health success does not prove HMAC pairing, Redis write ACLs, callback routing,
cookie transmission, provider delivery, OTP verification or backup-code recovery.
Redis may permit PING while denying session/nonce writes. A guarded health route
may need an allowed operator network; do not disable its access policy to get a
pass. No active authentication or smoke mode is implemented: even authenticated
GETs can consume replay nonces and generate audit records. Do not present them as
read-only. Run any functional smoke separately with an explicitly owned synthetic
account and isolated state.

## Output and next steps

Human and JSON output use stable check IDs and `pass`, `fail`, `unknown` or
`skipped` statuses. Exit 0 means the requested scoped checks passed; 1 means a
check failed; 2 means invalid arguments/input; 3 means incomplete checks. None of
these replaces the documented startup validation or browser login test.

- `gateway.host_cookie`/`gateway.callbacks`: correct the named configuration, then
  verify callback routing and cookies in a browser. Do not share cookie values.
- A service `.credentials` failure: compare the named client/server ID and selected
  authentication mode locally. An online 401 alone cannot distinguish a wrong ID,
  wrong secret, unsupported protocol or policy denial.
- A `.health` failure: check the configured route, TLS CA/server name/client cert,
  health access policy and the corresponding Redis dependency. The tool does not
  print remote error details or change server settings.
- `unknown`: supply the missing snapshot or perform the separately authorized
  browser/service test. It is not a pass and does not trigger automatic repair.
