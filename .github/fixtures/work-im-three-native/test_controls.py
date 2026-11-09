"""Pure offline runner controls. Never starts Go, HTTP listeners or components."""
import importlib.util
from pathlib import Path
import tempfile
import unittest
import hashlib
import io
import json
import tarfile
import subprocess
from unittest.mock import patch

spec=importlib.util.spec_from_file_location('three',Path(__file__).with_name('run_three.py'))
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)

class Controls(unittest.TestCase):
 def test_exact_test_and_children_required(self):
  top='TestFixed';valid=b'=== RUN   TestFixed\n--- PASS: TestFixed (0.01s)\n    --- PASS: TestFixed/required (0.00s)\nPASS\n'
  self.assertTrue(m.exact_pass(valid,top,('required',)))
  for raw in (valid.replace(b'TestFixed/required',b'TestFixed/other'),valid.replace(b'    --- PASS: TestFixed/required (0.00s)\n',b''),valid+ b'    --- PASS: TestFixed/required (0.00s)\n',valid+b'--- SKIP: Other (0.00s)\n',valid.replace(b'\nPASS\n',b'\n')):
   self.assertFalse(m.exact_pass(raw,top,('required',)))
 def test_loopback_only(self):
  self.assertEqual(m.endpoint(b'http://127.0.0.1:1234'),'http://127.0.0.1:1234')
  for value in (b'http://localhost:1234',b'https://127.0.0.1:1234',b'http://127.0.0.1:1234/',b'http://x@127.0.0.1:1234',b'http://127.0.0.1:1234?q=x',b'http://127.0.0.1:01234'):
   with self.assertRaises(Exception):m.endpoint(value)
 def test_real_files_exact_read_and_refuse_alias(self):
  with tempfile.TemporaryDirectory() as root:
   p=Path(root)/'owned';m.write_new(p,b'public synthetic')
   self.assertEqual(m.read_regular(p,100),b'public synthetic')
   with self.assertRaises(Exception):m.read_regular(p,2)
   with self.assertRaises(FileExistsError):m.write_new(p,b'replace')
   link=Path(root)/'alias';link.symlink_to(p)
   with self.assertRaises(Exception):m.read_regular(link,100)
 def test_transport_and_native_seams_are_explicit(self):
  root=Path(__file__).parent
  adapter=(root/'adapter_formal_bridge_test.go').read_text()
  self.assertLess(adapter.index('http.DefaultTransport = closed'),adapter.index('Setup(app,'))
  self.assertIn('r.URL.Host != "oapi.dingtalk.com"',adapter)
  self.assertNotIn('DefaultTransport.RoundTrip',adapter)
  self.assertIn('NewRouterWithClientAndHandlersE', (root/'herald_formal_bridge_test.go').read_text())
  sg=(root/'stargate_formal_bridge_test.go').read_text()
  for symbol in ('SendVerifyCodeAPI()', 'LoginAPI(store)', 'auth.IsAuthenticated(s)', 'auth.GetUserID(s)'):
   self.assertIn(symbol,sg)

class SourceBindingControls(unittest.TestCase):
 def test_manifest_tamper_refused(self):
  self.assertEqual(set(m.expected_inputs()),set(m.SOURCES))
  raw=Path(__file__).with_name('source-inputs.json').read_bytes()
  with patch.object(m,'read_regular',return_value=raw+b' '):
   with self.assertRaises(m.Refusal):m.expected_inputs()
 def test_archive_and_blob_tamper_refused(self):
  stream=io.BytesIO()
  with tarfile.open(fileobj=stream,mode='w') as archive:
   row=tarfile.TarInfo('go.mod');row.size=3;archive.addfile(row,io.BytesIO(b'abc'))
  raw=stream.getvalue();expected={'archive_size':len(raw),'archive_sha256':m.sha(raw),'internal_source_and_test_git_blob_sha256':{'go.mod':m.sha(b'abc')}}
  m.validate_archive(raw,expected)
  with self.assertRaises(m.Refusal):m.validate_archive(raw[:-1]+b'X',expected)
  expected['internal_source_and_test_git_blob_sha256']['go.mod']=m.sha(b'wrong')
  with self.assertRaises(m.Refusal):m.validate_archive(raw,expected)
 def test_real_git_replace_cannot_change_selected_bytes(self):
  with tempfile.TemporaryDirectory() as root:
   repo=Path(root)
   def git(*args,input=None):return subprocess.check_output(['/usr/bin/git','-C',root,*args],input=input,stderr=subprocess.DEVNULL)
   git('init','-q')
   def tree(data):
    blob=git('hash-object','-w','--stdin',input=data).decode().strip()
    return git('mktree',input=('100644 blob '+blob+'\tgo.mod\n').encode()).decode().strip()
   original=tree(b'original');replacement=tree(b'replacement')
   before=m.git(repo,'archive','--format=tar',original)
   git('replace',original,replacement)
   replaced=git('archive','--format=tar',original)
   self.assertNotEqual(before,replaced)
   self.assertEqual(m.git(repo,'archive','--format=tar',original),before)
   expected={'archive_size':len(before),'archive_sha256':m.sha(before),'internal_source_and_test_git_blob_sha256':{'go.mod':m.sha(b'original')}}
   m.validate_archive(before,expected)
   with self.assertRaises(m.Refusal):m.validate_archive(replaced,expected)
 def test_modified_overlay_refuses_materialization(self):
  root=Path(__file__).parent
  repo=root.parent/'herald-dingtalk127-source-current'
  expected=m.expected_inputs()['adapter'];original=m.read_regular
  def capture(path,limit):
   raw=original(path,limit)
   return raw+b'\n// changed overlay\n' if Path(path).name=='adapter_formal_bridge_test.go' else raw
  with tempfile.TemporaryDirectory() as temp,patch.object(m,'read_regular',side_effect=capture):
   with self.assertRaises(m.Refusal):m.materialize(repo,'adapter',Path(temp),expected)
 def test_overlay_bytes_pinned(self):
  root=Path(__file__).parent
  for name,row in m.SOURCES.items():self.assertEqual(m.sha((root/row[3]).read_bytes()),m.EXPECTED_OVERLAYS[name])

if __name__=='__main__':unittest.main()
