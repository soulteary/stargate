# Mail OTP and authenticator example

This example connects the published Stargate 1.3.5, Warden 1.6.2, Herald 1.4.4
and herald-totp 2.0.3 services. It uses actual SMTP delivery, shared persistent
Redis, explicit HMAC key IDs and three independent service credentials. It does
not build an application or expose a test-code endpoint. For a password-only
local demo, use the [root quick start](../../README.md#-quick-start).

## Local mail OTP

Requirements: Python 3, Docker Compose v2, and free loopback ports 8088 and 8025.
The `*.test.localhost` names must resolve to loopback in your browser. If they do
not, add `127.0.0.1 auth.test.localhost hello.test.localhost` to your hosts file.

```sh
cd examples/full-stack
python3 configure.py --mode mail
docker compose config --quiet
docker compose up -d
```

Open **http://hello.test.localhost:8088**. Enter `demo@example.test`, send the
email code, then open **http://127.0.0.1:8025** to read the Mailpit inbox. Submit
the delivered code and confirm that the browser returns to the protected demo.
The email goes through Herald's real SMTP provider. Mailpit captures it locally;
this is not delivery to an external mailbox. No SMS provider is configured.

Open `http://auth.test.localhost:8088/_login` to inspect the authentication page.
Logout uses `POST /_logout` on the authentication host; use the browser logout
control or a same-origin form, not a GET link. Confirm that the protected service
requires authentication again. Do not clear Redis or remove volumes to test this.

For a different port or DNS names, choose them **before** initial configuration:

```sh
python3 configure.py --port 9090 --mailbox-port 9025 \
  --auth-host auth.demo.localhost --app-host app.demo.localhost
```

This generates separate DNS-only proxy rules and host:port callback authorities.
Open `http://app.demo.localhost:9090` and `http://127.0.0.1:9025`. Port 80/443
uses its canonical authority without a port suffix. `--subnet` can select an
unused private /24 when the default edge network conflicts with a local network.

## Add an authenticator

Start with a fresh configuration using `python3 configure.py --mode totp` instead
of `--mode mail`. This enables the TOTP service and both required proxy flags.
First log in by email OTP. Then open `http://auth.test.localhost:8088/totp/enroll`,
choose Start enrollment, then scan the QR code or enter the setup key in your authenticator, and confirm using
its dynamic code. Save the displayed backup codes privately before leaving.

Log out and test the authenticator login option, then test one backup code. A
used backup code must fail with a fresh login challenge. There is no initial
shared TOTP secret: credentials belong to the stable `demo-user-001` account ID.
Changing an email address must not silently create a new account ID.

## Configuration and persistence

`configure.py` creates `.env` with mode 0600 and a private `state/` directory.
It never prints secrets or overwrites existing configuration. Running it again
retains all keys and user IDs; it does **not** apply changed command-line options.
A partial setup fails for manual inspection. Never delete an existing setup to
change a port or enable TOTP: back it up, edit the non-secret settings together,
and keep all stored credentials. To enable TOTP in an existing mail setup, set
`COMPOSE_PROFILES=totp` and `TOTP_ENABLED=true`, retaining every generated key.

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
private Compose network; for multiple hosts or an untrusted network, configure
TLS/mTLS in the respective services and SDKs before deployment. mTLS does not
replace Herald/TOTP request authentication.

1. Place `cert.pem`, `key.pem` and a copy of `tls.yml.example` named `tls.yml` in
   a protected certificate directory. Keep the private key out of Git.
2. Set `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`, `SMTP_USER`, `SMTP_PASSWORD` and
   `TLS_DIR` in the shell environment. Do not place a password in command-line
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

The generator validates the certificate/key pair and required settings before
writing. It does not prove certificate hostname coverage, SMTP deliverability,
volume backup quality or network policy. Verify those independently. `COOKIE_SECURE`
is true and the proxy listens with TLS on port 443. Open `https://app.example.com`
and confirm delivery to the controlled mailbox before adding other users.
Replace the `hello` demo with your application and retain its ForwardAuth middleware.
The example proxy uses generated file-provider routes and never mounts the Docker
socket. Every service has `traefik.enable=false` so an unrelated Docker-discovery
proxy does not automatically route these containers. Choose a unique `--project`
and an unused `--subnet`; do not reuse another deployment’s project or network.

## Stop and diagnose

`docker compose ps` and `docker compose logs --tail=50 SERVICE` identify startup
failures; handle logs as sensitive. Healthy processes alone do not prove login.
Check DNS/port, TLS and Cookie transport, exact callback authority, Warden stable
ID/source freshness, all three HMAC key-ID pairs, Redis and actual delivery in that
order. Do not disable verification, switch to soft delivery or reset quotas to
make the example pass. See [deployment guidance](../../docs/enUS/DEPLOYMENT.md).
