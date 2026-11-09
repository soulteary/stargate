#!/usr/bin/env python3
"""Fixed-schema local namespace metadata projection. Never open private payloads."""
import argparse
import importlib.util
import json
from pathlib import Path
import re

_spec=importlib.util.spec_from_file_location('fixed_namespace_runner',Path(__file__).with_name('run_transition.py'))
_runner=importlib.util.module_from_spec(_spec);_spec.loader.exec_module(_runner)
HEX=re.compile(r'^[0-9a-f]{64}$')
FALSE_FLAGS=('actual_upgrade','physical_Redis','real_TTL_wait','TOTP','Warden','IM','clear_Redis')
TRUE_FLAGS=('unsafe_old_prefix_witness_preserved','fresh_disjoint_rollback_prefix_required')
NATIVE_PHASES=[(i,n,p) for i,(n,p) in enumerate(_runner.PHASES) if n!='clock']
CHILDREN={**{n+'-modules':False for n in _runner.SOURCES},**{n+'-build':False for n in _runner.SOURCES},'readiness-publication':True,'redis-fixture':True,**{'phase-%02d'%i:True for i,_,_ in NATIVE_PHASES}}
class Refusal(Exception):pass

def need(ok):
 if not ok:raise Refusal('safe namespace metadata refused')

def read(root,name):return json.loads(_runner.capture(root/(name+'.json'),128*1024))

def sources(value):
 expected=_runner.expected();need(type(value)is dict and set(value)==set(expected));out={}
 for name,row in value.items():
  need(type(row)is dict and set(row)=={'commit','tree','archive_sha256'})
  need(row=={k:expected[name][k] for k in ('commit','tree','archive_sha256')})
  out[name]={k:row[k] for k in ('commit','tree','archive_sha256')}
 return out

def terminal(value,native):
 need(type(value)is dict and set(value)=={'exit_code','stdout_sha256','stderr_sha256','exact_pass'})
 need(type(value['exit_code'])is int and value['exit_code']==0)
 need(value['exact_pass']is True if native else value['exact_pass']is None)
 for key in ('stdout_sha256','stderr_sha256'):need(type(value[key])is str and HEX.fullmatch(value[key]))
 return {k:value[k] for k in ('exit_code','stdout_sha256','stderr_sha256','exact_pass')}

def phases(value,children):
 need(type(value)is list and len(value)==len(_runner.PHASES));out=[]
 for i,((component,phase),row) in enumerate(zip(_runner.PHASES,value)):
  need(type(row)is dict)
  if component=='clock':
   need(set(row)=={'phase','simulated_clock_only'} and row['phase']==phase and row['simulated_clock_only']is True)
   out.append({'phase':phase,'simulated_clock_only':True});continue
  need(set(row)=={'component','phase','exit_code','stdout_sha256','stderr_sha256','exact_pass'})
  need(row['component']==component and row['phase']==phase)
  result=terminal({k:row[k] for k in ('exit_code','stdout_sha256','stderr_sha256','exact_pass')},True)
  need(result==children['phase-%02d'%i]);out.append({'component':component,'phase':phase,**result})
 return out

def project(root,output):
 need(root.is_absolute() and output.is_absolute() and output.parent.resolve()==output.parent and not output.exists());output.mkdir(mode=0o700)
 failure={'passed':False,'scope':'released-native-namespace-regression','state':'failed-or-unknown','raw_private_data_exported':False,**{k:False for k in FALSE_FLAGS}}
 try:
  source=sources(read(root,'source-inputs'))
  children={name:terminal(read(root,name+'-terminal'),native) for name,native in CHILDREN.items()}
  result=read(root,'result')
  need(type(result)is dict and set(result)=={'passed','scope','phases',*FALSE_FLAGS,*TRUE_FLAGS})
  need(result['passed']is True and result['scope']=='native-namespace-regression-only')
  need(all(result[k]is False for k in FALSE_FLAGS) and all(result[k]is True for k in TRUE_FLAGS))
  ordered=phases(result['phases'],children)
  public={'passed':True,'scope':'released-native-namespace-regression','current_HEAD_qualification':False,'native_phases':len(NATIVE_PHASES),'synthetic_clock_phases':1,'known_zero_children':len(children),**{k:True for k in TRUE_FLAGS},**{k:False for k in FALSE_FLAGS}}
 except Exception:
  _runner.record(output,'failure',failure);return False
 _runner.record(output,'source-inputs',source)
 _runner.record(output,'native-results',{'children':children,'ordered_phases':ordered})
 _runner.record(output,'result',public)
 return True

if __name__=='__main__':
 ap=argparse.ArgumentParser();ap.add_argument('--run',type=Path,required=True);ap.add_argument('--output',type=Path,required=True);args=ap.parse_args()
 try:ok=project(args.run,args.output)
 except Exception:ok=False
 if not ok:raise SystemExit(1)
