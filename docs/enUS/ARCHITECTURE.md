# Stargate Architecture Document

This document describes the technical architecture and design decisions of the Stargate project.

## Technology Stack

- **Language**: Go 1.25
- **Web Framework**: [Fiber v2.52.10](https://github.com/gofiber/fiber)
- **Template Engine**: [Fiber Template v1.7.5](https://github.com/gofiber/template)
- **Session Management**: Fiber Session Middleware
- **Logging**: [Logrus v1.9.3](https://github.com/sirupsen/logrus)
- **Terminal Output**: [Pterm v0.12.82](https://github.com/pterm/pterm)
- **Testing Framework**: [Testza v0.5.2](https://github.com/MarvinJWendt/testza)

## Project Structure

```
codes/src/
├── cmd/stargate/          # Application entry point
│   ├── main.go            # Main function, initializes configuration and starts server
│   ├── server.go          # Server configuration and route setup
│   └── constants.go       # Route and configuration constants
│
├── internal/              # Internal packages (not exposed externally)
│   ├── auth/              # Authentication logic
│   │   ├── auth.go        # Authentication core functionality
│   │   └── auth_test.go   # Authentication tests
│   │
│   ├── config/            # Configuration management
│   │   ├── config.go      # Configuration variable definitions and initialization
│   │   ├── validation.go  # Configuration validation logic
│   │   └── config_test.go # Configuration tests
│   │
│   ├── handlers/          # HTTP request handlers
│   │   ├── check.go       # Authentication check handler
│   │   ├── login.go       # Login handler
│   │   ├── logout.go      # Logout handler
│   │   ├── session_share.go # Session sharing handler
│   │   ├── health.go      # Health check handler
│   │   ├── index.go       # Root path handler
│   │   ├── utils.go       # Handler utility functions
│   │   └── handlers_test.go # Handler tests
│   │
│   ├── i18n/              # Internationalization support
│   │   └── i18n.go        # Multi-language translations
│   │
│   ├── middleware/        # HTTP middleware
│   │   └── log.go         # Logging middleware
│   │
│   ├── secure/            # Password encryption algorithms
│   │   ├── interface.go   # Encryption algorithm interface
│   │   ├── plaintext.go   # Plain text password (testing only)
│   │   ├── bcrypt.go      # BCrypt algorithm
│   │   ├── md5.go         # MD5 algorithm
│   │   ├── sha512.go      # SHA512 algorithm
│   │   └── secure_test.go # Encryption algorithm tests
│   │
│   └── web/               # Web resources
│       └── templates/     # HTML templates
│           ├── login.html # Login page template
│           └── assets/   # Static resources
│               └── favicon.ico
```

## Core Components

### 1. Authentication System (`internal/auth`)

The authentication system is responsible for:
- Password verification (supports multiple encryption algorithms)
- Session management (create, verify, destroy)
- Authentication status checking

**Key Functions:**
- `CheckPassword(password string) bool`: Verifies password
- `Authenticate(session *session.Session) error`: Marks session as authenticated
- `IsAuthenticated(session *session.Session) bool`: Checks if session is authenticated
- `Unauthenticate(session *session.Session) error`: Destroys session

### 2. Configuration System (`internal/config`)

The configuration system provides:
- Environment variable management
- Configuration validation
- Default value support

**Configuration Variables:**
- `AUTH_HOST`: Authentication hostname (required)
- `PASSWORDS`: Password configuration (algorithm:password list) (required)
- `DEBUG`: Debug mode (default: false)
- `LANGUAGE`: Interface language (default: en, supports en/zh)
- `COOKIE_DOMAIN`: Cookie domain (optional, for cross-domain session sharing)
- `LOGIN_PAGE_TITLE`: Login page title (default: Stargate - Login)
- `LOGIN_PAGE_FOOTER_TEXT`: Login page footer text (default: Copyright © 2024 - Stargate)
- `USER_HEADER_NAME`: User header name set after successful authentication (default: X-Forwarded-User)
- `PORT`: Service listening port (local development only, default: 80)

### 3. Request Handlers (`internal/handlers`)

Handlers are responsible for processing HTTP requests:

- **CheckRoute**: Traefik Forward Auth authentication check
- **LoginRoute/LoginAPI**: Login page and login processing
- **LogoutRoute**: Logout processing
- **SessionShareRoute**: Cross-domain session sharing
- **HealthRoute**: Health check
- **IndexRoute**: Root path processing

### 4. Password Encryption (`internal/secure`)

Supports multiple password encryption algorithms:
- `plaintext`: Plain text (testing only)
- `bcrypt`: BCrypt hash
- `md5`: MD5 hash
- `sha512`: SHA512 hash

All algorithms implement the `HashResolver` interface:
```go
type HashResolver interface {
    Check(h string, password string) bool
}
```

## Workflow

### Authentication Flow

1. **User accesses protected resource**
   - Traefik intercepts the request
   - Forwards to Stargate `/_auth` endpoint

2. **Stargate checks authentication**
   - First checks `Stargate-Password` header (API authentication)
   - If header authentication fails, checks `stargate_session_id` cookie (Web authentication)

3. **Authentication succeeds**
   - Sets `X-Forwarded-User` header (or configured user header name) with value "authenticated"
   - Returns 200 OK
   - Traefik allows the request to continue

4. **Authentication fails**
   - HTML requests: Redirects to login page (`/_login?callback=<originalURL>`)
   - API requests (JSON/XML): Returns 401 Unauthorized

### Login Flow

1. **User accesses login page**
   - `GET /_login?callback=<url>`
   - If already logged in, redirects to session exchange endpoint
   - If domain differs, stores callback in cookie (`stargate_callback`)

2. **Submit login form**
   - `POST /_login` with password
   - Verifies password
   - Creates session and sets cookie
   - **Callback retrieval priority**:
     1. From cookie (if previously set)
     2. From form data
     3. From query parameters
     4. If none of the above, and origin domain differs from authentication service domain, use origin domain as callback

3. **Session exchange**
   - If callback exists, redirects to `{callback}/_session_exchange?id=<session_id>`
   - `GET /_session_exchange?id=<session_id>`
   - Sets session cookie (if `COOKIE_DOMAIN` is configured, sets to specified domain)
   - Redirects to root path `/`

## Security Considerations

### Session Security

- Cookies use `HttpOnly` flag to prevent XSS attacks
- Cookies use `SameSite=Lax` to prevent CSRF attacks
- Cookie path is set to `/`, allowing use across the entire domain
- Session expiration time: 24 hours (`config.SessionExpiration`)
- Supports custom cookie domain (for cross-domain scenarios)
- Session IDs are generated using UUID to ensure uniqueness and security

### Password Security

- Supports multiple encryption algorithms (recommend using bcrypt or sha512)
- Password configuration passed via environment variables, not stored in code
- Password normalization during verification (remove spaces, convert to uppercase)

### Request Security

- Authentication check endpoint supports two authentication methods:
  - Header authentication (`Stargate-Password`): For API requests
  - Cookie authentication: For Web requests
- Distinguishes between HTML and API requests, returns appropriate responses

## Extensibility

### Adding New Password Algorithms

1. Create new algorithm implementation in `internal/secure/`
2. Implement `HashResolver` interface
3. Register algorithm in `config/validation.go`

### Adding New Languages

1. Add language constant in `internal/i18n/i18n.go`
2. Add translation mappings
3. Add language option in configuration

### Customizing Login Page

Modify the `internal/web/templates/login.html` template file.

## Performance Optimization

- Uses Fiber framework, based on fasthttp, excellent performance
- Sessions stored in memory for fast access
- Static resources served via Fiber static file service
- Supports debug mode, can be disabled in production

## Deployment Architecture

### Docker Deployment

- Multi-stage build to reduce image size
- Uses `golang:1.25-alpine` as build stage
- Uses `scratch` base image as runtime stage to minimize security risks
- Template files copied from `src/internal/web/templates` to `/app/web/templates` in image
- Uses Chinese mirror source (`GOPROXY=https://goproxy.cn`) to accelerate dependency downloads
- Uses `-ldflags "-s -w"` during compilation to reduce binary size
- Application automatically finds template paths (supports `./internal/web/templates` for local development and `./web/templates` for production)

### Traefik Integration

- Integrated via Forward Auth middleware
- Supports HTTP and HTTPS
- Supports multiple domains and path rules

## Logging and Monitoring

- Uses Logrus for logging
- Supports debug mode (DEBUG=true)
- All critical operations are logged
- Health check endpoint available for monitoring

## Testing

- Unit tests cover core functionality
- Test files located in `*_test.go` files in each package
- Uses `testza` for assertions
- Test coverage includes:
  - Authentication logic (`internal/auth/auth_test.go`)
  - Configuration validation (`internal/config/config_test.go`)
  - Password encryption algorithms (`internal/secure/secure_test.go`)
  - HTTP handlers (`internal/handlers/handlers_test.go`)

## Future Improvements

- [ ] Support more password encryption algorithms
- [ ] Support OAuth2/OpenID Connect
- [ ] Support multi-user and role management
- [ ] Add admin interface
- [ ] Support external session storage (Redis, etc.)
- [ ] Add Prometheus metrics export
- [ ] Support configuration files (YAML/JSON)
