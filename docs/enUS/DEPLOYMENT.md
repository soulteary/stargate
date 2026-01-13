# Deployment Guide

This document provides a detailed deployment guide for Stargate Forward Auth Service.

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

## Docker Deployment

### Build Image

#### Build from Source

```bash
cd codes
docker build -t stargate:latest .
```

#### Build Parameters

- **Base Image**: `golang:1.25-alpine` (build stage)
- **Runtime Image**: `scratch` (minimal image)
- **Working Directory**: `/app`
- **Exposed Port**: `80`

### Run Container

#### Basic Run

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=plaintext:yourpassword \
  stargate:latest
```

#### Full Configuration Run

```bash
docker run -d \
  --name stargate \
  -p 80:80 \
  -e AUTH_HOST=auth.example.com \
  -e PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy \
  -e DEBUG=false \
  -e LANGUAGE=zh \
  -e LOGIN_PAGE_TITLE=My Auth Service \
  -e LOGIN_PAGE_FOOTER_TEXT=© 2024 My Company \
  -e COOKIE_DOMAIN=.example.com \
  --restart unless-stopped \
  stargate:latest
```

#### Parameter Description

- `-d`: Run in background
- `--name stargate`: Container name
- `-p 80:80`: Port mapping (host port:container port)
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

# Stop and remove
docker rm -f stargate
```

## Docker Compose Deployment

### Basic Configuration

The project provides a `docker-compose.yml` example file:

```yaml
services:
  stargate:
    image: stargate
    environment:
      - AUTH_HOST=auth.test.localhost
      - PASSWORDS=plaintext:test1234|test1337
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=proxy
      - traefik.http.routers.auth.entrypoints=http
      - traefik.http.routers.auth.rule=Host(`auth.test.localhost`) || Path(`/_session_exchange`)
      - traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth

  whoami:
    image: traefik/whoami
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.docker.network=proxy
      - traefik.http.routers.whoami.entrypoints=http
      - traefik.http.routers.whoami.rule=Host(`whoami.test.localhost`)
      - "traefik.http.routers.whoami.middlewares=stargate"

networks:
  traefik:
    external: true
```

### Start Services

```bash
cd codes
docker-compose up -d
```

### Stop Services

```bash
docker-compose down
```

### View Logs

```bash
# View all service logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f stargate
```

### Custom Configuration

Edit `docker-compose.yml` and modify environment variables:

```yaml
services:
  stargate:
    image: stargate
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$...
      - DEBUG=false
      - LANGUAGE=zh
      - COOKIE_DOMAIN=.example.com
```

## Traefik Integration

### Basic Configuration

Stargate is designed to integrate with Traefik, providing authentication through Forward Auth middleware.

#### 1. Configure Stargate Service

Configure Stargate in `docker-compose.yml`:

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      - AUTH_HOST=auth.example.com
      - PASSWORDS=bcrypt:$2a$10$...
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.docker.network=traefik"
      - "traefik.http.routers.auth.entrypoints=http,https"
      - "traefik.http.routers.auth.rule=Host(`auth.example.com`) || Path(`/_session_exchange`)"
      - "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"
      - "traefik.http.middlewares.stargate.forwardauth.authResponseHeaders=X-Forwarded-User"
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
      - "traefik.docker.network=traefik"
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
```

2. Ensure all related domains are routed to Stargate via Traefik

3. Login flow:
   - User logs in at `auth.example.com`
   - Redirects to `app.example.com/_session_exchange?id=<session_id>`
   - Session cookie is set to the `.example.com` domain
   - All `*.example.com` subdomains can use this session

## Production Deployment

### Security Recommendations

#### 1. Use Strong Password Algorithms

**Not Recommended:**

```bash
PASSWORDS=plaintext:yourpassword
```

**Recommended:**

```bash
PASSWORDS=bcrypt:$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
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
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost/health"]
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

**Note:** Stargate uses in-memory session storage, sessions are not shared between instances. If multi-instance deployment is needed, it is recommended to:

- Use load balancer session persistence (Sticky Session)
- Or wait for external session storage (Redis) support

#### 2. Load Balancing

Add a load balancer before Traefik:

```yaml
services:
  traefik:
    labels:
      - "traefik.http.services.stargate.loadbalancer.server.port=80"
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

Use the `/health` endpoint for monitoring:

```bash
# Health check script
#!/bin/bash
if curl -f http://localhost/health > /dev/null 2>&1; then
  exit 0
else
  exit 1
fi
```

#### 3. Prometheus Integration

(To be implemented) Future versions will support Prometheus metrics export.

## Monitoring and Maintenance

### Log Management

#### View Logs

```bash
# Docker
docker logs -f stargate

# Docker Compose
docker-compose logs -f stargate
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
time curl http://auth.example.com/health
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
- "traefik.http.middlewares.stargate.forwardauth.address=http://stargate/_auth"
```

### Debugging Tips

#### 1. Enable Debug Mode

```bash
DEBUG=true
```

#### 2. Check Network Connection

```bash
# Test from inside container
docker exec stargate wget -O- http://localhost/health
```

#### 3. View Traefik Logs

```bash
docker logs traefik
```

#### 4. Test API Endpoints

```bash
# Test health check
curl http://auth.example.com/health

# Test authentication (using Header)
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

1. **Backup Configuration**: Save current environment variable configuration

2. **Stop Service:**

```bash
docker stop stargate
```

3. **Pull New Image:**

```bash
docker pull stargate:latest
```

4. **Start New Container:**

```bash
docker run -d \
  --name stargate \
  ...(use backed up configuration)
  stargate:latest
```

5. **Verify Service:**

```bash
curl http://auth.example.com/health
```

### Rollback

If problems occur after upgrade:

```bash
# Stop new container
docker stop stargate

# Start with old image
docker run -d \
  --name stargate \
  ...(use backed up configuration)
  stargate:<old-version>
```
