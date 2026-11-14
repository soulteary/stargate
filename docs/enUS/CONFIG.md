# Configuration Reference

This document details all configuration options for Stargate.

## Table of Contents

- [Configuration Methods](#configuration-methods)
- [Configuration Quick Reference](#configuration-quick-reference)
- [Required Configuration](#required-configuration)
- [Optional Configuration](#optional-configuration)
- [Password Configuration](#password-configuration)
- [Configuration Examples](#configuration-examples)

## Configuration Methods

Stargate is configured via environment variables. All configuration items are set through environment variables, no configuration file is needed.

### Setting Environment Variables

**Linux/macOS:**

```bash
export AUTH_HOST=auth.example.com
export PASSWORDS='plaintext:yourpassword'
```

**Docker:**

```bash
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword ghcr.io/soulteary/stargate:1.3.8
```

**Docker Compose:**

```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword
```

## Configuration Quick Reference

Below are all environment variables used in code. "Required" means the service will not start without it when applicable.

| Variable | Type/Values | Default | Required |
|----------|-------------|---------|----------|
| `AUTH_HOST` | String | — | Yes |
| `PASSWORDS` | See password config | — | Yes when Warden disabled |
| `PASSWORD_HEADER_AUTH_ENABLED` | true/false | false | No |
| `DEBUG` | true/false | false | No |
| `LOG_LEVEL` | debug/info/warn/error | info | No |
| `LOGIN_PAGE_TITLE` | String | Stargate - Login | No |
| `LOGIN_PAGE_FOOTER_TEXT` | String | Copyright © 2024 - Stargate | No |
| `LOGIN_SUPPORT_URL` | String | `""` | No |
| `USER_HEADER_NAME` | String | X-Forwarded-User | No |
| `TRUSTED_PROXIES` | comma-separated IPs/CIDRs | empty | No |
| `PROXY_HEADER` | HTTP header name | X-Forwarded-For | No |
| `COOKIE_DOMAIN` | String | empty | No |
| `COOKIE_SECURE` | true/false | true | No |
| `CALLBACK_ALLOWED_HOSTS` | comma-separated hosts | empty | No |
| `SESSION_EXCHANGE_SECRET` | Secret string (32+ chars) | empty | No |
| `SESSION_EXCHANGE_REDIRECT_PATH` | Path | `/` | No |
| `LANGUAGE` | en, zh, fr, it, ja, de, ko | en | No |
| `PORT` | String | empty (:80) | No |
| `WARDEN_ENABLED` | true/false | false | No |
| `WARDEN_URL` | String | empty | No |
| `WARDEN_API_KEY` | String | empty | No |
| `WARDEN_HMAC_KEY_ID` | String | empty | No |
| `WARDEN_HMAC_SECRET` | String | empty | No |
| `WARDEN_TLS_CA_CERT_FILE` | path | empty | No |
| `WARDEN_TLS_CLIENT_CERT_FILE` | path | empty | No |
| `WARDEN_TLS_CLIENT_KEY_FILE` | path | empty | No |
| `WARDEN_TLS_SERVER_NAME` | String | empty | No |
| `HEADER_AUTH_ENABLED` | true/false | false | No |
| `HEADER_AUTH_SHARED_SECRET` | Secret string (at least 32 characters) | empty | Yes when header auth enabled |
| `HEADER_AUTH_SECRET_HEADER` | HTTP header name | X-Stargate-Header-Auth | No |
| `WARDEN_CACHE_TTL` | positive integer (seconds); no effect | 300 | No |
| `WARDEN_SOURCE_MAX_AGE` | duration | 0s (compatibility) | No |
| `WARDEN_SOURCE_CLOCK_SKEW` | duration | 0s | No |
| `METRICS_API_KEY` | secret | empty (metrics disabled) | No |
| `HERALD_ENABLED` | true/false | false | No |
| `HERALD_URL` | String | empty | No |
| `HERALD_API_KEY` | String | empty | No |
| `HERALD_HMAC_KEY_ID` | String | empty | No |
| `HERALD_HMAC_SECRET` | String | empty | No |
| `HERALD_CLIENT_IP_HEADER` | HTTP header name | empty | No |
| `HERALD_TLS_CA_CERT_FILE` | path | empty | No |
| `HERALD_TLS_CLIENT_CERT_FILE` | path | empty | No |
| `HERALD_TLS_CLIENT_KEY_FILE` | path | empty | No |
| `HERALD_TLS_SERVER_NAME` | String | empty | No |
| `HERALD_TOTP_ENABLED` | true/false | false | No |
| `LOGIN_SMS_ENABLED` | true/false | true | No |
| `LOGIN_EMAIL_ENABLED` | true/false | true | No |
| `LOGIN_DINGTALK_REQUIRE_USERID` | true/false | false | No |
| `LOGIN_UNIFORM_RESPONSES` | true/false | false | No |
| `LOGIN_UNIFORM_RESEND_COOLDOWN` | duration | 60s | No |
| `SESSION_STORAGE_ENABLED` | true/false | false | No |
| `SESSION_STORAGE_REDIS_ADDR` | String | localhost:6379 | No |
| `SESSION_STORAGE_REDIS_PASSWORD` | String | empty | No |
| `SESSION_STORAGE_REDIS_DB` | String | 0 | No |
| `SESSION_STORAGE_REDIS_KEY_PREFIX` | String | stargate:session: | No |
| `AUDIT_LOG_ENABLED` | true/false | true | No |
| `AUDIT_LOG_FORMAT` | json/text | json | No |
| `STEP_UP_ENABLED` | true/false | false | No |
| `STEP_UP_PATHS` | comma-separated paths | empty | No |
| `OTLP_ENABLED` | true/false | false | No |
| `OTLP_ENDPOINT` | String | empty | No |
| `AUTH_REFRESH_ENABLED` | true/false | true | No |
| `AUTH_REFRESH_INTERVAL` | duration | 5m | No |
| `REQUEST_CONTEXT_TIMEOUT` | duration | 10s | No |
| `RATE_LIMIT_LOGIN_MAX` | integer | 10 | No |
| `RATE_LIMIT_VERIFICATION_MAX` | integer | 5 | No |
| `RATE_LIMIT_WINDOW` | duration | 1m | No |

## Required Configuration

The following configuration items are required. Failure to set them will cause the service to fail to start.

### `AUTH_HOST`

Hostname of the authentication service.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | Yes |
| **Default** | None |
| **Example** | `auth.example.com` |

**Description:**

- Used to build login callback URLs
- Usually set to the hostname of the Stargate service
- Supports wildcard `*` (not recommended for production use)

**Example:**

```bash
AUTH_HOST=auth.example.com
```

### `PASSWORDS`

Password configuration, specifying the password encryption algorithm and password list.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | Yes |
| **Default** | None |
| **Format** | `algorithm:password1|password2|password3` |

**Description:**

- Format: `algorithm:password1|password2|password3`
- Supports multiple passwords, separated by `|`
- Any password that passes verification allows login
- Supported algorithms see [Password Configuration](#password-configuration) section

**Examples:**

```bash
# Single plain text password
PASSWORDS='plaintext:test123'

# Multiple plain text passwords
PASSWORDS='plaintext:test123|admin456|user789'

# BCrypt hash
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

```

## Optional Configuration

The following configuration items are optional. Default values are used if not set.

### `DEBUG`

Enable debug mode.

| Attribute | Value |
|-----------|-------|
| **Type** | Boolean |
| **Required** | No |
| **Default** | `false` |
| **Possible Values** | `true`, `false` |

**Description:**

- When enabled, log level is set to `DEBUG`
- Outputs more detailed debug information
- Recommended to set to `false` in production

**Example:**

```bash
DEBUG=true
```

### `LANGUAGE`

Interface language.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | `en` |
| **Possible Values** | `en` (English), `zh` (Chinese), `fr` (French), `it` (Italian), `ja` (Japanese), `de` (German), `ko` (Korean) |

**Description:**

- Affects the language of error messages and interface text
- Case-insensitive (`EN`, `en`, `En` all work)

**Example:**

```bash
LANGUAGE=zh
```

### `LOGIN_PAGE_TITLE`

Login page title.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | `Stargate - Login` |

**Description:**

- Displayed at the title position of the login page
- Supports HTML tags (not recommended)

**Example:**

```bash
LOGIN_PAGE_TITLE='My Auth Service'
```

### `LOGIN_PAGE_FOOTER_TEXT`

Login page footer text.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | `Copyright © 2024 - Stargate` |

**Description:**

- Displayed at the footer position of the login page
- Supports HTML tags (not recommended)

**Example:**

```bash
LOGIN_PAGE_FOOTER_TEXT='© 2024 My Company'
```

### `USER_HEADER_NAME`

User header name set after successful authentication.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | `X-Forwarded-User` |

**Description:**

- After successful authentication, Stargate sets this header in the response
- Header value is the authenticated user ID when available; password-only authentication uses `authenticated`
- Backend services can determine if a user is authenticated via this header
- Must be a non-empty string

**Example:**

```bash
USER_HEADER_NAME=X-Authenticated-User
```

### `TRUSTED_PROXIES` and `PROXY_HEADER`

`TRUSTED_PROXIES` is the comma-separated allowlist of reverse-proxy source IPs or CIDRs. It is empty by default, so Stargate ignores `X-Forwarded-Host`, `X-Forwarded-Uri`, `X-Forwarded-Proto`, and the configured client-IP header until the immediate peer is trusted. `PROXY_HEADER` selects the header Fiber uses for the client IP and defaults to `X-Forwarded-For`.

```bash
TRUSTED_PROXIES=10.20.0.10,10.30.0.0/24
PROXY_HEADER=X-Forwarded-For
```

Configure the narrowest proxy addresses possible. The proxy must remove client-supplied forwarding headers and write its own values before forwarding the request.

Every element of the client-IP header must be a bare IPv4 or IPv6 address. Since v1.3.7 (Fiber v3.7.0) Stargate reads the header from right to left and stops at the first element that is not one, such as `203.0.113.7:51234` or `unknown`. The client is then taken to be the nearest trusted proxy, so all users behind that proxy share one login and verification rate limit and appear under the proxy address in audit records. Earlier versions skipped such an element and used the next one to its left.

Send forwarding headers and browser headers such as `Origin` in the header section of the request. Since v1.3.7 a field sent as an HTTP trailer is not read as a header, and a request that announces more than 16 trailer names is answered with `431`.

### `COOKIE_DOMAIN`

Cookie domain, used for cross-domain session sharing.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | Empty (not set) |

**Description:**

- If set, session cookies will be set to the specified domain
- Supports cross-subdomain session sharing
- Format: `.example.com` (note the leading dot)
- When set to empty, cookies are only valid for the current domain

**Example:**

```bash
# Allow session sharing across all *.example.com subdomains
COOKIE_DOMAIN=.example.com
```

**Cross-Domain Session Sharing Scenario:**

Assume the following domains:
- `auth.example.com` - Authentication service
- `app1.example.com` - Application 1
- `app2.example.com` - Application 2

After setting `COOKIE_DOMAIN=.example.com`:
1. User logs in at `auth.example.com`
2. Session cookie is set to the `.example.com` domain
3. User can use the same session on `app1.example.com` and `app2.example.com`

**Cookie and session behavior (current implementation)**:
- **Session expiration**: Fixed to 24 hours in code; there is no `SESSION_TTL` (or similar) env variable.
- **Cookie Secure**: Controlled by `COOKIE_SECURE` and enabled by default. Set it to `false` only for local HTTP development.
- **Cookie SameSite**: Fixed to `Lax`; there is no `COOKIE_SAME_SITE` env variable.

### `COOKIE_SECURE`

Controls the `Secure` attribute on session and session-exchange cookies. The default `true` is appropriate for HTTPS deployments. Plain HTTP browsers will not return a Secure cookie, so the provided local Compose file and `start-local.sh` explicitly set `COOKIE_SECURE=false`.

```bash
# Local HTTP development only
COOKIE_SECURE=false
```

### `CALLBACK_ALLOWED_HOSTS`

Comma-separated allowlist of exact hosts that may receive a post-login callback. Ports, when used, must be listed explicitly. Subdomains covered by `COOKIE_DOMAIN` are also allowed.

```bash
CALLBACK_ALLOWED_HOSTS=app.example.com,admin.example.com
```

Stargate rejects callback hosts outside this list with HTTP 400. Do not add domains that can be registered or controlled by untrusted users.

### `SESSION_EXCHANGE_SECRET`

Encryption key for short-lived session exchange tickets. This setting is required and must contain at least 32 random characters whenever `COOKIE_DOMAIN` or `CALLBACK_ALLOWED_HOSTS` enables cross-domain callbacks, and use the same value on every Stargate replica.

```bash
SESSION_EXCHANGE_SECRET=<at-least-32-random-characters>
```

The value is never logged. Without it, session exchange requests fail closed. The value `local-development-session-secret-change-me` used by the repository's local HTTP demo is public: Stargate refuses to start with it unless `COOKIE_SECURE=false`. When Redis session storage is enabled, consumed ticket hashes are also stored in Redis with their remaining lifetime so replay protection is shared by every Stargate replica. In-memory deployments enforce single use only within one process.

### `SESSION_EXCHANGE_REDIRECT_PATH`

Landing path on the callback origin after a verified session exchange. The default `/` preserves existing behavior. Set a native application login handshake path such as `/auth/login/example/` when the application must create its own session after ForwardAuth. This is administrator configuration, not a request parameter. Only absolute same-origin paths without queries, fragments, percent escapes or control characters are accepted; login/exchange loops are rejected. It does not change allowed callback hosts or ticket replay protection.

### `PORT`

Service listening port for native and container deployments. Managed by the config package along with other env-based options.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | Empty for a native binary (`:80`); official container images set `PORT=8080` |

**Description:**

- Can be set in development or production deployments
- Official container images listen on `8080` by default. Use a bare number such as `PORT=8080` for container overrides: the image healthcheck inserts it into a URL, so the native `:8080` form is unsuitable there.
- Native binary format: port number (e.g., `8080`) or `:port` (e.g., `:8080`)

**Example:**

```bash
PORT=8080
```

### Warden Integration (Optional)

Warden is the user whitelist/authorized user information service. **This is optional** - if you don't need user whitelist functionality, you don't need to enable it. The following configuration options are available for integrating with Warden.

#### `WARDEN_ENABLED`

Enable Warden integration for user whitelist authentication functionality.

| Attribute | Value |
|-----------|-------|
| **Type** | Boolean |
| **Required** | No |
| **Default** | `false` |
| **Possible Values** | `true`, `false` |

**Description:**

- When enabled, Stargate will use Warden service for user whitelist verification
- Requires `WARDEN_URL` to be set
- This is an optional feature, Stargate can be used independently with password authentication

**Example:**

```bash
WARDEN_ENABLED=true
```

#### `WARDEN_URL`

Base URL of the Warden service.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No (required if `WARDEN_ENABLED=true`) |
| **Default** | Empty |
| **Example** | `http://warden:8081` or `https://warden.example.com` |

**Description:**

- Full base URL of the Warden service (without trailing slash)
- Must be set if `WARDEN_ENABLED=true`
- Used for querying user information and whitelist verification

**Example:**

```bash
WARDEN_URL=http://warden:8081
```

#### `WARDEN_API_KEY`

API key for authenticating with Warden service.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | Empty |

**Description:**

- Simple authentication method using API key
- Suitable for development and production environments
- If not set, Warden client may not authenticate properly

**Example:**

```bash
WARDEN_API_KEY=your-warden-api-key-here
```

#### `WARDEN_CACHE_TTL`

Accepted for compatibility. It has no effect on Stargate's behaviour.

| Attribute | Value |
|-----------|-------|
| **Type** | Integer |
| **Required** | No |
| **Default** | `300` |
| **Unit** | Seconds |

**Description:**

- The value is passed to the Warden client as the lifetime of its user-list cache. Stargate never requests the user list: sign-in, sending a code, trusted header authentication and the authorization refresh each look up one user by phone number or mail address, and those lookups are not cached. Every one of them is a request to Warden, whatever this value is.
- It does not delay revocation. How soon a disabled or removed user loses an existing session is set by [`AUTH_REFRESH_INTERVAL`](#auth_refresh_interval), which is also the setting that bounds how often a session causes a Warden request.
- With `HEADER_AUTH_ENABLED=true`, every forwarded request that carries a trusted identity header causes one Warden lookup.
- The value must still be a positive integer; Stargate does not start otherwise.

**Example:**

```bash
WARDEN_CACHE_TTL=300
```

The legacy Warden global OTP settings were removed. Use Herald-backed TOTP with `HERALD_TOTP_ENABLED`.

### Herald Integration (Optional)

Herald is the OTP/verification code service. **This is optional** - if you don't need verification code functionality, you don't need to enable it. The following configuration options are available for integrating with Herald.

#### `HERALD_ENABLED`

Enable Herald integration for OTP/verification code functionality.

| Attribute | Value |
|-----------|-------|
| **Type** | Boolean |
| **Required** | No |
| **Default** | `false` |
| **Possible Values** | `true`, `false` |

**Description:**

- When enabled, Stargate will use Herald service for sending and verifying OTP codes
- Requires `HERALD_URL` to be set
- This is an optional feature, Stargate can be used independently with password authentication

**Example:**

```bash
HERALD_ENABLED=true
```

#### `LOGIN_SMS_ENABLED`

Allow verification code login via SMS when Herald is enabled.

| Attribute | Value |
|-----------|-------|
| **Type** | Boolean |
| **Required** | No |
| **Default** | `true` |
| **Possible Values** | `true`, `false` |

**Description:**

- When `false`, SMS is not offered as a deliver option and requests for SMS codes are rejected (or fall back to email if enabled)
- Only applies when `HERALD_ENABLED=true` and Warden login is used

**Example:**

```bash
LOGIN_SMS_ENABLED=false
```

#### `LOGIN_DINGTALK_REQUIRE_USERID`

**Default:** `false` (compatibility mode). With explicit DingTalk delivery,
Stargate first uses the canonical Warden `dingtalk_userid`, then retains the
existing canonical-phone fallback when this option is false.

Set `true` to require an explicit Warden-managed userid binding. The value is
trimmed and its opaque case is preserved. Missing/whitespace-only bindings,
commas, CR/LF or remaining control characters return the existing
`dingtalk_not_bound` response before any Herald request; they never fall back to
phone or email. Numeric userid values are not guessed to be phone numbers.
The same helper checks pending login bindings before verification: a former
phone-derived or changed-contact binding is refused without consuming the code.
This option does not expire, migrate or delete existing sessions or bindings.

This is a destination constraint, not proof that an administrator verified the
real person's mapping. It does not change native Warden active-status checks,
directory-cache/session revocation boundaries, email-verification claims or
application permissions. It does not turn DingTalk login into verified email.
See [the isolated userid-only profile](DINGTALK_USERID_ONLY.md) before opting in.

#### `LOGIN_EMAIL_ENABLED`

Allow verification code login via email when Herald is enabled.

| Attribute | Value |
|-----------|-------|
| **Type** | Boolean |
| **Required** | No |
| **Default** | `true` |
| **Possible Values** | `true`, `false` |

**Description:**

- When `false`, email is not offered as a deliver option and requests for email codes are rejected (or fall back to SMS if enabled)
- Only applies when `HERALD_ENABLED=true` and Warden login is used

**Example:**

```bash
LOGIN_EMAIL_ENABLED=false
```

#### `LOGIN_UNIFORM_RESPONSES`

Make `POST /_send_verify_code` and Warden sign-in through `POST /_login` answer
with the same status, headers and body whether or not the submitted phone
number or email address is on the Warden allow list.

| Attribute | Value |
|-----------|-------|
| **Type** | Boolean |
| **Required** | No |
| **Default** | `false` |
| **Possible Values** | `true`, `false` |

**Description:**

With the default `false`, the two endpoints answer differently for a listed
and an unlisted identifier, and for a listed account they report whether a
channel is usable and whether TOTP is enrolled. With `true`:

- `POST /_send_verify_code` answers `200` with a `challenge_id` and
  `next_resend_in` for every identifier that passes the request checks and the
  resend cooldown, whether or not a code was sent. Nothing is sent for an
  unknown or inactive identifier. The login page shows "If this account can
  receive codes, a code has been sent." in every case, also for an account
  without a usable destination, a Herald limit or lock, and a Herald outage.
- `challenge_id` is a handle minted by Stargate, not the Herald challenge id.
  `expires_in` and `debug_code` are not returned, and the request's
  `Idempotency-Key` is not forwarded to Herald.
- A failed Warden sign-in answers `401` with one message, whatever the reason:
  unknown identifier, wrong, expired or locked code, TOTP not enrolled, or a
  Herald failure.
- Requests that are malformed, or that the configuration cannot serve, keep
  their own answers, which are given before the Warden lookup. A failed Warden
  lookup answers `503` for every identifier.
- On both endpoints a request that supplies `phone` is handled by the phone
  number alone, as if `mail` had not been sent. The default mode falls back to
  `mail` when the phone is not listed. In this mode the fallback would show
  the owner of one listed mail address whether a phone number is listed: next
  to an unlisted phone, and only then, a code would reach that mailbox and
  sign-in with it would succeed. The login page asks for one identifier in
  this mode and sends nothing while both of its fields are filled in.

Limits:

- The mode unifies response content, not response time. A send for a listed
  identifier waits for Herald, so a single request can still be told apart by
  latency. See [SECURITY.md](SECURITY.md#uniform-responses) for what else is
  not covered.
- What happened to a request is visible only in the log and the audit log.
- The mode concerns Warden sign-in through Herald. When it is `true` while
  `WARDEN_ENABLED` or `HERALD_ENABLED` is `false`, Stargate logs a startup
  warning, and requests with `auth_method=warden` are refused with the same
  error for every identifier.
- `debug_code` is never returned in this mode. Stargate logs a startup warning
  when the debug-code relay (`DEBUG=true` with `COOKIE_SECURE=false`) is
  active.
- Password sign-in, `/_auth`, re-authentication, step-up and TOTP management
  are not affected.

Deployment:

- Set `HERALD_CLIENT_IP_HEADER`, together with `HERALD_TRUSTED_PROXY_HEADER`
  and `HERALD_TRUSTED_PROXIES` on Herald. Without it Herald counts its
  `RATE_LIMIT_PER_IP` against Stargate's address for all users, so one client
  can use up that limit for everyone, and in this mode a send that Herald
  refuses is still shown as sent. Stargate logs a startup warning when the
  header is empty.
- Enable the setting only after every replica runs a version that has it. A
  replica without it does not know handles and would send them to Herald as
  challenge ids, so sign-ins that reach it fail.
- With more than one replica, `LOGIN_UNIFORM_RESPONSES`,
  `LOGIN_UNIFORM_RESEND_COOLDOWN` and the secret behind the digest key must be
  identical on all of them. Store keys carry a keyed digest in place of the
  identifier. The key is derived from the first of these that is set, in this
  order: `SESSION_EXCHANGE_SECRET`, `HERALD_HMAC_SECRET`, `HERALD_API_KEY`,
  `WARDEN_HMAC_SECRET`, `WARDEN_API_KEY`. Replicas that derive different keys
  keep the resend cooldown per replica. So does a deployment that sets none of
  them: each process then uses a random key.

**Example:**

```bash
LOGIN_UNIFORM_RESPONSES=true
```

#### `LOGIN_UNIFORM_RESEND_COOLDOWN`

Resend window that Stargate enforces for each identifier when
`LOGIN_UNIFORM_RESPONSES=true`.

| Attribute | Value |
|-----------|-------|
| **Type** | Duration |
| **Required** | No |
| **Default** | `60s` |
| **Possible Values** | A Go duration of at least `1s`, for example `60s`, `90s`, `2m` |

**Description:**

- Used only in uniform mode. The value is validated at startup in either case.
- The window starts with the first admitted `POST /_send_verify_code` for a
  phone number or an email address, and later requests do not extend it.
  Inside it the endpoint answers `429` with `Retry-After` and `next_resend_in`
  and contacts neither Warden nor Herald. Phone and mail are counted
  separately. A request that supplies both is counted against the phone only,
  because the mail is not used in that case.
- An admitted request reports the window as `next_resend_in`, in whole seconds
  rounded up.
- Set it to at least Herald's `RESEND_COOLDOWN` (default `60s`). If it is
  lower, a resend that Stargate admits while Herald's cooldown is still running
  shows "sent" and sends nothing. The code delivered earlier keeps working for
  a client that presents the `challenge_id` it holds with the resend, which the
  login page does.
- The windows are kept in the same store as the rate limits: shared through
  Redis when `SESSION_STORAGE_ENABLED=true`, otherwise per process.

**Example:**

```bash
LOGIN_UNIFORM_RESEND_COOLDOWN=60s
```

#### `HERALD_URL`

Base URL of the Herald service.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No (required if `HERALD_ENABLED=true`) |
| **Default** | Empty |
| **Example** | `http://herald:8082` or `https://herald.example.com` |

**Description:**

- Full base URL of the Herald service (without trailing slash)
- Must be set if `HERALD_ENABLED=true`
- Used for creating challenges and verifying codes

**Example:**

```bash
HERALD_URL=http://herald:8082
```

#### `HERALD_API_KEY`

API key for authenticating with Herald service (simple authentication method).

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | Empty |

**Description:**

- Simple authentication method using API key
- Suitable for development environments
- If both `HERALD_API_KEY` and `HERALD_HMAC_SECRET` are set, HMAC takes precedence
- Must be set if `HERALD_HMAC_SECRET` is not set

**Example:**

```bash
HERALD_API_KEY=your-api-key-here
```

#### `HERALD_HMAC_SECRET`

HMAC secret for service-to-service authentication with Herald (recommended for production).

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | Empty |

**Description:**

- HMAC-SHA256 signature-based authentication (more secure than API key)
- Recommended for production environments
- If both `HERALD_API_KEY` and `HERALD_HMAC_SECRET` are set, HMAC takes precedence
- Provides better security for service-to-service communication
- Must match the HMAC secret configured in Herald service

The current Herald SDK signs HMAC v3 requests, binding `Idempotency-Key` into
the signature along with the method, path, query, and body. Deploy Herald
v1.3.7 or newer before upgrading an HMAC-enabled Stargate instance. The SDK
does not automatically downgrade the signature version.

**Authentication Methods:**

Stargate supports two authentication methods for Herald:

1. **API Key** (Simple, for development):
   - Set `HERALD_API_KEY` only
   - Less secure but easier to configure
   - Suitable for development and testing

2. **HMAC Signature** (Secure, for production):
   - Set `HERALD_HMAC_SECRET`
   - More secure, uses HMAC-SHA256 signatures
   - Recommended for production environments
   - Provides timestamp-based request signing

**Example:**

```bash
# Production: Use HMAC
HERALD_HMAC_SECRET=your-hmac-secret-key-here

# Development: Use API Key
HERALD_API_KEY=your-api-key-here
```

**Note:** If neither `HERALD_API_KEY` nor `HERALD_HMAC_SECRET` is set, Herald client may not authenticate properly and requests may fail.

#### Herald mTLS (Optional)

In production, Herald requires the selected HMAC or API-key request credentials; optional mTLS adds transport protection and does not replace them.

#### `HERALD_TLS_CA_CERT_FILE`

Path to CA certificate file for Herald server verification.

| Attribute | Value |
|-----------|-------|
| **Type** | String (file path) |
| **Required** | No |
| **Default** | Empty |

#### `HERALD_TLS_CLIENT_CERT_FILE`

Path to client certificate file (mTLS).

| Attribute | Value |
|-----------|-------|
| **Type** | String (file path) |
| **Required** | No |
| **Default** | Empty |

#### `HERALD_TLS_CLIENT_KEY_FILE`

Path to client private key file (mTLS).

| Attribute | Value |
|-----------|-------|
| **Type** | String (file path) |
| **Required** | No |
| **Default** | Empty |

#### `HERALD_CLIENT_IP_HEADER`

Name of the request header in which Stargate sends the end-user address to Herald on challenge and verification requests, for example `X-Forwarded-For`.

| Attribute | Value |
|-----------|-------|
| **Type** | String (HTTP header name) |
| **Required** | No |
| **Default** | Empty (no header is sent) |

Herald applies `RATE_LIMIT_PER_IP` to the address it sees for the caller. Without this setting that is Stargate's address, so every user shares one per-IP quota at Herald. To have the limit counted per end user, set all three of the following:

```bash
# Stargate
HERALD_CLIENT_IP_HEADER=X-Forwarded-For

# Herald
HERALD_TRUSTED_PROXY_HEADER=X-Forwarded-For
HERALD_TRUSTED_PROXIES=10.20.0.15   # the address Herald sees for Stargate
```

- The header name must match Herald's `HERALD_TRUSTED_PROXY_HEADER`. Herald ignores the header from any peer that is not in `HERALD_TRUSTED_PROXIES`, so setting only the Stargate side has no effect.
- The value sent is the client address Stargate itself resolved. It is only as accurate as Stargate's own `TRUSTED_PROXIES` and `PROXY_HEADER` configuration; see the client-IP header format under [`TRUSTED_PROXIES` and `PROXY_HEADER`](#trusted_proxies-and-proxy_header).
- If a proxy sits between Stargate and Herald, that proxy is the peer Herald sees. List it in `HERALD_TRUSTED_PROXIES` and make sure it passes the header on unchanged.
- The header is sent on verification-code and code-verification requests. TOTP requests do not carry it.
- A name that is not a valid header name, or one the Herald client sets itself (such as `X-Signature` or `X-API-Key`), stops startup.
- Requires Herald v1.6.0 or later on the server.

#### `HERALD_TLS_SERVER_NAME`

Server Name Indication (SNI) for TLS handshake and certificate validation.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | Empty |

#### Herald TOTP (Optional, per-user 2FA)

TOTP binding and verification are performed via **Herald** (Herald proxies to herald-totp internally). Separate from Herald OTP (SMS/email verification codes). Stargate only needs Herald URL/auth and this flag; herald-totp URL and auth are configured on the Herald service.

#### `HERALD_TOTP_ENABLED`

Enable TOTP (bind/verify TOTP codes). When `true`, Stargate calls Herald's `/v1/totp/*` API using the Herald client.

| Attribute | Value |
|-----------|-------|
| **Type** | Boolean |
| **Required** | No |
| **Default** | `false` |
| **Possible Values** | `true`, `false` |

**Note:** Herald must be enabled (`HERALD_ENABLED`, `HERALD_URL`) and Herald must be configured to proxy herald-totp (e.g. Herald's `HERALD_TOTP_ENABLED`, `HERALD_TOTP_BASE_URL`).

### Session Storage (Redis, Optional)

When enabled, server-side sessions are stored in Redis for sharing across instances and survival across Stargate restarts, subject to Redis persistence and availability. When disabled, sessions are stored in the current process's memory and are lost on restart. The browser cookie carries only the session identifier; it does not store session data. Use shared Redis storage for multiple replicas.

#### `SESSION_STORAGE_ENABLED`

Enable Redis session storage.

| Attribute | Value |
|-----------|-------|
| **Type** | Boolean |
| **Required** | No |
| **Default** | `false` |
| **Possible Values** | `true`, `false` |

#### `SESSION_STORAGE_REDIS_ADDR`

Redis address.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | `localhost:6379` |

#### `SESSION_STORAGE_REDIS_PASSWORD`

Redis password (empty if none).

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | Empty |

#### `SESSION_STORAGE_REDIS_DB`

Redis database index.

| Attribute | Value |
|-----------|-------|
| **Type** | Integer (e.g. 0–15) |
| **Required** | No |
| **Default** | `0` |

#### `SESSION_STORAGE_REDIS_KEY_PREFIX`

Key prefix for session keys (to separate multiple Stargate deployments or other apps).

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | `stargate:session:` |

### Audit Log (Optional)

#### `AUDIT_LOG_ENABLED`

Enable audit logging.

| Attribute | Value |
|-----------|-------|
| **Type** | Boolean |
| **Required** | No |
| **Default** | `true` |
| **Possible Values** | `true`, `false` |

#### `AUDIT_LOG_FORMAT`

Audit log output format.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | `json` |
| **Possible Values** | `json`, `text` |

### Step-up Authentication (Optional)

Require re-authentication for selected paths. A Warden user proves it with a current TOTP code when TOTP is enrolled, otherwise with a one-time code sent through Herald to the contact registered in Warden. Only a session created with the shared password re-enters `PASSWORDS`.

#### `STEP_UP_ENABLED`

Enable step-up authentication. Requires `PASSWORDS` or `HERALD_ENABLED=true`.

When `WARDEN_ENABLED=true`, `HERALD_ENABLED=true` and `PASSWORDS` are all set, a session created by the shared-password login still passes step-up with that password, and Stargate logs a warning at startup. Remove `PASSWORDS` to require a per-user proof for every session.

| Attribute | Value |
|-----------|-------|
| **Type** | Boolean |
| **Required** | No |
| **Default** | `false` |
| **Possible Values** | `true`, `false` |

#### `STEP_UP_PATHS`

Paths that require step-up; comma-separated; prefix matching supported.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | Empty |

**Example:** `/admin,/api/sensitive`

At least one path is required when `STEP_UP_ENABLED=true`. Matching uses the request path only; query parameters do not affect protection.

`TRUSTED_PROXIES` must also be configured. The trusted ForwardAuth proxy must send the original request URI in `X-Forwarded-Uri`; Stargate rejects the Step-up check when this trusted path provenance is missing or invalid. A plain path matches itself and child path segments, so `/admin` protects both `/admin` and `/admin/users` but not `/administrator`. Explicit `*` and `?` glob patterns remain supported. Matching is case-insensitive and ignores `;` path parameters in every segment, so `/Admin`, `/admin;x` and `/admin;x/users` are also protected; a path whose parameters hide a dot segment (such as `/public/..;/admin`) requires step-up.

### OpenTelemetry (Optional)

#### `OTLP_ENABLED`

Enable OTLP telemetry export.

| Attribute | Value |
|-----------|-------|
| **Type** | Boolean |
| **Required** | No |
| **Default** | `false` |
| **Possible Values** | `true`, `false` |

#### `OTLP_ENDPOINT`

OTLP collector endpoint (e.g. Jaeger/OTLP Collector).

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | Empty |

**Example:** `http://jaeger:4318/v1/traces`

### Auth Refresh (Optional)

#### `AUTH_REFRESH_ENABLED`

Periodically refresh user/authorization info from Warden and update session during its lifetime.

A refresh may update roles and scopes only when the session's nonempty `user_id`
exactly matches Warden's nonempty `user_id`. Email addresses and phone numbers
are lookup inputs and may be reassigned; they cannot rebind a session to another
account. A mismatch or missing ID rejects the request without replacing identity,
permissions or the refresh timestamp. The stored session is retained for expiry
and investigation; the client must sign in again. Legacy sessions without
`auth_method` remain compatible only when their stored `user_id` matches.
Warden records used with authorization refresh must provide stable user IDs.

Replace every instance before relying on this boundary; old instances remain
vulnerable during mixed-version operation or rollback. Pause protected traffic
for a coordinated replacement. No key-format, persistent-volume or automatic
record migration is introduced. Already-rebound sessions cannot be detected or
repaired automatically: review directory reassignment history, retain audit
and session evidence under access controls, and revoke or expire affected
sessions before restoring access. If session integrity cannot be established,
require fresh Warden authentication for all existing sessions. A rollback
restores the old rebinding behavior and must not resume affected traffic.


| Attribute | Value |
|-----------|-------|
| **Type** | Boolean |
| **Required** | No |
| **Default** | `true` |
| **Possible Values** | `true`, `false` |

#### `AUTH_REFRESH_INTERVAL`

Refresh interval (Go duration, e.g. `5m`, `1h`).

| Attribute | Value |
|-----------|-------|
| **Type** | String (duration) |
| **Required** | No |
| **Default** | `5m` |

**Example:** `AUTH_REFRESH_INTERVAL=10m`

#### `REQUEST_CONTEXT_TIMEOUT`

Maximum lifetime of the standard Go context propagated through tracing to
Warden, Herald, and other request-scoped work. The value must be a positive Go
duration. The default is `10s`, matching the existing Herald client timeout and
remaining below Stargate's 15-second response write timeout.

Fiber/fasthttp does not provide per-request cancellation when an ordinary HTTP
client disconnects. This setting therefore supplies the reliable deadline;
Stargate also cancels the context when the handler returns. It does not add or
promise immediate client-disconnect cancellation.

| Attribute | Value |
|-----------|-------|
| **Type** | String (duration) |
| **Required** | No |
| **Default** | `10s` |

**Example:** `REQUEST_CONTEXT_TIMEOUT=10s`

#### `RATE_LIMIT_LOGIN_MAX`

Maximum number of requests one client may send to `/_login` and `/_step_up`
within `RATE_LIMIT_WINDOW`. The same quota bounds repeated `Stargate-Password`
header failures. Set it to `0` to disable the login quota.

A client is identified by its IPv4 address or, for IPv6, by its /64 prefix.
The same rule applies to `RATE_LIMIT_VERIFICATION_MAX`.

| Attribute | Value |
|-----------|-------|
| **Type** | Integer |
| **Required** | No |
| **Default** | `10` |

**Example:** `RATE_LIMIT_LOGIN_MAX=10`

#### `RATE_LIMIT_VERIFICATION_MAX`

Maximum number of requests one client may send to `/_send_verify_code`,
`/_send_reauth_code` and the TOTP enrollment and revocation endpoints within
`RATE_LIMIT_WINDOW`. Set it to
`0` to disable the verification quota.

| Attribute | Value |
|-----------|-------|
| **Type** | Integer |
| **Required** | No |
| **Default** | `5` |

**Example:** `RATE_LIMIT_VERIFICATION_MAX=5`

#### `RATE_LIMIT_WINDOW`

Fixed window over which both quotas are counted. The value must be a positive
Go duration.

| Attribute | Value |
|-----------|-------|
| **Type** | String (duration) |
| **Required** | No |
| **Default** | `1m` |

**Example:** `RATE_LIMIT_WINDOW=1m`

#### Rate-limit state and client attribution

Redis counters use an atomic Lua script with millisecond expiry. Requests do not
extend an existing window, including its final second. The Redis account must
permit `EVAL`, `EVALSHA`, `PTTL`, `SET`, `PEXPIRE`, and `INCR` on the rate-limit
key prefix. A missing script is reloaded automatically after Redis restarts or
`SCRIPT FLUSH`. Existing integer counters and key prefixes remain compatible;
a counter without expiry is given the configured window without losing its count.
No new persistent volume or data migration is required.

Old instances can still extend a window during a rolling upgrade or rollback.
To obtain the corrected fixed-window behavior consistently, replace every
instance. For a coordinated transition, pause authentication traffic, replace
instances, wait one configured rate-limit window, then resume traffic. Do not
delete counters to bypass an active quota. Rolling back restores the previous
window behavior; it does not require rewriting session or counter data.


Rate-limit counters are shared through Redis when `SESSION_STORAGE_ENABLED=true`,
using the same connection and `SESSION_STORAGE_REDIS_KEY_PREFIX` as sessions. Every
replica then enforces one quota. Without Redis the counters stay process-local, so
a deployment with N replicas allows up to N times the configured value. If Redis
becomes unreachable, Stargate falls back to process-local counting and logs a
warning rather than removing the limit or rejecting every request.

Quotas are keyed by client address, and the client address is the direct peer
unless the peer is listed in `TRUSTED_PROXIES`. Behind a reverse proxy that is not
listed there, every client is attributed to the proxy and the whole deployment
shares a single quota. Stargate warns at startup when `TRUSTED_PROXIES` is empty
and again when it first receives forwarded headers from an untrusted peer. Set
`TRUSTED_PROXIES` to the reverse-proxy source IPs or CIDRs so quotas apply per
client.


## Password Configuration

Stargate supports plaintext (development only) and bcrypt password verification. Password configuration format: `algorithm:password1|password2|password3`. Password bytes are case-sensitive and whitespace is significant.

### Supported Algorithms

#### `plaintext` - Plain Text Password

**Description:**

- Stored in plain text, no encryption
- **Testing environment only**
- Strongly not recommended for production use
- Stargate logs a startup warning when it is used while `COOKIE_SECURE` is `true`

**Example:**

```bash
PASSWORDS='plaintext:test123|admin456'
```

#### `bcrypt` - BCrypt Hash

**Description:**

- Uses BCrypt algorithm for hashing
- High security, recommended for production
- Password must use BCrypt hash value

**Generate BCrypt Hash:**

```bash
# `htpasswd` is provided by apache2-utils on Debian/Ubuntu and Alpine.
# Enter and confirm the password at the prompts. Copy the hash after `stargate:`.
htpasswd -nBC 10 stargate
```

**Example:**

```bash
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
```

MD5 and unsalted SHA-512 password hashes are rejected because they are fast, unsalted primitives and are not password hashing functions.

### Password Verification Rules

1. **Password Normalization**: Passwords are compared as opaque, case-sensitive values; whitespace and case are preserved
2. **Multiple Password Support**: Multiple passwords can be configured, any password that passes verification is acceptable
3. **Algorithm Consistency**: All passwords must use the same algorithm

## Configuration Examples

### Stargate + Redis Only (Password Auth, Sessions in Redis)

```bash
# Required
AUTH_HOST=auth.example.com
PASSWORDS='plaintext:test123'

# Session storage in Redis (multi-instance or persistent sessions)
SESSION_STORAGE_ENABLED=true
SESSION_STORAGE_REDIS_ADDR=redis:6379
SESSION_STORAGE_REDIS_PASSWORD=
SESSION_STORAGE_REDIS_DB=0
SESSION_STORAGE_REDIS_KEY_PREFIX=stargate:session:

# Optional
DEBUG=false
LANGUAGE=zh
```

Use when you want password-only login but need sessions in Redis for multiple instances or to survive restarts.

### Basic Configuration (Password Authentication)

```bash
# Required configuration
AUTH_HOST=auth.example.com
PASSWORDS='plaintext:test123'

# Optional configuration
DEBUG=false
LANGUAGE=en
```

### Production Configuration (Password Authentication)

```bash
# Required configuration
AUTH_HOST=auth.example.com
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

# Optional configuration
DEBUG=false
LANGUAGE=zh
LOGIN_PAGE_TITLE='My Auth Service'
LOGIN_PAGE_FOOTER_TEXT='© 2024 My Company'
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

### Warden + Herald OTP Authentication Configuration (Recommended for Production)

```bash
# Required configuration
AUTH_HOST=auth.example.com

# Warden configuration
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8081
WARDEN_API_KEY=your-warden-api-key
WARDEN_CACHE_TTL=300

# Herald configuration
HERALD_ENABLED=true
HERALD_URL=http://herald:8082
HERALD_HMAC_SECRET=your-herald-hmac-secret

# Optional configuration
DEBUG=false
LANGUAGE=zh
LOGIN_PAGE_TITLE='My Auth Service'
LOGIN_PAGE_FOOTER_TEXT='© 2024 My Company'
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

**Description:**

- This configuration demonstrates how to use both Warden and Herald integrations together
- Warden provides user whitelist verification
- Herald provides OTP/verification code sending and verification
- Production environment recommends using HMAC signature (`HERALD_HMAC_SECRET`) instead of API Key
- **Note**: This is an optional configuration, Stargate can be used independently with password authentication without these services

### Docker Compose Configuration

**Password Authentication Mode:**

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:1.3.8
    environment:
      # Required configuration
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      
      # Optional configuration
      - DEBUG=false
      - LANGUAGE=zh
      - LOGIN_PAGE_TITLE=My Auth Service
      - LOGIN_PAGE_FOOTER_TEXT=© 2024 My Company
      - COOKIE_DOMAIN=.example.com
```

**Warden + Herald OTP Authentication Mode:**

This is the Stargate side of an API-key deployment. Deploy
[Warden](https://github.com/soulteary/warden/blob/v1.4.4/docs/enUS/DEPLOYMENT.md)
and [Herald](https://github.com/soulteary/herald/blob/v1.3.13/docs/enUS/DEPLOYMENT.md)
using their versioned guides, including their data and persistence requirements.
Supply URLs reachable from the Stargate container and matching credentials;
the upstream default ports are Warden `8081` and Herald `8082`. Configure shared
networks or routable addresses explicitly. For Herald HMAC mode, use the matching
`HERALD_HMAC_SECRET`/`HERALD_HMAC_KEY_ID` settings instead of the API-key example.
Use stable Warden user IDs and retain the deployment's session/replay persistence.
This documentation does not migrate or delete existing provider data or volumes.

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:1.3.8
    environment:
      - AUTH_HOST=auth.example.com
      - WARDEN_ENABLED=true
      - WARDEN_URL=${WARDEN_URL:?set the reachable Warden URL}
      - WARDEN_API_KEY=${WARDEN_API_KEY:?set the matching Warden API key}
      - HERALD_ENABLED=true
      - HERALD_URL=${HERALD_URL:?set the reachable Herald URL}
      - HERALD_API_KEY=${HERALD_API_KEY:?set the matching Herald API key}
      - CALLBACK_ALLOWED_HOSTS=app.example.com
      - COOKIE_DOMAIN=.example.com
      - SESSION_EXCHANGE_SECRET=${SESSION_EXCHANGE_SECRET:?set at least 32 random characters}
```

### Local Development Configuration

```bash
# Required configuration
AUTH_HOST=localhost
PASSWORDS='plaintext:test123|admin456'

# Optional configuration
DEBUG=true
LANGUAGE=zh
PORT=8080
```

## Configuration Validation

Stargate validates all configuration items at startup:

1. **Required Configuration Check**: If required configuration is not set, the service will fail to start and display an error message
2. **Format Validation**: Incorrect password configuration format will cause startup failure
3. **Algorithm Validation**: Unsupported password algorithms will cause startup failure
4. **Value Validation**: Some configuration items have value restrictions (e.g., `LANGUAGE`, `DEBUG`)

**Error Examples:**

```bash
# Missing required configuration
Error: Configuration error: environment variable 'AUTH_HOST' is required but not set.

# Incorrect password format
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'invalid_format'

# Unsupported algorithm
Error: Configuration error: invalid value for environment variable 'PASSWORDS': 'unknown:password'
```

## Configuration Dependencies

### Configuration Item Dependencies

- **Warden Integration**:
  - When `WARDEN_ENABLED=true`, must set `WARDEN_URL`
  - `WARDEN_API_KEY` is recommended (for service authentication)

- **Herald Integration (OTP SMS/email)**:
  - When `HERALD_ENABLED=true`, must set `HERALD_URL`
  - Must set either `HERALD_API_KEY` or `HERALD_HMAC_SECRET` (recommend HMAC for production)
  - Optional: mTLS via `HERALD_TLS_*`; can be used alongside API Key/HMAC

- **Herald TOTP Integration (per-user 2FA)**:
  - Stargate: set `HERALD_TOTP_ENABLED=true` only (TOTP is via Herald proxy)
  - Herald service must be configured with `HERALD_TOTP_ENABLED`, `HERALD_TOTP_BASE_URL`, and `HERALD_TOTP_API_KEY` or `HERALD_TOTP_HMAC_SECRET`

- **Session Storage**:
  - When `SESSION_STORAGE_ENABLED=true`, Redis must be reachable (default `SESSION_STORAGE_REDIS_ADDR=localhost:6379`)

- **Step-up Authentication**:
  - When `STEP_UP_ENABLED=true`, use `STEP_UP_PATHS` to define paths that require re-authentication. Warden users confirm with their TOTP code or a delivered one-time code; sessions created with the shared password confirm with that password, which is not a separate authentication factor

- **Auth Refresh**:
  - Authorization refresh is enabled by default for Warden sessions; it is a no-op in standalone password mode. Use `AUTH_REFRESH_INTERVAL` to tune the refresh interval or set `AUTH_REFRESH_ENABLED=false` only when delayed revocation is acceptable.

- **Warden + Herald Combined Usage** (Optional):
  - When OTP authentication is needed, can optionally enable both Warden and Herald
  - Warden provides user whitelist verification and user information
  - Herald provides verification code sending and verification
  - **Note**: These integrations are optional, Stargate can be used independently

### Authentication Mode Selection

1. **Password Authentication Mode** (Simple, suitable for development/testing):
   - Only need to set `PASSWORDS`
   - Does not require Warden and Herald

2. **Warden + Herald OTP Authentication Mode** (Optional, suitable for scenarios requiring advanced features):
   - Need to enable `WARDEN_ENABLED=true` and `HERALD_ENABLED=true`
   - Provides user whitelist management and OTP verification code functionality
   - More secure, supports rate limiting and auditing
   - **Note**: This is an optional feature, Stargate can be used independently with password authentication

## Configuration Best Practices

1. **Production Security**:
   - Use `bcrypt`; avoid `plaintext`
   - Set `DEBUG=false`
   - Use strong passwords (password authentication mode)
   - Or use Warden + Herald OTP authentication (recommended)

2. **Inter-Service Security** (if optional service integrations are enabled):
   - Production environment use `HERALD_HMAC_SECRET` instead of `HERALD_API_KEY`
   - Ensure Warden and Herald services are accessible
   - Configure appropriate network policies and firewall rules

3. **Cross-Domain Sessions**:
   - If you need to share sessions across subdomains, set `COOKIE_DOMAIN`
   - Format: `.example.com` (note the leading dot)

4. **Multi-Language Support**:
   - Set `LANGUAGE` according to user base
   - Supports `en`, `zh`, `fr`, `it`, `ja`, `de`, `ko`

5. **Custom Interface**:
   - Use `LOGIN_PAGE_TITLE` and `LOGIN_PAGE_FOOTER_TEXT` to customize the login page

6. **Monitoring and Debugging**:
   - Set `DEBUG=true` in development environment for detailed logs
   - Set `DEBUG=false` in production environment to reduce log output

7. **Performance Optimization**:
   - Raise `AUTH_REFRESH_INTERVAL` to reduce requests to Warden; a longer interval also delays the effect of disabling a user
   - `WARDEN_CACHE_TTL` does not reduce them: per-user lookups are not cached

## Legacy Password Header Authentication

`Stargate-Password` authentication is disabled by default. To opt in for a trusted legacy client:

```bash
PASSWORD_HEADER_AUTH_ENABLED=true
PASSWORDS='bcrypt:<hash>'
```

Treat the header as a credential: require HTTPS, apply rate limits, and configure the reverse proxy to remove `Stargate-Password` before forwarding the request to the backend. Cookie sessions remain the recommended browser authentication method.

## Trusted Identity Header Authentication

Trusted identity headers are disabled by default. This mode requires Warden and is intended only for a reverse proxy that authenticates the caller before forwarding the request.

```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8081
HEADER_AUTH_ENABLED=true
HEADER_AUTH_SHARED_SECRET=<at-least-32-random-characters>
# Optional; this is the default header name:
HEADER_AUTH_SECRET_HEADER=X-Stargate-Header-Auth
```

The proxy sends `X-User-Phone` or `X-User-Mail` together with the configured secret header. Stargate removes the secret header before forwarding and discards the identity headers when the secret does not match.

Configured header names must be valid HTTP field names and must not reuse routing, authentication, identity, or hop-by-hop headers. In particular, do not configure the shared-secret header as `X-Forwarded-Uri`, `Stargate-Password`, `X-User-Phone`, or `X-User-Mail`; Stargate rejects these combinations at startup.

Treat all three headers as trust-boundary inputs: use HTTPS, generate a strong shared secret, and configure the edge proxy to remove any client-supplied `X-User-Phone`, `X-User-Mail`, and header named by `HEADER_AUTH_SECRET_HEADER` before adding its own. Never expose this mode directly to untrusted clients.


## Source freshness and scrape credentials

Set `WARDEN_SOURCE_MAX_AGE` to a positive duration to require negotiated Warden
snapshot metadata on every directory lookup. For example, use `90s` with
`WARDEN_SOURCE_CLOCK_SKEW=1s` when the directory refreshes every 15–30 seconds.
Choose values from the measured update interval and the required revocation bound.
Warden must support `Warden-Snapshot-Metadata: v1`; older servers fail closed when
this option is enabled. The default `0s` retains compatibility with older Warden
versions and does not bound source age.

The source deadline is the earlier of Warden's deadline and the source success
time plus `WARDEN_SOURCE_MAX_AGE`, minus the clock allowance. Sessions retain that
deadline and refresh when either it or `AUTH_REFRESH_INTERVAL` expires. Enabling
source checks requires `AUTH_REFRESH_ENABLED=true`. Existing Warden sessions
without a deadline revalidate on their next ForwardAuth request. Changing either
source-age setting also revalidates sessions created under the previous policy. Source errors
return HTTP 503 without destroying the session or extending its authorization;
confirmed missing/disabled identities and changed stable user IDs invalidate it.
Trusted-header authentication performs a fresh directory lookup for each request.
Snapshot metadata is not signed. Use verified HTTPS for cross-host calls;
plaintext HTTP requires a trusted, isolated local network. Metadata from an
untrusted HTTP intermediary cannot establish source freshness.

This uses a metadata-checking transport with the existing SDK; no SDK cache TTL
is treated as evidence of source freshness.

`/metrics` now returns HTTP 404 unless `METRICS_API_KEY` is configured. Supply a
random secret of at least 32 bytes and configure the scraper to send
`Authorization: Bearer <secret>`. Scrape credentials are independent of login,
Warden and Herald credentials. Reusing configured service/session keys or plaintext login passwords
is rejected at startup; never reuse a hashed login password either. Restrict this path at the network or proxy layer as
well. Existing anonymous scrapers must be updated before deploying this change.


### `LOGIN_SUPPORT_URL`

Optional HTTPS support link or one `mailto:help@example.com` mailbox. Empty hides the link. Userinfo, control characters and mailto headers are rejected.

## Scoped deployment diagnostics

Use [`stargate doctor`](DIAGNOSTICS.md) for explicit offline configuration inspection and optional health/readiness checks. Health success does not prove HMAC or a working login.
