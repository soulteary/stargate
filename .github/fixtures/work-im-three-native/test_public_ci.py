"""Offline metadata/CI controls, no components, listeners or module downloads."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

root=Path(__file__).parent
spec=importlib.util.spec_from_file_location('public_projection',root/'public_results.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)

class PublicCIControls(unittest.TestCase):
 def sample(self,run):
  expected=m.expected_inputs();source={}
  for n,row in expected.items():
   source[n]={'commit':row['commit'],'tree':row['tree'],'archive_sha256':row['archive_sha256'],'overlay_sha256':m.EXPECTED_OVERLAYS[n],'go_mod_sha256':row['internal_source_and_test_git_blob_sha256']['go.mod'],'go_sum_sha256':row['internal_source_and_test_git_blob_sha256']['go.sum']}
  components={n:{'exit_code':0,'exact_pass':True,'stdout_sha256':'a'*64,'stderr_sha256':'b'*64} for n in source}
  result={'passed':True,'scope':'native-three-service-synthetic-transport','Warden_runtime':False,'Redis_migration':False,'real_IM':False,'deployment':False,'source_inputs':source,'components':components}
  for n,v in [('source-inputs',source),('native-results',components),('result',result)]:
   (run/(n+'.json')).write_text(json.dumps(v))
  return source,components,result
 def test_valid_projection_never_reads_or_exports_private_files(self):
  with tempfile.TemporaryDirectory() as t:
   p=Path(t).resolve();run=p/'private';run.mkdir();self.sample(run)
   canary='NEVER_EXPORT_OTP_654321_RECIPIENT_PRIVATE_ENDPOINT'
   (run/'adapter.stdout.private').write_text(canary)
   (run/'adapter.ready.private').write_text(canary)
   self.assertTrue(m.project(run,p/'public'))
   self.assertEqual({x.name for x in (p/'public').iterdir()},{'result.json','source-inputs.json','native-results.json'})
   self.assertNotIn(canary,''.join(x.read_text() for x in (p/'public').iterdir()))
 def test_invalid_source_or_extra_secret_only_exports_fixed_failure(self):
  for mode in ('wrongtree','extra','claim','exit','skip'):
   with self.subTest(mode=mode),tempfile.TemporaryDirectory() as t:
    p=Path(t).resolve();run=p/'private';run.mkdir();source,native,result=self.sample(run)
    if mode=='wrongtree':source['sg']['tree']='0'*40
    elif mode=='extra':source['sg']['secret']='NEVER_EXPORT_CANARY'
    elif mode=='claim':result['real_IM']=True
    elif mode=='exit':native['adapter']['exit_code']=1
    else:native['sg']['exact_pass']=False
    for n,v in [('source-inputs',source),('native-results',native),('result',result)]:
     (run/(n+'.json')).write_text(json.dumps(v))
    self.assertFalse(m.project(run,p/'public'))
    self.assertEqual([x.name for x in (p/'public').iterdir()],['failure.json'])
    self.assertNotIn('NEVER_EXPORT_CANARY',(p/'public/failure.json').read_text())
 def test_missing_unknown_and_symlink_fail_closed(self):
  for mode in ('missing','symlink'):
   with self.subTest(mode=mode),tempfile.TemporaryDirectory() as t:
    p=Path(t).resolve();run=p/'private';run.mkdir()
    if mode=='symlink':
     self.sample(run);(run/'result.json').unlink();(p/'canary').write_text('NEVER_EXPORT_CANARY');(run/'result.json').symlink_to(p/'canary')
    self.assertFalse(m.project(run,p/'public'))
    self.assertEqual([x.name for x in (p/'public').iterdir()],['failure.json'])
 def test_path_scoped_read_only_fixed_sources_and_explicit_artifacts(self):
  workflow=root.parents[1]/'workflows/released-work-im-native-three.yml'
  text=workflow.read_text()
  self.assertIn("paths:\n      - '.github/fixtures/work-im-three-native/**'",text)
  self.assertNotIn('\n  push:',text)
  self.assertIn('contents: read',text)
  self.assertEqual(text.count('persist-credentials: false'),4)
  for row in m.SOURCES.values():self.assertIn('ref: '+row[0],text)
  for value in ('go mod download && go mod verify',"go-version: '1.27.3'",'GOTOOLCHAIN: local'):self.assertIn(value,text)
  upload=text.split('path: |')[-1]
  self.assertNotIn('*',upload)
  for name in ('stdout','stderr','ready','started','intent'):self.assertNotIn(name,upload)
  for name in ('result.json','native-results.json','source-inputs.json','failure.json'):self.assertIn(name,upload)

if __name__=='__main__':unittest.main()
