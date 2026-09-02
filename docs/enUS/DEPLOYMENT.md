# Deployment Guide

This document provides a detailed deployment guide for Stargate Forward Auth Service.

Upgrading from v0.12.0? Read the [v1.0.0 migration guide](MIGRATION_V1.md) before changing production traffic.

## Table of Contents

- [Deployment Methods](#deployment-methods)
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

## Service Dependencies

Stargate can integrate with the following optional services:

### Warden Service

**Function:** User whitelist management and user information provision

**Deployment Requirements:**
- Requires database (PostgreSQL/MySQL/SQLite)
- Provides HTTP API interface
- Supports API Key authentication

**Configuration:**
```bash
WARDEN_ENABLED=true
WARDEN_URL=http://warden:8080
WARDEN_API_KEY=your-api-key
```

### Herald Service

**Function:** OTP/verification code sending and verification

**Deployment Requirements:**
- Requires Redis (stores challenges and rate limit state)
- Provides HTTP API interface
- Supports HMAC signature or mTLS authentication (recommended for production)

**Configuration:**
```bash
HERALD_ENABLED=true
HERALD_URL=http://herald:8080
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

- **Base Image**: `golang:1.27.0-alpine3.24` (build stage)
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
docker compose config
docker compose up -d
```

### Using an Existing External Traefik Network

Use the separate example below when Traefik already runs on a shared external Docker network. Its actual network name and inspected CIDR drive the Compose network, every `traefik.docker.network` label, and `TRUSTED_PROXIES`.

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    environment:
      - AUTH_HOST=auth.test.localhost
      - PASSWORDS=plaintext:test1234|test1337
      - CALLBACK_ALLOWED_HOSTS=whoami.test.localhost
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

  whoami:
    image: traefik/whoami
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=${TRAEFIK_NETWORK_NAME:-traefik}
      - traefik.http.routers.whoami.entrypoints=http
      - traefik.http.routers.whoami.rule=Host(`whoami.test.localhost`)
      - "traefik.http.routers.whoami.middlewares=stargate"

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

docker compose config
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
    image: ghcr.io/soulteary/stargate:v1.0.0
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
    image: ghcr.io/soulteary/stargate:v1.0.0
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
