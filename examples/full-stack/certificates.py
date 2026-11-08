#!/usr/bin/env python3
"""Create private local SMTP certificates; never modify host trust or existing keys."""
import argparse
from pathlib import Path
import shutil
import ssl
import subprocess


def openssl(*args):
    executable = shutil.which('openssl')
    if not executable:
        raise ValueError('OpenSSL is required to generate and verify local SMTP TLS certificates')
    result = subprocess.run([executable, *map(str, args)], capture_output=True, timeout=30)
    if result.returncode:
        raise ValueError('SMTP TLS certificate generation or verification failed; no existing files were overwritten')


def generate(directory):
    directory = Path(directory)
    if not shutil.which('openssl'):
        raise ValueError('OpenSSL is required to generate and verify local SMTP TLS certificates')
    expected = ['ca.pem', 'ca-key.pem', 'smtp-cert.pem', 'smtp-key.pem']
    if directory.exists():
        if directory.is_symlink() or not all((directory/name).is_file() for name in expected):
            raise ValueError('Partial SMTP TLS configuration exists; inspect it before continuing')
        if any((directory/name).stat().st_mode & 0o077 for name in ['ca-key.pem', 'smtp-key.pem']):
            raise ValueError('SMTP TLS private keys must have private 0600 permissions')
        try:
            ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER).load_cert_chain(directory/'smtp-cert.pem', directory/'smtp-key.pem')
        except (ssl.SSLError, OSError):
            raise ValueError('Existing SMTP certificate and private key do not match; nothing was overwritten') from None
        openssl('verify', '-CAfile', directory/'ca.pem', '-verify_hostname', 'mailpit', directory/'smtp-cert.pem')
        return False
    directory.mkdir(parents=True, mode=0o700)
    try:
        openssl('req', '-x509', '-newkey', 'ec', '-pkeyopt', 'ec_paramgen_curve:P-256', '-nodes',
                '-sha256', '-days', '365', '-subj', '/CN=Local SMTP example CA',
                '-addext', 'basicConstraints=critical,CA:TRUE', '-addext', 'keyUsage=critical,keyCertSign,cRLSign',
                '-keyout', directory/'ca-key.pem', '-out', directory/'ca.pem')
        (directory/'ca-key.pem').chmod(0o600)
        openssl('req', '-new', '-newkey', 'ec', '-pkeyopt', 'ec_paramgen_curve:P-256', '-nodes',
                '-subj', '/CN=mailpit', '-keyout', directory/'smtp-key.pem', '-out', directory/'smtp.csr')
        (directory/'smtp-key.pem').chmod(0o600)
        extension = directory/'smtp.ext'
        extension.write_text('basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature\nextendedKeyUsage=serverAuth\nsubjectAltName=DNS:mailpit\n')
        openssl('x509', '-req', '-in', directory/'smtp.csr', '-CA', directory/'ca.pem', '-CAkey', directory/'ca-key.pem',
                '-CAcreateserial', '-days', '365', '-sha256', '-extfile', extension, '-out', directory/'smtp-cert.pem')
        for name in ['ca.pem', 'smtp-cert.pem']:
            (directory/name).chmod(0o644)
        openssl('verify', '-CAfile', directory/'ca.pem', '-verify_hostname', 'mailpit', directory/'smtp-cert.pem')
    except Exception:
        # Preserve partial state for inspection rather than silently replacing any generated keys.
        raise
    return True


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', required=True)
    args = parser.parse_args()
    try:
        created = generate(args.output)
    except (ValueError, OSError, subprocess.TimeoutExpired) as error:
        print(str(error) if isinstance(error, ValueError) else 'Local SMTP certificate setup failed')
        raise SystemExit(1)
    print('Local SMTP TLS files created.' if created else 'Existing valid SMTP TLS files retained.')
