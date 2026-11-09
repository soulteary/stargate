"""Only offline source/file controls; no native build, Redis or listeners."""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
import os
from unittest.mock import patch

root=Path(__file__).parent
spec=importlib.util.spec_from_file_location('transition',root/'run_transition.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
class Controls(unittest.TestCase):
 def test_manifest_and_overlay_pin_tampering(self):
  self.assertEqual(set(m.expected()),set(m.SOURCES))
  with patch.object(m,'capture',return_value=(root/'source-inputs.json').read_bytes()+b' '):
   with self.assertRaises(m.Refusal):m.expected()
  for name,pin in m.OVERLAYS.items():self.assertEqual(m.sha((root/name).read_bytes()),pin)
 def test_exact_test_receipt_refuses_skip_wrong_or_duplicate(self):
  valid=b'=== RUN   TestNativeNamespaceSession\n--- PASS: TestNativeNamespaceSession (0.01s)\nPASS\n'
  self.assertTrue(m.exact_pass(valid,'TestNativeNamespaceSession'))
  for bad in (valid.replace(b'TestNativeNamespaceSession',b'TestOther'),valid+b'--- SKIP: Other (0.00s)\n',valid+valid,valid.replace(b'\nPASS\n',b'\n')):
   self.assertFalse(m.exact_pass(bad,'TestNativeNamespaceSession'))
 def test_fixed_control_endpoint_not_ambient(self):
  self.assertEqual(m.local_url('http://127.0.0.1:1234'),'http://127.0.0.1:1234')
  for bad in ('https://127.0.0.1:1234','http://localhost:1234','http://x@127.0.0.1:1234','http://127.0.0.1:1234/path','http://127.0.0.1:1234?q=secret'):
   with self.assertRaises(Exception):m.local_url(bad)
 def test_pending_name_hides_partial_bytes_until_actual_rename(self):
  # Actual host filesystem operations, not a claim that the Go helper has run.
  with tempfile.TemporaryDirectory() as t:
   p=Path(t);pending=p/'redis-ready.pending.private.json';final=p/'redis-ready.private.json'
   fd=os.open(pending,os.O_WRONLY|os.O_CREAT|os.O_EXCL,0o600)
   os.write(fd,b'{"redis_addr":')
   self.assertFalse(final.exists())
   os.write(fd,b'"127.0.0.1:1234"}')
   os.close(fd);self.assertFalse(final.exists());os.rename(pending,final)
   self.assertEqual(json.loads(final.read_text()),{'redis_addr':'127.0.0.1:1234'})
  source=(root/'redis_transition_test.go').read_text()
  body=source.split('func publishMigrationReady')[1].split('func TestNamespaceReadyPublication')[0]
  self.assertIn('os.OpenFile(pending,',body)
  self.assertLess(body.index('f.Close()'),body.index('os.Rename(pending, final)'))
 def test_held_file_and_exclusive_writes(self):
  with tempfile.TemporaryDirectory() as t:
   p=Path(t)/'owned';m.write(p,b'synthetic')
   self.assertEqual(m.capture(p,32),b'synthetic')
   with self.assertRaises(Exception):m.capture(p,2)
   with self.assertRaises(FileExistsError):m.write(p,b'overwrite')
   alias=Path(t)/'alias';alias.symlink_to(p)
   with self.assertRaises(Exception):m.capture(alias,32)
 def test_real_native_seams_and_unsafe_witness_not_removed(self):
  session=(root/'session_transition_test.go').read_text();otp=(root/'otp_transition_test.go').read_text();redis=(root/'redis_transition_test.go').read_text()
  self.assertIn('store, rc := setupSessionStore()',session)
  self.assertIn('case "unsafe-old-prefix-witness":',session)
  self.assertIn('check("old-session", 204)',session)
  self.assertIn('prefix = "migration:rollback-fresh:"',session)
  self.assertIn('NewHandlersWithError(rc, testLogger())',otp)
  self.assertIn('h.challengeManager.VerifyWithOptions',otp)
  self.assertIn('verify("old-otp-witness", true)',otp)
  self.assertIn('mr.FastForward(301 * time.Second)',redis)
  for text in (session,otp,redis):
   self.assertNotIn('FlushAll',text);self.assertNotIn('FLUSHALL',text);self.assertNotIn('FLUSHDB',text)

if __name__=='__main__':unittest.main()
