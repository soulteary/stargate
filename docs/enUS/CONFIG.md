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
| **Possible Values** | `en` (English), `zh` (Chinese) |

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

Service listening port (local development only).

| Attribute | Value |
|-----------|-------|
| **Type** | String |
| **Required** | No |
| **Default** | `80` |

**Description:**

- Only for local development environment
- Usually not needed in Docker containers (uses default port 80)
- Format: port number (e.g., `8080`) or `:port` (e.g., `:8080`)

**Example:**

```bash
PORT=8080
```

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

### Basic Configuration

```bash
# Required configuration
AUTH_HOST=auth.example.com
PASSWORDS=plaintext:test123

# Optional configuration
DEBUG=false
LANGUAGE=en
```

### Production Configuration

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

### Docker Compose Configuration

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

## Configuration Best Practices

1. **Production Security**:
   - Use `bcrypt` or `sha512` algorithms, avoid `plaintext`
   - Set `DEBUG=false`
   - Use strong passwords

2. **Cross-Domain Sessions**:
   - If you need to share sessions across subdomains, set `COOKIE_DOMAIN`
   - Format: `.example.com` (note the leading dot)

3. **Multi-Language Support**:
   - Set `LANGUAGE` according to user base
   - Supports `en` and `zh`

4. **Custom Interface**:
   - Use `LOGIN_PAGE_TITLE` and `LOGIN_PAGE_FOOTER_TEXT` to customize the login page

5. **Monitoring and Debugging**:
   - Set `DEBUG=true` in development environment for detailed logs
   - Set `DEBUG=false` in production environment to reduce log output
