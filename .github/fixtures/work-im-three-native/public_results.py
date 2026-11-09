#!/usr/bin/env python3
"""Project only fixed safe metadata; never open raw logs or readiness files."""
import argparse
import json
from pathlib import Path
import re
import os
import importlib.util
_spec=importlib.util.spec_from_file_location('fixed_native_three_runner',Path(__file__).with_name('run_three.py'))
_runner=importlib.util.module_from_spec(_spec);_spec.loader.exec_module(_runner)
read_regular,write_new,SOURCES,EXPECTED_OVERLAYS,expected_inputs=(_runner.read_regular,_runner.write_new,_runner.SOURCES,_runner.EXPECTED_OVERLAYS,_runner.expected_inputs)

HEX=re.compile(r'^[0-9a-f]{64}$')
class Refusal(Exception):pass

def need(ok):
 if not ok:raise Refusal('safe metadata projection refused')

def hash_value(v):
 need(type(v)is str and HEX.fullmatch(v));return v

def sources(value):
 expected=expected_inputs();need(type(value)is dict and set(value)==set(SOURCES));out={}
 for name,row in value.items():
  need(type(row)is dict and set(row)=={'commit','tree','archive_sha256','overlay_sha256','go_mod_sha256','go_sum_sha256'})
  need(row['commit']==SOURCES[name][0] and row['tree']==SOURCES[name][1])
  need(row['archive_sha256']==expected[name]['archive_sha256'] and row['overlay_sha256']==EXPECTED_OVERLAYS[name])
  blobs=expected[name]['internal_source_and_test_git_blob_sha256']
  need(row['go_mod_sha256']==blobs['go.mod'] and row['go_sum_sha256']==blobs['go.sum'])
  out[name]={key:row[key] for key in row}
 return out

def components(value):
 need(type(value)is dict and set(value)==set(SOURCES));out={}
 for name,row in value.items():
  need(type(row)is dict and set(row)=={'exit_code','stdout_sha256','stderr_sha256','exact_pass'})
  need(type(row['exit_code'])is int and 0<=row['exit_code']<=255 and type(row['exact_pass'])is bool)
  out[name]={'exit_code':row['exit_code'],'exact_pass':row['exact_pass'],'stdout_sha256':hash_value(row['stdout_sha256']),'stderr_sha256':hash_value(row['stderr_sha256'])}
 return out

def project(root,output):
 need(root.is_absolute() and output.is_absolute() and not output.exists() and output.parent.resolve()==output.parent)
 output.mkdir(mode=0o700)
 # An absent/failed/unknown run always exports only a fixed failure statement.
 failure={'passed':False,'scope':'released-native-three-qualification','state':'failed-or-unknown','raw_logs_exported':False,'actual_Warden':False,'physical_Redis':False,'real_IM':False,'deployment':False}
 try:
  source=sources(json.loads(read_regular(root/'source-inputs.json',65536)))
  native=components(json.loads(read_regular(root/'native-results.json',65536)))
  result=json.loads(read_regular(root/'result.json',65536))
  need(type(result)is dict and set(result)=={'passed','scope','Warden_runtime','Redis_migration','real_IM','deployment','source_inputs','components'})
  need(result['passed']is True and result['scope']=='native-three-service-synthetic-transport')
  need(all(result[x]is False for x in ('Warden_runtime','Redis_migration','real_IM','deployment')))
  need(sources(result['source_inputs'])==source and components(result['components'])==native)
  need(all(x['exit_code']==0 and x['exact_pass']is True for x in native.values()))
  public={'passed':True,'scope':'released-native-three-qualification','current_HEAD_qualification':False,'actual_Warden':False,'physical_Redis':False,'real_IM':False,'deployment':False}
 except Exception:
  write_new(output/'failure.json',(json.dumps(failure,sort_keys=True)+'\n').encode())
  return False
 for name,value in [('source-inputs',source),('native-results',native),('result',public)]:
  write_new(output/(name+'.json'),(json.dumps(value,sort_keys=True,indent=2)+'\n').encode())
 return True

if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--run',type=Path,required=True);p.add_argument('--output',type=Path,required=True);a=p.parse_args()
 try:ok=project(a.run,a.output)
 except Exception:ok=False
 if not ok:raise SystemExit(1)
