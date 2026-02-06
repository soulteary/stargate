# API Documentation

This document describes in detail all API endpoints of the Stargate Forward Auth Service.

## Table of Contents

- [Authentication Check Endpoint](#authentication-check-endpoint)
- [Login Endpoint](#login-endpoint)
- [Send Verification Code Endpoint](#send-verification-code-endpoint)
- [Logout Endpoint](#logout-endpoint)
- [Session Exchange Endpoint](#session-exchange-endpoint)
- [Health Check Endpoint](#health-check-endpoint)
- [Root Endpoint](#root-endpoint)
- [Authentication Flows](#authentication-flows)

## Authentication Check Endpoint

### `GET /_auth`

The main authentication check endpoint for Traefik Forward Auth. This endpoint is the core functionality of Stargate, used to verify whether a user has been authenticated.

#### Authentication Methods

Stargate supports two authentication methods, checked in the following priority order:

1. **Header Authentication** (API requests)
   - Request header: `Stargate-Password: <password>`
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
2. **Warden + Herald OTP Authentication Mode**: Verifies code and creates session

#### Request Body

Form data (`application/x-www-form-urlencoded`):

**Password Authentication Mode:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `auth_method` | String | No | Authentication method, value is `password` (default) |
| `password` | String | Yes | User password |
| `callback` | String | No | Callback URL after successful login |

**Warden + Herald OTP Authentication Mode:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `auth_method` | String | Yes | Authentication method, value is `warden` |
| `phone` | String | No | User phone number (one of `phone` or `mail`) |
| `mail` | String | No | User email (one of `phone` or `mail`) |
| `challenge_id` | String | Yes | challenge_id returned by Herald |
| `code` | String | Yes | Verification code entered by user |
| `callback` | String | No | Callback URL after successful login |

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
   - Redirects to `{callback}/_session_exchange?id={session_id}`
   - Status code: `302 Found`

2. **Without callback**:
   - **HTML request**: Returns an HTML page with meta refresh, automatically redirecting to the origin domain
   - **API request**: Returns JSON response
     ```json
     {
       "success": true,
       "message": "Login successful",
       "session_id": "<session_id>"
     }
     ```

**Failure Response**

| Status Code | Description | Response Body |
|-------------|-------------|---------------|
| `401 Unauthorized` | Incorrect password | Error message in JSON/XML/text format based on Accept header |
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

**Warden + Herald OTP Authentication:**

```bash
# Submit login form (with verification code)
curl -X POST \
     -d "auth_method=warden&mail=user@example.com&challenge_id=ch_xxx&code=123456&callback=app.example.com" \
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

## Send Verification Code Endpoint

### `POST /_send_verify_code`

Send verification code request. This endpoint is used in the Warden + Herald OTP authentication flow.

#### Request Headers (optional)

| Header | Description |
|--------|-------------|
| `Idempotency-Key` | Optional. If present, Stargate forwards it to Herald; Herald returns the same challenge response for duplicate requests with the same key within TTL. |

#### Request Body

Form data (`application/x-www-form-urlencoded`) or JSON (`application/json`):

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `phone` | String | No | User phone number (one of `phone` or `mail`) |
| `mail` | String | No | User email (one of `phone` or `mail`) |

#### Request Headers (optional)

| Header | Description |
|--------|-------------|
| `Idempotency-Key` | Passed through to Herald; same key within TTL returns cached challenge (no duplicate send). Omit for each new send. |

#### Processing Flow

1. **Stargate → Warden**: Query user information
   - Verify if user is in whitelist
   - Check user status (if active)
   - Get user's email and phone

2. **Stargate → Herald**: Create challenge and send verification code
   - Use email/phone returned by Warden as destination
   - Call Herald API to create challenge
   - Herald sends verification code (SMS or Email)

3. **Return Result**: Return challenge_id and related information

#### Response

**Success Response (200 OK)**

```json
{
  "success": true,
  "challenge_id": "ch_xxxxxxxxxxxx",
  "expires_in": 300,
  "next_resend_in": 60,
  "channel": "email",
  "destination": "u***@example.com"
}
```

When `DEBUG=true`, the response may also include `debug_code` (the verification code) so the login page can display it for local/testing. **Do not enable DEBUG in production.**

**Failure Response**

| Status Code | Description | Response Body |
|-------------|-------------|---------------|
| `400 Bad Request` | Invalid request parameters (missing phone or mail) | Error message |
| `404 Not Found` | User not in Warden whitelist | Error message |
| `429 Too Many Requests` | Rate limit triggered | Error message |
| `500 Internal Server Error` | Server error or Herald service unavailable | Error message |

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

# Using JSON format
curl -X POST \
     -H "Content-Type: application/json" \
     -d '{"mail":"user@example.com"}' \
     http://auth.example.com/_send_verify_code
```

#### Notes

- Requires `WARDEN_ENABLED=true` and `HERALD_ENABLED=true`
- User must be in Warden whitelist to send verification code
- Herald performs rate limiting, with frequency limits for same user/phone/email
- Code expiration time is determined by Herald configuration (default 300 seconds)
- Resend cooldown is determined by Herald configuration (default 60 seconds)
- **Debug mode:** When `DEBUG=true`, Stargate includes the verification code in the response (from Herald's create response or via `GET /v1/test/code/:id`) and the login page displays it (e.g. "Verification code (debug): 123456"). Use only for local/testing; set `DEBUG=false` in production.

## Logout Endpoint

### `GET /_logout`

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
curl -b cookies.txt http://auth.example.com/_logout
```

## Session Exchange Endpoint

### `GET /_session_exchange`

Used for cross-domain session sharing. Sets the specified session ID cookie and redirects to the root path.

This endpoint is primarily used to share authentication sessions across multiple domains/subdomains. After a user logs in on one domain, this endpoint can be used to set the session cookie on another domain.

#### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | String | Yes | Session ID to set |

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
| `400 Bad Request` | Missing session ID | Error message |

#### Cookie Domain

If the `COOKIE_DOMAIN` environment variable is configured, the cookie will be set to the specified domain, enabling cross-subdomain sharing.

#### Example

```bash
# Set session cookie (for cross-domain scenarios)
curl "http://auth.example.com/_session_exchange?id=<session_id>"
```

**Typical Usage Scenario:**

1. User logs in at `auth.example.com`
2. After successful login, redirects to `app.example.com/_session_exchange?id=<session_id>`
3. Session cookie is set to the `.example.com` domain (if `COOKIE_DOMAIN=.example.com` is configured)
4. Redirects to `app.example.com/`
5. User can use this session across all `*.example.com` subdomains

## Health Check Endpoint

### `GET /health`

Service health check endpoint. Used to monitor service status.

#### Response

**Success Response (200 OK)**

```
HTTP/1.1 200 OK
```

#### Example

```bash
curl http://auth.example.com/health
```

**Typical Uses:**

- Docker health checks
- Kubernetes liveness probes
- Load balancer health checks

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
6. Redirects to `https://app.example.com/_session_exchange?id=<session_id>`
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
12. Redirects to `https://app.example.com/_session_exchange?id=<session_id>`
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
3. **Password verification**: Passwords are normalized before verification (remove spaces, convert to uppercase)
4. **Multiple password support**: Multiple passwords can be configured, any password that passes verification is acceptable
5. **Cross-domain sessions**: The `COOKIE_DOMAIN` environment variable must be configured to enable cross-domain session sharing
