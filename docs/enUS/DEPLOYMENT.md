# Deployment Guide

For a complete first deployment with mail OTP or mail OTP plus TOTP, see the [full-stack example](../../examples/full-stack/README.md). It separates local SMTP capture from the HTTPS production template.

For v1.3.5, read [authentication priority upgrade and rollback](#authentication-priority-upgrade-and-rollback-v135). Valid explicit credentials no longer depend on refreshing an unrelated existing cookie.

For v1.3.4, read [dependency patch upgrade and rollback](#dependency-patch-upgrade-and-rollback-v134). The update changes masking behavior and dependency versions, while preserving existing session and verification state.


Current deployment examples use v1.3.5. Before replacing a pre-v1.3 deployment, read the [revocable-session migration](#revocable-session-migration) and [browser-bound session exchange](#browser-bound-session-exchange) requirements. Coordinate all replicas; preserve required Redis state and account for the documented fresh-login boundary.

This document provides a detailed deployment guide for Stargate Forward Auth Service.

For v1.3.3, read [Herald credential selection](#herald-credential-selection-v133) before updating a client with both API-key and HMAC settings. First set and verify explicit key IDs on all existing callers, then upgrade Herald v1.4.3, followed by Stargate v1.3.3. This patch preserves Redis records and session deadlines.

For v1.3.2, see [OTP verification error handling](#otp-verification-error-handling-v132) for the 503/409 change and rollback behavior. Upgrade herald-totp v2.0.2, then Herald v1.4.2, before publishing the gateway update. Existing v1.3.1 records and session deadlines are preserved.

For v1.3.1, upgrade herald-totp to v2.0.1, then Herald to v1.4.1, before following [bound login challenge upgrade and rollback](#bound-login-challenge-upgrade-and-rollback). Keep existing v1.3.0 revocable-session records, tombstones, configured lifetime and namespace; this patch does not require a session-format migration or renew an existing deadline.

For v1.3.0, read the [session migration](#revocable-session-migration) and [release notes](../../CHANGELOG.md). Existing sessions require a new login; backend service upgrades and mixed-version SSO require coordination.

Upgrading from v0.12.0? Read the [v1.0.0 migration guide](MIGRATION_V1.md) before changing production traffic.

## Table of Contents

- [Deployment Methods](#deployment-methods)
- [Direct Binaries](#direct-binaries)
- [Docker Deployment](#docker-deployment)
- [Docker Compose Deployment](#docker-compose-deployment)
- [Traefik Integration](#traefik-integration)
- [Production Deployment](#production-deployment)
- [Monitoring and Maintenance](#monitoring-and-maintenance)
- [Troubleshooting](#troubleshooting)

## Deployment Methods

Stargate supports the following deployment methods:

1. **Docker Container** (Recommended) - Simplest and most common
2. **Docker Compose** - Suitable for local development and testing
3. **Kubernetes** - Suitable for large-scale production environments
4. **Direct Binary Execution** - Suitable for special scenarios

This document mainly introduces Docker and Docker Compose deployment methods.

## Direct Binaries

Release binaries target Linux, macOS and Windows on amd64/arm64. Container
images target Linux amd64/arm64. The binary does not embed HTML templates or
static assets: install the entire `src/internal/web/templates` directory from
the matching source tag, including its `assets` subdirectory.

Template and asset paths are relative to the process working directory. The
lookup order is `./internal/web/templates`, `./web/templates`, then
`./src/internal/web/templates`. Run from the source checkout or provide the
matching `web/templates` tree in the installation working directory. Published
container images already contain these files under `/app/web/templates`.
A successful `/healthz` response does not verify template availability; also
check the unauthenticated login page with
`curl -fsS -o /dev/null http://127.0.0.1:8080/_login`.

## Service Dependencies

Stargate can integrate with the following optional services:

### Warden Service

**Function:** User whitelist management and user information provision

**Deployment Requirements:**
- Uses local JSON data and/or a remote API; no SQL database is required by Warden.
- Provides HTTP API interface
- Supports API Key authentication

**Configuration:**
```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8081
WARDEN_API_KEY=your-api-key
```

### Herald Service

**Function:** OTP/verification code sending and verification

**Deployment Requirements:**
- Requires Redis (stores challenges and rate limit state)
- Provides HTTP API interface
- In production, Herald requires the selected HMAC or API-key request credentials; optional mTLS adds transport protection and does not replace them.

**Configuration:**
```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8082
HERALD_HMAC_SECRET=your-hmac-secret  # Recommended for production
```

### Inter-Service Communication Security

**Production Environment Requirements:**

1. **HMAC Signature Authentication** (Recommended):
   - Stargate ↔ Herald uses HMAC-SHA256 signature
   - Configure `HERALD_HMAC_SECRET`
   - Includes timestamp verification (prevents replay attacks)

2. **mTLS Authentication** (Optional, more secure):
   - Configure TLS client certificate
   - Set `HERALD_TLS_CLIENT_CERT_FILE` and `HERALD_TLS_CLIENT_KEY_FILE`
   - Configure CA certificate verification

3. **Network Isolation:**
   - Inter-service communication should be on internal network
   - Use firewall rules to restrict access
   - Avoid exposing services to public network

## Docker Deployment

### Build Image

#### Build from Source

From the project root directory:

```bash
docker build -f docker/Dockerfile -t stargate:latest .
```

#### Build Parameters

- **Base Image**: `golang:1.27.3-alpine3.24` (build stage)
- **Runtime Image**: `alpine:3.24` (runtime stage; includes CA certificates and BusyBox `wget` for HTTPS and health checks)
- **Working Directory**: `/app`
- **Exposed Port**: `8080`

### Run Container

#### Basic Run

```bash
docker run -d \
  --name stargate \
  -p 8080:8080 \
  -e AUTH_HOST=auth.example.com \
  -e 'PASSWORDS=plaintext:yourpassword' \
  stargate:latest
```

#### Full Configuration Run

```bash
docker run -d \
  --name stargate \
  -p 8080:8080 \
  -e AUTH_HOST=auth.example.com \
  -e 'PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy' \
  -e DEBUG=false \
  -e LANGUAGE=zh \
  -e 'LOGIN_PAGE_TITLE=My Auth Service' \
  -e 'LOGIN_PAGE_FOOTER_TEXT=© 2024 My Company' \
  -e COOKIE_DOMAIN=.example.com \
  --restart unless-stopped \
  stargate:latest
```

#### Parameter Description

- `-d`: Run in background
- `--name stargate`: Container name
- `-p 8080:8080`: Port mapping (host port:container port)
- `-e`: Environment variable
- `--restart unless-stopped`: Auto-restart policy

### View Logs

```bash
# View real-time logs
docker logs -f stargate

# View last 100 lines of logs
docker logs --tail 100 stargate
```

### Stop and Remove

```bash
# Stop container
docker stop stargate

# Remove container
docker rm stargate

```

## Docker Compose Deployment

### Basic Configuration

The repository root `docker-compose.yml` is the bundled stack: it starts Traefik, Stargate, and the protected demo on the Compose-managed `stargate-traefik` network (`172.30.0.0/24`). It does not require a pre-existing external network.

```bash
docker compose config >/dev/null
docker compose up -d
```

The validation commands discard stdout because resolved Compose configuration can
contain credentials. Earlier terminal or CI output is not rewritten. If actual
credentials were captured, restrict access, apply retention/cleanup to logs and
exported copies, and rotate the exposed credentials.

### Using an Existing External Traefik Network

Use the separate example below when Traefik already runs on a shared external Docker network. Its actual network name and inspected CIDR drive the Compose network, every `traefik.docker.network` label, and `TRUSTED_PROXIES`.

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:1.3.5
    environment:
      - AUTH_HOST=auth.test.localhost
      - PASSWORDS=plaintext:test1234|test1337
      - CALLBACK_ALLOWED_HOSTS=hello.test.localhost
      - SESSION_EXCHANGE_SECRET=local-development-session-secret-change-me
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
      - COOKIE_SECURE=false # Local HTTP only; omit for HTTPS.
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}
      - traefik.http.routers.auth.entrypoints=http
      - traefik.http.routers.auth.rule=Host(`auth.test.localhost`) || Path(`/_session_exchange`)
      - traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"

  hello:
    image: ghcr.io/soulteary/hello:2.3.0
    command: ["-listen", ":8080"]
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}
      - traefik.http.routers.hello.entrypoints=http
      - traefik.http.routers.hello.rule=Host(`hello.test.localhost`)
      - "traefik.http.routers.hello.middlewares=stargate"
      - traefik.http.services.hello.loadbalancer.server.port=8080

networks:
  traefik:
    name: ${TRAEFIK_NETWORK_NAME:-traefik}
    external: true
```

### Inspect or Create the External Network

If the network exists, the commands inspect it and reuse all allocated CIDRs; do not delete or recreate an existing production network. If it does not exist, they create it with the documented subnet.

```bash
export TRAEFIK_NETWORK_NAME="${TRAEFIK_NETWORK_NAME:-traefik}"

if docker network inspect "$TRAEFIK_NETWORK_NAME" >/dev/null 2>&1; then
  export TRAEFIK_NETWORK_CIDR="$(
    docker network inspect "$TRAEFIK_NETWORK_NAME" \
      --format '{{range .IPAM.Config}}{{println .Subnet}}{{end}}' |
      awk 'NF { cidr = cidr separator $0; separator = "," } END { print cidr }'
  )"
  if [ -z "$TRAEFIK_NETWORK_CIDR" ]; then
    echo "No subnet found for Docker network $TRAEFIK_NETWORK_NAME" >&2
    exit 1
  fi
else
  docker network create \
    --driver bridge \
    --subnet 172.30.0.0/24 \
    "$TRAEFIK_NETWORK_NAME"
  export TRAEFIK_NETWORK_CIDR=172.30.0.0/24
fi

docker compose config >/dev/null
docker compose up -d
```

### Stop Services

```bash
docker compose down
```

### View Logs

```bash
# View all service logs
docker compose logs -f

# View specific service logs
docker compose logs -f stargate
```

### Custom Configuration

Edit `docker-compose.yml` and modify environment variables:

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:1.3.5
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$...
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
      - DEBUG=false
      - LANGUAGE=zh
      - COOKIE_DOMAIN=.example.com
      - SESSION_EXCHANGE_SECRET=replace-with-at-least-32-random-characters
```

## Traefik Integration

### Basic Configuration

Stargate is designed to integrate with Traefik, providing authentication through Forward Auth middleware.

#### 1. Configure Stargate Service

Configure Stargate in `docker-compose.yml`:

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:1.3.5
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$$2a$$10$$...
      - TRUSTED_PROXIES=${TRAEFIK_NETWORK_CIDR:?set TRAEFIK_NETWORK_CIDR from docker network inspect}
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}"
      - "traefik.http.routers.auth.entrypoints=http,https"
      - "traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)"
      - "traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth"
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"
```

#### 2. Configure Protected Services

Apply Stargate middleware to services that require authentication:

```yaml
services:
  your-app:
    image: your-app:latest
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}"
      - "traefik.http.routers.your-app.entrypoints=http,https"
      - "traefik.http.routers.your-app.rule=Host(`app.example.com`)"
      - "traefik.http.routers.your-app.middlewares=stargate"  # Apply authentication middleware
```

### HTTPS Configuration

#### Using Let's Encrypt

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls.certresolver=letsencrypt"
      - "traefik.http.routers.auth.tls=true"
```

#### Using Custom Certificates

```yaml
services:
  stargate:
    labels:
      - "traefik.http.routers.auth.entrypoints=https"
      - "traefik.http.routers.auth.tls=true"
      - "traefik.http.routers.auth.tls.certfile=/path/to/cert.pem"
      - "traefik.http.routers.auth.tls.keyfile=/path/to/key.pem"
```

### Cross-Domain Session Sharing

If you need to share sessions across subdomains:

1. Set the `COOKIE_DOMAIN` environment variable:

```yaml
services:
  stargate:
    environment:
      - COOKIE_DOMAIN=.example.com
      - SESSION_EXCHANGE_SECRET=replace-with-at-least-32-random-characters
```

2. Ensure all related domains are routed to Stargate via Traefik

3. Login flow:
   - User logs in at `auth.example.com`
   - Redirects to `app.example.com/_session_exchange?ticket=<opaque_ticket>`
   - Session cookie is set to the `.example.com` domain
   - All `*.example.com` subdomains can use this session

## Production Deployment

### Security Recommendations

#### 1. Use Strong Password Algorithms

**Not Recommended:**

```bash
PASSWORDS='plaintext:yourpassword'
```

**Recommended:**

```bash
PASSWORDS='bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
```

#### 2. Enable HTTPS

- Configure HTTPS via Traefik
- Use Let's Encrypt automatic certificates
- Force HTTPS redirect

#### 3. Disable Debug Mode

```bash
DEBUG=false
```

#### 4. Set Resource Limits

```yaml
services:
  stargate:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 128M
        reservations:
          cpus: '0.25'
          memory: 64M
```

#### 5. Use Health Checks

```yaml
services:
  stargate:
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "-", "http://127.0.0.1:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### High Availability Deployment

#### 1. Multi-Instance Deployment

```yaml
services:
  stargate:
    deploy:
      replicas: 3
```

**Session storage scope:**

- **One process / one replica, including cross-domain callbacks:** in-memory storage (`SESSION_STORAGE_ENABLED=false`) is supported when ticket issuance, ticket redemption, and subsequent session use all reach the same live Stargate process. Session records and consumed-ticket hashes exist only in that process. A restart loses both stores, invalidates active sessions, and means single-use ticket state is not retained across the restart.
- **Multiple processes or replicas:** Redis is required whenever a session or exchange ticket may be handled by different processes. Set `SESSION_STORAGE_ENABLED=true`, configure the same `SESSION_STORAGE_REDIS_*` namespace, and use the same `SESSION_EXCHANGE_SECRET` on every replica. Sticky sessions are only a routing optimization; they do not provide shared state or cross-process replay protection.
- **Rate limits across replicas:** Rate-limit counters are shared through the same Redis connection. Without `SESSION_STORAGE_ENABLED=true`, each replica enforces its own quota and a deployment with N replicas allows up to N times `RATE_LIMIT_LOGIN_MAX` and `RATE_LIMIT_VERIFICATION_MAX`. Quotas are keyed by client address, so `TRUSTED_PROXIES` must list the reverse proxy; otherwise every client is attributed to the proxy and shares one quota.
- **Rolling upgrade:** Redis is required for an uninterrupted rolling replacement because old and new processes overlap. Without Redis, use a stop-then-start upgrade and accept session invalidation and fresh login. Never mix v0.12.0 and v1.0.0 in one serving pool.
- **State across a process restart:** Redis is required whenever sessions or consumed-ticket replay state must survive a process replacement or restart.

#### 2. Load Balancing

Add a load balancer before Traefik:

```yaml
services:
  traefik:
    labels:
      - "traefik.http.services.stargate.loadbalancer.server.port=8080"
```

### Monitoring Configuration

#### 1. Log Collection

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### 2. Health Check Endpoint

Use `/healthz` for process liveness. Use `/readyz` separately when traffic should only be sent after Redis, Warden, and Herald dependencies are ready. The legacy `/health` endpoint remains a deprecated readiness alias.

```bash
# Health check script
#!/bin/bash
if curl -f http://localhost:8080/healthz > /dev/null 2>&1; then
  exit 0
else
  exit 1
fi
```

#### 3. Prometheus Integration

Stargate exposes Prometheus metrics at `GET /metrics`. Configure your Prometheus server to scrape this endpoint. The metrics endpoint is excluded from request logging and tracing by default.

## Monitoring and Maintenance

### Log Management

#### View Logs

```bash
# Docker
docker logs -f stargate

# Docker Compose
docker compose logs -f stargate
```

#### Log Levels

- `DEBUG=true`: Detailed debug information
- `DEBUG=false`: Only critical information

#### Log Rotation

Configure Docker log driver:

```yaml
services:
  stargate:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### Performance Monitoring

#### Resource Usage

```bash
# View container resource usage
docker stats stargate
```

#### Response Time

Monitor response time using the health check endpoint:

```bash
time curl http://auth.example.com/healthz
```

### Regular Maintenance

1. **Update Images**: Regularly pull the latest images
2. **Check Logs**: Regularly check error logs
3. **Monitor Resources**: Monitor CPU and memory usage
4. **Backup Configuration**: Backup environment variable configuration

## Troubleshooting

### Common Issues

#### 1. Service Fails to Start

**Problem:** Container exits immediately after starting

**Troubleshooting Steps:**

```bash
# View container logs
docker logs stargate

# Check configuration
docker inspect stargate | grep -A 20 Env
```

**Common Causes:**

- Missing required configuration (`AUTH_HOST`, `PASSWORDS`)
- Incorrect password configuration format
- Port is occupied

#### 2. Authentication Fails

**Problem:** Users cannot log in

**Troubleshooting Steps:**

1. Check if password configuration is correct
2. Check if password algorithm matches
3. View service logs: `docker logs stargate`

**Common Causes:**

- Incorrect password configuration
- Password algorithm mismatch (e.g., configured bcrypt but used plain text password)
- Incorrect cookie domain configuration

#### 3. Cross-Domain Sessions Not Working

**Problem:** Cannot share sessions across subdomains

**Troubleshooting Steps:**

1. Check `COOKIE_DOMAIN` configuration
2. Confirm cookie domain format is correct (`.example.com`)
3. Check browser cookie settings

**Solution:**

```bash
# Ensure COOKIE_DOMAIN is set
COOKIE_DOMAIN=.example.com
```

#### 4. Traefik Integration Issues

**Problem:** Traefik cannot correctly forward authentication requests

**Troubleshooting Steps:**

1. Check Traefik label configuration
2. Confirm network configuration is correct
3. Check Forward Auth middleware address

**Solution:**

```yaml
# Ensure middleware address is correct
- "traefik.http.middlewares.stargate.forwardauth.address=http://stargate:8080/_auth"
- "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User,X-Auth-User,X-Auth-Email,X-Auth-Name,X-Auth-Scopes,X-Auth-Role,X-Auth-AMR"
```

### Debugging Tips

#### 1. Enable Debug Mode

```bash
DEBUG=true
```

#### 2. Check Network Connection

```bash
# Test from inside container
docker exec stargate wget -q -O - http://127.0.0.1:8080/healthz
```

#### 3. View Traefik Logs

```bash
docker logs traefik
```

#### 4. Test API Endpoints

```bash
# Test health check
curl http://auth.example.com/healthz

# Test authentication (using Header)
# Server prerequisite: PASSWORD_HEADER_AUTH_ENABLED=true
curl -H "Stargate-Password: yourpassword" http://auth.example.com/_auth

# Test authentication (using Cookie)
curl -H "Cookie: stargate_session_id=<session_id>" http://auth.example.com/_auth
```

### Getting Help

If you encounter problems:

1. View logs: `docker logs stargate`
2. Check configuration: Confirm all environment variables are correct
3. View documentation: [API Documentation](API.md), [Configuration Reference](CONFIG.md)
4. Submit Issue: Submit a problem report in the project repository

## Upgrade Guide

### Upgrade Steps

1. **Create separate rollback and v1 environment files:** Keep the existing `stargate.env` unchanged. The commands below refuse to overwrite either output file and remove the retired Warden OTP settings only from the new v1 file.

```bash
set -eu
old_env=./stargate.env
rollback_env=./stargate-v0.12.0.env
v1_env=./stargate-v1.env
old_container=${STARGATE_OLD_CONTAINER:-stargate}

test ! -e "$rollback_env"
test ! -e "$v1_env"
umask 077

# If the file is missing, export the existing container environment securely.
if [ ! -e "$old_env" ]; then
  export_tmp=$(mktemp "${old_env}.tmp.XXXXXX")
  trap 'rm -f "$export_tmp"' 0 1 2 15
  docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$old_container" > "$export_tmp"
  ln "$export_tmp" "$old_env"
  rm -f "$export_tmp"
  trap - 0 1 2 15
fi

test -f "$old_env"
chmod 600 "$old_env"
(set -C; cat "$old_env" > "$rollback_env")
(set -C; awk '
  /^[[:space:]]*WARDEN_OTP_(ENABLED|SECRET_KEY)[[:space:]]*(=|$)/ { next }
  /^[[:space:]]*PORT[[:space:]]*(=|$)/ { print "PORT=8080"; next }
  { print }
' "$old_env" > "$v1_env")
```

Herald-backed TOTP requires Stargate to resolve authenticated users through Warden, so also set `WARDEN_ENABLED=true` and `WARDEN_URL`.

v1 rejects `WARDEN_OTP_ENABLED` and `WARDEN_OTP_SECRET_KEY` whenever either name is present, including empty or `false` values. Do not copy them back. If TOTP is needed, configure Stargate in `stargate-v1.env` with `HERALD_ENABLED=true`, `HERALD_TOTP_ENABLED=true`, `HERALD_URL`, and one supported Herald service-auth method. Configure Herald's TOTP proxy and herald-totp's independent `HERALD_TOTP_ENCRYPTION_KEY`; do not automatically reuse the retired Warden secret. See [Configuration](CONFIG.md).

2. **Keep the previous container and its original environment for rollback:**

```bash
docker stop stargate
docker rename stargate stargate-previous
```

3. **Pull New Image:**

```bash
docker pull ghcr.io/soulteary/stargate:v1.0.0
```

4. **Start New Container:**

```bash
docker run -d \
  --name stargate \
  --env-file ./stargate-v1.env \
  -p 8080:8080 \
  --restart unless-stopped \
  ghcr.io/soulteary/stargate:v1.0.0
```

5. **Verify Service:**

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
```

### Rollback

If problems occur after upgrade:

```bash
# Remove the new container and restore the unchanged previous container.
# Keep stargate-v0.12.0.env as the rollback configuration record.
docker rm -f stargate
docker rename stargate-previous stargate
docker start stargate
```

## Revocable session migration

This version stores sessions under `<SESSION_STORAGE_REDIS_KEY_PREFIX>revocable:v1:`
(the base prefix is normalized to end in `:`). Existing sessions and in-flight SSO
tickets from the ordinary namespace require a new login. Deploy all gateway
instances with this version; do not configure an older instance to write into the
new namespace. Leave ordinary records to expire instead of copying them.

Session identifiers now carry an absolute 24-hour deadline. Logout atomically
marks the identifier revoked until that deadline; an older request cannot restore
it by saving its authenticated snapshot afterward. Session writes and revocation
markers cannot extend the identifier deadline. Keep Redis and application clocks
synchronized, and allow the dedicated session keyspace's `EVAL`, `TIME`, `GET`,
`SET`, and `DEL` operations. This adds an absolute limit to the former idle timeout.

Factor enrollment, confirmation, revocation, and step-up now recheck Warden
identity before proceeding, even when background authorization refresh is
configured off. A directory/source failure returns 503 without performing the
factor operation. Confirmed disabled, deleted, or reassigned identities fail with
401. Password-only sessions continue to use their configured local credentials.

Rate-limit buckets use the registered method and route, so case variants and
trailing slashes share the same quota. Distinct registered endpoints retain
independent quotas.

Atomic revocation applies to the same identifier. Never rotate an old authenticated
snapshot into a new ID as a substitute for reauthentication: obtain fresh
credential proof first, then clear/reset identity data before issuing a new
authenticated session. Anonymous logout only clears the browser cookie and does
not allocate a revocation marker for a never-stored identifier.

### Browser-bound session exchange

Cross-domain login now starts a handshake on the destination's `/_session_exchange` endpoint. The destination creates a ten-minute host-only, HttpOnly, SameSite=Lax nonce cookie. Its sealed request binds the destination origin, the configured `SESSION_EXCHANGE_REDIRECT_PATH`, and the nonce hash. The auth host issues a two-minute single-use ticket bound to that request; the destination checks its own cookie before importing the session. Direct login with a callback automatically visits the destination to establish this binding. Existing proxy routing for `/_session_exchange` remains required on each allowed destination.

Production deployments must use HTTPS with `COOKIE_SECURE=true`. Nonce cookies use the `__Host-` prefix, have no Domain attribute, and cannot be shared across destinations. HTTP development uses an unprefixed cookie with `COOKIE_SECURE=false`; use this only on a trusted local development network. Keep `SESSION_EXCHANGE_SECRET` equal across participating replicas and configure a shared Redis replay store for multiple replicas.

Previously issued tickets are rejected. Complete an in-progress login again after upgrading all replicas; do not mix old and new exchange handlers during a rolling upgrade. Caller-provided return URLs and nonce values are not accepted. The final landing remains the configured same-origin path. Parallel exchanges to the same destination may replace the pending nonce and require restarting one login.

### Herald credential precedence

When `HERALD_HMAC_SECRET` is nonempty, Stargate passes only HMAC credentials to the Herald SDK, even if `HERALD_API_KEY` is also set. Failed HMAC authentication does not fall back to the API key. For a Herald server using `REQUEST_AUTH_MODE=api_key`, leave `HERALD_HMAC_SECRET` empty and configure `HERALD_API_KEY`. Before upgrading a dual-configured client, ensure the server accepts its HMAC credentials or explicitly select API-key-only mode.

### Warden credential precedence

When both `WARDEN_API_KEY` and `WARDEN_HMAC_KEY_ID` / `WARDEN_HMAC_SECRET` are configured, the updated SDK sends only HMAC credentials. It does not retry authentication with the API key. Configure matching HMAC credentials on Warden before upgrading this combination. For an API-key-only Warden deployment, remove both client HMAC settings and keep `WARDEN_API_KEY`. API-key-only authentication and mTLS remain supported; this change does not alter the server's authentication policy.


## Bound login challenge upgrade and rollback

OTP login now requires the gateway's trusted challenge context to match the
current Warden user, selected channel and directory contact. The gateway sends
Herald a `/v2/otp/verifications` request bound to that subject, the `login`
purpose and channel, and checks those fields and the challenge ID in a successful
response. There is no fallback to unbound `/v1/otp/verifications`. Configure a
Herald server supporting the bound route before changing gateway traffic.

Preserve the existing challenge-context Redis namespace and share it across all
gateway replicas. Set `SESSION_STORAGE_ENABLED=true` and configure the session
Redis connection on every replica, using the same Redis database and
`SESSION_STORAGE_REDIS_KEY_PREFIX`. The server installs Redis-backed challenge
context only when that Redis client is initialized; without it, context remains
process-local. Process-local context is suitable only for a single process:
a restart or another replica cannot validate a challenge whose binding it never
stored. Existing pending codes need no format conversion when their binding
survives and the directory contact is unchanged. Missing bindings or changes to
the user's current contact require a new login challenge. Email local parts and
plus tags retain their identity; domain case and supported phone formatting are
normalized without treating different contacts as equivalent.

If Herald delivered a code but saving its binding fails, creation returns
`503/binding_unavailable` without a challenge ID. Preserve and retry the original
idempotency key after restoring the context backend; do not create a new key to
bypass uncertain delivery. A conflicting stored binding also fails closed.

Pause new code creation and drain in-flight login attempts while replacing all
gateway instances. Older gateways retain unbound verification, so mixing them
keeps the older acceptance path available. A rollback reintroduces that behavior;
do not use rollback as a way to accept a rejected or unrelated-purpose code.
Do not delete provider idempotency, nonce or audit state during either direction,
and follow each backend's separate OTP/TOTP storage migration requirements.

TOTP enrollment-confirmation quotas now propagate through Herald and Stargate as
429 with fixed error fields. The local enrollment binding remains available for
a later retry. Backend SDK pins alone do not upgrade the independently deployed
TOTP service. Subject quotas are shared with enrollment start, so provision room
for both steps and failed attempts; a subject limit of one cannot complete the
flow before its hourly budget resets.

The session-kit clock-bound fix changes only the future identifier plausibility
check. It neither extends immutable deadlines nor permits saving a revoked
identifier. Keep clocks synchronized. Older adapters can reject a session from a
slightly ahead replica after rollback; leave the revocable namespace and absolute
deadline enforcement intact rather than copying records into an older namespace.

## OTP verification error handling (v1.3.2)

OTP login reports Herald server errors as 503 and its published
`409/lock_contention` response as 409. Invalid credentials and mismatched
challenge bindings retain their rejection behavior. This is an existing
error-classification correction; proof validation is unchanged. The current SDK does not expose upstream retry headers;
Stargate does not synthesize `Retry-After` or retry verification automatically.

Keep the same Redis dataset and namespaces, pending gateway bindings, session
records and revocation markers when upgrading from v1.3.1. No record conversion,
new login, or session-deadline extension is required. Rolling replicas use the
same records, though older replicas still classify these failures as 401.
Clients must accept that status difference until the rollout finishes.

For a provider failure known to occur before consumption, retrying after
recovery can use the existing challenge. A failed response alone does not prove
that verification was uncommitted: preserve state and reconcile uncertain
results before restarting a flow. Follow the separate TOTP/Herald upgrade guides
and retain provider audit, replay and idempotency records.

A gateway rollback restores the earlier 401 classification. Keep the same
records and deadlines, and do not independently downgrade provider services.
Upgrades from releases before v1.3.1 still require their documented session and
bound-challenge migrations.

## Herald credential selection (v1.3.3)

Stargate selects HMAC when `HERALD_HMAC_SECRET` is nonempty and sends no
`HERALD_API_KEY` on signed requests. Before upgrading any Herald verifier that uses a key map, configure and verify
its existing key ID through `HERALD_HMAC_KEY_ID` on every current Stargate caller.
Then upgrade Herald to v1.4.3, followed by the Stargate v1.3.3 binary. For `REQUEST_AUTH_MODE=api_key`, leave
the client HMAC secret empty and keep its API key. A failed HMAC request is not
retried with the API key. Deploy and verify Herald v1.4.3 before Stargate v1.3.3;
updating the Go SDK dependency does not update the separate service.

Keep shared Redis data, pending OTP bindings, replay markers and existing session
namespaces and deadlines. No record-format migration, forced reauthentication or
session renewal is required from v1.3.2. No OTP verification retry is automatic;
503 and the specific lock-contention 409 retain their v1.3.2 meanings.

Before rolling back, check the chosen authentication mode and active keys. An old
gateway can send both configured credentials again; rollback does not restore a
retired key or remove secrets from configuration. An old Herald service restores
its old key-map selection behavior and may not support the new proxy key-ID
configuration. Follow its rollback guidance, preserve state and verify the same
login and TOTP flows before forwarding traffic. Retain the v1.3.0/v1.3.1 migration
requirements for older deployments.

## Dependency patch upgrade and rollback (v1.3.4)

Deploy the verified Warden v1.6.2 and Herald v1.4.4 service releases, including
Herald's documented herald-totp v2.0.3 rollout, before upgrading Stargate to
v1.3.4. Updating the gateway SDK modules does not deploy these services. Existing
HMAC key IDs, credential precedence and no-fallback behavior remain unchanged;
follow the v1.3.3 staged key-ID guidance when upgrading older deployments.

The secure-kit/v2 v2.1.4 update preserves Unicode code points in masked log text
and handles oversized masking bounds safely. Non-ASCII masked output can change;
existing ASCII output is preserved. Review log consumers that depend on exact
masked strings. This does not change the login proof or session record format.

Keep Redis records, pending challenge bindings, replay and revocation markers,
session namespaces and absolute deadlines across mixed-version rollout. The
patch does not renew sessions or automatically retry OTP verification. Existing
503 and lock-contention 409 responses retain their meanings.

Before rollback to v1.3.3, verify the active credentials and provider versions.
The verified old binary or image restores its compiled dependency and masking
behavior. For a source or SDK rollback, restore the pre-upgrade application
`go.mod` and `go.sum`, check the effective `go list -m all` versions, then rebuild.
A parent-module downgrade alone may retain newer direct pins or transitive
minimums under Go minimal version selection. Rollback does not rewrite existing
logs, restore retired keys or undo committed OTP consumption.
Keep shared state and reconcile uncertain provider results. Do not independently
downgrade provider services or discard replay records.

## Authentication priority upgrade and rollback (v1.3.5)

Upgrade the gateway binary or image from v1.3.4 to v1.3.5 with the same
configuration and service versions. No dependency, Redis schema, session namespace
or absolute-deadline change is required. Preserve pending challenge bindings,
revocation and replay records; do not clear Redis during rollout or rollback.

A valid password header or trusted identity header is evaluated independently of
an unrelated old Warden session. If session authentication is selected, directory
refresh and source freshness still fail closed. An invalid password header can
still fall back to an authenticated session, which must pass those checks. This
existing password fallback is separate from service HMAC authentication, which
does not retry with an API key. Stateless credentials cannot borrow a session's
step-up verification marker. No automatic OTP retry or session renewal is added.

Rollback to the verified v1.3.4 binary or image preserves existing state but
restores the previous ordering: refreshing an old cookie can reject a request
with otherwise valid explicit credentials. For source builds, check out the v1.3.4 source
(`19d2997ff0bb6c61ff8a302cf4b99287c8b27008`) and rebuild; restoring only
`go.mod` and `go.sum` cannot undo this source change because dependencies are
unchanged. Verify selected module versions before rebuilding. The v1.3.4 dependency and older migration guidance above still applies.
