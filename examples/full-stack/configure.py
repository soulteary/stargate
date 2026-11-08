#!/usr/bin/env python3
"""Create a private, persistent example configuration; never rotate existing keys."""
import argparse
import ipaddress
import json
import os
from pathlib import Path
import re
import secrets
import ssl
import sys

HERE = Path(__file__).resolve().parent
SECRET_KEYS = ('SESSION_EXCHANGE_SECRET', 'STARGATE_WARDEN_HMAC', 'STARGATE_HERALD_HMAC',
               'HERALD_TOTP_HMAC', 'HERALD_IDEMPOTENCY_SECRET', 'HERALD_PII_PEPPER', 'REDIS_PASSWORD')


def hostname(value):
    if len(value) > 253 or not re.fullmatch(r'[a-zA-Z0-9](?:[a-zA-Z0-9.-]*[a-zA-Z0-9])?', value):
        raise ValueError('hostnames must be DNS names without scheme, port or path')
    if any(not part or len(part) > 63 or part.startswith('-') or part.endswith('-') for part in value.split('.')):
        raise ValueError('invalid DNS label')
    try:
        ipaddress.ip_address(value)
    except ValueError:
        return value.lower()
    raise ValueError('use DNS hostnames, not IP addresses')


def port(value):
    result = int(value)
    if not 1 <= result <= 65535:
        raise ValueError('ports must be between 1 and 65535')
    return result


def authority(host, selected_port, default_port):
    return host if selected_port == default_port else f'{host}:{selected_port}'


def dotenv(values):
    lines = []
    for key, value in values.items():
        value = str(value)
        if any(c in value for c in '\n\r\x00'):
            raise ValueError(f'{key} must be a single line')
        lines.append(key + "='" + value.replace('\\', '\\\\').replace("'", "\\'") + "'")
    return '\n'.join(lines) + '\n'


def private_write(path, text):
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, 'w') as output:
        output.write(text)


def configure(args, environ=None):
    environ = os.environ if environ is None else environ
    output = Path(args.output).resolve()
    env_file = output / '.env'
    state = output / 'state'
    # Existing deployments are never rewritten, including encryption keys and user IDs.
    if env_file.exists() or state.exists():
        if env_file.is_file() and (state / 'users.json').is_file() and (state / 'routes.yml').is_file():
            if env_file.stat().st_mode & 0o077:
                raise ValueError('existing .env is not private; restrict its permissions to 0600')
            return False
        raise ValueError('partial configuration exists; inspect it before continuing (nothing overwritten)')
    production = args.environment == 'production'
    auth = hostname(args.auth_host or ('auth.test.localhost' if not production else ''))
    app = hostname(args.app_host or ('hello.test.localhost' if not production else ''))
    if auth == app:
        raise ValueError('authentication and application hostnames must differ')
    if production and any(host == 'localhost' or host.endswith(('.localhost', '.test', '.invalid')) for host in (auth, app)):
        raise ValueError('production requires real DNS hostnames')
    selected_port = port(args.port or (443 if production else 8088))
    mailbox_port = port(args.mailbox_port)
    if not production and selected_port == mailbox_port:
        raise ValueError('web and mailbox ports must differ')
    subnet = ipaddress.ip_network(args.subnet, strict=True)
    if subnet.version != 4 or subnet.prefixlen != 24 or not any(subnet.subnet_of(ipaddress.ip_network(block)) for block in ('10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16')):
        raise ValueError('use an unused private IPv4 /24 for the example edge network')
    email = args.email or ('' if production else 'demo@example.test')
    if not re.fullmatch(r'[^\s@]+@[^\s@]+\.[^\s@]+', email):
        raise ValueError('set --email to the initial user email address')
    uid = args.user_id or ('' if production else 'demo-user-001')
    if not re.fullmatch(r'[a-zA-Z0-9_-]{1,128}', uid):
        raise ValueError('set --user-id to a stable account identifier (letters, digits, _ or -)')
    values = {key: secrets.token_hex(32) for key in SECRET_KEYS}
    values.update(TOTP_ENCRYPTION_KEY=secrets.token_urlsafe(24),
                  COMPOSE_FILE=str(HERE / 'compose.yml') + os.pathsep + str(HERE / ('compose.production.yml' if production else 'compose.local.yml')),
                  COMPOSE_PROFILES='totp' if args.mode == 'totp' else '',
                  COMPOSE_PROJECT_NAME=args.project,
                  STATE_DIR=str(state), EDGE_NETWORK=args.project + '-edge', EDGE_SUBNET=str(subnet),
                  AUTH_HOSTNAME=auth, APP_HOSTNAME=app,
                  AUTH_AUTHORITY=authority(auth, selected_port, 443 if production else 80),
                  APP_AUTHORITY=authority(app, selected_port, 443 if production else 80),
                  PUBLIC_PORT=selected_port, MAILBOX_PORT=mailbox_port,
                  COOKIE_SECURE='true' if production else 'false',
                  TOTP_ENABLED='true' if args.mode == 'totp' else 'false')
    if not re.fullmatch(r'[a-z0-9][a-z0-9_-]{0,50}', args.project):
        raise ValueError('invalid Compose project name')
    if production:
        for key in ('SMTP_HOST', 'SMTP_PORT', 'SMTP_FROM', 'SMTP_USER', 'SMTP_PASSWORD', 'TLS_DIR'):
            if not environ.get(key):
                raise ValueError(f'production requires {key} in the environment')
            values[key] = environ[key]
        port(values['SMTP_PORT'])
        tls_dir = Path(values['TLS_DIR']).resolve()
        # Validate the configured certificate/key pair before persisting any configuration.
        context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        context.load_cert_chain(tls_dir / 'cert.pem', tls_dir / 'key.pem')
        if not (tls_dir / 'tls.yml').is_file():
            raise ValueError('TLS_DIR must contain tls.yml (see README)')
        values['TLS_DIR'] = str(tls_dir)
    else:
        values.update(SMTP_HOST='mailpit', SMTP_PORT=1025, SMTP_FROM='login@example.test', SMTP_USER='', SMTP_PASSWORD='')
    routing = {
        'http': {
            'routers': {
                'auth': {'rule': f'Host(`{auth}`) || Path(`/_session_exchange`)', 'entryPoints': ['web'], 'service': 'auth'},
                'app': {'rule': f'Host(`{app}`)', 'entryPoints': ['web'], 'service': 'app', 'middlewares': ['stargate']},
            },
            'services': {
                'auth': {'loadBalancer': {'servers': [{'url': 'http://stargate:8080'}]}},
                'app': {'loadBalancer': {'servers': [{'url': 'http://hello:8080'}]}},
            },
            'middlewares': {'stargate': {'forwardAuth': {
                'address': 'http://stargate:8080/_auth',
                'authResponseHeaders': ['X-Forwarded-User', 'X-Auth-User', 'X-Auth-Email', 'X-Auth-Name', 'X-Auth-Scopes', 'X-Auth-Role', 'X-Auth-AMR'],
            }}},
        },
    }
    if production:
        for router in routing['http']['routers'].values():
            router['tls'] = {}
    encoded = dotenv(values)
    output.mkdir(parents=True, exist_ok=True, mode=0o700)
    state.mkdir(mode=0o700)
    # User directory contains no password/OTP/TOTP secret. Its private parent protects it on the host;
    # read permission lets the non-root Warden container read the mounted directory file.
    users = state / 'users.json'
    private_write(users, json.dumps([{'user_id': uid, 'mail': email, 'status': 'active', 'scope': ['demo'], 'role': 'user'}], indent=2) + '\n')
    users.chmod(0o644)
    routes = state / 'routes.yml'
    private_write(routes, json.dumps(routing, indent=2) + '\n')
    routes.chmod(0o644)
    private_write(env_file, encoded)
    return True


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--environment', choices=('local', 'production'), default='local')
    parser.add_argument('--mode', choices=('mail', 'totp'), default='mail')
    parser.add_argument('--output', default=str(HERE))
    parser.add_argument('--auth-host')
    parser.add_argument('--app-host')
    parser.add_argument('--port', type=int)
    parser.add_argument('--mailbox-port', type=int, default=8025)
    parser.add_argument('--project', default='stargate-example')
    parser.add_argument('--subnet', default='172.31.86.0/24')
    parser.add_argument('--email')
    parser.add_argument('--user-id')
    args = parser.parse_args()
    try:
        created = configure(args)
    except (ValueError, OSError, ssl.SSLError) as error:
        # Configuration values (particularly SMTP passwords) must not appear in diagnostics.
        print('Configuration failed: ' + (str(error) if isinstance(error, ValueError) else type(error).__name__), file=sys.stderr)
        return 1
    print('Configuration created.' if created else 'Existing configuration retained; no keys or user IDs changed.')
    print('Run docker compose from the output directory. Do not share its .env file or rendered configuration.')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
