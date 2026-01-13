# API Documentation

This document describes in detail all API endpoints of the Stargate Forward Auth Service.

## Table of Contents

- [Authentication Check Endpoint](#authentication-check-endpoint)
- [Login Endpoint](#login-endpoint)
- [Logout Endpoint](#logout-endpoint)
- [Session Exchange Endpoint](#session-exchange-endpoint)
- [Health Check Endpoint](#health-check-endpoint)
- [Root Endpoint](#root-endpoint)

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

Handles login requests, verifies password, and creates a session.

#### Request Body

Form data (`application/x-www-form-urlencoded`):

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `password` | String | Yes | User password |
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

```bash
# Submit login form (with callback)
curl -X POST \
     -d "password=yourpassword&callback=app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login

# Submit login form (without callback, will auto-infer)
curl -X POST \
     -d "password=yourpassword" \
     -H "X-Forwarded-Host: app.example.com" \
     -c cookies.txt \
     http://auth.example.com/_login
```

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

## Authentication Flow Examples

### Web Application Authentication Flow

1. User accesses protected resource (e.g., `https://app.example.com/dashboard`)
2. Traefik intercepts the request and forwards it to `https://auth.example.com/_auth`
3. Stargate checks the session in the cookie
4. If not authenticated, redirects to `https://auth.example.com/_login?callback=app.example.com`
5. User enters password and submits
6. Stargate verifies password, creates session, sets cookie
7. Redirects to `https://app.example.com/_session_exchange?id=<session_id>`
8. Session cookie is set to the `app.example.com` domain
9. User accesses protected resource again, authentication succeeds

### API Authentication Flow

1. API client sends request to protected resource
2. Traefik intercepts the request and forwards it to `https://auth.example.com/_auth`
3. API client includes `Stargate-Password: <password>` in the request header
4. Stargate verifies password
5. If verification succeeds, sets `X-Forwarded-User` header and returns 200
6. Traefik allows the request to continue to the backend service

## Notes

1. **Session expiration time**: Default 24 hours, requires re-login after expiration
2. **Cookie security**: All cookies are set with `HttpOnly` and `SameSite=Lax` flags
3. **Password verification**: Passwords are normalized before verification (remove spaces, convert to uppercase)
4. **Multiple password support**: Multiple passwords can be configured, any password that passes verification is acceptable
5. **Cross-domain sessions**: The `COOKIE_DOMAIN` environment variable must be configured to enable cross-domain session sharing
