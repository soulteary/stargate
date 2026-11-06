# Verified email assertions

Stargate's `CheckRoute` ForwardAuth endpoint returns `X-Auth-Email-Verified: true`
only for a Warden session whose successful Herald **email-channel** challenge
was bound server-side to the same challenge ID, explicit Warden user ID, and
current directory email address. A successful check without that evidence returns
`false`. Authentication failures do not export this header.

This proves current email control, not historical mail ownership or a guarantee
that the user's ID will never be reassigned. Applications such as OwlMail still
need a stable, non-recycled subject and their own historical-ownership policy.
Warden can require explicit user IDs; operators remain responsible for lifecycle.

## Evidence and lifetime

After successful challenge verification, the server-side session stores the
subject, normalized email, verification timestamp, source
(`herald-email-challenge`) and format version. Domains use IDNA and lowercase;
ASCII local parts remain exact, including case and plus tags. Unsupported address
forms receive no assertion. Evidence is not inferred from AMR.

Evidence expires after 24 hours (`SessionExpiration`); session access or
authorization refresh cannot extend it or the session's absolute deadline.
Each response checks
the authenticated result's subject and current email. A directory refresh that
changes/removes the email permanently deletes the evidence; changing back cannot
resurrect it. Signing out destroys the session; a new login resets old identity
state. Within the current revocable namespace, authenticated sessions without
evidence return `false`. Sessions from the pre-v1.3.0 namespace require a new
login; see the [session migration](DEPLOYMENT.md#revocable-session-migration).

SMS, TOTP, shared-password login, password-header authentication, trusted identity
headers, a listed directory email and generic `otp`/`email` AMR do not create this
evidence. Missing/unavailable challenge context does not break an otherwise valid
existing login flow, but cannot produce `true`.

Directory revocation still follows existing authorization-refresh settings and
their effective lookup/cache window. This feature does not make revocation
instantaneous. Protected deployments must enable and test the relevant refresh
policy and fail-closed behavior.

## Reverse proxy integration

Copy this header only from a successful trusted Stargate authentication response,
alongside `X-Auth-User` and `X-Auth-Email`. Add `X-Auth-Email-Verified` to the
proxy's explicit authentication response-header allowlist. Strip client-supplied
identity headers from application requests and restrict direct backend access.
Never set verification unconditionally in proxy configuration. Stargate removes
an incoming verification header from its own authentication-check request too.

OwlMail's public multiuser release gate and real identity/notification acceptance
remain pending actual deployment tests. Component tests here do not establish
historical ownership or complete the deployed Stargate/Warden/Herald/Phorge chain.
