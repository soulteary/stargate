"""Read-only release downloads and isolated real-service compatibility checks.

Only formal Linux/amd64 binaries are executed. OCI references are recorded, not
executed or cryptographically verified here; the release publication gate owns
image signature/provenance verification. No application is built by this job.
"""
import base64
from contextlib import contextmanager
import hashlib
import hmac
import http.cookies
import io
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import signal
import socket
import struct
import subprocess
import tarfile
import time
import urllib.error
import urllib.parse
import urllib.request

REPOS = {'warden': 'warden', 'herald': 'herald', 'totp': 'herald-totp', 'gateway': 'stargate'}
ORDER = ['totp', 'warden', 'herald', 'gateway']
ROTATION_SUBJECTS = ('chain-c@example.test', 'chain-d@example.test')
MAX_DOWNLOAD = 160 * 1024 * 1024
DEADLINE = None
OUT = Path('compatibility-evidence').resolve()
RUNTIME = OUT / 'runtime'
processes = {}
log_handles = []
events = []
result = {'status': 'in_progress', 'checks': events, 'production_access': False,
          'image_execution': False, 'image_signature_verification': 'separate publication gate',
          'delivery_scope': 'Actual Herald OTP manager; synthetic email, soft delivery and authenticated test-only code listener',
          'harness_source': os.environ.get('GITHUB_SHA'), 'run_id': os.environ.get('GITHUB_RUN_ID')}


def validate_manifest(value):
    if not isinstance(value, dict) or set(value) not in ({'mode', 'old'}, {'mode', 'old', 'new'}):
        raise ValueError('manifest must contain only mode, old and optionally new')
    if value['mode'] not in ('baseline', 'upgrade') or ('new' in value) != (value['mode'] == 'upgrade'):
        raise ValueError('mode does not match old/new tuples')
    for phase in ('old', 'new'):
        if phase not in value:
            continue
        entries = value[phase]
        if not isinstance(entries, dict) or set(entries) != set(REPOS):
            raise ValueError('each tuple must contain exactly the four fixed services')
        for service, item in entries.items():
            if not isinstance(item, dict) or set(item) != {'version', 'source', 'sha256', 'image'}:
                raise ValueError('release entry must provide version/source/sha256/image')
            patterns = {'version': r'[1-9][0-9]*\.[0-9]+\.[0-9]+', 'source': r'[0-9a-f]{40}',
                        'sha256': r'[0-9a-f]{64}', 'image': r'ghcr\.io/soulteary/' + REPOS[service] + r'@sha256:[0-9a-f]{64}'}
            if any(not isinstance(item[k], str) or re.fullmatch(p, item[k]) is None for k, p in patterns.items()):
                raise ValueError('invalid release identity')
    return value


def budget(seconds=20):
    if DEADLINE is None:
        return seconds
    remaining = DEADLINE - time.monotonic()
    if remaining <= 0:
        raise TimeoutError('whole verification deadline exceeded')
    return min(seconds, remaining)


def save():
    OUT.mkdir(exist_ok=True)
    (OUT / 'results.json').write_text(json.dumps(result, indent=2) + '\n')


def fetch(url, limit=MAX_DOWNLOAD, api=False):
    headers = {'User-Agent': 'stargate-release-compatibility', 'Accept': 'application/vnd.github+json' if api else '*/*'}
    if api and os.environ.get('GH_TOKEN'):
        headers['Authorization'] = 'Bearer ' + os.environ['GH_TOKEN']
    with urllib.request.urlopen(urllib.request.Request(url, headers=headers), timeout=budget(30)) as response:
        data = response.read(limit + 1)
    if len(data) > limit:
        raise ValueError('release response exceeds size limit')
    return data


def api(repo, path):
    return json.loads(fetch('https://api.github.com/repos/soulteary/' + repo + '/' + path, 4 * 1024 * 1024, True))


def download_tuple(phase, entries):
    paths = {}
    for service, entry in entries.items():
        repo = REPOS[service]
        tag = 'v' + entry['version']
        ref = api(repo, 'git/ref/tags/' + tag)['object']
        for _ in range(3):
            if ref['type'] == 'commit':
                break
            if ref['type'] != 'tag' or not re.fullmatch(r'[0-9a-f]{40}', ref['sha']):
                raise ValueError('invalid annotated tag target')
            ref = api(repo, 'git/tags/' + ref['sha'])['object']
        if ref['type'] != 'commit' or ref['sha'] != entry['source']:
            raise ValueError('formal tag source mismatch')
        release = api(repo, 'releases/tags/' + tag)
        if release['draft'] or release['prerelease'] or release['tag_name'] != tag:
            raise ValueError('a stable published release is required')
        name = repo + '-linux-amd64'
        names = [a['name'] for a in release['assets']]
        if names.count(name) != 1 or names.count('checksums.txt') != 1:
            raise ValueError('expected release assets absent or duplicated')
        base = 'https://github.com/soulteary/' + repo + '/releases/download/' + tag + '/'
        checksums = fetch(base + 'checksums.txt', 1024 * 1024).decode()
        matches = [line.split()[0] for line in checksums.splitlines() if len(line.split()) == 2 and line.split()[1].lstrip('*') == name]
        if matches != [entry['sha256']]:
            raise ValueError('pinned checksum and published manifest disagree')
        data = fetch(base + name)
        if hashlib.sha256(data).hexdigest() != entry['sha256']:
            raise ValueError('downloaded binary checksum mismatch')
        directory = OUT / phase / service
        directory.mkdir(parents=True, exist_ok=True)
        path = directory / name
        path.write_bytes(data)
        path.chmod(0o755)
        info = subprocess.check_output(['go', 'version', '-m', str(path)], text=True, timeout=budget(20))
        if 'GOOS=linux' not in info or 'GOARCH=amd64' not in info:
            raise ValueError('binary platform mismatch')
        if service in ('warden', 'gateway'):
            if '\tgithub.com/soulteary/' + repo + '\tv' + entry['version'] + '\t' not in info:
                raise ValueError('binary module version mismatch')
            if 'vcs.revision=' + entry['source'] not in info or 'vcs.modified=false' not in info:
                raise ValueError('binary source metadata mismatch')
        if service == 'totp' and ('Version=' + entry['version'] not in info or 'Commit=' + entry['source'] not in info):
            raise ValueError('TOTP embedded version/source mismatch')
        # Verified Herald 1.4.3 and 1.4.4 binaries omit VCS/version build metadata. Their source
        # binding is the pinned, separately verified asset hash plus formal tag.
        (directory / 'buildinfo.txt').write_text(info)
        (directory / 'checksums.txt').write_text(checksums)
        paths[service] = path.resolve()
        events.append({'check': 'formal binary identity', 'phase': phase, 'service': service,
                       **entry, 'release_id': release['id'], 'path': str(path),
                       'embedded_source_present': entry['source'] in info,
                       'embedded_version_present': entry['version'] in info,
                       'verification_scope': 'Published stable tag/source, pinned SHA, checksum manifest and Go metadata; signatures verified separately'})
        save()
    return paths


def download_templates(source, phase):
    data = fetch('https://api.github.com/repos/soulteary/stargate/tarball/' + source)
    target = OUT / phase / 'templates'
    target.mkdir(parents=True, exist_ok=True)
    count = total = 0
    with tarfile.open(fileobj=io.BytesIO(data), mode='r:gz') as archive:
        for item in archive:
            parts = Path(item.name).parts
            if len(parts) < 5 or parts[1:5] != ('src', 'internal', 'web', 'templates'):
                continue
            relative = Path(*parts[5:])
            if not item.isfile():
                if not item.isdir():
                    raise ValueError('nonregular template archive entry')
                continue
            if not relative.parts or relative.is_absolute() or '..' in relative.parts:
                raise ValueError('unsafe template archive path')
            count += 1
            total += item.size
            if count > 1000 or total > 16 * 1024 * 1024:
                raise ValueError('template archive too large')
            destination = target / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(archive.extractfile(item).read())
    if count == 0:
        raise ValueError('formal source templates missing')
    events.append({'check': 'matching formal template source', 'phase': phase, 'source': source,
                   'files': count, 'archive_sha256': hashlib.sha256(data).hexdigest()})
    return target


def resp(parts, db=0):
    with socket.create_connection(('127.0.0.1', 6379), budget(3)) as connection:
        stream = connection.makefile('rb')
        def command(values):
            encoded = [str(v).encode() for v in values]
            connection.sendall(b'*' + str(len(encoded)).encode() + b'\r\n' + b''.join(b'$' + str(len(v)).encode() + b'\r\n' + v + b'\r\n' for v in encoded))
            line = stream.readline()
            kind, value = line[:1], line[1:-2]
            if kind == b'+': return value.decode()
            if kind == b':': return int(value)
            if kind == b'-': raise RuntimeError(value.decode())
            if kind == b'$':
                size = int(value)
                if size < 0: return None
                data = stream.read(size)
                if stream.read(2) != b'\r\n': raise ValueError('incomplete Redis reply')
                return data.decode()
            raise ValueError('unexpected Redis response')
        if command(['SELECT', db]) != 'OK': raise ValueError('Redis SELECT failed')
        return command(parts)


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


http_observer = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())
key_w = 'synthetic-release-compat-warden-secret-123456'
key_h = 'synthetic-release-compat-herald-secret-123456'
key_t = 'synthetic-release-compat-totp-secret-123456'
test_key = 'synthetic-release-compat-test-key-123456'
SIGNERS = {key_w: ('warden', 'gw-warden-r8'), key_h: ('herald', 'gw-herald-r8'), key_t: ('totp', 'herald-totp-r8')}
warden, herald, totp, gateway = ['http://127.0.0.1:' + str(port) for port in (38081, 38082, 38084, 38080)]
browser_cookies = {}


def hmac_headers(secret,method,target,body,nonce_override=None,key_id_override=None):
 parsed=urllib.parse.urlsplit(target);path=parsed.path;query=parsed.query;ts=str(int(time.time()));nonce=nonce_override or secrets.token_hex(16);bodyhash=hashlib.sha256(body).hexdigest();service='stargate'
 if SIGNERS[secret][0]=='warden':
  kid=SIGNERS[secret][1] if key_id_override is None else key_id_override;version='v2';canonical='\n'.join([method,path+('?' +query if query else ''),kid,ts,nonce,bodyhash])
 elif SIGNERS[secret][0]=='herald':
  kid=SIGNERS[secret][1] if key_id_override is None else key_id_override;version='v3';canonical='\n'.join(['HERALD-HMAC-V3',method,path,query,ts,nonce,service,kid,hashlib.sha256(b'').hexdigest(),bodyhash])
 elif SIGNERS[secret][0]=='totp':
  kid=SIGNERS[secret][1] if key_id_override is None else key_id_override;version='v2';fields=['v2',method,path,query,kid,ts,service,nonce,bodyhash];canonical=''.join(str(len(f.encode()))+'\n'+f+'\n' for f in fields)
 else:raise AssertionError('unexpected signing credential')
 return {'X-Signature-Version':version,'X-Signature':hmac.new(secret.encode(),canonical.encode(),hashlib.sha256).hexdigest(),'X-Timestamp':ts,'X-Nonce':nonce,'X-Service':service,'X-Key-Id':kid}


def request(base,path,body=None,key=None,opener=None,nonce=None,key_id=None):
 headers={'Accept':'application/json'}
 data=None
 if body is not None:headers['Content-Type']='application/x-www-form-urlencoded' if isinstance(body,str) else 'application/json';data=body.encode() if isinstance(body,str) else json.dumps(body).encode()
 if key:headers.update(hmac_headers(key,'GET' if data is None else 'POST',path,data or b'',nonce,key_id))
 if opener and browser_cookies:headers['Cookie']='; '.join(k+'='+v for k,v in browser_cookies.items())
 headers['Host']=urllib.parse.urlsplit(base).netloc
 assert urllib.parse.urlsplit(base).hostname=='127.0.0.1' and urllib.parse.urlsplit(base).scheme=='http','observer targets only own loopback published ports'
 req=urllib.request.Request(base+path,data=data,headers=headers)
 try:response=http_observer.open(req,timeout=budget(10))
 except urllib.error.HTTPError as error:response=error
 with response:
  reply={'status':response.code,'body':response.read().decode(),'headers':dict(response.headers),'cookies':response.headers.get_all('Set-Cookie',[])}
 status=reply['status'];raw=reply['body'].encode();hdr=reply['headers']
 if opener:
  for cookie in reply['cookies']:
   parsed=http.cookies.SimpleCookie();parsed.load(cookie)
   for k,v in parsed.items():browser_cookies[k]=v.value
 try:value=json.loads(raw)
 except ValueError:value=raw.decode(errors='replace')
 return status,value,{k.lower():v for k,v in hdr.items()}


def otp(secret):
 key=base64.b32decode(secret+'='*((-len(secret))%8));digest=hmac.new(key,struct.pack('>Q',int(time.time())//30),hashlib.sha1).digest();offset=digest[-1]&15
 return str((struct.unpack('>I',digest[offset:offset+4])[0]&0x7fffffff)%1000000).zfill(6)


def expect(label, response, status):
    if response[0] != status:
        raise AssertionError(f'{label}: HTTP {response[0]} expected {status}')
    events.append({'check': label, 'status': status})
    save()
    return response[1]


def child_environment(env):
    # Repository/runner tokens are only for the parent download process.
    temporary = RUNTIME / 'tmp'
    temporary.mkdir(parents=True, exist_ok=True)
    return {'PATH': os.environ.get('PATH', '/usr/bin:/bin'), 'HOME': str(RUNTIME),
            'TMPDIR': str(temporary), 'LANG': 'C.UTF-8', 'TZ': 'UTC', **env}


def start(service, path, env, templates=None):
    if service in processes:
        raise ValueError('process already started')
    if templates is not None:
        destination = RUNTIME / 'src/internal/web/templates'
        shutil.rmtree(destination, ignore_errors=True)
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copytree(templates, destination)
    handle = (OUT / f'{service}-{len(events)}.log').open('wb')
    log_handles.append(handle)
    processes[service] = subprocess.Popen([str(path)], env=child_environment(env), cwd=RUNTIME,
                                          stdout=handle, stderr=handle, start_new_session=True)
    limit = time.monotonic() + budget(35)
    base = {'warden': warden, 'herald': herald, 'totp': totp, 'gateway': gateway}[service]
    endpoint = '/health' if service == 'warden' else '/healthz'
    while time.monotonic() < limit:
        if processes[service].poll() is not None:
            raise RuntimeError(service + ' exited during startup')
        try:
            if request(base, endpoint)[0] == 200:
                return
        except (OSError, urllib.error.URLError):
            pass
        time.sleep(.2)
    raise TimeoutError(service + ' health deadline exceeded')


def stop(service):
    process = processes[service]
    if process.poll() is not None:
        processes.pop(service)
        return
    os.killpg(process.pid, signal.SIGTERM)
    try:
        process.wait(8)
    except subprocess.TimeoutExpired:
        os.killpg(process.pid, signal.SIGKILL)
        process.wait(5)
    processes.pop(service)


def environments(prefix):
    common = {'OTLP_ENABLED': 'false', 'LOG_LEVEL': 'debug'}
    return {
        'warden': {**common, 'PORT': '38081', 'ENVIRONMENT': 'production', 'MERGE_MODE': 'ONLY_LOCAL', 'CONFIG': '',
                   'DATA_FILE': str(RUNTIME / 'users.json'), 'REDIS_ENABLED': 'true', 'REDIS': '127.0.0.1:6379',
                   'INTERVAL': '3600', 'SNAPSHOT_MAX_AGE': '1h', 'API_KEY': '', 'WARDEN_HMAC_KEYS': json.dumps({'gw-warden-r8': key_w})},
        'totp': {**common, 'PORT': ':38084', 'REDIS_ADDR': '127.0.0.1:6379', 'API_KEY': '',
                 'HERALD_TOTP_HMAC_KEYS': json.dumps({'herald-totp-r8': key_t}),
                 'HERALD_TOTP_ENCRYPTION_KEY': '0123456789abcdef0123456789abcdef',
                 'RATE_LIMIT_PER_SUBJECT': '100', 'RATE_LIMIT_PER_IP': '1000'},
        'herald': {**common, 'PORT': ':38082', 'ENVIRONMENT': 'test', 'REDIS_ADDR': '127.0.0.1:6379', 'API_KEY': '',
                   'REQUEST_AUTH_MODE': 'hmac_v2', 'HMAC_V1_ENABLED': 'false', 'HERALD_HMAC_KEYS': json.dumps({'gw-herald-r8': key_h}),
                   'HERALD_IDEMPOTENCY_SECRET': 'synthetic-chain-idempotency-secret-123456', 'AUDIT_ENABLED': 'false',
                   'HERALD_TEST_MODE': 'true', 'HERALD_TEST_API_KEY': test_key, 'HERALD_TEST_LISTENER_ADDR': '127.0.0.1:38083',
                   'HERALD_METRICS_API_KEY': 'synthetic-metrics-only-secret-123456', 'HERALD_TOTP_ENABLED': 'true',
                   'HERALD_TOTP_BASE_URL': totp, 'HERALD_TOTP_API_KEY': '', 'HERALD_TOTP_HMAC_SECRET': key_t,
                   'HERALD_TOTP_HMAC_KEY_ID': 'herald-totp-r8', 'HERALD_TOTP_HMAC_SERVICE': 'stargate',
                   'PROVIDER_FAILURE_POLICY': 'soft', 'SMTP_HOST': '', 'HERALD_SMTP_API_URL': '', 'HERALD_DINGTALK_API_URL': ''},
        'gateway': {**common, 'PORT': '38080', 'AUTH_HOST': '127.0.0.1:38080', 'COOKIE_SECURE': 'false', 'COOKIE_DOMAIN': '',
                    'SESSION_EXCHANGE_SECRET': 'synthetic-compat-exchange-secret-123456', 'PASSWORDS': 'plaintext:synthetic-unused-password',
                    'WARDEN_ENABLED': 'true', 'WARDEN_URL': warden, 'WARDEN_API_KEY': '', 'WARDEN_HMAC_SECRET': key_w,
                    'WARDEN_HMAC_KEY_ID': 'gw-warden-r8', 'WARDEN_CACHE_TTL': '1', 'HERALD_ENABLED': 'true', 'HERALD_URL': herald,
                    'HERALD_API_KEY': '', 'HERALD_HMAC_SECRET': key_h, 'HERALD_HMAC_KEY_ID': 'gw-herald-r8', 'HERALD_TOTP_ENABLED': 'true',
                    'SESSION_STORAGE_ENABLED': 'true', 'SESSION_STORAGE_REDIS_ADDR': '127.0.0.1:6379', 'SESSION_STORAGE_REDIS_DB': '1',
                    'SESSION_STORAGE_REDIS_KEY_PREFIX': prefix, 'LOGIN_SMS_ENABLED': 'false', 'LOGIN_EMAIL_ENABLED': 'true',
                    'RATE_LIMIT_LOGIN_MAX': '100', 'RATE_LIMIT_VERIFICATION_MAX': '100', 'AUTH_REFRESH_ENABLED': 'false',
                    'AUDIT_LOG_ENABLED': 'false', 'LANGUAGE': 'en'}}


def send(subject):
    response = expect('gateway sends bound OTP', request(gateway, '/_send_verify_code',
                      'mail=' + urllib.parse.quote(subject) + '&deliver_via=email', opener=True), 200)
    if not response['success']:
        raise AssertionError('OTP creation failed')
    req = urllib.request.Request('http://127.0.0.1:38083/v1/test/code/' + response['challenge_id'], headers={'X-Test-Api-Key': test_key})
    with http_observer.open(req, timeout=budget(5)) as reply:
        code = json.load(reply)
    return response['challenge_id'], code['code']


def login(subject, challenge, code):
    response = expect('gateway verifies bound OTP', request(gateway, '/_login', urllib.parse.urlencode({
        'auth_method': 'warden', 'mail': subject, 'challenge_id': challenge, 'verify_code': code}), opener=True), 200)
    if not response['success']:
        raise AssertionError('OTP login rejected')


def nonce_seed(service):
    base, path, secret = {'warden': (warden, '/user?mail=chain-a%40example.test', key_w),
                          'herald': (herald, '/v1/totp/status?subject=chain-user-a', key_h),
                          'totp': (totp, '/v1/status?subject=chain-user-a', key_t)}[service]
    nonce = secrets.token_hex(16)
    expect(service + ' accepts fresh nonce', request(base, path, key=secret, nonce=nonce), 200)
    if service == 'warden':
        redis_key = 'warden:hmac:replay:' + hashlib.sha256(('gw-warden-r8:' + nonce).encode()).hexdigest()
    elif service == 'herald':
        redis_key = 'otp:nonce:' + hmac.new(b'synthetic-chain-idempotency-secret-123456', ('stargate\x00gw-herald-r8\x00' + nonce).encode(), hashlib.sha256).hexdigest()
    else:
        redis_key = 'totp:auth:nonce:' + hmac.new(key_t.encode(), ('totp-auth-nonce:v2\n8\nstargate\n' + nonce).encode(), hashlib.sha256).hexdigest()
    if resp(['PTTL', redis_key]) <= 0:
        raise AssertionError('accepted nonce marker missing')
    return service, base, path, secret, nonce, redis_key


def replay_metric():
    body = expect('Warden replay metric', request(warden, '/metrics', key=key_w), 200)
    match = re.search(r'(?:^|\n)[^ #\n]*hmac_replay_rejected_total(?:\{[^}]*\})? ([0-9.eE+\-]+)', body)
    if not match:
        raise AssertionError('replay metric absent')
    return float(match.group(1))


def nonce_check(seed, stage):
    service, base, path, secret, nonce, redis_key = seed
    ttl = resp(['PTTL', redis_key])
    if ttl <= 0:
        raise AssertionError('transition exceeded nonce lifetime')
    expect(service + ' fresh nonce control', request(base, path, key=secret), 200)
    before = replay_metric() if service == 'warden' else None
    body = expect(service + ' retained nonce rejected', request(base, path, key=secret, nonce=nonce), 401)
    if service == 'herald' and body['reason'] != 'replayed_nonce':
        raise AssertionError('wrong replay rejection reason')
    if service == 'warden' and replay_metric() != before + 1:
        raise AssertionError('401 was not replay rejection')
    events.append({'check': 'fresh-timestamp replay proof', 'stage': stage, 'service': service, 'marker_ttl_ms': ttl})


def session_snapshot(prefix):
    token = next(urllib.parse.unquote(v) for v in browser_cookies.values() if urllib.parse.unquote(v).startswith('r1.'))
    key = prefix + 'revocable:v1:' + token
    ttl = resp(['PTTL', key], 1)
    if ttl <= 0:
        raise AssertionError('session record absent')
    return {'cookies': dict(browser_cookies), 'token': token, 'deadline_ms': int(token.split('.')[1], 36),
            'redis_key': key, 'observed_deadline': time.time() + ttl / 1000}


def assert_session(snapshot, stage):
    global browser_cookies
    previous = browser_cookies
    browser_cookies = dict(snapshot['cookies'])
    response = request(gateway, '/_auth', opener=True)
    expect('existing session survives ' + stage, response, 200)
    if response[2]['x-auth-user'] != 'chain-user-a' or browser_cookies != snapshot['cookies']:
        raise AssertionError('session identity or cookie changed')
    ttl = resp(['PTTL', snapshot['redis_key']], 1)
    if ttl <= 0 or time.time() + ttl / 1000 > snapshot['observed_deadline'] + 2:
        raise AssertionError('session expiry extended or lost')
    browser_cookies = previous



def directory_users():
    return [{'user_id': 'chain-user-' + suffix, 'mail': 'chain-' + suffix + '@example.test',
             'status': 'active', 'scope': ['read'], 'role': 'viewer'} for suffix in ('a', 'b', 'c', 'd')]


@contextmanager
def isolated_browser():
    global browser_cookies
    previous = browser_cookies
    browser_cookies = {}
    try:
        yield
    finally:
        browser_cookies = previous


def rotation_login(subject):
    # Each independent credential case gets one recipient and one fresh browser.
    # Preserve the pending USER_B browser and all server-side cooldown state.
    with isolated_browser():
        challenge, code = send(subject)
        login(subject, challenge, code)



def backup_hash(code):
    return hashlib.sha256(code.strip().replace('-', '').upper().encode()).hexdigest()


def backup_state(subject):
    key = 'totp:backup:' + subject
    raw = resp(['GET', key])
    if not isinstance(raw, str):
        raise AssertionError('backup state absent')
    entries = json.loads(raw)
    if not isinstance(entries, list) or len(entries) != 10:
        raise AssertionError('unexpected backup entry count')
    for entry in entries:
        if (set(entry) != {'code_hash', 'used_at'} or re.fullmatch(r'[0-9a-f]{64}', entry['code_hash']) is None
                or type(entry['used_at']) is not int or entry['used_at'] < 0):
            raise AssertionError('invalid backup record')
    ttl = resp(['PTTL', key])
    if ttl != -1:
        raise AssertionError('backup record must retain its persistent lifetime')
    summary = {'entry_count': len(entries), 'used_count': sum(e['used_at'] > 0 for e in entries),
               'used_at_by_index': [e['used_at'] for e in entries], 'record_sha256': hashlib.sha256(raw.encode()).hexdigest(),
               'key_ttl_ms': ttl}
    return raw, entries, summary


def assert_backup_unchanged(reference, stage):
    current = backup_state('chain-user-a')
    if current[0] != reference[0]:
        raise AssertionError('backup state mutated during ' + stage)
    events.append({'check': 'backup state preserved', 'stage': stage, **current[2]})


def consume_backup(code, challenge, label):
    body = expect(label, request(totp, '/v1/verify', {'subject': 'chain-user-a', 'code': code,
                  'challenge_id': challenge}, key_t), 200)
    if (body.get('ok') is not True or body.get('subject') != 'chain-user-a'
            or len(body.get('amr', [])) != 2 or set(body['amr']) != {'totp', 'backup_code'}):
        raise AssertionError('backup verification did not use the backup-code branch')


def rejected_backup(code, challenge, label, status, reason):
    body = expect(label, request(totp, '/v1/verify', {'subject': 'chain-user-a', 'code': code,
                  'challenge_id': challenge}, key_t), status)
    if body.get('ok') is not False or body.get('reason') != reason:
        raise AssertionError('wrong backup rejection branch')



def replay_marker_ttl(challenge):
    ttl = resp(['PTTL', 'totp:ch_used:' + challenge])
    if type(ttl) is not int or (ttl < 0 and ttl != -2):
        raise AssertionError('unexpected persistent or invalid challenge replay marker')
    return ttl


def run(manifest):
    global browser_cookies
    paths = {phase: download_tuple(phase, manifest[phase]) for phase in ('old', 'new') if phase in manifest}
    templates = {phase: download_templates(manifest[phase]['gateway']['source'], phase) for phase in paths}
    for port in (38080, 38081, 38082, 38083, 38084):
        with socket.socket() as probe:
            probe.bind(('127.0.0.1', port))
    if resp(['PING']) != 'PONG':
        raise AssertionError('real Redis unavailable')
    prefix = 'release-compat:' + secrets.token_hex(8) + ':'
    users = directory_users()
    RUNTIME.mkdir(parents=True, exist_ok=True)
    (RUNTIME / 'users.json').write_text(json.dumps(users))
    env = environments(prefix)
    result['configuration'] = {'business_api_keys_empty': True, 'distinct_hmac_credentials': True,
                               'Redis_lifetime': 'one workflow service process for all stages', 'session_prefix': prefix}
    for service in ORDER:
        start(service, paths['old'][service], env[service], templates['old'] if service == 'gateway' else None)
    challenge_a, code_a = send('chain-a@example.test')
    login('chain-a@example.test', challenge_a, code_a)
    snapshot = session_snapshot(prefix)
    result['original_session'] = snapshot
    enrollment = expect('real TOTP enrollment start', request(herald, '/v1/totp/enroll/start', {'subject': 'chain-user-a'}, key_h), 200)
    secret = urllib.parse.parse_qs(urllib.parse.urlsplit(enrollment['otpauth_uri']).query)['secret'][0]
    confirmed = expect('real TOTP enrollment confirm', request(herald, '/v1/totp/enroll/confirm',
                       {'enroll_id': enrollment['enroll_id'], 'code': otp(secret)}, key_h), 200)
    if not confirmed['totp_enabled']:
        raise AssertionError('credential not enabled')
    codes = confirmed['backup_codes']
    if len(codes) != 10 or not all(isinstance(code, str) for code in codes):
        raise AssertionError('enrollment backup codes missing')
    used_backup, spare_backup = codes[:2]
    before_backup = backup_state('chain-user-a')
    if before_backup[2]['used_count'] != 0:
        raise AssertionError('new enrollment already contains consumed backups')
    used_index = next(i for i, entry in enumerate(before_backup[1]) if entry['code_hash'] == backup_hash(used_backup))
    spare_index = next(i for i, entry in enumerate(before_backup[1]) if entry['code_hash'] == backup_hash(spare_backup))
    backup_challenge = prefix + 'backup-first'
    consume_backup(used_backup, backup_challenge, 'consume initial backup code')
    retained_backup = backup_state('chain-user-a')
    if retained_backup[2]['used_count'] != 1 or retained_backup[1][used_index]['used_at'] <= 0:
        raise AssertionError('initial backup consumption not persisted')
    expected = [dict(entry) for entry in before_backup[1]]
    expected[used_index]['used_at'] = retained_backup[1][used_index]['used_at']
    if retained_backup[1] != expected:
        raise AssertionError('initial backup consumption changed unrelated entries')
    result['backup_state_before_rollout'] = retained_backup[2]
    result['backup_quota_profile'] = {'subject_limit': 100, 'IP_limit': 1000, 'state_reset': False}
    # Enrollment confirmation consumes its TOTP counter. Use a fresh next-step
    # code to seed an explicit challenge replay marker that survives each stage.
    step = int(time.time()) // 30
    while int(time.time()) // 30 <= step:
        budget()
        time.sleep(.2)
    totp_proof = {'subject': 'chain-user-a', 'code': otp(secret), 'challenge_id': prefix + 'totp-proof'}
    verified = expect('real TOTP verification', request(totp, '/v1/verify', totp_proof, key_t), 200)
    if not verified['ok']:
        raise AssertionError('fresh TOTP rejected')
    # A consumed OTP is retained as proof throughout subsequent process changes.
    verification = {'challenge_id': challenge_a, 'code': code_a, 'expected_user_id': 'chain-user-a',
                    'expected_purpose': 'login', 'expected_channel': 'email'}
    browser_cookies = {}
    pending_id, pending_code = send('chain-b@example.test')
    binding_key = prefix + 'challenge:context:' + pending_id
    if resp(['EXISTS', binding_key], 1) != 1:
        raise AssertionError('pending binding absent')
    binding_deadline = time.time() + resp(['PTTL', binding_key], 1) / 1000
    result['pending_binding_deadline'] = binding_deadline
    stages = [('baseline', None, 'old')]
    if manifest['mode'] == 'upgrade':
        stages += [('upgrade', service, 'new') for service in ORDER]
        stages += [('rollback', service, 'old') for service in reversed(ORDER)]
    for direction, service, phase in stages:
        stage = direction + (' ' + service if service else '')
        seeds = [nonce_seed(name) for name in ('warden', 'herald', 'totp')]
        if service:
            stop(service)
            start(service, paths[phase][service], env[service], templates[phase] if service == 'gateway' else None)
        for seed in seeds:
            nonce_check(seed, stage)
        assert_session(snapshot, stage)
        status = expect('TOTP credential survives ' + stage, request(herald, '/v1/totp/status?subject=chain-user-a', key=key_h), 200)
        if not status['totp_enabled']:
            raise AssertionError('credential lost')
        fresh_backup_challenge = prefix + 'backup-reuse-' + stage.replace(' ', '-')
        rejected_backup(used_backup, fresh_backup_challenge, 'used backup with fresh challenge ' + stage, 401, 'invalid')
        if resp(['EXISTS', 'totp:ch_used:' + fresh_backup_challenge]) != 0:
            raise AssertionError('failed backup reuse claimed a fresh challenge')
        assert_backup_unchanged(retained_backup, stage)
        marker_ttl = replay_marker_ttl(backup_challenge)
        if marker_ttl > 0:
            rejected_backup(spare_backup, backup_challenge, 'same consumed challenge rejects spare backup ' + stage, 400, 'replay')
            assert_backup_unchanged(retained_backup, 'same-challenge rejection ' + stage)
        else:
            events.append({'check': 'same-challenge marker lifetime', 'stage': stage, 'marker_ttl_ms': marker_ttl,
                           'scope': 'marker missing or expiring; durable fresh-challenge used-code rejection still checked'})
        totp_replay = expect('consumed TOTP challenge rejected ' + stage, request(totp, '/v1/verify', totp_proof, key_t), 400)
        if totp_replay['reason'] != 'replay':
            raise AssertionError('wrong TOTP replay branch')
        rejected = expect('consumed OTP rejected ' + stage, request(herald, '/v2/otp/verifications', verification, key_h), 401)
        if rejected['reason'] != 'expired':
            raise AssertionError('wrong consumed-OTP rejection')
        binding_ttl = resp(['PTTL', binding_key], 1)
        if binding_ttl <= 0 or time.time() + binding_ttl / 1000 > binding_deadline + 2:
            raise AssertionError('pending OTP binding lost or extended during rollout')
        events.append({'check': 'version stage', 'stage': stage, 'product_source': manifest[phase][service]['source'] if service else None})
        if direction == 'upgrade' and service == 'gateway':
            rotate_credentials(paths['new'], templates['new'], env, snapshot, 'new formal tuple')
    if manifest['mode'] == 'baseline':
        rotate_credentials(paths['old'], templates['old'], env, snapshot, 'old formal baseline tuple')
    assert_backup_unchanged(retained_backup, 'completed rollback and key configuration phases')
    consume_backup(spare_backup, prefix + 'backup-after-rollback', 'consume retained spare backup after rollback')
    after_backup = backup_state('chain-user-a')
    expected = [dict(entry) for entry in retained_backup[1]]
    expected[spare_index]['used_at'] = after_backup[1][spare_index]['used_at']
    if (after_backup[1] != expected or after_backup[2]['used_count'] != 2
            or after_backup[1][spare_index]['used_at'] <= 0):
        raise AssertionError('spare backup consumption changed unrelated state')
    rejected_backup(spare_backup, prefix + 'backup-after-rollback-reuse', 'retained spare remains single-use', 401, 'invalid')
    assert_backup_unchanged(after_backup, 'post-rollback spare reuse')
    result['backup_state_after_rollback'] = after_backup[2]
    login('chain-b@example.test', pending_id, pending_code)
    if resp(['EXISTS', binding_key], 1) != 0:
        raise AssertionError('consumed binding not removed')
    result['status'] = 'passed'
    result['limits'] = ['No production traffic or real delivery provider.', 'Formal native binaries executed; OCI references/signatures handled separately.',
                        'Nonce checks use new seeds before each adjacent transition and freshly signed current timestamps, within positive TTL.',
                        'Code-version stages preserve credentials. A separate labeled stage changes synthetic keys through overlap, retirement and explicit configuration rollback.']



def rotate_credentials(paths, templates, original_env, snapshot, code_tuple):
    """Explicit configuration rotation, separate from code-version replacement."""
    env = {name: dict(values) for name, values in original_env.items()}
    old = {'warden': (key_w, 'gw-warden-r8'), 'herald': (key_h, 'gw-herald-r8'), 'totp': (key_t, 'herald-totp-r8')}
    fresh = {name: ('synthetic-rotated-' + name + '-secret-123456789', 'rotated-' + name) for name in old}
    for name, (secret, kid) in fresh.items():
        SIGNERS[secret] = (name, kid)
    map_names = {'warden': 'WARDEN_HMAC_KEYS', 'herald': 'HERALD_HMAC_KEYS', 'totp': 'HERALD_TOTP_HMAC_KEYS'}
    for name in old:
        env[name][map_names[name]] = json.dumps({old[name][1]: old[name][0], fresh[name][1]: fresh[name][0]})
    endpoints = {'warden': (warden, '/user?mail=chain-a%40example.test'),
                 'herald': (herald, '/v1/totp/status?subject=chain-user-a'),
                 'totp': (totp, '/v1/status?subject=chain-user-a')}
    def restart_all(stage):
        for name in ORDER:
            stop(name)
            start(name, paths[name], env[name], templates if name == 'gateway' else None)
        assert_session(snapshot, stage)
    restart_all('key overlap')
    for name, (base, path) in endpoints.items():
        expect(name + ' old key during overlap', request(base, path, key=old[name][0]), 200)
        expect(name + ' new key during overlap', request(base, path, key=fresh[name][0]), 200)
    # Switch callers only after both old and new server credentials work.
    env['herald'].update(HERALD_TOTP_HMAC_SECRET=fresh['totp'][0], HERALD_TOTP_HMAC_KEY_ID=fresh['totp'][1])
    env['gateway'].update(WARDEN_HMAC_SECRET=fresh['warden'][0], WARDEN_HMAC_KEY_ID=fresh['warden'][1],
                          HERALD_HMAC_SECRET=fresh['herald'][0], HERALD_HMAC_KEY_ID=fresh['herald'][1])
    for name in ('herald', 'gateway'):
        stop(name)
        start(name, paths[name], env[name], templates if name == 'gateway' else None)
    assert_session(snapshot, 'rotated caller configuration')
    # Real SDK traffic selects the rotated keys for all three legs.
    rotation_login(ROTATION_SUBJECTS[0])
    status = expect('rotated Herald SDK reaches TOTP', request(herald, '/v1/totp/status?subject=chain-user-a', key=fresh['herald'][0]), 200)
    if not status['totp_enabled']:
        raise AssertionError('TOTP credential lost on key rotation')
    for name in old:
        env[name][map_names[name]] = json.dumps({fresh[name][1]: fresh[name][0]})
    restart_all('old key retirement')
    for name, (base, path) in endpoints.items():
        expect(name + ' retired key with fresh timestamp rejected', request(base, path, key=old[name][0]), 401)
        expect(name + ' active key after retirement', request(base, path, key=fresh[name][0]), 200)
    rotation_login(ROTATION_SUBJECTS[1])
    # Only an explicit configuration rollback restores the old keys. Merely
    # replacing an executable is never described as restoring retired secrets.
    env = {name: dict(values) for name, values in original_env.items()}
    restart_all('explicit key configuration rollback')
    for name, (base, path) in endpoints.items():
        expect(name + ' restored configured key', request(base, path, key=old[name][0]), 200)
        expect(name + ' removed rotated key rejected', request(base, path, key=fresh[name][0]), 401)
    events.append({'check': 'three-leg key rotation', 'phases': ['overlap', 'caller switch', 'retirement', 'explicit configuration rollback'],
                   'code_tuple': code_tuple, 'secret_values_recorded': False})


def main():
    global DEADLINE
    DEADLINE = time.monotonic() + 18 * 60
    def alarm_handler(_signal, _frame):
        raise TimeoutError('whole verification deadline exceeded')
    signal.signal(signal.SIGALRM, alarm_handler)
    signal.alarm(18 * 60)
    OUT.mkdir(exist_ok=True)
    if (OUT / 'results.json').exists():
        raise ValueError('refusing to overwrite previous evidence')
    try:
        raw = os.environ.get('COMPAT_MANIFEST_JSON', '')
        if len(raw) > 32 * 1024:
            raise ValueError('manifest too large')
        manifest = validate_manifest(json.loads(raw) if raw else json.loads((Path(__file__).parent / 'manifests/baseline.json').read_text()))
        result['manifest'] = manifest
        save()
        run(manifest)
    except Exception as error:
        result.update(status='failed', failure=repr(error))
        raise
    finally:
        signal.alarm(0)
        cleanup_errors = []
        for service in list(reversed(processes)):
            try:
                stop(service)
            except Exception as error:
                cleanup_errors.append({'service': service, 'error': repr(error)})
        for handle in log_handles:
            handle.close()
        result['cleanup'] = {'status': 'passed' if not cleanup_errors and not processes else 'failed',
                             'remaining_child_processes': list(processes), 'errors': cleanup_errors,
                             'Redis': 'GitHub Actions owns removal of its ephemeral service container and network'}
        if cleanup_errors:
            result['status'] = 'failed'
        save()
        if cleanup_errors:
            raise RuntimeError('child process cleanup failed')


if __name__ == '__main__':
    main()
