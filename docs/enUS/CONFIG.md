# Configuration Reference

This document details all configuration options for Stargate.

## Table of Contents

- [Configuration Methods](#configuration-methods)
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
export PASSWORDS=plaintext:yourpassword
```

**Docker:**

```bash
docker run -e AUTH_HOST=auth.example.com -e PASSWORDS=plaintext:yourpassword stargate:latest
```

**Docker Compose:**

```yaml
services:
  stargate:
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=plaintext:yourpassword
```

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
PASSWORDS=plaintext:test123

# Multiple plain text passwords
PASSWORDS=plaintext:test123|admin456|user789

# BCrypt hash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# SHA512 hash
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
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
LOGIN_PAGE_TITLE=My Auth Service
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
LOGIN_PAGE_FOOTER_TEXT=© 2024 My Company
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
- Header value is `authenticated`
- Backend services can determine if a user is authenticated via this header
- Must be a non-empty string

**Example:**

```bash
USER_HEADER_NAME=X-Authenticated-User
```

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

### `PORT`

Service listening port (local development only). Managed by the config package along with other env-based options.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | Empty (when empty, server uses default port `:80`) |

**Description:**

- Only for local development environment
- Usually not needed in Docker containers (uses default port 80)
- Format: port number (e.g., `8080`) or `:port` (e.g., `:8080`)

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
| **Example** | `http://warden:8080` or `https://warden.example.com` |

**Description:**

- Full base URL of the Warden service (without trailing slash)
- Must be set if `WARDEN_ENABLED=true`
- Used for querying user information and whitelist verification

**Example:**

```bash
WARDEN_URL=http://warden:8080
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

TTL (Time To Live) for Warden user information cache.

| Attribute | Value |
|-----------|-------|
| **Type** | Integer |
| **Required** | No |
| **Default** | `300` (5 minutes) |
| **Unit** | Seconds |

**Description:**

- Time to cache user information locally
- Reduces request frequency to Warden service
- Improves authentication performance

**Example:**

```bash
WARDEN_CACHE_TTL=300
```

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

#### `HERALD_URL`

Base URL of the Herald service.

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No (required if `HERALD_ENABLED=true`) |
| **Default** | Empty |
| **Example** | `http://herald:8080` or `https://herald.example.com` |

**Description:**

- Full base URL of the Herald service (without trailing slash)
- Must be set if `HERALD_ENABLED=true`
- Used for creating challenges and verifying codes

**Example:**

```bash
HERALD_URL=http://herald:8080
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

## Password Configuration

Stargate supports multiple password encryption algorithms. Password configuration format: `algorithm:password1|password2|password3`

### Supported Algorithms

#### `plaintext` - Plain Text Password

**Description:**

- Stored in plain text, no encryption
- **Testing environment only**
- Strongly not recommended for production use

**Example:**

```bash
PASSWORDS=plaintext:test123|admin456
```

#### `bcrypt` - BCrypt Hash

**Description:**

- Uses BCrypt algorithm for hashing
- High security, recommended for production
- Password must use BCrypt hash value

**Generate BCrypt Hash:**

```bash
# Using Go
go run -c 'golang.org/x/crypto/bcrypt' <<< 'password'

# Using online tools or other tools
```

**Example:**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
```

#### `md5` - MD5 Hash

**Description:**

- Uses MD5 algorithm for hashing
- Lower security, not recommended for production
- Password must use MD5 hash value (32-character hexadecimal string)

**Generate MD5 Hash:**

```bash
# Linux/macOS
echo -n "password" | md5sum

# Or use online tools
```

**Example:**

```bash
PASSWORDS=md5:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

#### `sha512` - SHA512 Hash

**Description:**

- Uses SHA512 algorithm for hashing
- High security, recommended for production
- Password must use SHA512 hash value (128-character hexadecimal string)

**Generate SHA512 Hash:**

```bash
# Linux/macOS
echo -n "password" | shasum -a 512

# Or use online tools
```

**Example:**

```bash
PASSWORDS=sha512:5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8
```

### Password Verification Rules

1. **Password Normalization**: Spaces are removed and converted to uppercase before verification
2. **Multiple Password Support**: Multiple passwords can be configured, any password that passes verification is acceptable
3. **Algorithm Consistency**: All passwords must use the same algorithm

## Configuration Examples

### Basic Configuration (Password Authentication)

```bash
# Required configuration
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123

# Optional configuration
DEBUG=false
LANGUAGE=en
```

### Production Configuration (Password Authentication)

```bash
# Required configuration
AUTH_HOST=auth.example.com
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy

# Optional configuration
DEBUG=false
LANGUAGE=zh
LOGIN_PAGE_TITLE=My Auth Service
LOGIN_PAGE_FOOTER_TEXT=© 2024 My Company
USER_HEADER_NAME=X-Forwarded-User
COOKIE_DOMAIN=.example.com
```

### Warden + Herald OTP Authentication Configuration (Recommended for Production)

```bash
# Required configuration
AUTH_HOST=auth.example.com

# Warden configuration
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-warden-api-key
WARDEN_CACHE_TTL=300

# Herald configuration
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
HERALD_HMAC_SECRET=your-herald-hmac-secret

# Optional configuration
DEBUG=false
LANGUAGE=zh
LOGIN_PAGE_TITLE=My Auth Service
LOGIN_PAGE_FOOTER_TEXT=© 2024 My Company
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
    image: stargate:latest
    environment:
      # Required configuration
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
      
      # Optional configuration
      - DEBUG=false
      - LANGUAGE=zh
      - LOGIN_PAGE_TITLE=My Auth Service
      - LOGIN_PAGE_FOOTER_TEXT=© 2024 My Company
      - COOKIE_DOMAIN=.example.com
```

**Warden + Herald OTP Authentication Mode:**

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      # Required configuration
      - AUTH_HOST=auth.example.com
      
      # Warden configuration
      - WARDEN_ENABLED=true
      - WARDEN_URL=http://warden:8080
      - WARDEN_API_KEY=your-warden-api-key
      - WARDEN_CACHE_TTL=300
      
      # Herald configuration
      - HERALD_ENABLED=true
      - HERALD_URL=http://herald:8080
      - HERALD_HMAC_SECRET=your-herald-hmac-secret
      
      # Optional configuration
      - DEBUG=false
      - LANGUAGE=zh
      - LOGIN_PAGE_TITLE=My Auth Service
      - LOGIN_PAGE_FOOTER_TEXT=© 2024 My Company
      - COOKIE_DOMAIN=.example.com
    depends_on:
      - warden
      - herald

  warden:
    image: warden:latest
    environment:
      - WARDEN_DB_URL=postgres://user:pass@db:5432/warden
    # ... other configuration

  herald:
    image: herald:latest
    environment:
      - HERALD_REDIS_URL=redis://redis:6379/0
    # ... other configuration
```

### Local Development Configuration

```bash
# Required configuration
AUTH_HOST=localhost
PASSWORDS=plaintext:test123|admin456

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

- **Herald Integration**:
  - When `HERALD_ENABLED=true`, must set `HERALD_URL`
  - Must set either `HERALD_API_KEY` or `HERALD_HMAC_SECRET` (recommend HMAC for production)

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
   - Use `bcrypt` or `sha512` algorithms, avoid `plaintext`
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
   - Set `WARDEN_CACHE_TTL` to reduce requests to Warden
   - Adjust cache time according to actual needs
