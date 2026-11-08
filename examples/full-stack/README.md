# Mail OTP and authenticator example

This example builds Stargate from this checkout and connects the published Warden
1.6.2, Herald 1.5.0 and herald-totp 2.0.3 services. It uses actual SMTP delivery, shared persistent
Redis, explicit HMAC key IDs and three independent service credentials. It does not expose a test-code endpoint. For a password-only
local demo, use the [root quick start](../../README.md#-quick-start).

## Local mail OTP

Requirements: Python 3, OpenSSL, Docker Compose v2 on a Unix-compatible host (use WSL2 on Windows), and free loopback ports 8088 and 8025.
The `*.test.localhost` names must resolve to loopback in your browser. If they do
not, add `127.0.0.1 auth.test.localhost hello.test.localhost` to your hosts file.

```sh
cd examples/full-stack
python3 configure.py --mode mail
docker compose config --quiet
docker compose up -d --build
```

Open **http://hello.test.localhost:8088**. Enter `demo@example.test`, send the
email code, then open **http://127.0.0.1:8025** to read the Mailpit inbox. Submit
the delivered code and confirm that the browser returns to the protected demo.
The local override explicitly uses Herald development mode while retaining
HMAC, strict delivery and `HERALD_TEST_MODE=false`. The email goes through Herald's real SMTP provider. Mailpit captures it locally;
this is not delivery to an external mailbox. No SMS provider is configured.

Open `http://auth.test.localhost:8088/_login` to inspect the authentication page.
Visit **http://auth.test.localhost:8088/_account** and use its logout button, which
submits a same-origin `POST /_logout`. Revisit the protected service and confirm
it requires authentication again. Do not clear Redis or remove volumes. This
account page comes from the source build; it is not in the old v1.3.5 image.

For a different port or DNS names, choose them **before** initial configuration:

```sh
python3 configure.py --port 9090 --mailbox-port 9025 \
  --auth-host auth.demo.localhost --app-host app.demo.localhost
```

This generates separate DNS-only proxy rules and host:port callback authorities.
Open `http://app.demo.localhost:9090` and `http://127.0.0.1:9025`. Port 80/443
uses its canonical authority without a port suffix. `--subnet` can select an
unused private /24 when the default edge network conflicts with a local network.

### Shared-VM bind mounts on macOS

The non-root service must be able to read its specific mounted public directory
files. Some shared-VM filesystems deny access through a private host ancestor
(such as a Documents directory), even when `users.json` itself is 0644. Check the
actual container error; do not treat a timeout as a permission diagnosis.

Do not relax `.env`, certificate-key or private-parent permissions. For the
**synthetic local demo only**, an operator may stage only `state/users.json`, `state/routes.yml` and the public
`state/provider-ca.pem` in an owned temporary directory outside Documents, make that
public-fixture parent 0755 and those public files 0644, and run Compose with an
explicit `STATE_DIR` override pointing there. Keep the original private `.env`
and every credential unchanged. Never copy real user data, `.env` or private
keys into that public fixture directory. Keep SMTP private keys at their original private path. Verify the exact mounted files are
readable before calling startup successful; record this as a local filesystem
accommodation, not a production configuration default. Remove only the owned
fixture directory after stopping its stack. Production should use an approved
host/mount mechanism that preserves its access-control requirements.

### Local SMTP TLS and existing setups

Herald requires verified SMTP STARTTLS even in the local HTTP walkthrough.
Setup generates a private local CA and a server certificate with SAN `mailpit`.
Mailpit requires STARTTLS; only the public CA bundle is mounted into Herald via
`SSL_CERT_FILE`. The CA private key is never mounted. No host trust store is
changed. Both `mail` and `totp` modes require OpenSSL; the separate root
password-only Compose example does not.

For an older local configuration without these files, keep `.env`, Redis,
account IDs and every application/HMAC/TOTP key unchanged. Run `certificates.py`
with `--output` pointing to a new `local-smtp` directory under its private state
directory. Copy its public `ca.pem` to `STATE_DIR/provider-ca.pem` (0644), and add
`SMTP_TLS_DIR` (the absolute certificate directory), `FIXTURE_UID`/`FIXTURE_GID`
(your host numeric UID/GID) and `HERALD_CA_FILE="/certs/provider-ca.pem"` to the
existing private `.env`. Recreate Herald and Mailpit to reload trust. Do not
initialize another application configuration or rotate credentials.

Local SMTP certificates last 365 days. An expired, partial or mismatched set is
rejected rather than overwritten. To renew explicitly, retain a private backup
of only the old `local-smtp` certificate directory, create a fresh certificate set
at the original path, update the public provider CA bundle and recreate Herald
and Mailpit. Do not change users, Redis, HMAC or TOTP encryption keys. These are
local test certificates, not a production CA-management procedure.

## Add an authenticator

Start with a fresh configuration using `python3 configure.py --mode totp` instead
of `--mode mail`. This enables the TOTP service and both required proxy flags.
First log in by email OTP. Then open `http://auth.test.localhost:8088/totp/enroll`,
choose Start enrollment, then scan the QR code or enter the setup key in your authenticator, and confirm using
its dynamic code. Save the displayed backup codes privately before leaving.

Log out and test the authenticator login option, then use the explicit recovery
mode to test one backup code. A used backup code must fail with a fresh login
challenge. The revised recovery form comes from this source checkout, not the
old v1.3.5 image. There is no initial
shared TOTP secret: credentials belong to the stable `demo-user-001` account ID.
Changing an email address must not silently create a new account ID.

## Configuration and persistence

`configure.py` creates `.env` with mode 0600 and a private `state/` directory.
It never prints secrets or overwrites existing configuration. Running it again
retains all keys and user IDs; it does **not** apply changed command-line options.
A partial setup fails for manual inspection. Never delete an existing setup to
change a port or enable TOTP: back it up, edit the non-secret settings together,
and keep all stored credentials. For an existing **local** mail setup, set `COMPOSE_PROFILES=totp` and
`TOTP_ENABLED=true`, retaining every generated key. Production also needs the
TLS extension and routing file described below; these two flags alone are not sufficient.

The generated user directory has no password or code; it is readable by the
non-root Warden container, while its private parent restricts host access. The
Redis AOF and Herald audit spool use named volumes. Restart or `docker compose down`
retains them. **Do not use `down -v` on a deployment you intend to retain.**
Keep this source checkout available: `COMPOSE_FILE` records its absolute paths.
Back up Redis and `.env` together; losing the TOTP encryption key makes retained
credentials unreadable. Keep account IDs, prefixes, replay state and session
absolute deadlines across upgrades and rollback. Never commit `.env`, print
`docker compose config` without `--quiet` into logs, or share a browser trace
containing codes/cookies.

The three HMAC legs are Stargate→Warden (`stargate-directory`), Stargate→Herald
(`stargate-otp`) and Herald→TOTP (`herald-totp`). API keys are intentionally empty;
HMAC rejection does not fall back to them. Rates, one-use consumption and strict
delivery failure handling remain enabled. Directory snapshots are refreshed every
15 seconds, expire after 60 seconds and are accepted by Stargate for at most
90 seconds with a one-second clock allowance; keep clocks synchronized.

## Production template

The production override is a starting template for a single trusted host, not a
certificate issuer or delivery-provider setup tool. It requires real DNS, a TLS
certificate valid for both hosts and authenticated SMTP. It does not start Mailpit
or publish Redis/Warden/Herald/TOTP ports. Internal service HTTP is confined to the
private Compose network. Herald’s production TOTP client requires verified HTTPS
even on that network: the TOTP production extension adds an internal TLS proxy and
a private CA trusted only by the Herald container. For multiple hosts or an
untrusted network, configure TLS/mTLS for the remaining service links before
deployment. mTLS does not
replace Herald/TOTP request authentication.

1. Place `cert.pem`, `key.pem` and a copy of `tls.yml.example` named `tls.yml` in
   a protected certificate directory. Keep the private key out of Git and limit
   its host permissions to 0600. For TOTP, also provide `totp-cert.pem` (SAN
   `DNS:totp-tls`), `totp-key.pem` and the issuing `totp-ca.pem`. Keep the CA private
   key offline; it is not mounted. The TOTP server certificate/key must match,
   and its chain must verify against that CA.
2. Set `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, `SMTP_USER`, `SMTP_PASSWORD`,
   `TLS_DIR` and `STARGATE_IMAGE` in the shell environment. `STARGATE_IMAGE` must
   identify a tested gateway image by `@sha256:...` digest, built from source that
   contains the account and recovery UI. A repository tag alone is not accepted. Do not place a password in command-line
   arguments or shell history. This template requires authenticated SMTP; adapt
   it explicitly if your approved relay uses another authentication mechanism.
3. From this directory, initialize a **new** output directory:

   ```sh
   python3 configure.py --environment production --mode totp \
     --auth-host auth.example.com --app-host app.example.com \
     --email operator@example.com --user-id account-001 \
     --output /protected/path/stargate-config
   cd /protected/path/stargate-config
   docker compose config --quiet
   docker compose up -d
   ```

The production override does not build Stargate. Build and verify your gateway
image from the reviewed source separately, publish it to your approved registry,
and set its immutable digest. Do not substitute the old v1.3.5 image, which lacks
the account and revised recovery UI.

For production TOTP, configuration also selects `compose.totp-production.yml`.
Herald calls `https://totp-tls:8443`; the proxy forwards unchanged signed requests
to the private TOTP service. No TOTP proxy port is published. `SSL_CERT_FILE` is
scoped to the Herald container, with no host trust-store changes or TLS bypass.
For a private SMTP issuer, provide `SMTP_CA_FILE` during initial production setup.
The generator combines that public CA with the TOTP issuer in `state/provider-ca.pem`;
without a custom SMTP CA, production mail uses the image’s normal public trust roots.
Validate an authenticated TOTP operation through Herald after startup; a healthy
process or rendered Compose file does not prove this TLS/authentication path.

The generator validates the certificate/key pairs, parses the CA and checks required settings before
writing. It does not prove certificate hostname coverage, SMTP deliverability,
volume backup quality or network policy. Verify those independently. `COOKIE_SECURE`
is true and the proxy listens with TLS on port 443. Open `https://app.example.com`
and confirm delivery to the controlled mailbox before adding other users.
Replace the `hello` demo with your application and retain its ForwardAuth middleware.
The example proxy uses generated file-provider routes and never mounts the Docker
socket. Every service has `traefik.enable=false` so an unrelated Docker-discovery
proxy does not automatically route these containers. Choose a unique `--project`
and an unused `--subnet`; do not reuse another deployment’s project or network.

### Existing provider trust-bundle migration

An older configuration without `state/provider-ca.pem` is rejected explicitly;
setup will not create it silently or replace existing credentials. For production
mail using public SMTP certificates and no TOTP, create an empty 0644 provider
bundle and keep `HERALD_CA_FILE=""` so normal public roots remain in use. For a
private SMTP issuer, put only its public CA certificates in that file. With TOTP,
include its public issuer too, retaining existing SMTP issuers, and set
`HERALD_CA_FILE="/certs/provider-ca.pem"`. Never put a CA private key in the bundle.
Recreate Herald after changing trust. Retain all application keys and Redis.
Generated path values use JSON double quotes and literal dollars escaped as `$$`;
use absolute paths, not shell expressions, when editing them.

## Enable TOTP on an existing production mail deployment

Do not rerun setup expecting it to change an existing deployment, and do not
regenerate any secrets. Back up the existing `.env` and Redis first.

1. In the existing `TLS_DIR`, provision the TOTP certificate with SAN `DNS:totp-tls`,
   matching `totp-key.pem` (host mode 0600) and issuing `totp-ca.pem` described above.
2. Copy this checkout's `totp-tls.yml.example` to the existing `STATE_DIR/totp-tls.yml`
   and make that non-secret routing file readable by the proxy (0644). Add the
   public TOTP issuer to `STATE_DIR/provider-ca.pem`, retaining any existing SMTP CA,
   and set `HERALD_CA_FILE="/certs/provider-ca.pem"` in the private `.env`. Do not copy
   or change the user directory, encryption key or HMAC credentials.
3. Edit the existing private `.env`: set `COMPOSE_PROFILES="totp"` and
   `TOTP_ENABLED="true"`; append this checkout's absolute
   `compose.totp-production.yml` path to `COMPOSE_FILE`, after the base and
   production override paths, using the platform's path separator (`:` on Unix).
   Retain `TLS_DIR`, the original `TOTP_ENCRYPTION_KEY`, all HMAC secrets/key IDs,
   Redis password, exchange secret, prefixes and account IDs. Do not create a
   second configuration or overwrite this file from a fresh setup.
4. Run `docker compose config --quiet` and restart the selected services. Verify
   an authenticated TOTP operation through Herald with certificate verification
   enabled, then complete enrollment in the browser. Startup health alone is not
   proof of the new TLS/request-authentication path.

This selects the TLS proxy and CA mount as well as enabling the feature. On
rollback, keep the encryption key, credential/replay state and original account
IDs even if TOTP is disabled; do not erase Redis or regenerate configuration.

## Stop and diagnose

`docker compose ps` and `docker compose logs --tail=50 SERVICE` identify startup
failures; handle logs as sensitive. Healthy processes alone do not prove login.
Check DNS/port, TLS and Cookie transport, exact callback authority, Warden stable
ID/source freshness, all three HMAC key-ID pairs, Redis and actual delivery in that
order. Do not disable verification, switch to soft delivery or reset quotas to
make the example pass. See [deployment guidance](../../docs/enUS/DEPLOYMENT.md).

## Source identity and published versions

Local setup builds the checked-out Stargate source with `docker/Dockerfile`;
`--build` is required, especially after pulling updates. Record `git rev-parse HEAD`
and `git status --short` before building, then record the resulting image ID with
`docker compose images`. A modified tree must be recorded as modified; never
claim a source build is a published artifact. The three backend images remain
pinned to published versions and immutable digests.

The old published Stargate v1.3.5 does not contain the new account/recovery UI.
Use the source build for this local walkthrough. Production requires your own
tested immutable gateway image until a suitable formal release is available;
this example does not publish or silently select one.

### Build context boundary

The repository `.dockerignore` allows Go source, the explicit runtime web resources
and the two packaged Linux binaries. Git credentials, example configuration,
`.env*`, audit spools and private key/certificate files do not enter the build
context. New custom runtime resources must be explicitly allowed and tested; do
not remove the private-file exclusions. CI verifies this with synthetic canaries
in an actual scratch-image `COPY`, without copying real private data.
