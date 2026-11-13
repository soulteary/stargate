# Security Documentation

> 🌐 **Language / 语言**: [English](SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

This document explains Stargate's security features, security configuration, and best practices.

## Implemented Security Features

1. **Forward Auth Protection**: Centralized authentication layer for protecting backend services
2. **Password Verification**: Support for bcrypt and plaintext (local development only); MD5 and unsalted SHA-512 are rejected
3. **Secure Session Management**: Cookie-based sessions with configurable domain and expiration
4. **Service Integration Security**: In production, Herald requires the selected HMAC or API-key request credentials; optional mTLS adds transport protection and does not replace them. Warden retains its own configured API-key/HMAC/mTLS policy.
5. **Session Sharing Security**: Secure cross-domain session exchange mechanism
6. **Input Validation**: Strict validation of all input parameters
7. **Error Handling**: Error messages and other response data are endpoint-specific; `DEBUG` is not a general non-disclosure guarantee
8. **Security Response Headers**: Automatically adds security-related HTTP response headers
9. **HTTPS Enforcement**: Production environments should use HTTPS
10. **OTP Integration**: Secure integration with Herald for OTP/verification code authentication

## Security Best Practices

### 1. Production Environment Configuration

**Required Configuration**:
- Use bcrypt for password authentication in production
- Configure `COOKIE_DOMAIN` for proper session management
- Use HTTPS via reverse proxy (Traefik, Nginx, etc.)
- Set `TRUSTED_PROXIES` to the reverse proxy's source IPs or CIDRs
- Configure secure session cookie settings

**Configuration Example**:
```bash
export AUTH_HOST=auth.example.com
export PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
export COOKIE_DOMAIN=.example.com
export TRUSTED_PROXIES=10.20.0.10
```

**Note**: Cookie Secure is enabled by default and configurable through `COOKIE_SECURE`; SameSite remains fixed to `Lax`. `COOKIE_SAME_SITE` and `SESSION_TTL` are not implemented. See [CONFIG.md](CONFIG.md).

### 2. Password Security

**Recommended Practices**:
- ✅ Use bcrypt password hashes in production
- ✅ Store password hashes in environment variables
- ✅ Use different passwords for different environments
- ✅ Regularly rotate passwords

**Not Recommended**:
- ❌ Use plaintext passwords in production
- ❌ Hardcode passwords in configuration files
- ❌ Configure MD5 or unsalted SHA-512 hashes; Stargate rejects both
- ❌ Share passwords across environments

**Password Algorithm Comparison**:
- `bcrypt`: Supported and recommended for production
- `plaintext`: Supported only for local development and testing
- `sha512`: Rejected; fast, unsalted general-purpose hashes are unsuitable for password storage
- `md5`: Rejected; it is unsuitable for password storage

### 3. Session Security

**Session Configuration** (per current implementation; full options in [CONFIG.md](CONFIG.md)):
- **Cookie Domain**: Set `COOKIE_DOMAIN` to share sessions across subdomains
- **Secure Flag**: Enabled by default through `COOKIE_SECURE=true`; disable only for local HTTP development
- **SameSite**: Fixed to `Lax`; no `COOKIE_SAME_SITE` env var
- **HttpOnly**: Cookies are automatically HttpOnly to prevent XSS attacks
- **Expiration**: Fixed to 24 hours in code; no `SESSION_TTL` env var

**Configuration Example**:
```bash
export COOKIE_DOMAIN=.example.com
```

### 4. Network Security

**Required Configuration**:
- Production environments must use HTTPS
- Configure reverse proxy (Traefik, Nginx) to handle SSL/TLS
- Use firewall rules to restrict access
- Regularly update dependencies to fix known vulnerabilities

**Recommended Configuration**:
- Use Traefik with Let's Encrypt for automatic SSL certificates
- Configure the narrowest possible `TRUSTED_PROXIES` allowlist; forwarded headers are ignored by default
- Ensure the proxy overwrites forwarding headers instead of appending untrusted client values
- Use network policies to restrict service access
- Monitor and log authentication attempts

### 5. Service Integration Security

When integrating with Warden and Herald services:

**Recommended: mTLS**
- Use mutual TLS certificates for highest security
- Configure client certificates for Stargate
- Verify server certificates for Warden and Herald

**Alternative: HMAC Signature**
- Use HMAC-SHA256 signatures for secure communication
- Configure shared secrets securely
- Use timestamp validation to prevent replay attacks

**Configuration Example (HMAC)**:
```bash
export WARDEN_ENABLED=true
export WARDEN_URL=https://warden:8080
export WARDEN_HMAC_SECRET=your-secret-key

export HERALD_ENABLED=true
export HERALD_URL=https://herald:8082
export HERALD_HMAC_SECRET=your-secret-key
```

## API Security

### Authentication Methods

Stargate supports two authentication methods:

1. **Header Authentication** (API requests)
   - Request header: `Stargate-Password: <password>`
   - Disabled by default. Set `PASSWORD_HEADER_AUTH_ENABLED=true` only for trusted legacy clients.
   - Suitable for API requests, automation scripts
   - Password is validated against configured password hashes

2. **Cookie Authentication** (Web requests)
   - Cookie: `stargate_session_id=<session_id>`
   - Suitable for web applications accessed via browsers
   - Session is validated against stored session data

### Forward Auth Endpoint

The main authentication endpoint `GET /_auth`:

- **Success (200 OK)**: Sets `X-Forwarded-User` header and returns 200
- **Failure (401 Unauthorized)**: Redirects to login page (HTML) or returns 401 (API)

### Rate Limiting

Consider implementing rate limiting at the reverse proxy level:
- Limit login attempts per IP
- Limit forward auth requests per IP
- Use Traefik middleware or Nginx rate limiting

## Data Security

### Session Storage

Stargate supports multiple session storage backends:

1. **Redis** (Recommended for production)
   - Distributed session storage
   - Supports session sharing across instances
   - Configure with password protection

2. **In-Memory** (Development only)
   - Simple, no external dependencies
   - Not suitable for production (lost on restart)

**Redis Configuration**:
```bash
export SESSION_STORAGE_ENABLED=true
export SESSION_STORAGE_REDIS_ADDR=redis:6379
export SESSION_STORAGE_REDIS_PASSWORD=your-redis-password
```

### Sensitive Information Management

**Recommended Practices**:
- ✅ Use environment variables for passwords and secrets
- ✅ Use password files for sensitive configuration
- ✅ Never log passwords or session tokens
- ✅ Use secure key management services in production

**Not Recommended**:
- ❌ Hardcode passwords in configuration files
- ❌ Pass passwords via command line arguments
- ❌ Commit sensitive information to version control
- ❌ Log sensitive user data

## Error Responses and Debug Verification Codes

Stargate does not expose a `MODE` setting. Error responses are endpoint-specific.
Operational details are normally recorded through the logger, but do not treat
`DEBUG` as a general response-sanitization or non-disclosure guarantee.

When Stargate runs with `DEBUG=true` and `COOKIE_SECURE=false`, and the configured Herald service runs in
test mode (`HERALD_TEST_MODE`) and returns a non-empty `debug_code`, Stargate includes
that test verification code as `debug_code` in the successful `POST /_send_verify_code`
response; `POST /_send_reauth_code` relays it under the same conditions. The bundled login page displays the code and fills it into the verification
code field automatically. With `COOKIE_SECURE=true` (the default, used by HTTPS
deployments) the code is never relayed, regardless of `DEBUG`.

This combination exposes a credential-equivalent verification code to the requesting
client. Use it only for local development or testing, never in production. In production,
set Stargate `DEBUG=false` and keep Herald `HERALD_TEST_MODE` disabled.

## Security Response Headers

Stargate automatically adds the following security-related HTTP response headers:

- `X-Content-Type-Options: nosniff` - Prevents MIME type sniffing
- `X-Frame-Options: DENY` - Prevents clickjacking
- `X-XSS-Protection: 0` - Disables the deprecated legacy browser XSS filter
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Content-Security-Policy: default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'` - Stargate's pages load scripts and styles only from their own `/assets` path and use no inline script or style. `img-src` allows `data:` for the TOTP enrollment QR code. `form-action` is not set, because a successful login or step-up redirects to the callback host.
- `Strict-Transport-Security: max-age=31536000` - Sent only when `COOKIE_SECURE=true` (the default). It is omitted for local HTTP deployments with `COOKIE_SECURE=false` and does not include `includeSubDomains`.

`Content-Security-Policy` and `Strict-Transport-Security` are not added to `/_auth` responses. The reverse proxy may relay a denied ForwardAuth response to the browser under the protected application's origin, and those two headers describe the Stargate host only.

## Cross-Domain Session Sharing

Stargate supports secure cross-domain session sharing:

- **Session Exchange Endpoint**: `GET /_session_exchange`
- **Opaque ticket**: AES-GCM ticket contains no raw session identifier
- **Audience validation**: Tickets are bound to an allowlisted callback host
- **Replay protection**: Tickets expire quickly and can be redeemed only once

**Security Notes**:
- Only share sessions between trusted domains
- Use HTTPS for session exchange
- Monitor session exchange attempts

## OTP/Verification Code Security

When using Herald integration for OTP authentication:

- **Challenge-Based**: Uses challenge-verify model
- In production, Herald requires the selected HMAC or API-key request credentials; optional mTLS adds transport protection and does not replace them.
- **Rate Limiting**: Herald handles rate limiting
- **Audit Logging**: Herald maintains audit logs

## Uniform Responses

By default `POST /_send_verify_code` and Warden sign-in through `POST /_login`
answer differently for an identifier that is on the Warden allow list and one
that is not. For a listed account they also report whether a delivery channel
is usable and whether TOTP is enrolled. A client that is not signed in can use
these differences to learn which phone numbers and email addresses are listed.

`LOGIN_UNIFORM_RESPONSES=true` removes these differences from the response
content. For one request, the two endpoints answer with the same status,
headers and body whether the identifier is unknown, inactive or listed,
whatever the state of the listed account, and whatever Herald answers. The
behavior is described in [CONFIG.md](CONFIG.md) and [API.md](API.md).

What is covered:

- The status code, headers and body of `POST /_send_verify_code` and of
  `POST /_login` with `auth_method=warden`, in the JSON, XML and text formats.
- `challenge_id`: a random handle is issued for every admitted request, so
  neither its presence nor its form marks a listed identifier. The Herald
  challenge id stays on the server.
- Herald's own answers (resend cooldown, per-user and per-destination limits,
  locked user, outage) and the `Idempotency-Key` behavior, which are not
  passed on.
- The debug verification code: `debug_code` is never returned in this mode,
  also with `DEBUG=true` and `COOKIE_SECURE=false`.
- The fallback from `phone` to `mail`. The default mode consults `mail` when
  the `phone` of the same request is not listed. With identical responses the
  fallback would still show the owner of one listed mail address whether a
  phone number is listed: next to an unlisted phone, and only then, a code
  would reach that mailbox and sign-in with it would succeed. In this mode a
  request that supplies `phone` is handled by the phone number alone, and
  Warden is asked about one identifier per request.

What is not covered:

- **Response time.** It is not equalized. A send for a listed identifier calls
  Herald and waits for the delivery, and a sign-in for a listed identifier
  calls Herald to verify. Both take longer than the same request for an
  unlisted identifier, and a single send request can be told apart by latency.
  The mode removes the exact signal in the response; it does not hide the
  allow list from a client that measures time.
- **The message.** A listed user receives the code. Anyone who knows an
  identifier can cause that message, as before.
- **Herald quota.** A request for an unlisted identifier never reaches Herald
  and consumes none of its limits. A client that knows several listed
  identifiers and receives the codes of one can fill Herald's per-IP limit with
  sends to them, request a code for another identifier, and learn from whether
  its own next code still arrives. Without `HERALD_CLIENT_IP_HEADER` Herald
  counts that limit against Stargate's address for all users: one client can
  then use it up for everyone, and a send that Herald refuses is still shown as
  sent. Set `HERALD_CLIENT_IP_HEADER` together with Herald's
  `HERALD_TRUSTED_PROXY_HEADER` and `HERALD_TRUSTED_PROXIES`; Stargate logs a
  startup warning when the header is empty in this mode. That keeps the limit
  per client address. It does not remove the signal for such a client: the
  signal needs Herald's per-IP limit to be reached. With the default
  `RATE_LIMIT_WINDOW` of one minute, one client address can request at most 20
  codes through Stargate within any minute: twice
  `RATE_LIMIT_VERIFICATION_MAX` on each of `POST /_send_verify_code` and
  `POST /_send_reauth_code`. A Herald `RATE_LIMIT_PER_IP` at or above the
  number that applies to the deployment keeps its limit out of reach. Herald's
  default of 5 does not.
- **Direct access to Warden or Herald.** The mode concerns Stargate's
  responses only.

Consequences to plan for:

- The page shows "sent" when nothing was sent and one text for every failed
  sign-in. The reason is recorded in the log and the audit log only.
- A request that supplies both `phone` and `mail` is handled by the phone
  number alone. A client that sends both receives no code when the phone
  number is not listed, also when the mail is. The login page has both fields:
  in this mode its hint asks for one of them, and while both are filled in it
  shows that hint and sends nothing.
- The resend cooldown is counted per submitted identifier before any lookup.
  Anyone who knows an identifier can keep it inside its window, and the
  owner's own request is then answered `429` until the window ends. Herald's
  cooldown already allows the same for listed identifiers. The per-client
  limit `RATE_LIMIT_VERIFICATION_MAX` bounds how many identifiers one client
  can hold.
- A `challenge_id` is bound either to the challenge that its own request
  created, or to the challenge of a `challenge_id` that the requester
  presented. When Herald refuses a resend, the code delivered earlier
  therefore stays usable only for a client that presents the handle it holds,
  which the login page does. A request that presents nothing, or a handle for
  another user, channel or destination, receives a `challenge_id` that is
  bound to nothing. Asking for a code for an identifier never yields a handle
  for a challenge that another client holds.

## Re-authentication for Sensitive Operations

Step-up (`/_step_up`), TOTP enrollment and TOTP revocation require a proof that belongs to the session's user, not only a valid session cookie:

- A Warden user with TOTP enrolled must present a current TOTP code.
- Any other Warden user must present a one-time code sent through Herald to the contact registered in Warden. The request cannot name a destination.
- A session created with the shared `PASSWORDS` secret has no user identity and re-enters that password. This is the only case in which the shared password is accepted.

A session cookie that is stolen shortly after login can therefore not enroll an authenticator, and the shared password cannot pass step-up or remove TOTP for a Warden user. TOTP enrollment accepts a proof for 5 minutes; step-up records its own result for 10 minutes.

Limits of this control:

- In a deployment that enables both Warden and `PASSWORDS`, anyone who holds the shared password can sign in with it and pass step-up in that session. Such a session carries no user identity. Remove `PASSWORDS` to close this path; `STEP_UP_ENABLED` no longer requires it when Herald is enabled.
- A user who has lost the authenticator and its backup codes cannot pass step-up or remove TOTP. An administrator must remove the factor in Herald.
- Re-authentication codes are created in Herald with purpose `login`. Stargate keeps them apart from sign-in codes with its own challenge binding, but the delivered message and Herald's limits are those of a sign-in code.

## Vulnerability Reporting

If you discover a security vulnerability, please report it through:

1. **GitHub Security Advisory** (Preferred)
   - Go to the [Security tab](https://github.com/soulteary/stargate/security) in the repository
   - Click on "Report a vulnerability"
   - Fill out the security advisory form

2. **Email** (If GitHub Security Advisory is not available)
   - Send an email to the project maintainers
   - Include a detailed description of the vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

## Related Documentation

- [API Documentation](API.md) - Learn about API security features
- [Architecture Documentation](ARCHITECTURE.md) - Learn about security architecture
- [Configuration Reference](CONFIG.md) - Learn about security-related configuration options
- [Deployment Guide](DEPLOYMENT.md) - Learn about production deployment security recommendations

## Deployment Trust Boundary

Forward-auth protects only requests that actually pass through the configured reverse proxy. It does not firewall the backend service.

- Do not publish protected backend ports on public or untrusted networks.
- Allow backend traffic only from the reverse proxy network or security group.
- Remove client-supplied identity and credential headers before proxying; allow only the values produced by the trusted authentication path.
- Expose Stargate health and metrics endpoints only to the intended operational network.


### Source age and monitoring migration

For revocation-sensitive deployments, enable `WARDEN_SOURCE_MAX_AGE` with a Warden
server supporting snapshot metadata v1 and enforce source freshness on Warden as
well. A five-minute session refresh does not bound the age of the underlying
directory. See [the configuration reference](CONFIG.md#source-freshness-and-scrape-credentials)
for the new source checks and separate `/metrics` bearer credential.
