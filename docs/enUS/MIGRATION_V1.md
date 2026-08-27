# Migrating from v0.12.0 to v1.0.0

Stargate v1.0.0 establishes the first stable deployment and HTTP contracts. It also changes several security defaults. Test the upgrade in a staging environment before changing production traffic.

## Before upgrading

1. Record the current image, environment variables, proxy routes, callback hosts, and health probes.
2. Back up the deployment configuration and keep the v0.12.0 image available for rollback.
3. Identify every caller of `/_logout`, `/totp/enroll`, and the cross-domain session exchange route.
4. If more than one Stargate replica is running, prepare Redis-backed session storage before upgrading.

## Required deployment changes

### Official container port and user

| Contract | v0.12.0 | v1.0.0 |
| --- | --- | --- |
| Official container port | `80` | `8080` |
| Runtime user | root | UID/GID `10001` |
| Liveness probe | `/health` | `/healthz` |
| Dependency readiness probe | `/health` | `/readyz` |

Update reverse-proxy targets and container health checks together. A minimal Compose service is:

```yaml
services:
  stargate:
    image: ghcr.io/soulteary/stargate:v1.0.0
    ports:
      - "8080:8080"
    read_only: true
    tmpfs:
      - /tmp
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://127.0.0.1:8080/healthz"]
```

When Stargate is used as Traefik ForwardAuth, change the target to `http://stargate:8080/_auth`.

### Cross-domain session exchange

v1.0.0 replaces the raw `?id=` session transfer with an encrypted, short-lived, single-use `?ticket=` value that is bound to the destination host.

- Configure `CALLBACK_ALLOWED_HOSTS` with every permitted callback host.
- Configure the same `SESSION_EXCHANGE_SECRET` on all replicas. It must contain at least 32 characters when cross-domain callbacks are enabled.
- Keep `COOKIE_SECURE=true` for HTTPS deployments.
- Do not construct exchange URLs in downstream applications. Follow the redirect returned by Stargate.
- Use Redis-backed session storage when tickets or sessions must be accepted by multiple replicas.

Old `?id=` exchange links are not compatible with v1.0.0.

### Proxy trust

Forwarded headers are ignored unless the immediate peer matches `TRUSTED_PROXIES`. Set this allowlist to the IP addresses or CIDRs of the reverse proxies that connect directly to Stargate. Use `PROXY_HEADER` to select the client-IP header; its default is `X-Forwarded-For`.

Do not place arbitrary client networks in `TRUSTED_PROXIES`.

### Authentication modes

- The login form remains available through `POST /_login`.
- Authentication through the `Stargate-Password` request header is disabled by default. Set `PASSWORD_HEADER_AUTH_ENABLED=true` only for a trusted proxy integration, and make the proxy remove any client-supplied copy of that header.
- Trusted identity headers require `HEADER_AUTH_ENABLED=true`, a matching `HEADER_AUTH_SHARED_SECRET` of at least 32 characters, and Warden. The proxy must remove client-supplied identity and secret headers before adding its own values.
- Invalid Warden or Herald client configuration now fails at startup instead of producing a partially working service.
- The legacy Warden global OTP fallback has been removed. Use Herald-backed TOTP enrollment and verification.

## HTTP contract changes

| Previous contract | v1.0.0 contract | Action |
| --- | --- | --- |
| `GET /_logout` | `POST /_logout` | Update links and clients to submit a POST request. |
| `GET /totp/enroll` | `POST /totp/enroll` | Start enrollment with POST. |
| `GET /health` for all probes | `GET /healthz` and `GET /readyz` | Separate liveness from dependency readiness. |
| `GET /metrics` | `GET /metrics` (unchanged) | Keep the existing Prometheus-compatible scrape target. |
| Session exchange `?id=` | Session exchange `?ticket=` | Follow Stargate redirects; do not expose session IDs. |

`GET /health` remains as a deprecated compatibility alias for readiness. New deployments should not use it. Browser requests that change authentication state are subject to same-origin checks and endpoint-specific rate limits.

## Configuration behavior changes

- `AUTH_REFRESH_ENABLED` now defaults to `true`. Set it to `false` only when delayed Warden revocation is acceptable.
- `COOKIE_SECURE` defaults to `true`; local HTTP testing must opt out explicitly.
- Startup validation now rejects malformed URLs, ports, durations, Redis database values, undersized exchange secrets, incomplete TLS certificate/key pairs, and incomplete HMAC key ID/secret pairs.
- `CALLBACK_ALLOWED_HOSTS` is required to permit explicit cross-domain callback destinations.
- Warden and Herald HMAC/TLS settings must be complete when enabled. `HERALD_HMAC_KEY_ID` may be omitted only when Herald has an effective default key ID; configure it explicitly for multi-key deployments without a suitable default.

Run the v1.0.0 image with the production environment before cutover and resolve every startup validation error rather than bypassing it.

## Rollout procedure

1. Deploy v1.0.0 to a separate pool with the updated port and probes.
2. For multiple replicas, enable and verify shared Redis session storage before sending user traffic.
3. Exercise password, Warden, header, and TOTP flows that are enabled in the deployment.
4. Verify same-domain and every allowed cross-domain callback.
5. Move traffic only after `/healthz` and `/readyz` both succeed.
6. Keep v0.12.0 and v1.0.0 in separate pools during the transition; their session-exchange contracts must not be mixed.

## Verification checklist

```bash
curl --fail http://127.0.0.1:8080/healthz
curl --fail http://127.0.0.1:8080/readyz
curl --fail http://127.0.0.1:8080/metrics
```

- The container runs as UID/GID `10001` and does not require a writable root filesystem.
- The reverse proxy reaches `http://stargate:8080/_auth`.
- Login succeeds and logout uses `POST /_logout`.
- TOTP enrollment, confirmation, verification, and revocation succeed when Herald is configured.
- An unlisted callback host and a reused or expired exchange ticket are rejected.
- A client-supplied password or identity header cannot bypass the trusted proxy.
- Dependency failure keeps `/healthz` live while `/readyz` reports not ready.

## Rollback

Restore the v0.12.0 image, proxy port, and old probe together. A v1.0.0 exchange ticket cannot be consumed by v0.12.0, so invalidate in-flight cross-domain exchanges and require a fresh login after rollback. Do not assume active sessions are portable between mixed-version pools.
