#!/usr/bin/env python3
"""Root-invoked, fixed-source local-only integration. No implicit invocation."""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import tarfile
import time
import urllib.request
import urllib.parse

SOURCES = {
    'sg': ('6f04912db1dafa953d244e3f7350bd9ead94cc80', 'a36ef6feec9a32410e290eb63bb1a9b831ecaf66', './src/internal/handlers', 'stargate_formal_bridge_test.go', 'src/internal/handlers/formal_three_bridge_test.go', 'TestFormalThreeServiceBridge'),
    'herald': ('a4e611f3763d670adec310d222cefed33930765e', '1b031fbaae91bff03a1d25f7a880160b2d2b9c8f', './internal/router', 'herald_formal_bridge_test.go', 'internal/router/formal_three_bridge_test.go', 'TestFormalHeraldServe'),
    'adapter': ('47c2d029f8c4f2a27c291f1b36b318e2de41c23f', 'ed729f80b5b6eaf6affc706f4fcd609faf457270', './internal/router', 'adapter_formal_bridge_test.go', 'internal/router/formal_three_bridge_test.go', 'TestFormalAdapterServe'),
}
EXPECTED_MANIFEST_SHA256 = 'cdd899c07caac2f0af5881019066ba8d4cc28533bd357e9d9f7bfc57ecf821a7'
EXPECTED_OVERLAYS = {'sg': 'cad04e8c64a9ac490c42b9dd1a8b3e153dad6450c30a6a2ea0ec05ebf8137bda', 'herald': 'fbc4263869a2bf8645d9b76a5b307d031a39aeaaeff83708016013326e4e7d6b', 'adapter': '78a7727f2af4a8671fa61045225dbe75a10db66e373f78e8857953e013196be2'}
SUBTESTS = ('opaque_mapping_and_client_injection', 'same_request_idempotent', 'wrong_user_cannot_consume', 'owner_session_and_consumed_replay', 'unmapped_user_no_delivery', 'adapter_auth_and_idempotency', 'provider_failure_refuses_login_challenge')
class Refusal(Exception):
    pass

def need(ok):
    if not ok:
        raise Refusal('fixed fixture gate refused')

def sha(raw):
    return hashlib.sha256(raw).hexdigest()

def read_regular(path, limit):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    try:
        a = os.fstat(fd)
        need(stat.S_ISREG(a.st_mode) and a.st_nlink == 1 and a.st_size <= limit)
        raw = bytearray()
        while len(raw) < a.st_size:
            block = os.read(fd, min(65536, a.st_size-len(raw)))
            need(bool(block))
            raw.extend(block)
        need(os.read(fd, 1) == b'')
        stamp = lambda s: (s.st_dev,s.st_ino,s.st_mode,s.st_uid,s.st_gid,s.st_nlink,s.st_size,s.st_mtime_ns,s.st_ctime_ns)
        need(stamp(a) == stamp(os.fstat(fd)) == stamp(os.lstat(path)))
        return bytes(raw)
    finally:
        os.close(fd)

def write_new(path, raw):
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'wb') as f:
        f.write(raw)
        f.flush()
        os.fsync(f.fileno())

def record(root, name, value):
    write_new(root / (name+'.json'), (json.dumps(value,sort_keys=True,indent=2)+'\n').encode())

def exact_pass(raw, top, children=()):
    text = raw.decode('utf-8', 'strict')
    names = re.findall(r'^--- PASS: (\S+) \(',text,re.M)
    names += re.findall(r'^    --- PASS: (\S+) \(',text,re.M)
    expected = [top]+[top+'/'+x for x in children]
    return all(names.count(x)==1 for x in expected) and len(names)==len(expected) and text.splitlines().count('PASS')==1 and not re.search(r'--- (?:FAIL|SKIP):|^FAIL(?:\s|$)',text,re.M)

def endpoint(raw):
    value=raw.decode('ascii');u=urllib.parse.urlsplit(value)
    need(u.scheme=='http' and u.hostname=='127.0.0.1' and u.username is None and u.password is None and u.path=='' and not u.query and not u.fragment and u.port is not None and 0<u.port<65536)
    need(value=='http://127.0.0.1:'+str(u.port))
    return value

def git(repo, *args):
    # Explicit fixed public object reads; no fetch/submodule/smudge hooks.
    env={'PATH':'/usr/bin:/bin','GIT_TERMINAL_PROMPT':'0','GIT_CONFIG_NOSYSTEM':'1','GIT_CONFIG_GLOBAL':'/dev/null','GIT_NO_LAZY_FETCH':'1','GIT_NO_REPLACE_OBJECTS':'1'}
    p=subprocess.run(['/usr/bin/git','-C',str(repo),'-c','core.attributesFile=/dev/null',*args],env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=30)
    need(p.returncode==0)
    return p.stdout

def expected_inputs():
    raw = read_regular(Path(__file__).parent/'source-inputs.json', 1024*1024)
    need(sha(raw) == EXPECTED_MANIFEST_SHA256)
    value = json.loads(raw)
    need(set(value) == set(SOURCES))
    for name, row in value.items():
        need(row['commit'] == SOURCES[name][0] and row['tree'] == SOURCES[name][1])
    return value

def validate_archive(raw, expected):
    need(len(raw) == expected['archive_size'] and sha(raw) == expected['archive_sha256'])
    actual = {}
    with tarfile.open(fileobj=io.BytesIO(raw), mode='r:') as archive:
        for row in archive:
            if row.name in expected['internal_source_and_test_git_blob_sha256']:
                need(row.isfile() and row.name not in actual)
                actual[row.name] = sha(archive.extractfile(row).read())
    need(actual == expected['internal_source_and_test_git_blob_sha256'])

def materialize(repo, name, out, expected):
    rev,tree,_,overlay,destination,_ = SOURCES[name]
    need(git(repo,'rev-parse',rev+'^{tree}').decode().strip()==tree)
    raw=git(repo,'archive','--format=tar',rev)
    need(len(raw)<=128*1024*1024)
    validate_archive(raw, expected)
    target=out/name;target.mkdir(mode=0o700)
    seen=set();total=0
    with tarfile.open(fileobj=io.BytesIO(raw),mode='r:') as archive:
        for row in archive:
            p=Path(row.name)
            need(not p.is_absolute() and all(x not in ('..','.git') for x in p.parts) and row.name not in seen)
            seen.add(row.name);need(len(seen)<=20000)
            dest=target/p
            if row.isdir():
                dest.mkdir(mode=0o700,parents=True,exist_ok=True)
                continue
            need(row.isfile() and row.size<=16*1024*1024)
            total+=row.size;need(total<=128*1024*1024)
            dest.parent.mkdir(mode=0o700,parents=True,exist_ok=True)
            f=archive.extractfile(row);need(f is not None);data=f.read(row.size+1);need(len(data)==row.size)
            write_new(dest,data)
    overlay_bytes=read_regular(Path(__file__).parent/overlay,128*1024)
    need(sha(overlay_bytes) == EXPECTED_OVERLAYS[name])
    write_new(target/destination,overlay_bytes)
    # Existing module/source is unchanged; only one new test per module.
    return target,{'commit':rev,'tree':tree,'archive_sha256':sha(raw),'overlay_sha256':sha(overlay_bytes),'go_mod_sha256':sha(read_regular(target/'go.mod',1024*1024)),'go_sum_sha256':sha(read_regular(target/'go.sum',4*1024*1024))}

def child(root,name,argv,cwd,env):
    stdout=open(root/(name+'.stdout.private'),'xb');stderr=open(root/(name+'.stderr.private'),'xb')
    os.chmod(stdout.name,0o600);os.chmod(stderr.name,0o600)
    try:
        p=subprocess.Popen(argv,cwd=cwd,env=env,stdin=subprocess.DEVNULL,stdout=stdout,stderr=stderr)
    finally:
        stdout.close();stderr.close()
    record(root,name+'-started',{'pid':p.pid,'argv':argv,'state':'started-not-terminal'})
    return p

def terminal(root,name,p,budget):
    try:
        code=p.wait(timeout=budget)
    except subprocess.TimeoutExpired:
        record(root,name+'-unknown',{'pid':p.pid,'state':'unknown-retained','kill_requested':False,'retry':False})
        raise Refusal('owned child terminal unknown')
    record(root,name+'-terminal',{'pid':p.pid,'state':'terminal-known','exit_code':code})
    return code

def wait_ready(root,name,p):
    path=root/(name+'.ready.private');deadline=time.monotonic()+15
    while time.monotonic()<deadline:
        need(p.poll() is None)
        if path.exists():
            return endpoint(read_regular(path,128))
        time.sleep(.02)
    raise Refusal('owned readiness absent')

def stop(url):
    r=urllib.request.Request(url+'/_fixture/stop',data=b'',method='POST',headers={'X-Fixture-Key':'synthetic-fixture-control'})
    opener=urllib.request.build_opener(urllib.request.ProxyHandler({}))
    with opener.open(r,timeout=3) as response:
        need(response.status==204)

def main():
    ap=argparse.ArgumentParser()
    for name in SOURCES:ap.add_argument('--'+name+'-source',type=Path,required=True)
    ap.add_argument('--go',type=Path,required=True);ap.add_argument('--go-mod-cache',type=Path,required=True);ap.add_argument('--output',type=Path,required=True)
    args=ap.parse_args();out=args.output
    need(out.is_absolute() and not out.exists() and out.parent.resolve()==out.parent and args.go.is_absolute())
    out.mkdir(mode=0o700)
    record(out,'intent',{'scope':'synthetic-three-native-components','source_commits':{n:v[0] for n,v in SOURCES.items()},'real_IM':False,'deployment':False,'Warden_runtime':False,'Redis_migration':False})
    try:
        # No inherited configuration, credentials, proxies, GOFLAGS, workspace or toolchain download.
        env={'PATH':str(args.go.parent)+':/usr/bin:/bin','GOPROXY':'off','GOSUMDB':'off','GOTOOLCHAIN':'local','GOENV':'off','GOWORK':'off','GOMODCACHE':str(args.go_mod_cache),'GOCACHE':str(out/'go-cache'),'TMPDIR':str(out),'LANG':'C','TZ':'UTC'}
        version=subprocess.run([str(args.go),'version'],env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=10)
        need(version.returncode==0 and version.stdout.decode().split()[2]=='go1.27.3')
        expected = expected_inputs()
        copies={};pins={}
        for name in SOURCES:copies[name],pins[name]=materialize(getattr(args,name+'_source'),name,out,expected[name])
        record(out,'source-inputs',pins)
        bins={}
        for name,(_,_,package,_,_,_) in SOURCES.items():
            bins[name]=out/(name+'.test')
            p=child(out,name+'-build',[str(args.go),'test','-mod=readonly','-c','-o',str(bins[name]),package],copies[name],env)
            need(terminal(out,name+'-build',p,180)==0)
        adapter=child(out,'adapter',[str(bins['adapter']),'-test.v','-test.run=^TestFormalAdapterServe$'],copies['adapter'],dict(env,FORMAL_ADAPTER_READY=str(out/'adapter.ready.private')))
        adapter_url=wait_ready(out,'adapter',adapter)
        herald=child(out,'herald',[str(bins['herald']),'-test.v','-test.run=^TestFormalHeraldServe$'],copies['herald'],dict(env,FORMAL_ADAPTER_URL=adapter_url,FORMAL_HERALD_READY=str(out/'herald.ready.private')))
        herald_url=wait_ready(out,'herald',herald)
        sg=child(out,'sg',[str(bins['sg']),'-test.v','-test.run=^TestFormalThreeServiceBridge$'],copies['sg'],dict(env,FORMAL_ADAPTER_URL=adapter_url,FORMAL_HERALD_URL=herald_url))
        sg_code=terminal(out,'sg',sg,60)
        # Only known completion permits orderly own fixture shutdown. No signal/kill/restart.
        stop(herald_url);herald_code=terminal(out,'herald',herald,10)
        stop(adapter_url);adapter_code=terminal(out,'adapter',adapter,10)
        checks={}
        for name,code in [('sg',sg_code),('herald',herald_code),('adapter',adapter_code)]:
            raw=read_regular(out/(name+'.stdout.private'),4*1024*1024)
            stderr=read_regular(out/(name+'.stderr.private'),4*1024*1024)
            checks[name]={'exit_code':code,'stdout_sha256':sha(raw),'stderr_sha256':sha(stderr),'exact_pass':exact_pass(raw,SOURCES[name][5],SUBTESTS if name=='sg' else ())}
        record(out,'native-results',checks)
        need(all(x['exit_code']==0 and x['exact_pass'] for x in checks.values()))
        record(out,'result',{'passed':True,'scope':'native-three-service-synthetic-transport','Warden_runtime':False,'Redis_migration':False,'real_IM':False,'deployment':False,'source_inputs':pins,'components':checks})
    except Exception:
        record(out,'failure',{'passed':False,'state':'failed-or-unknown-retained','raw_private_logs_retained':True,'automatic_kill_retry_cleanup':False})
        raise Refusal('fixed local fixture refused; private evidence retained') from None

if __name__=='__main__':
    try:main()
    except Exception:
        print('fixed local fixture refused; inspect owned private evidence',file=__import__('sys').stderr)
        raise SystemExit(1)
