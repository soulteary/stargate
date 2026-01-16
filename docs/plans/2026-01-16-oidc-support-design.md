# OAuth2/OIDC Support Design

**Goal:** Add OAuth2/OpenID Connect authentication support to Stargate, enabling integration with enterprise SSO providers like Keycloak, Azure AD, and self-hosted OIDC services.

**Architecture:** When OIDC is enabled, password authentication is automatically disabled. Users click a "Login with {Provider}" button, are redirected to the OIDC provider, and upon successful authentication, a session is created with user_id and email claims.

**Tech Stack:** Go 1.25, Fiber v2, coreos/go-oidc/v3

---

## Overview

Stargate currently supports password-based authentication only. This design adds OIDC support while maintaining the project's philosophy of simple configuration (environment variables only) and lightweight deployment.

**Key Design Decisions:**
- **Mutually exclusive modes** - OIDC or password, not both
- **Generic OIDC** - Works with any standard OIDC provider
- **Minimal user info** - Only user_id (sub) and email claims
- **Local JWT validation** - High performance, no remote introspection calls
- **Zero UI choices** - Single configurable login button

---

## Configuration

### New Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OIDC_ENABLED` | No | `false` | Enable OIDC authentication |
| `OIDC_ISSUER_URL` | Yes* | - | OIDC Provider Issuer URL (e.g., `https://keycloak.example.com/realms/myrealm`) |
| `OIDC_CLIENT_ID` | Yes* | - | Client ID from provider |
| `OIDC_CLIENT_SECRET` | Yes* | - | Client secret from provider |
| `OIDC_REDIRECT_URI` | No | Auto | Callback URL, defaults to `https://{AUTH_HOST}/_oidc/callback` |
| `OIDC_PROVIDER_NAME` | No | `OIDC` | Display name for login button |

*Required when `OIDC_ENABLED=true`

### Example docker-compose.yml

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      - AUTH_HOST=auth.example.com
      - OIDC_ENABLED=true
      - OIDC_ISSUER_URL=https://keycloak.example.com/realms/myrealm
      - OIDC_CLIENT_ID=stargate-client
      - OIDC_CLIENT_SECRET=your-secret-here
      - OIDC_PROVIDER_NAME=Company Account
```

### Configuration Validation

On startup, when `OIDC_ENABLED=true`:
1. Verify required fields are present
2. Fetch and validate provider metadata from `{ISSUER_URL}/.well-known/openid-configuration`
3. Fail startup with clear error message if validation fails

---

## Authentication Flow

### Login Sequence

```
1. User accesses protected resource
   ↓
2. Traefik forwards to /_auth
   ↓
3. No session → Redirect to /_login
   ↓
4. Display "Login with {OIDC_PROVIDER_NAME}" button
   ↓
5. User clicks button → GET /_oidc/login
   ↓
6. Generate state, store in session
   ↓
7. Redirect to provider's authorization endpoint
   ↓
8. User authenticates at provider
   ↓
9. Provider redirects to /_oidc/callback?code=xxx&state=yyy
   ↓
10. Validate state matches session
    ↓
11. Exchange code for ID token
    ↓
12. Verify JWT signature locally
    ↓
13. Extract sub (user_id) and email claims
    ↓
14. Create session with user info
    ↓
15. Redirect to /_session_exchange?id={session_id}
    ↓
16. Complete login flow
```

### Session Storage

When OIDC authentication succeeds, session contains:

```go
session.Set("authenticated", true)
session.Set("user_id", claims["sub"])      // OIDC sub claim
session.Set("email", claims["email"])      // User email
session.Set("provider", "oidc")            // Auth method标识
```

### X-Forwarded-User Header

The `/_auth` endpoint returns:
- `X-Forwarded-User: {user_id}` if available
- Falls back to `X-Forwarded-User: {email}` if no user_id
- Legacy behavior: `X-Forwarded-User: authenticated` if neither

---

## New API Endpoints

### GET /_oidc/login

Initiates OIDC authentication flow.

**Behavior:**
1. Generate cryptographically random state
2. Store state in session
3. Build authorization URL with:
   - `response_type=code`
   - `scope=openid email`
   - `client_id`
   - `redirect_uri`
   - `state`
4. Redirect to provider

### GET /_oidc/callback

Handles OAuth2 callback from provider.

**Query Parameters:**
- `code` (required): Authorization code
- `state` (required): State parameter for CSRF protection

**Success Response:**
- Validates state
- Exchanges code for ID token
- Verifies JWT signature
- Creates session with user info
- Redirects to `/_session_exchange?id={session_id}`

**Error Response:**
- Displays error page with retry button
- Logs detailed error for debugging

---

## Data Structures

### OIDC Configuration

```go
// src/internal/config/oidc.go
type OIDCConfig struct {
    Enabled      bool
    IssuerURL    string
    ClientID     string
    ClientSecret string
    RedirectURI  string
    ProviderName string
}

type ProviderMetadata struct {
    Issuer        string
    AuthEndpoint  string
    TokenEndpoint string
    JWKSURI       string
}
```

### Session Data

```go
type SessionData struct {
    Authenticated bool
    UserID        string  // OIDC sub claim
    Email         string
    Provider      string  // "oidc"
}
```

---

## Components

### New Package: `src/internal/oidc/`

**Purpose:** Core OIDC logic

**Files:**
- `oidc.go` - Provider discovery, client initialization
- ` verifier.go` - JWT verification logic
- `claims.go` - Claims extraction

### New Handler: `src/internal/handlers/oidc.go`

**Functions:**
- `OIDCLogin(ctx *fiber.Ctx) error` - Initiate login flow
- `OIDCCallback(ctx *fiber.Ctx) error` - Handle callback

### New Templates

**`src/internal/web/templates/login_oidc.html`**
- Single login button: "使用 {OIDC_PROVIDER_NAME} 登录"
- Auto-redirect option (configurable)

**`src/internal/web/templates/error.html`**
- Generic error page with retry button
- Supports i18n error messages

### Modified Files

**`src/internal/config/`**
- Add OIDC configuration variables
- Add OIDC validation logic

**`src/internal/handlers/login.go`**
- Check `config.OIDC.Enabled` to determine which page to show
- Disable POST handler when OIDC enabled

**`src/internal/handlers/check.go`**
- Modify `X-Forwarded-User` to use user_id or email from session

---

## Dependencies

### New Dependency

```
github.com/coreos/go-oidc/v3 v3.10.0
```

**Rationale:** Industry-standard OIDC client library with built-in:
- Provider discovery
- JWT signature verification
- Claims parsing

---

## Error Handling

| Error Scenario | Handling |
|----------------|----------|
| Missing required config | Startup fails with descriptive error |
| Provider unreachable | Startup fails, check network/access |
| Invalid issuer URL | Startup fails, "not a valid OIDC provider" |
| User denies authorization | Show error page with retry |
| Code exchange fails | Show error page, log details |
| Invalid JWT | Show error page, "authentication failed" |
| State mismatch | Show error page, "security validation failed" |
| Missing email claim | Use user_id only, log warning |

---

## Migration Path

### For Existing Users

No breaking changes. Existing password-based authentication continues to work unchanged.

### To Enable OIDC

1. Add OIDC environment variables to docker-compose.yml
2. Set `OIDC_ENABLED=true`
3. `PASSWORDS` config is ignored (but can remain for fallback)
4. Restart container

### To Revert to Password Auth

1. Set `OIDC_ENABLED=false` or remove variable
2. Restart container

---

## Security Considerations

### State Parameter
- Cryptographically random (16+ bytes)
- Stored in session
- Validated on callback
- Single-use only

### JWT Validation
- Verify signature using provider's JWKS
- Check issuer matches configured issuer
- Check audience includes client_id
- Check expiration (exp claim)

### Token Storage
- Only session ID stored in cookie (HttpOnly, SameSite=Lax)
- ID token not stored, only claims extracted to session
- Session expires in 24 hours

### Redirect URI
- Validated against allowed redirect URIs at provider
- Default uses AUTH_HOST to prevent open redirect

---

## Testing Strategy

### Unit Tests
- Provider discovery and metadata parsing
- JWT verification logic
- Claims extraction
- State generation/validation

### Integration Tests
- Full login flow with mock provider
- Error scenarios (invalid code, state mismatch)
- Session creation and user info storage

### Manual Testing
- Test with real Keycloak instance
- Test with Azure AD
- Verify callback flow
- Test error cases

---

## Future Enhancements (Out of Scope)

- Multiple OIDC providers simultaneously
- Group/role claims propagation
- Token refresh with refresh tokens
- Resource Owner Password Credentials flow
- Additional claims (profile, groups, etc.)
- Configuration file support (YAML/JSON)
