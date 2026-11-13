"""Verify the actual Docker COPY context using synthetic canaries only."""
import io
import os
from pathlib import Path
import re
import secrets
import shutil
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parents[2]
# The files the binary embeds are the files the image must carry: the source
# build compiles the same go:embed directive, and the release image serves the
# copied template directory in preference to the embedded one.
EMBED = re.search(r'^//go:embed (.+)$', (ROOT/'src/internal/web/resources.go').read_text(), re.MULTILINE)
if not EMBED:
    raise RuntimeError('go:embed directive not found in src/internal/web/resources.go')
RESOURCES = [name.removeprefix('templates/') for name in EMBED.group(1).split()]
allowed = ['go.mod', 'go.sum', 'dist/stargate-linux-amd64', 'dist/stargate-linux-arm64']
tracked = subprocess.check_output(['git', '-C', str(ROOT), 'ls-files', '-z', '--', 'src']).decode().split('\0')
allowed += [name for name in tracked if name.endswith('.go') and not name.endswith('_test.go')]
allowed += ['src/internal/web/templates/'+name for name in RESOURCES]
rejected = ['.git/config', '.env', 'examples/full-stack/.env', 'examples/full-stack/state/secret',
            'src/.env.private', 'src/audit-spool/events.jsonl', 'src/audit-spool/secret.go',
            'src/generated-outputs/credential.go', 'src/outputs/sensitive.go', 'src/internal/auth/private.key',
            'src/internal/auth/server.pem', 'src/internal/handlers/private_test.go']
image = 'stargate-context-proof:'+secrets.token_hex(8)
container = None


def run(args, timeout=90):
    process = subprocess.run(args, capture_output=True, timeout=timeout)
    if process.returncode:
        raise RuntimeError('Docker context proof failed; no actual private files were used')
    return process.stdout


with tempfile.TemporaryDirectory(prefix='stargate-context-proof-') as directory:
    context = Path(directory)
    shutil.copyfile(ROOT/'.dockerignore', context/'.dockerignore')
    for name in allowed+rejected:
        path = context/name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text('synthetic-noncredential-canary\n')
    (context/'Dockerfile').write_text('FROM scratch\nCOPY . /proof/\n')
    try:
        run(['docker', 'build', '--network=none', '-t', image, context])
        container = run(['docker', 'create', '--label', 'traefik.enable=false', image, '/not-executed']).decode().strip()
        # The scratch container is never started; inspect exported filesystem metadata.
        with tarfile.open(fileobj=io.BytesIO(run(['docker', 'export', container]))) as archive:
            names = {entry.name.removeprefix('proof/') for entry in archive if entry.isfile() and entry.name.startswith('proof/')}
        if not set(allowed) <= names or set(rejected) & names:
            raise AssertionError('Docker COPY included a rejected canary or omitted required code/resources')
        print('Docker COPY context proof passed: required source/resources/binaries present; private canaries excluded.')
    finally:
        if container:
            run(['docker', 'rm', '-f', container])
        run(['docker', 'image', 'rm', '-f', image])
