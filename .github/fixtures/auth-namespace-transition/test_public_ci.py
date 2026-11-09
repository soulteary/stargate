"""Offline synthetic metadata mutations; no Go/components/Redis/listeners."""
import copy
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

ROOT=Path(__file__).parent
spec=importlib.util.spec_from_file_location('namespace_public',ROOT/'public_results.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)

class PublicControls(unittest.TestCase):
 def sample(self,root):
  expected=m._runner.expected()
  source={n:{k:row[k] for k in ('commit','tree','archive_sha256')} for n,row in expected.items()}
  children={n:{'exit_code':0,'stdout_sha256':'a'*64,'stderr_sha256':'b'*64,'exact_pass':True if native else None} for n,native in m.CHILDREN.items()}
  phases=[]
  for i,(n,p) in enumerate(m._runner.PHASES):
   phases.append({'phase':p,'simulated_clock_only':True} if n=='clock' else {'component':n,'phase':p,**children['phase-%02d'%i]})
  result={'passed':True,'scope':'native-namespace-regression-only','phases':phases,**{k:False for k in m.FALSE_FLAGS},**{k:True for k in m.TRUE_FLAGS}}
  values={'source-inputs':source,'result':result,**{n+'-terminal':r for n,r in children.items()}}
  self.write(root,values);return values
 def write(self,root,values):
  for n,v in values.items():(root/(n+'.json')).write_text(json.dumps(v))
 def test_complete_projection_has_exact22_children_and_no_private_reads(self):
  with tempfile.TemporaryDirectory() as t:
   root=Path(t).resolve();run=root/'run';run.mkdir();self.sample(run)
   canary='NEVER_EXPORT_TOKEN_OTP_ENDPOINT_RECIPIENT'
   for n in ('old-session.private.json','new-otp.private.json','redis-ready.private.json','phase-00.stdout.private','redis-fixture-started.json'):(run/n).write_text(canary)
   original=m._runner.capture;reads=[]
   def captured(path,limit):reads.append(Path(path));return original(path,limit)
   with patch.object(m._runner,'capture',side_effect=captured):self.assertTrue(m.project(run,root/'public'))
   self.assertEqual({p.name for p in (root/'public').iterdir()},{'result.json','source-inputs.json','native-results.json'})
   self.assertEqual(len(json.loads((root/'public/native-results.json').read_text())['children']),22)
   self.assertTrue(all(not(p.name.endswith('.private.json')or'.stdout.'in p.name or'-started'in p.name)for p in reads))
   self.assertNotIn(canary,''.join(p.read_text()for p in(root/'public').iterdir()))
 def test_wrong_sources_and_secret_fields_refused(self):
  for mode in ('commit','tree','archive_sha256','extra'):
   with self.subTest(mode=mode),tempfile.TemporaryDirectory() as t:
    root=Path(t).resolve();run=root/'run';run.mkdir();v=self.sample(run)
    v['source-inputs']['sg_old'][mode]='NEVER_EXPORT_CANARY' if mode=='extra' else '0'*len(v['source-inputs']['sg_old'][mode]);self.write(run,v)
    self.assertFalse(m.project(run,root/'public'));self.fixed_failure(root)
 def fixed_failure(self,root):
  self.assertEqual([p.name for p in(root/'public').iterdir()],['failure.json'])
  text=(root/'public/failure.json').read_text();self.assertNotIn('NEVER_EXPORT_CANARY',text);self.assertFalse(json.loads(text)['passed'])
 def test_phase_order_missing_duplicate_and_terminal_mismatch_refused(self):
  for mode in ('order','missing','duplicate','wronglabel','mismatch','extrafield'):
   with self.subTest(mode=mode),tempfile.TemporaryDirectory() as t:
    root=Path(t).resolve();run=root/'run';run.mkdir();v=self.sample(run);ph=v['result']['phases']
    if mode=='order':ph[0],ph[1]=ph[1],ph[0]
    elif mode=='missing':ph.pop()
    elif mode=='duplicate':ph[-1]=copy.deepcopy(ph[0])
    elif mode=='wronglabel':ph[2]['phase']='rollback-fresh'
    elif mode=='mismatch':ph[0]['stdout_sha256']='c'*64
    else:ph[0]['secret']='NEVER_EXPORT_CANARY'
    self.write(run,v);self.assertFalse(m.project(run,root/'public'));self.fixed_failure(root)
 def test_each_child_must_be_known_zero_with_exact_native_pass(self):
  for mode in ('nonzero','unknown','nativefalse','buildtrue','badhex','missing'):
   with self.subTest(mode=mode),tempfile.TemporaryDirectory() as t:
    root=Path(t).resolve();run=root/'run';run.mkdir();v=self.sample(run)
    if mode=='nonzero':v['phase-00-terminal']['exit_code']=1
    elif mode=='unknown':v['phase-00-terminal']['exit_code']=None
    elif mode=='nativefalse':v['redis-fixture-terminal']['exact_pass']=False
    elif mode=='buildtrue':v['sg_old-build-terminal']['exact_pass']=True
    elif mode=='badhex':v['sg_new-modules-terminal']['stdout_sha256']='NEVER_EXPORT_CANARY'
    else:(run/'readiness-publication-terminal.json').unlink();del v['readiness-publication-terminal']
    self.write(run,v);self.assertFalse(m.project(run,root/'public'));self.fixed_failure(root)
 def test_unsafe_witnesses_and_realworld_claims_cannot_be_relabeled(self):
  for flag in (*m.FALSE_FLAGS,*m.TRUE_FLAGS,'passed'):
   with self.subTest(flag=flag),tempfile.TemporaryDirectory() as t:
    root=Path(t).resolve();run=root/'run';run.mkdir();v=self.sample(run);v['result'][flag]=not v['result'][flag];self.write(run,v)
    self.assertFalse(m.project(run,root/'public'));self.fixed_failure(root)
 def test_symlink_oversized_partial_unknown_only_fixed_failure(self):
  for mode in ('symlink','oversized','partial','absent'):
   with self.subTest(mode=mode),tempfile.TemporaryDirectory() as t:
    root=Path(t).resolve();run=root/'run';run.mkdir();self.sample(run);p=run/'result.json';p.unlink()
    if mode=='symlink':(root/'canary').write_text('NEVER_EXPORT_CANARY');p.symlink_to(root/'canary')
    elif mode=='oversized':p.write_bytes(b' '*131073)
    elif mode=='partial':p.write_text('{"secret":"NEVER_EXPORT_CANARY"')
    self.assertFalse(m.project(run,root/'public'));self.fixed_failure(root)
 def test_workflow_is_scoped_readonly_and_only_lists_safe_artifacts(self):
  workflow=ROOT.parents[1]/'workflows/released-auth-namespace-transition.yml';text=workflow.read_text()
  self.assertNotIn('\n  push:',text);self.assertIn('contents: read',text);self.assertEqual(text.count('persist-credentials: false'),5)
  for n,row in m._runner.SOURCES.items():self.assertIn('ref: '+row[0],text)
  for s in ('test_controls.py','test_public_ci.py','go mod download && go mod verify',"go-version: '1.27.3'","'.github/fixtures/auth-namespace-transition/**'"):self.assertIn(s,text)
  upload=text.split('path: |')[-1];self.assertNotIn('*',upload)
  for s in ('private','stdout','stderr','ready','started'):self.assertNotIn(s,upload)
  for s in ('result.json','native-results.json','source-inputs.json','failure.json'):self.assertIn(s,upload)

if __name__=='__main__':unittest.main()
