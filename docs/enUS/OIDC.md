# OIDC Authentication

Stargate supports OpenID Connect (OIDC) authentication for integration with enterprise SSO providers.

## Configuration

Enable OIDC by setting the following environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OIDC_ENABLED` | Yes | `false` | Set to `true` to enable OIDC |
| `OIDC_ISSUER_URL` | Yes* | - | Your OIDC provider's issuer URL |
| `OIDC_CLIENT_ID` | Yes* | - | Client ID from your OIDC provider |
| `OIDC_CLIENT_SECRET` | Yes* | - | Client secret from your OIDC provider |
| `OIDC_REDIRECT_URI` | No | `https://{AUTH_HOST}/_oidc/callback` | Callback URL for your application |
| `OIDC_PROVIDER_NAME` | No | `OIDC` | Display name for login button |

*Required when `OIDC_ENABLED=true`

## Example docker-compose.yml

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      - AUTH_HOST=auth.example.com
      - OIDC_ENABLED=true
      - OIDC_ISSUER_URL=https://keycloak.example.com/realms/myrealm
      - OIDC_CLIENT_ID=stargate-client
      - OIDC_CLIENT_SECRET=your-secret
      - OIDC_PROVIDER_NAME=Company Account
```

## Supported Providers

Any OIDC-compliant provider should work, including:
- Keycloak
- Azure AD / Entra ID
- Auth0
- Okta
- Google Workspace
- Self-hosted OIDC servers

## Notes

- When OIDC is enabled, password authentication is automatically disabled
- If you need `http` or a custom callback URL, set `OIDC_REDIRECT_URI` explicitly
- Only `openid` and `email` scopes are requested
- User ID (sub claim) and email are stored in the session
- The `X-Forwarded-User` header contains the user ID or email
