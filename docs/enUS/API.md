# API Documentation

This document describes in detail all API endpoints of the Stargate Forward Auth Service.

Upgrading from v0.12.0? Read the [v1.0.0 migration guide](MIGRATION_V1.md) before changing clients.

## Table of Contents

- [Authentication Check Endpoint](#authentication-check-endpoint)
- [Login Endpoint](#login-endpoint)
- [Send Verification Code Endpoint](#send-verification-code-endpoint)
- [Logout Endpoint](#logout-endpoint)
- [Session Exchange Endpoint](#session-exchange-endpoint)
- [TOTP Endpoints](#totp-endpoints)
- [Health Check Endpoint](#health-check-endpoints)
- [Metrics Endpoint](#metrics-endpoint)
- [Root Endpoint](#root-endpoint)
- [Authentication Flows](#authentication-flows)

## Authentication Check Endpoint

### `GET /_auth`

The main authentication check endpoint for Traefik Forward Auth. This endpoint is the core functionality of Stargate, used to verify whether a user has been authenticated.

#### Authentication Methods

Stargate supports three authentication methods, checked in the following priority order:

1. **Header Authentication** (API requests)
   - Request header: `Stargate-Password: <password>`
   - Disabled by default. Set `PASSWORD_HEADER_AUTH_ENABLED=true` only for trusted legacy clients.
   - Suitable for API requests, automation scripts, etc.

2. **Trusted identity headers**
   - Requires `HEADER_AUTH_ENABLED=true`, a matching `HEADER_AUTH_SHARED_SECRET` in `HEADER_AUTH_SECRET_HEADER` (default `X-Stargate-Header-Auth`), and Warden. `X-User-Phone` / `X-User-Mail` are resolved against the directory. The proxy must overwrite client-supplied identity and secret headers; never expose the shared secret to browsers.

3. **Cookie Authentication** (Web requests)
   - Cookie: `stargate_session_id=<session_id>`
   - Suitable for web applications accessed via browsers

#### Request Headers

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| `Stargate-Password` | String | No | Password authentication for API requests |
| `X-User-Phone`, `X-User-Mail` | String | — | Optional trusted identity; requires the configured proxy secret header |
| `Cookie` | String | No | Session cookie containing `stargate_session_id` |
| `Accept` | String | No | Used to determine request type (HTML/API) |

#### Response

**Success Response (200 OK)**

When authentication succeeds, Stargate sets the user information header and returns a 200 status code:

```
HTTP/1.1 200 OK
X-Forwarded-User: authenticated
```

The user header name can be configured via the `USER_HEADER_NAME` environment variable (default: `X-Forwarded-User`).

**Failure Response**

| Status Code | Description | Response Body |
|-------------|-------------|---------------|
| `401 Unauthorized` | Authentication failed | Error message (JSON format for API requests) or redirect to login page (HTML requests) |
| `403 Forbidden` | A request with `Stargate-Password` that did not arrive through a proxy in `TRUSTED_PROXIES` together with a client address. The password is not compared. Only with `PASSWORD_HEADER_REQUIRE_TRUSTED_PROXY=true` | Error message |
| `429 Too Many Requests` | `RATE_LIMIT_LOGIN_MAX` failed `Stargate-Password` attempts from this client in the current window. Sent with a `Retry-After` header (seconds) for every request that carries the header, with the correct password too, until the window ends. Only with `PASSWORD_HEADER_AUTH_ENABLED=true` | Error message |
| `503 Service Unavailable` | Directory authorization temporarily unavailable | — |
| `500 Internal Server Error` | Server error | Error message |

#### Request Type Handling

- **HTML requests**: Redirect to `/_login?callback=<originalURL>` on authentication failure
- **API requests** (JSON/XML): Return 401 error response on authentication failure
- **Requests with `Stargate-Password` over the failure quota**: Return 429 with `Retry-After` for HTML and API requests alike; there is no redirect

#### Examples

**Using Header Authentication (API Request)**

```bash
# Server prerequisite: PASSWORD_HEADER_AUTH_ENABLED=true
curl -H "Stargate-Password: yourpassword" \
     http://auth.example.com/_auth
```

**Using Cookie Authentication (Web Request)**

```bash
curl -H "Cookie: stargate_session_id=<session_id>" \
     http://auth.example.com/_auth
```

## Login Endpoint

### `GET /_login`

Displays the login page.

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `callback` | String | No | Callback URL after successful login (usually the domain of the original request) |

#### Behavior

- If the user is already logged in, automatically redirects to the session exchange endpoint
- If the user is not logged in, displays the login page
- If the URL contains a `callback` parameter and the domain differs, the callback is stored in the `stargate_callback` cookie (expires in 10 minutes)

#### Callback Retrieval Priority

1. **From query parameters**: The `callback` parameter in the URL (highest priority)
2. **From cookie**: If not in query parameters, retrieve from the `stargate_callback` cookie

#### Response

**200 OK** - Returns login page HTML

The page includes:
- Login form
- Customizable title (`LOGIN_PAGE_TITLE`)
- Customizable footer text (`LOGIN_PAGE_FOOTER_TEXT`)

#### Example

```bash
# Access login page
curl http://auth.example.com/_login?callback=app.example.com
```

### `POST /_login`

Handles login requests, supports two authentication modes:

1. **Password Authentication Mode**: Verifies password and creates session
2. **Warden Authentication Mode**: Verifies either a Herald challenge code or an enrolled TOTP code and creates a session

#### Request Body

Form data (`application/x-www-form-urlencoded`):

**Password Authentication Mode:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `auth_method` | String | No | Authentication method, value is `password` (default) |
| `password` | String | Yes | User password |
| `callback` | String | No | Callback URL after successful login |

**Warden Authentication Mode:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `auth_method` | String | Yes | Authentication method, value is `warden` |
| `phone` | String | No | User phone number; at least one of `phone` or `mail` is required |
| `mail` | String | No | User email; at least one of `mail` or `phone` is required |
| `challenge_id` | String | Conditional | Required with `verify_code`; optional when `use_otp=true` |
| `verify_code` | String | Conditional | Required for Herald challenge verification when `use_otp` is absent or `false` |
| `use_otp` | Boolean | No | Set to `true` to use an enrolled TOTP authenticator instead of a Herald challenge code |
| `otp_code` | String | Conditional | Required when `use_otp=true` |
| `callback` | String | No | Callback URL after successful login |

The handler also accepts `phone` + `mail` in the same Warden request.
The TOTP variant requires `HERALD_TOTP_ENABLED=true` and an already enrolled user. Otherwise, use the `challenge_id` + `verify_code` variant.

#### Callback Retrieval Priority

Login processing retrieves the callback in the following priority order:

1. **From form data**: The `callback` field in the POST request form data. The login page fills it with the callback it was opened for
2. **From cookie**: If the form has no `callback`, the `stargate_callback` cookie that was stored when the login page was opened for a different domain
3. **From query parameters**: The `callback` in the URL query parameters
4. **Auto-inference**: If none of the above exist, and the origin domain (`X-Forwarded-Host`) differs from the authentication service domain, use the origin domain as the callback

Whichever source supplies the value, it must pass the same callback host validation; a `callback` field that is not allowed is answered `400 Bad Request` and is not replaced by the cookie. The `stargate_callback` cookie is cleared after a successful login even when the form value was used.

#### Response

**Success Response (200 OK)**

The response varies depending on whether there is a callback and the request type:

1. **With callback**:
   - Redirects to `{callback}/_session_exchange?ticket={opaque_ticket}`
   - Status code: `302 Found`

2. **Without callback**:
   - **HTML request**: Returns an HTML page with meta refresh, automatically redirecting to the origin domain
   - **API request**: Returns JSON response
     ```json
     {
       "success": true,
       "message": "Login successful"
     }
     ```

**Failure Response**

| Status Code | Description | Response Body |
|-------------|-------------|---------------|
| `400 Bad Request` | Invalid or missing Warden identifier or selected verification fields, or TOTP is not enrolled | Error message in JSON/XML/text format based on Accept header |
| `401 Unauthorized` | Incorrect password, absent/inactive Warden user, or failed challenge/TOTP verification | Error message in JSON/XML/text format based on Accept header |
| `502 Bad Gateway` | Herald TOTP status lookup failed | Error message |
| `503 Service Unavailable` | Herald verification service is unavailable | Error message |
| `409 Conflict` | Published verifier lock contention; no automatic code retry | — |
| `429 Too Many Requests` | Request rate limit exceeded | — |
| `500 Internal Server Error` | Server error | Error message |

#### Uniform Responses

With `LOGIN_UNIFORM_RESPONSES=true`, the answer to a failed Warden sign-in
(`auth_method=warden`) does not depend on the identifier. Password sign-in and
the response to a successful Warden sign-in are unchanged.

When `phone` is supplied, `mail` is not used: the request is handled as if
`mail` had not been sent. The default mode consults `mail` when the phone is
not listed. With uniform responses the owner of one listed mail address could
then sign in with it next to a phone number exactly when that number is not
listed.

`challenge_id` is the handle that `POST /_send_verify_code` returns in this
mode. The Herald challenge id behind a handle is not accepted.

Every failed sign-in answers `401 Unauthorized` with the same message:

```json
{
  "error": "Sign-in failed. Check the account and the code, or request a new code.",
  "code": 401,
  "reason": "authentication_failed"
}
```

This covers an unknown or inactive identifier, a `challenge_id` that is
unknown, expired or issued for another account, a wrong, expired or locked
code, TOTP that is not enrolled or does not verify, and a Herald failure after
the Warden lookup. The reasons `invalid`, `expired`, `locked`,
`too_many_attempts`, `totp_not_enrolled` and `lock_contention`, and the status
codes `409 Conflict` and `502 Bad Gateway`, are not returned for a Warden
sign-in in this mode.

The following answers keep their status and body. They are decided before the
Warden lookup, or do not depend on its result, and are therefore the same for
every identifier:

| Status Code | Case |
|-------------|------|
| `400 Bad Request` | Invalid phone number, no identifier, `challenge_id` or `verify_code` missing (`verification_required`), `otp_code` missing, or `use_otp=true` while TOTP is disabled |
| `429 Too Many Requests` | Per-client request limit |
| `500 Internal Server Error` | Code variant: Herald is disabled or its client is not initialized |
| `503 Service Unavailable` | Code variant: the challenge context store cannot be read. TOTP variant: the Herald client is not initialized. Either variant: the Warden lookup failed |

The mode unifies response content. It does not equalize response time; see
[Uniform Responses](SECURITY.md#uniform-responses) in the security
documentation.

#### Examples

**Password Authentication:**

```bash
# Submit login form (with callback)
curl -X POST \
     -d "auth_method=password&password=yourpassword&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login
```

**Warden + Herald Challenge Authentication:**

```bash
# Submit login form (with verification code)
curl -X POST \
     -d "auth_method=warden&mail=user@example.com&challenge_id=ch_xxx&verify_code=123456&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login
```

**Warden + TOTP Authenticator Authentication:**

```bash
# Submit login form with an enrolled authenticator code
curl -X POST \
     -d "auth_method=warden&mail=user@example.com&use_otp=true&otp_code=123456&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login
```

#### Warden + Herald Authentication Flow Description

1. User enters email or phone number, calls `POST /_send_verify_code` to send verification code
2. Stargate calls Warden to query user information (whitelist verification, status check)
3. Stargate calls Herald to create challenge and send verification code
4. User enters verification code, calls `POST /_login` for verification
5. Stargate calls Herald to verify the code
6. After successful verification, Stargate creates session and returns

With the TOTP variant, the client sets `use_otp=true` and supplies `otp_code`; no prior `POST /_send_verify_code` call is required.

## Send Verification Code Endpoint

### `POST /_send_verify_code`

Send verification code request. This endpoint is used in the Warden + Herald OTP authentication flow.

#### Request Headers (optional)

| Header | Description |
|--------|-------------|
| `Idempotency-Key` | Optional. If present, Stargate forwards it to Herald; Herald returns the same challenge response for duplicate requests with the same key within TTL. It is not forwarded when `LOGIN_UNIFORM_RESPONSES=true`. |

<!-- api-contract: send-verify-code-request-body -->
#### Request Body

Supported request-body media types:

| Request media type | Supported |
|--------------------|-----------|
| `application/x-www-form-urlencoded` | ✅ |
| `multipart/form-data` | ✅ |
| `application/json` | ❌ |

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `phone` | String | No | User phone number; at least one of `phone` or `mail` is required |
| `mail` | String | No | User email; at least one of `mail` or `phone` is required |
| `deliver_via` | String | No | Preferred channel: `sms`, `email`, or `dingtalk`. If omitted, Stargate tries an enabled SMS destination first, then email. DingTalk is never selected implicitly; callers must send `deliver_via=dingtalk`. |
| `challenge_id` | String | No | Read only with `LOGIN_UNIFORM_RESPONSES=true`: the `challenge_id` the client holds for this identifier, sent with a resend. It lets the code delivered earlier keep working when no new code is sent, and it never changes the response; see Uniform Responses below. Ignored in the default mode and when longer than 128 bytes. |

The handler also accepts `phone` + `mail` in the same verification-code request.

#### Processing Flow

1. **Stargate → Warden**: Query user information
   - Verify if user is in whitelist
   - Check user status (if active)
   - Get user's email and phone

2. **Stargate → Herald**: Create challenge and send verification code
   - Use the email, phone, or DingTalk identifier returned by Warden as the destination
   - Call Herald API to create challenge
   - Herald sends the verification code through the selected SMS, email, or DingTalk channel

3. **Return Result**: Return challenge_id and related information

#### Response

**Success Response (200 OK)**

```json
{
  "success": true,
  "message": "Verification code sent",
  "challenge_id": "ch_xxxxxxxxxxxx",
  "expires_in": 300,
  "next_resend_in": 60
}
```

On a rate-limited send, a validated upstream retry interval is returned as optional integer `next_resend_in` (1–86400 seconds) and `Retry-After`. If the upstream omits timing or sends invalid/conflicting metadata, both are absent: clients must treat the wait as unknown, without inventing a countdown or automatically retrying the send.

**Failure Response**

| Status Code | Description | Response Body |
|-------------|-------------|---------------|
| `400 Bad Request` | Invalid request parameters (missing phone or mail) | Error message |
| `401 Unauthorized` | User is absent from Warden or is not active | JSON error with `success=false` |
| `429 Too Many Requests` | Rate limit triggered | Error message |
| `503 Service Unavailable` | Herald client or service is unavailable | JSON error with `success=false` |
| `500 Internal Server Error` | Herald rejected the send for another provider error | JSON error with `success=false` |

#### Uniform Responses

With `LOGIN_UNIFORM_RESPONSES=true` the answer does not depend on whether the
identifier is on the Warden allow list.

The identifier of a request is `phone` when it is supplied, and `mail`
otherwise. A `mail` next to a `phone` is not used: the request is handled as
if `mail` had not been sent. The default mode sends the code to a listed
`mail` when the `phone` is not listed. With uniform responses the arrival of
that code would tell the owner of the mail address that the phone is not
listed.

A request is handled in this order:

1. Request checks, unchanged: `400 Bad Request` for an invalid phone number, a missing identifier, or Herald not configured.
2. Configuration checks, for every identifier: `400 Bad Request` with reason `channel_disabled` when `deliver_via` names a disabled channel, and `503 Service Unavailable` with reason `connection_failed` when the Herald client is not initialized.
3. Resend cooldown, before any lookup. The identifier has a window of `LOGIN_UNIFORM_RESEND_COOLDOWN`. While it is open the answer is `429 Too Many Requests`, and neither Warden nor Herald is contacted.
4. One Warden lookup, by the identifier. If the lookup itself fails, the answer is `503 Service Unavailable` with reason `service_unavailable` for every identifier.
5. `200 OK`, whether or not a code was sent.

**Success Response (200 OK)**

```json
{
  "success": true,
  "message": "If this account can receive codes, a code has been sent.",
  "challenge_id": "ch_xxxxxxxxxxxxxxxxxxxxxx",
  "next_resend_in": 60
}
```

- `challenge_id` is a handle that Stargate mints for every request. It is not the Herald challenge id, which is not returned. Send it to `POST /_login` as `challenge_id`.
- `next_resend_in` is `LOGIN_UNIFORM_RESEND_COOLDOWN` in whole seconds. `expires_in` and `debug_code` are not returned.
- No code is sent for an unknown or inactive identifier, for an account without a usable destination (for example when DingTalk is requested and not bound), or when Herald refuses the request or cannot be reached (resend cooldown, limit, locked user, outage). The response is the same in each case.
- A resend may present the `challenge_id` the client holds, in the form field `challenge_id`. When Herald creates no new challenge and the presented handle belongs to a challenge that is still valid for the same account, channel and destination, the new `challenge_id` is bound to that challenge and is accepted with the code delivered earlier. Without the field, or when the presented value does not match, the new `challenge_id` is bound to nothing and fails at `POST /_login` like any unknown one. The field never changes the response to this request.
- `401 Unauthorized`, and `500 Internal Server Error` for a provider error, are not returned in this mode.
- `Idempotency-Key` is not forwarded to Herald.

**Cooldown Response (429 Too Many Requests)**

The `Retry-After` header carries the same number of seconds as `next_resend_in`: the time until the request is admitted again.

```json
{
  "success": false,
  "message": "Too many requests. Please try again later.",
  "reason": "rate_limited",
  "next_resend_in": 50
}
```

The mode unifies response content. It does not equalize response time: a
request for a listed identifier waits for Herald. See
[Uniform Responses](SECURITY.md#uniform-responses) in the security
documentation.

#### Examples

```bash
# Send verification code (using email)
curl -X POST \
     -d "mail=user@example.com" \
     -H "Content-Type: application/x-www-form-urlencoded" \
     http://auth.example.com/_send_verify_code

# Send verification code (using phone)
curl -X POST \
     -d "phone=13800138000" \
     -H "Content-Type: application/x-www-form-urlencoded" \
     http://auth.example.com/_send_verify_code

```

#### Notes

- Requires `WARDEN_ENABLED=true` and `HERALD_ENABLED=true`
- User must be in Warden whitelist to send verification code
- Herald performs rate limiting, with frequency limits for same user/phone/email
- Code expiration time is determined by Herald configuration (default 300 seconds)
- Resend cooldown is determined by Herald configuration (default 60 seconds)

## Logout Endpoint

### `POST /_logout`

Logs out the current user and destroys the session.

#### Response

**Success Response (200 OK)**

```
HTTP/1.1 200 OK
Content-Type: text/plain

Logged out
```

The session cookie will be cleared.

#### Example

```bash
curl -X POST -b cookies.txt http://auth.example.com/_logout
```

## Re-authentication

Step-up, TOTP enrollment and TOTP revocation require a fresh proof from the person who owns the session. Stargate selects exactly one proof from the session and from Herald's TOTP state; the client cannot choose it:

| Session | Proof | Form field |
|---------|-------|------------|
| Warden user with TOTP enrolled (`HERALD_TOTP_ENABLED=true`) | A current TOTP code, verified by Herald | `code` |
| Any other Warden user | A one-time code sent by `POST /_send_reauth_code` to the contact registered in Warden | `code` |
| Session created with the shared password (no user identity) | The shared `PASSWORDS` secret | `password` |

The shared password is never accepted for a session that carries a user identity. A user session for which no Herald client is available is refused with `403 Forbidden` (`reauthentication_unavailable`). If Herald's TOTP state cannot be read the request is refused with `502 Bad Gateway`; it does not fall back to a delivered code. A failed proof returns `401 Unauthorized`.

A successful proof records `reauth_verified_at` in the session. TOTP enrollment accepts that record for 5 minutes. Step-up and TOTP revocation always require the proof in the same request.

### `POST /_send_reauth_code`

Sends a one-time code for re-authentication to the session's user.

- **Authentication**: Required (session cookie). The session must be one whose proof is a delivered code (see the table above); other sessions receive `400 Bad Request`.
- **Request Body**: optional `deliver_via` (`sms`, `email` or `dingtalk`) selects the channel, as on `POST /_send_verify_code`. The destination always comes from Warden; identifiers or destinations in the request are ignored.
- **Response**: JSON in the same shape as `POST /_send_verify_code`, including `429 Too Many Requests` and `next_resend_in` when Herald applies its limits. The challenge is bound to the session that requested it, to the user and to re-authentication: it cannot be used to sign in, and a sign-in code cannot be used here.

The route is same-origin protected and shares the `RATE_LIMIT_VERIFICATION_MAX` quota. The challenge is created in Herald with purpose `login`, the only purpose a default Herald deployment allows, so the message text and Herald's per-user and per-destination limits are those of a sign-in code.

## Step-up Authentication Endpoint

### `GET /_step_up`

Displays the re-authentication form for an authenticated session: a TOTP code field, a "send code" button with a code field, or a password field, as selected in [Re-authentication](#re-authentication). The optional `callback` query parameter must be a local path; unsafe values fall back to `/`.

### `POST /_step_up`

Verifies the proof and records successful step-up authentication in the session for 10 minutes. Submit the form field `code` or `password` that applies to the session, and the optional local-path `callback`. This route is same-origin protected and rate limited. A failed proof is recorded in the audit log as `login_failed` with method `step_up`.

## Session Exchange Endpoint

### `GET /_session_exchange`

Starts or completes browser-bound cross-domain session exchange. Without a ticket or target nonce cookie, it creates a host-only nonce cookie and redirects to the auth host with a sealed `exchange_state`. The login form carries this state through authentication. Direct auth-host login with a callback first redirects here to start the same handshake.

A returned `ticket` is accepted only when it matches this browser's nonce cookie, destination audience and scheme, and configured `SESSION_EXCHANGE_REDIRECT_PATH`. It is encrypted, valid for at most two minutes, and consumed once across replicas when Redis is configured. The referenced session must still be authenticated. Older unbound ticket formats are rejected.

Successful completion sets `stargate_session_id`, clears the pending nonce cookie, and redirects to the configured same-origin landing path. `COOKIE_DOMAIN` applies only if the destination is within that domain; other allowed domains receive a host-only session cookie. Invalid, expired, replayed, or mismatched tickets return `401 Unauthorized`; invalid targets or insecure requests with secure-cookie mode return `400 Bad Request`.

Open this endpoint in the same browser that completes login, for example `https://app.example.com/_session_exchange?start=1`. A standalone ticket URL without that browser's nonce cannot import the session. See [deployment guidance](DEPLOYMENT.md#browser-bound-session-exchange) for cookie and migration requirements.

## TOTP Endpoints

When Herald TOTP is enabled (`HERALD_ENABLED=true`, `HERALD_TOTP_ENABLED=true`), Stargate provides TOTP (authenticator app) bind/unbind and verification as part of login. These endpoints require an authenticated session and a per-user proof as described in [Re-authentication](#re-authentication). A recent login is not sufficient. Enrollment must be confirmed within 5 minutes of the proof.

### `GET /totp/enroll`

Displays a confirmation page without creating enrollment state. Unless the session holds a proof from the last 5 minutes, the page includes the re-authentication fields that apply to the session. Unauthenticated users are redirected with `302 Found` to `/_login`; a session whose user the directory no longer authorizes receives `401 Unauthorized`. An enrollment started from this page must be confirmed within 10 minutes.

### `POST /totp/enroll`

Starts enrollment and displays the TOTP bind page. POST prevents cross-site navigation from creating enrollment state.

- **Authentication**: An authenticated session with either a proof from the last 5 minutes or a valid proof in this request (form field `code`). Unauthenticated users are redirected with `302 Found` to `/_login`; a missing or failed proof returns `401 Unauthorized`. The enrollment state this request creates is valid for 10 minutes.
- **Response**: 200 OK with enrollment page HTML; `302 Found` to `/_login` if not authenticated; `302 Found` to `/totp/revoke` when TOTP is already enrolled (after a valid TOTP code); or 400/502/503 on configuration or Herald errors.

Upstream TOTP status/start errors return `502 Bad Gateway`.

### `POST /totp/enroll/confirm`

Confirms TOTP binding with the code from the authenticator app.

- **Authentication**: An authenticated session holding a proof from the last 5 minutes. The proof is normally the one given to `POST /totp/enroll`. Without it the response is `401 Unauthorized`. Enrollment state older than 10 minutes is refused with `400 Bad Request`.

<!-- api-contract: totp-enroll-confirm-request-body -->
#### Request Body

| Request media type | Supported |
|--------------------|-----------|
| `application/x-www-form-urlencoded` | ✅ |
| `multipart/form-data` | ✅ |
| `application/json` | ❌ |

The form must contain the `enroll_id` returned by the enrollment page and the 6-digit TOTP `code`.

#### Response

JSON. Success returns `{"ok":true,"subject":"...","totp_enabled":true,"backup_codes":[...]}`; invalid or expired enrollment state returns `400 Bad Request`, and a missing proof or one older than 5 minutes returns `401 Unauthorized` with `recent_authentication_required`.

Confirmation quota returns `429 Too Many Requests` with `rate_limited`; pending enrollment is retained. An invalid provider response returns `502`. Do not automatically retry a code.

### `GET /totp/revoke`

Displays the TOTP unbind confirmation page.

- **Authentication**: Required (session cookie). Unauthenticated users are redirected with `302 Found` to `/_login`.
- **Response**: 200 OK with revoke confirmation page, including the re-authentication fields that apply to the session; or `302 Found` to `/_login` if not authenticated.

### `POST /totp/revoke`

Confirms TOTP unbind (removes TOTP from the account).

- **Authentication**: Required (session cookie).
- **Request Body**: `code` with a current TOTP code, or with a delivered code when the user has no TOTP enrolled. The `password` field is not read for a session that carries a user identity. See [Re-authentication](#re-authentication).
- **Response**: JSON. Success returns `{"ok":true,"subject":"..."}`; failed reauthentication returns `401 Unauthorized`, and an upstream revoke failure returns `502 Bad Gateway`. When Herald answers without confirming the revocation, the response is `502 Bad Gateway` with reason `revoke_not_confirmed`; the factor may still be bound.
- **Audit**: when audit logging is enabled, a successful enrollment or revocation is recorded as `user_updated` with `action` `totp_enrolled` or `totp_revoked`. Refused and unconfirmed revocations are recorded as failures, as are enrollment requests refused for a missing or failed proof.

**Notes:** TOTP creation and verification are performed by Herald (which may proxy to herald-totp). Stargate only orchestrates the UI and session; it does not implement OTP algorithms.

## Health Check Endpoints

### `GET /healthz`

Process liveness endpoint. It returns `200 OK` while Stargate is running and does not query Redis, Warden, or Herald. Use it for container health checks and Kubernetes liveness probes.

### `GET /readyz`

Dependency readiness endpoint. It aggregates the configured Redis, Warden, and Herald checks and returns a non-2xx response when a required dependency is unavailable. Use it for Kubernetes readiness probes and load-balancer routing.

```bash
curl http://auth.example.com/healthz
curl http://auth.example.com/readyz
```

`GET /health` remains a deprecated compatibility alias for `GET /readyz`.

The dependency checks run at most once every 5 seconds. Both routes answer from that result until it expires, so a dependency that fails or recovers is reported up to 5 seconds late, and requests that arrive while the checks run wait for them instead of starting their own.

## Metrics Endpoint

### `GET /metrics`

Prometheus metrics in text exposition format. Used for monitoring authentication requests, session creation/destruction, and Herald/Warden call latency and results.

- **Authentication**: `Authorization: Bearer <METRICS_API_KEY>`. The endpoint is disabled until `METRICS_API_KEY` is configured. Also restrict access via network or reverse proxy.
- **Response**: `200 OK` with `Content-Type: text/plain` and Prometheus exposition format; `404 Not Found` when `METRICS_API_KEY` is not set; `401 Unauthorized` when the token is missing or wrong.

**Typical Uses:**

- Prometheus scrape target
- Grafana dashboards

## Runtime Log Level Endpoint

### `GET /log/level`

Returns the current log level. `PUT /log/level` or `POST /log/level` changes it using JSON body `{"level":"debug"}` or query parameter `?level=debug`. Stargate restricts this endpoint to direct connections from `127.0.0.1`. Because a same-host reverse proxy or sidecar also connects from `127.0.0.1`, a request that carries any of `X-Forwarded-For`, `X-Forwarded-Host`, `X-Forwarded-Proto`, `Forwarded`, or `X-Real-IP` receives `404 Not Found`, and `PUT`/`POST` requests are subject to the same cross-site write protection as the login endpoints (`403 Forbidden`). A local `curl http://127.0.0.1:<port>/log/level` keeps working. This check relies on the proxy adding forwarding headers; still exclude `/log/level` from any route the public reverse proxy forwards.

## Root Endpoint

### `GET /`

Root path, displays service information.

#### Response

**200 OK** - Returns service information page

#### Example

```bash
curl http://auth.example.com/
```

## Error Response Format

All API error responses automatically select format based on the client's `Accept` header:

### JSON Format (`Accept: application/json`)

```json
{
  "error": "Error message",
  "code": 401
}
```

### XML Format (`Accept: application/xml`)

```xml
<errors>
  <error code="401">Error message</error>
</errors>
```

### Text Format (Default)

```
Error message
```

Error messages support internationalization, returning Chinese or English messages based on the `LANGUAGE` environment variable.

## Authentication Flows

### ForwardAuth Authentication Flow (Main Path)

**ForwardAuth checks the configured authentication method. Warden authorization refresh and trusted identity lookup can call Warden; it does not verify OTP codes through Herald on this path.**

1. User accesses protected resource (e.g., `https://app.example.com/dashboard`)
2. Traefik intercepts the request and forwards it to `https://auth.example.com/_auth`
3. Stargate loads the server-side session using its cookie identifier, or checks the configured password/trusted identity headers. Warden sessions may require directory authorization refresh; unavailable directory evidence returns 503.
4. If authenticated, sets `X-Forwarded-User` header and returns 200
5. If not authenticated, redirects to login page (HTML requests) or returns 401 (API requests)

### Password Authentication Login Flow

The password and OTP summaries below use the browser-bound [session exchange](#session-exchange-endpoint) described above; the destination nonce handshake must precede ticket issuance.

1. User accesses protected resource
2. Traefik → Stargate `/_auth`: Check session, not logged in
3. Redirects to `https://auth.example.com/_login?callback=app.example.com`
4. User enters password and submits `POST /_login` (`auth_method=password`)
5. Stargate verifies password, creates session, sets cookie
6. Redirects to `https://app.example.com/_session_exchange?ticket=<opaque_ticket>`
7. Session cookie is set to the `app.example.com` domain
8. User accesses protected resource again, forwardAuth verifies session, authentication succeeds

### Warden + Herald OTP Authentication Login Flow

1. User accesses protected resource
2. Traefik → Stargate `/_auth`: Check session, not logged in
3. Redirects to `https://auth.example.com/_login?callback=app.example.com`
4. User enters email or phone number, submits `POST /_send_verify_code`
5. **Stargate → Warden**: Query user (whitelist verification, status check), get user_id + email/phone
6. **Stargate → Herald**: Create challenge and send verification code (SMS or Email)
7. Herald returns challenge_id, expires_in, next_resend_in
8. User enters verification code, submits `POST /_login` (`auth_method=warden`)
9. **Stargate → Herald**: verify(challenge_id, code)
10. Herald returns ok + user_id (+ optional amr/authentication strength)
11. Stargate creates a server-side session with Warden user information and sets its opaque cookie identifier; it does not issue a JWT
12. Redirects to `https://app.example.com/_session_exchange?ticket=<opaque_ticket>`
13. Session cookie is set to the `app.example.com` domain
14. User accesses the protected resource again; ForwardAuth checks the session and refreshes Warden authorization when required, without re-verifying the OTP through Herald

### API Authentication Flow

1. API client sends request to protected resource
2. Traefik intercepts the request and forwards it to `https://auth.example.com/_auth`
3. API client includes `Stargate-Password: <password>` in the request header
4. Stargate verifies password (**does not call Warden/Herald**)
5. If verification succeeds, sets `X-Forwarded-User` header and returns 200
6. Traefik allows the request to continue to the backend service

## Notes

1. **Session expiration time**: Default 24 hours, requires re-login after expiration
2. **Cookie security**: All cookies are set with `HttpOnly` and `SameSite=Lax` flags
3. **Password verification**: Passwords are opaque, case-sensitive values; whitespace and case are preserved during verification
4. **Multiple password support**: Multiple passwords can be configured, any password that passes verification is acceptable
5. **Cross-domain sessions**: Allowed destinations use the browser-bound session exchange described above. `COOKIE_DOMAIN` is optional and only shares a cookie within that DNS domain; other allowed domains receive host-only cookies


### `GET /_account`

Authenticated session-management page with links to TOTP setup and a same-origin `POST /_logout` form. Unauthenticated requests redirect to `/_login`; directory freshness checks still apply. This GET does not start enrollment. The root endpoint remains unchanged.
