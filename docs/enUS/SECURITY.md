# Security Documentation

> 🌐 **Language / 语言**: [English](SECURITY.md) | [中文](../zhCN/SECURITY.md) | [Français](../frFR/SECURITY.md) | [Italiano](../itIT/SECURITY.md) | [日本語](../jaJP/SECURITY.md) | [Deutsch](../deDE/SECURITY.md) | [한국어](../koKR/SECURITY.md)

This document explains Stargate's security features, security configuration, and best practices.

## Implemented Security Features

1. **Forward Auth Protection**: Centralized authentication layer for protecting backend services
2. **Multiple Password Algorithms**: Support for bcrypt, SHA512, MD5, and plaintext (development only)
3. **Secure Session Management**: Cookie-based sessions with configurable domain and expiration
4. **Service Integration Security**: Secure communication with Warden and Herald services using mTLS or HMAC
5. **Session Sharing Security**: Secure cross-domain session exchange mechanism
6. **Input Validation**: Strict validation of all input parameters
7. **Error Handling**: Production mode hides detailed error information
8. **Security Response Headers**: Automatically adds security-related HTTP response headers
9. **HTTPS Enforcement**: Production environments should use HTTPS
10. **OTP Integration**: Secure integration with Herald for OTP/verification code authentication

## Security Best Practices

### 1. Production Environment Configuration

**Required Configuration**:
- Must set strong passwords using secure algorithms (bcrypt or SHA512)
- Set `MODE=production` to enable production mode
- Configure `COOKIE_DOMAIN` for proper session management
- Use HTTPS via reverse proxy (Traefik, Nginx, etc.)
- Configure secure session cookie settings

**Configuration Example**:
```bash
export MODE=production
export AUTH_HOST=auth.example.com
export PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
export COOKIE_DOMAIN=.example.com
export COOKIE_SECURE=true
export COOKIE_SAME_SITE=Strict
```

### 2. Password Security

**Recommended Practices**:
- ✅ Use strong password hashing algorithms (bcrypt or SHA512)
- ✅ Store password hashes in environment variables
- ✅ Use different passwords for different environments
- ✅ Regularly rotate passwords

**Not Recommended**:
- ❌ Use plaintext passwords in production
- ❌ Hardcode passwords in configuration files
- ❌ Use weak password algorithms (MD5) in production
- ❌ Share passwords across environments

**Password Algorithm Comparison**:
- `bcrypt`: Recommended for production (slow, secure)
- `sha512`: Good for production (fast, secure)
- `md5`: Not recommended for production (fast, less secure)
- `plaintext`: Development only (no security)

### 3. Session Security

**Session Configuration**:
- **Cookie Domain**: Set `COOKIE_DOMAIN` to share sessions across subdomains
- **Secure Flag**: Set `COOKIE_SECURE=true` in production (requires HTTPS)
- **SameSite**: Set `COOKIE_SAME_SITE=Strict` to prevent CSRF attacks
- **HttpOnly**: Cookies are automatically HttpOnly to prevent XSS attacks
- **Expiration**: Configure session expiration via `SESSION_TTL`

**Configuration Example**:
```bash
export COOKIE_DOMAIN=.example.com
export COOKIE_SECURE=true
export COOKIE_SAME_SITE=Strict
export SESSION_TTL=86400  # 24 hours
```

### 4. Network Security

**Required Configuration**:
- Production environments must use HTTPS
- Configure reverse proxy (Traefik, Nginx) to handle SSL/TLS
- Use firewall rules to restrict access
- Regularly update dependencies to fix known vulnerabilities

**Recommended Configuration**:
- Use Traefik with Let's Encrypt for automatic SSL certificates
- Configure `TRUSTED_PROXY_IPS` if behind a reverse proxy
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
export REDIS_ENABLED=true
export REDIS_ADDR=redis:6379
export REDIS_PASSWORD=your-redis-password
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

## Error Handling

### Production Mode

In production mode (`MODE=production` or `MODE=prod`):

- Hide detailed error information to prevent information leakage
- Return generic error messages
- Detailed error information is only recorded in logs
- Redirect to login page on authentication failure

### Development Mode

In development mode:

- Display detailed error information for debugging
- Include stack trace information
- More verbose logging

## Security Response Headers

Stargate automatically adds the following security-related HTTP response headers:

- `X-Content-Type-Options: nosniff` - Prevents MIME type sniffing
- `X-Frame-Options: DENY` - Prevents clickjacking
- `X-XSS-Protection: 1; mode=block` - XSS protection

## Cross-Domain Session Sharing

Stargate supports secure cross-domain session sharing:

- **Session Exchange Endpoint**: `GET /_session_exchange`
- **Secure Token**: Uses secure token for session exchange
- **Domain Validation**: Validates target domain before sharing session
- **Expiration**: Exchange tokens expire after short TTL

**Security Notes**:
- Only share sessions between trusted domains
- Use HTTPS for session exchange
- Monitor session exchange attempts

## OTP/Verification Code Security

When using Herald integration for OTP authentication:

- **Challenge-Based**: Uses challenge-verify model
- **Secure Communication**: Uses mTLS or HMAC for Herald communication
- **Rate Limiting**: Herald handles rate limiting
- **Audit Logging**: Herald maintains audit logs

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
