#!/usr/bin/env python3
"""Fixed, local-only old/new native namespace regression; never clears Redis."""
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

# Filled from independently readable public fixed Git object bytes, not runtime.
MANIFEST_SHA256 = 'a61dbb841c9ae6baa51c067238a001a6953631a096b557521a7cecdaab57d1a5'
OVERLAYS = {'redis_transition_test.go': 'c32f53cd14810d4b3e8fae53b5f4fe455291ed1fcc03a2dcb15e4c350ddfc3fb', 'session_transition_test.go': '048c89db6d3a268f39244efa15a383f0ada19147e33dbd569db53735ba87834b', 'otp_transition_test.go': 'bcf2c9bd3d1ce4b6cd0e3f91c62dc6c58e80f8adfa46241ecd88e2b465ea2665'}
SOURCES = {
 'sg_old':('791b6ccaf506f3197b19f7b406f6e56be668b116','./src/cmd/stargate',[('session_transition_test.go','src/cmd/stargate/namespace_transition_test.go')]),
 'sg_new':('6f04912db1dafa953d244e3f7350bd9ead94cc80','./src/cmd/stargate',[('session_transition_test.go','src/cmd/stargate/namespace_transition_test.go'),('redis_transition_test.go','src/cmd/stargate/namespace_redis_test.go')]),
 'herald_old':('0f90a5ec6d4cb557cb16e0ec70feedb9d5c7bf90','./internal/handlers',[('otp_transition_test.go','internal/handlers/namespace_transition_test.go')]),
 'herald_new':('a4e611f3763d670adec310d222cefed33930765e','./internal/handlers',[('otp_transition_test.go','internal/handlers/namespace_transition_test.go')]),
}
PHASES=(('sg_old','old-issue'),('sg_new','new-isolate'),('sg_old','unsafe-old-prefix-witness'),('sg_old','rollback-fresh'),('sg_new','new-after-rollback'),('herald_old','old-issue'),('herald_new','new-reject-old'),('herald_old','unsafe-old-verifier-witness'),('clock','advance-301-seconds'),('herald_old','old-expired'),('herald_new','new-issue'),('herald_old','old-reject-new'),('herald_new','new-verify'))
class Refusal(Exception):pass

def need(value):
 if not value:raise Refusal('fixed namespace fixture refused')
def sha(value):return hashlib.sha256(value).hexdigest()
def stamp(s):return(s.st_dev,s.st_ino,s.st_mode,s.st_uid,s.st_gid,s.st_nlink,s.st_size,s.st_mtime_ns,s.st_ctime_ns)
def capture(path,limit):
 fd=os.open(path,os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK)
 try:
  a=os.fstat(fd);need(stat.S_ISREG(a.st_mode) and a.st_nlink==1 and 0<=a.st_size<=limit);raw=bytearray()
  while len(raw)<a.st_size:
   block=os.read(fd,min(65536,a.st_size-len(raw)));need(bool(block));raw.extend(block)
  need(os.read(fd,1)==b'' and stamp(a)==stamp(os.fstat(fd))==stamp(os.lstat(path)));return bytes(raw)
 finally:os.close(fd)
def write(path,value):
 fd=os.open(path,os.O_WRONLY|os.O_CREAT|os.O_EXCL|os.O_NOFOLLOW,0o600)
 with os.fdopen(fd,'wb') as f:f.write(value);f.flush();os.fsync(f.fileno())
def record(root,name,value):write(root/(name+'.json'),(json.dumps(value,sort_keys=True,indent=2)+'\n').encode())
def expected():
 raw=capture(Path(__file__).with_name('source-inputs.json'),1024*1024);need(sha(raw)==MANIFEST_SHA256);v=json.loads(raw);need(set(v)==set(SOURCES))
 for name,row in v.items():need(row['commit']==SOURCES[name][0])
 return v

def source_git(repo,*args):
 env={'PATH':'/usr/bin:/bin','GIT_CONFIG_GLOBAL':'/dev/null','GIT_CONFIG_NOSYSTEM':'1','GIT_TERMINAL_PROMPT':'0','GIT_NO_LAZY_FETCH':'1','GIT_NO_REPLACE_OBJECTS':'1'}
 p=subprocess.run(['/usr/bin/git','-C',str(repo),'-c','core.attributesFile=/dev/null',*args],env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=30);need(p.returncode==0);return p.stdout

def materialize(repo,name,out,pin):
 rev,_,overlays=SOURCES[name];need(source_git(repo,'rev-parse',rev+'^{tree}').decode().strip()==pin['tree'])
 raw=source_git(repo,'archive','--format=tar',rev);need(len(raw)==pin['archive_size'] and sha(raw)==pin['archive_sha256'])
 dst=out/name;dst.mkdir(mode=0o700);seen=set();actual={};total=0
 with tarfile.open(fileobj=io.BytesIO(raw),mode='r:') as archive:
  for row in archive:
   path=Path(row.name);need(not path.is_absolute() and all(x not in ('..','.git') for x in path.parts) and row.name not in seen);seen.add(row.name);need(len(seen)<=20000)
   p=dst/path
   if row.isdir():p.mkdir(mode=0o700,parents=True,exist_ok=True);continue
   need(row.isfile() and row.size<=16*1024*1024);total+=row.size;need(total<=128*1024*1024);data=archive.extractfile(row).read(row.size+1);need(len(data)==row.size)
   if row.name in pin['blobs']:actual[row.name]=sha(data)
   p.parent.mkdir(mode=0o700,parents=True,exist_ok=True);write(p,data)
 need(actual==pin['blobs'])
 for filename,path in overlays:
  raw=capture(Path(__file__).with_name(filename),128*1024);need(sha(raw)==OVERLAYS[filename]);write(dst/path,raw)
 return dst

def exact_pass(raw,name):
 text=raw.decode('utf-8','strict');names=re.findall(r'^--- PASS: (\S+) \(',text,re.M)
 return names==[name] and text.splitlines().count('PASS')==1 and not re.search(r'--- (?:SKIP|FAIL):|^FAIL(?:\s|$)',text,re.M)
def start(root,name,argv,cwd,env):
 with open(root/(name+'.stdout.private'),'xb') as out,open(root/(name+'.stderr.private'),'xb') as err:
  os.chmod(out.name,0o600);os.chmod(err.name,0o600);p=subprocess.Popen(argv,cwd=cwd,env=env,stdin=subprocess.DEVNULL,stdout=out,stderr=err)
 record(root,name+'-started',{'pid':p.pid,'state':'started-not-terminal'});return p

def wait(root,name,p,budget,test=None):
 try:code=p.wait(timeout=budget)
 except subprocess.TimeoutExpired:
  record(root,name+'-unknown',{'pid':p.pid,'state':'unknown-retained','kill_requested':False});raise Refusal('owned terminal unknown')
 raw=capture(root/(name+'.stdout.private'),4*1024*1024);err=capture(root/(name+'.stderr.private'),4*1024*1024)
 result={'exit_code':code,'stdout_sha256':sha(raw),'stderr_sha256':sha(err),'exact_pass':exact_pass(raw,test) if test else None};record(root,name+'-terminal',result);need(code==0 and (test is None or result['exact_pass']));return result

def local_url(value):
 u=urllib.parse.urlsplit(value);need(u.scheme=='http' and u.hostname=='127.0.0.1' and u.port is not None and 0<u.port<65536 and u.netloc=='127.0.0.1:'+str(u.port) and not(u.path or u.query or u.fragment));return value

def control(url,phase):
 r=urllib.request.Request(local_url(url)+'/'+phase,data=b'',headers={'X-Fixture-Key':'synthetic-namespace-control'},method='POST')
 with urllib.request.build_opener(urllib.request.ProxyHandler({})).open(r,timeout=3) as response:need(response.status==204)

def main():
 ap=argparse.ArgumentParser()
 for name in SOURCES:ap.add_argument('--'+name.replace('_','-')+'-source',type=Path,required=True)
 ap.add_argument('--go',type=Path,required=True);ap.add_argument('--go-mod-cache',type=Path,required=True);ap.add_argument('--output',type=Path,required=True);a=ap.parse_args();root=a.output
 need(root.is_absolute() and root.parent.resolve()==root.parent and not root.exists() and a.go.is_absolute());root.mkdir(mode=0o700)
 record(root,'intent',{'scope':'isolated-native-session-OTP-namespace-regression','actual_upgrade':False,'physical_Redis':False,'real_TTL_wait':False,'TOTP':False,'clear_Redis':False})
 try:
  pins=expected();env={'PATH':str(a.go.parent)+':/usr/bin:/bin','GOPROXY':'off','GOSUMDB':'off','GOTOOLCHAIN':'local','GOENV':'off','GOWORK':'off','GOMODCACHE':str(a.go_mod_cache),'GOCACHE':str(root/'go-cache'),'TMPDIR':str(root),'LANG':'C','TZ':'UTC'}
  v=subprocess.run([str(a.go),'version'],env=env,capture_output=True,timeout=10);need(v.returncode==0 and v.stdout.decode().split()[2]=='go1.27.3')
  copies={n:materialize(getattr(a,n+'_source'),n,root,pins[n]) for n in SOURCES};bins={}
  record(root,'source-inputs',{n:{k:r[k] for k in ('commit','tree','archive_sha256')} for n,r in pins.items()})
  for name,(_,package,_) in SOURCES.items():
   verify=start(root,name+'-modules',[str(a.go),'mod','verify'],copies[name],env);wait(root,name+'-modules',verify,30)
   bins[name]=root/(name+'.test');p=start(root,name+'-build',[str(a.go),'test','-mod=readonly','-c','-o',str(bins[name]),package],copies[name],env);wait(root,name+'-build',p,180)
  runtime=dict(env,MIGRATION_ROOT=str(root));publication=start(root,'readiness-publication',[str(bins['sg_new']),'-test.v','-test.run=^TestNamespaceReadyPublication$'],copies['sg_new'],runtime);wait(root,'readiness-publication',publication,10,'TestNamespaceReadyPublication')
  server=start(root,'redis-fixture',[str(bins['sg_new']),'-test.v','-test.run=^TestMigrationRedisServe$'],copies['sg_new'],runtime)
  deadline=time.monotonic()+15;ready=root/'redis-ready.private.json'
  while not ready.exists() and time.monotonic()<deadline:need(server.poll()is None);time.sleep(.02)
  info=json.loads(capture(ready,512));need(set(info)=={'redis_addr','control_url'} and re.fullmatch(r'127\.0\.0\.1:[1-9][0-9]{0,4}',info['redis_addr']));local_url(info['control_url']);runtime['MIGRATION_REDIS']=info['redis_addr'];results=[]
  for index,(component,phase) in enumerate(PHASES):
   if component=='clock':control(info['control_url'],phase);results.append({'phase':phase,'simulated_clock_only':True});continue
   test='TestNativeNamespaceSession' if component.startswith('sg_') else 'TestNativeNamespaceOTP';name='phase-%02d'%index;p=start(root,name,[str(bins[component]),'-test.v','-test.run=^'+test+'$'],copies[component],dict(runtime,MIGRATION_MODE=phase));result=wait(root,name,p,20,test);results.append({'component':component,'phase':phase,**result})
  control(info['control_url'],'stop');wait(root,'redis-fixture',server,10,'TestMigrationRedisServe')
  record(root,'result',{'passed':True,'scope':'native-namespace-regression-only','phases':results,'unsafe_old_prefix_witness_preserved':True,'fresh_disjoint_rollback_prefix_required':True,'actual_upgrade':False,'physical_Redis':False,'real_TTL_wait':False,'TOTP':False,'Warden':False,'IM':False,'clear_Redis':False})
 except Exception:
  record(root,'failure',{'passed':False,'state':'failed-or-unknown-retained','automatic_kill_restart_cleanup':False});raise Refusal('fixed namespace fixture refused')from None

if __name__=='__main__':
 try:main()
 except Exception:
  print('fixed namespace fixture refused; owned private evidence retained',file=__import__('sys').stderr);raise SystemExit(1)
