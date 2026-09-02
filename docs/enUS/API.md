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
- [Health Check Endpoint](#health-check-endpoint)
- [Metrics Endpoint](#metrics-endpoint)
- [Root Endpoint](#root-endpoint)
- [Authentication Flows](#authentication-flows)

## Authentication Check Endpoint

### `GET /_auth`

The main authentication check endpoint for Traefik Forward Auth. This endpoint is the core functionality of Stargate, used to verify whether a user has been authenticated.

#### Authentication Methods

Stargate supports two authentication methods, checked in the following priority order:

1. **Header Authentication** (API requests)
   - Request header: `Stargate-Password: <password>`
   - Disabled by default. Set `PASSWORD_HEADER_AUTH_ENABLED=true` only for trusted legacy clients.
   - Suitable for API requests, automation scripts, etc.

2. **Cookie Authentication** (Web requests)
   - Cookie: `stargate_session_id=<session_id>`
   - Suitable for web applications accessed via browsers

#### Request Headers

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| `Stargate-Password` | String | No | Password authentication for API requests |
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
| `500 Internal Server Error` | Server error | Error message |

#### Request Type Handling

- **HTML requests**: Redirect to `/_login?callback=<originalURL>` on authentication failure
- **API requests** (JSON/XML): Return 401 error response on authentication failure

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

1. **From cookie**: If the domain differed when previously accessing the login page, the callback is stored in the `stargate_callback` cookie
2. **From form data**: The `callback` field in the POST request form data
3. **From query parameters**: The `callback` in the URL query parameters
4. **Auto-inference**: If none of the above exist, and the origin domain (`X-Forwarded-Host`) differs from the authentication service domain, use the origin domain as the callback

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
| `500 Internal Server Error` | Server error | Error message |

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
| `Idempotency-Key` | Optional. If present, Stargate forwards it to Herald; Herald returns the same challenge response for duplicate requests with the same key within TTL. |

#### Request Body

Form data using either `application/x-www-form-urlencoded` or `multipart/form-data`. JSON request bodies are not supported.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `phone` | String | No | User phone number; at least one of `phone` or `mail` is required |
| `mail` | String | No | User email; at least one of `mail` or `phone` is required |
| `deliver_via` | String | No | Preferred channel: `sms`, `email`, or `dingtalk`. If omitted, Stargate tries an enabled SMS destination first, then email. DingTalk is never selected implicitly; callers must send `deliver_via=dingtalk`. |

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

**Failure Response**

| Status Code | Description | Response Body |
|-------------|-------------|---------------|
| `400 Bad Request` | Invalid request parameters (missing phone or mail) | Error message |
| `401 Unauthorized` | User is absent from Warden or is not active | JSON error with `success=false` |
| `429 Too Many Requests` | Rate limit triggered | Error message |
| `503 Service Unavailable` | Herald client or service is unavailable | JSON error with `success=false` |
| `500 Internal Server Error` | Herald rejected the send for another provider error | JSON error with `success=false` |

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

## Step-up Authentication Endpoint

### `GET /_step_up`

Displays a password re-verification form for an authenticated session. The optional `callback` query parameter must be a local path; unsafe values fall back to `/`.

### `POST /_step_up`

Re-verifies the password and records successful step-up authentication in the session for 10 minutes. Submit form fields `password` and optional local-path `callback`. This route is same-origin protected and rate limited.

## Session Exchange Endpoint

### `GET /_session_exchange`

Used for cross-domain session sharing. Redeems a short-lived encrypted ticket, verifies that its session is authenticated, sets the cookie, and redirects to the root path.

This endpoint is primarily used to share authentication sessions across multiple domains/subdomains. After a user logs in on one domain, this endpoint can be used to set the session cookie on another domain.

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `ticket` | String | Yes | Encrypted, audience-bound exchange ticket returned by login |

#### Response

**Success Response (302 Redirect)**

```
HTTP/1.1 302 Found
Location: /
Set-Cookie: stargate_session_id=<session_id>; Path=/; HttpOnly; SameSite=Lax; Domain=<cookie_domain>; Expires=<expiry>
```

**Failure Response**

| Status Code | Description | Response Body |
|-------------|-------------|---------------|
| `400 Bad Request` | Missing ticket | Error message |
| `401 Unauthorized` | Invalid, expired, replayed, or misdirected ticket | Error message |

#### Cookie Domain

If the `COOKIE_DOMAIN` environment variable is configured, the cookie will be set to the specified domain, enabling cross-subdomain sharing.

#### Example

```bash
# Redeem the ticket returned by the login response
curl "https://app.example.com/_session_exchange?ticket=<opaque_ticket>"
```

**Typical Usage Scenario:**

1. User logs in at `auth.example.com`
2. After successful login, redirects to `app.example.com/_session_exchange?ticket=<opaque_ticket>`
3. Session cookie is set to the `.example.com` domain (if `COOKIE_DOMAIN=.example.com` is configured)
4. Redirects to `app.example.com/`
5. User can use this session across all `*.example.com` subdomains

## TOTP Endpoints

When Herald TOTP is enabled (`HERALD_ENABLED=true`, `HERALD_TOTP_ENABLED=true`), Stargate provides TOTP (authenticator app) bind/unbind and verification as part of login. These endpoints require an authenticated session. Enrollment must start and finish within 10 minutes of the login that created the session.

### `GET /totp/enroll`

Displays a confirmation page without creating enrollment state. It requires a session created by a successful login within the last 10 minutes; unauthenticated users are redirected with `302 Found` to `/_login`, while an older session receives `401 Unauthorized`.

### `POST /totp/enroll`

Starts enrollment and displays the TOTP bind page. POST prevents cross-site navigation from creating enrollment state.

- **Authentication**: A session created by a successful login within the last 10 minutes. Unauthenticated users are redirected with `302 Found` to `/_login`; older sessions receive `401 Unauthorized`.
- **Response**: 200 OK with enrollment page HTML; `302 Found` to `/_login` if not authenticated; or 400/503 on configuration or Herald errors.

### `POST /totp/enroll/confirm`

Confirms TOTP binding with the code from the authenticator app.

- **Authentication**: A session created by a successful login within the last 10 minutes.
- **Request Body**: Form data using either `application/x-www-form-urlencoded` or `multipart/form-data`, containing the `enroll_id` returned by the enrollment page and the 6-digit TOTP `code`. JSON request bodies are not supported.
- **Response**: JSON. Success returns `{"ok":true,"subject":"...","totp_enabled":true,"backup_codes":[...]}`; invalid or expired enrollment state returns `400 Bad Request`, and missing recent authentication returns `401 Unauthorized`.

### `GET /totp/revoke`

Displays the TOTP unbind confirmation page.

- **Authentication**: Required (session cookie). Unauthenticated users are redirected with `302 Found` to `/_login`.
- **Response**: 200 OK with revoke confirmation page; or `302 Found` to `/_login` if not authenticated.

### `POST /totp/revoke`

Confirms TOTP unbind (removes TOTP from the account).

- **Authentication**: Required (session cookie).
- **Request Body**: `password` or a valid current TOTP `code` is required to prove recent authentication.
- **Response**: JSON. Success returns `{"ok":true,"subject":"..."}`; failed reauthentication returns `401 Unauthorized`, and an upstream revoke failure returns `502 Bad Gateway`.

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

## Metrics Endpoint

### `GET /metrics`

Prometheus metrics in text exposition format. Used for monitoring authentication requests, session creation/destruction, and Herald/Warden call latency and results.

- **Authentication**: None (endpoint is typically not exposed to the public; restrict access via network or reverse proxy if needed).
- **Response**: 200 OK with `Content-Type: text/plain` and Prometheus exposition format.

**Typical Uses:**

- Prometheus scrape target
- Grafana dashboards

## Runtime Log Level Endpoint

### `GET /log/level`

Returns the current log level. `PUT /log/level` or `POST /log/level` changes it using JSON body `{"level":"debug"}` or query parameter `?level=debug`. Stargate restricts this endpoint to `127.0.0.1`; it must not be exposed through the public reverse proxy.

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

**Key Principle: forwardAuth main path only verifies session, does not call Warden/Herald**

1. User accesses protected resource (e.g., `https://app.example.com/dashboard`)
2. Traefik intercepts the request and forwards it to `https://auth.example.com/_auth`
3. Stargate **only verifies Session** (reads from Cookie or Redis)
   - **Does not call Warden**
   - **Does not call Herald**
4. If authenticated, sets `X-Forwarded-User` header and returns 200
5. If not authenticated, redirects to login page (HTML requests) or returns 401 (API requests)

### Password Authentication Login Flow

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
11. Stargate issues session (cookie/JWT), gets user information from Warden and writes to session claims
12. Redirects to `https://app.example.com/_session_exchange?ticket=<opaque_ticket>`
13. Session cookie is set to the `app.example.com` domain
14. User accesses protected resource again, forwardAuth **only verifies Stargate session**, does not trigger Warden/Herald

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
5. **Cross-domain sessions**: The `COOKIE_DOMAIN` environment variable must be configured to enable cross-domain session sharing
