"""Manifest boundary tests: these must never contact GitHub or execute binaries."""
import copy
import importlib.util
import json
from pathlib import Path
import unittest
from unittest import mock
import tempfile

ROOT = Path(__file__).parent
spec = importlib.util.spec_from_file_location('compatibility', ROOT / 'release_compatibility.py')
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class ManifestTests(unittest.TestCase):
    def setUp(self):
        self.manifest = json.loads((ROOT / 'manifests/baseline.json').read_text())

    def test_committed_baseline_and_complete_upgrade(self):
        self.assertEqual(module.validate_manifest(self.manifest)['mode'], 'baseline')
        manifest = copy.deepcopy(self.manifest)
        manifest.update(mode='upgrade', new=copy.deepcopy(manifest['old']))
        self.assertEqual(module.validate_manifest(manifest)['new'], manifest['old'])

    def test_rejects_incomplete_or_unknown_service_tuples(self):
        for key in ('old',):
            manifest = copy.deepcopy(self.manifest)
            del manifest[key]['totp']
            with self.assertRaises(ValueError):
                module.validate_manifest(manifest)
        manifest = copy.deepcopy(self.manifest)
        manifest['old']['other-repository'] = manifest['old'].pop('warden')
        with self.assertRaises(ValueError):
            module.validate_manifest(manifest)
        manifest = copy.deepcopy(self.manifest)
        manifest['mode'] = 'upgrade'
        with self.assertRaises(ValueError):
            module.validate_manifest(manifest)

    def test_rejects_paths_shell_fragments_unpinned_images_and_bad_sources(self):
        invalid = {'version': ['1.3.3; touch /tmp/unexpected', '../v1.3.3', '1.3.3-rc1'],
                   'source': ['main', 'a' * 39, 'a' * 40 + '/../../x'],
                   'sha256': ['a' * 63, 'a' * 64 + '\n'],
                   'image': ['ghcr.io/soulteary/stargate:latest',
                             'ghcr.io/other/stargate@sha256:' + 'a' * 64,
                             'ghcr.io/soulteary/herald@sha256:' + 'a' * 64]}
        for field, values in invalid.items():
            for value in values:
                with self.subTest(field=field, value=value):
                    manifest = copy.deepcopy(self.manifest)
                    manifest['old']['gateway'][field] = value
                    with self.assertRaises(ValueError):
                        module.validate_manifest(manifest)

    def test_rejects_arbitrary_download_fields(self):
        manifest = copy.deepcopy(self.manifest)
        manifest['old']['gateway']['url'] = 'https://example.invalid/app'
        with self.assertRaises(ValueError):
            module.validate_manifest(manifest)


    def test_tag_source_mismatch_stops_before_download(self):
        entry = self.manifest['old']['gateway']
        with mock.patch.object(module, 'api', return_value={'object': {'type': 'commit', 'sha': 'f' * 40}}), mock.patch.object(module, 'fetch') as fetch:
            with self.assertRaisesRegex(ValueError, 'source mismatch'):
                module.download_tuple('old', {'gateway': entry})
            fetch.assert_not_called()

    def test_published_checksum_mismatch_stops_before_binary_execution(self):
        entry = self.manifest['old']['gateway']
        release = {'draft': False, 'prerelease': False, 'tag_name': 'v' + entry['version'],
                   'assets': [{'name': 'stargate-linux-amd64'}, {'name': 'checksums.txt'}]}
        replies = [{'object': {'type': 'commit', 'sha': entry['source']}}, release]
        with mock.patch.object(module, 'api', side_effect=replies), mock.patch.object(module, 'fetch', return_value=('f' * 64 + '  stargate-linux-amd64').encode()) as fetch:
            with self.assertRaisesRegex(ValueError, 'manifest disagree'):
                module.download_tuple('old', {'gateway': entry})
            self.assertEqual(fetch.call_count, 1)

    def test_child_services_do_not_inherit_runner_credentials(self):
        with tempfile.TemporaryDirectory() as temporary, mock.patch.object(module, 'RUNTIME', Path(temporary)), mock.patch.dict(module.os.environ, {'GH_TOKEN': 'runner-token', 'GITHUB_TOKEN': 'other-token', 'AWS_SECRET_ACCESS_KEY': 'runner-key'}):
            env = module.child_environment({'API_KEY': '', 'HERALD_HMAC_SECRET': 'synthetic'})
        self.assertNotIn('GH_TOKEN', env)
        self.assertNotIn('GITHUB_TOKEN', env)
        self.assertNotIn('AWS_SECRET_ACCESS_KEY', env)
        self.assertEqual(env['HERALD_HMAC_SECRET'], 'synthetic')


    def test_rotation_recipients_are_distinct_and_present_in_directory(self):
        self.assertEqual(len(set(module.ROTATION_SUBJECTS)), 2)
        self.assertTrue(set(module.ROTATION_SUBJECTS).isdisjoint({'chain-a@example.test', 'chain-b@example.test'}))
        directory = {user['mail']: user for user in module.directory_users()}
        for subject in module.ROTATION_SUBJECTS:
            self.assertEqual(directory[subject]['status'], 'active')

    def test_rotation_login_preserves_pending_browser_even_on_failure(self):
        pending = {'stargate_session_id': 'pending-browser', 'stargate_callback': 'pending-target'}
        def fake_send(subject):
            self.assertEqual(module.browser_cookies, {})
            module.browser_cookies['temporary'] = 'synthetic'
            return 'challenge', '123456'
        def fake_login(subject, challenge, code):
            module.browser_cookies['stargate_session_id'] = 'rotation-session'
            raise RuntimeError('upstream failure')
        with mock.patch.object(module, 'browser_cookies', pending), mock.patch.object(module, 'send', side_effect=fake_send) as send, mock.patch.object(module, 'login', side_effect=fake_login) as login:
            with self.assertRaisesRegex(RuntimeError, 'upstream failure'):
                module.rotation_login(module.ROTATION_SUBJECTS[0])
            self.assertIs(module.browser_cookies, pending)
            self.assertEqual(pending, {'stargate_session_id': 'pending-browser', 'stargate_callback': 'pending-target'})
            send.assert_called_once_with(module.ROTATION_SUBJECTS[0])
            login.assert_called_once_with(module.ROTATION_SUBJECTS[0], 'challenge', '123456')


    def test_backup_summary_redacts_codes_and_individual_hashes(self):
        entries = [{'code_hash': format(i, '064x'), 'used_at': 123 if i == 0 else 0} for i in range(10)]
        raw = json.dumps(entries)
        with mock.patch.object(module, 'resp', side_effect=[raw, -1]) as redis:
            stored, private_entries, summary = module.backup_state('synthetic-user')
        self.assertEqual(stored, raw)
        self.assertEqual(private_entries, entries)
        self.assertEqual(summary['used_count'], 1)
        self.assertEqual(summary['entry_count'], 10)
        self.assertEqual(summary['key_ttl_ms'], -1)
        self.assertNotIn('code_hash', json.dumps(summary))
        self.assertNotIn(entries[1]['code_hash'], json.dumps(summary))
        self.assertEqual(redis.call_args_list, [mock.call(['GET', 'totp:backup:synthetic-user']), mock.call(['PTTL', 'totp:backup:synthetic-user'])])

    def test_backup_state_mutation_and_unexpected_expiry_fail_closed(self):
        entries = [{'code_hash': format(i, '064x'), 'used_at': 0} for i in range(10)]
        with mock.patch.object(module, 'resp', side_effect=[json.dumps(entries), 30000]):
            with self.assertRaisesRegex(AssertionError, 'persistent lifetime'):
                module.backup_state('synthetic-user')
        with mock.patch.object(module, 'backup_state', return_value=('changed', entries, {})):
            with self.assertRaisesRegex(AssertionError, 'mutated'):
                module.assert_backup_unchanged(('original', entries, {}), 'upgrade')

    def test_backup_hash_matches_case_and_separator_normalization(self):
        self.assertEqual(module.backup_hash(' abcd-efgh '), module.backup_hash('ABCDEFGH'))

    def test_unexpected_response_does_not_write_raw_backup_codes_to_error(self):
        with self.assertRaises(AssertionError) as raised:
            module.expect('enrollment', (201, {'backup_codes': ['private-synthetic-code']}, {}), 200)
        self.assertNotIn('private-synthetic-code', str(raised.exception))


    def test_backup_response_requires_subject_strength_and_rejection_ok(self):
        valid = {'ok': True, 'subject': 'chain-user-a', 'amr': ['totp', 'backup_code']}
        invalid = [dict(valid, subject='another-user'), dict(valid, amr=['backup_code']), dict(valid, ok=False)]
        for body in invalid:
            with self.subTest(body=body), mock.patch.object(module, 'request'), mock.patch.object(module, 'expect', return_value=body):
                with self.assertRaisesRegex(AssertionError, 'backup-code branch'):
                    module.consume_backup('synthetic', 'challenge', 'verify')
        with mock.patch.object(module, 'request'), mock.patch.object(module, 'expect', return_value={'ok': True, 'reason': 'invalid'}):
            with self.assertRaisesRegex(AssertionError, 'rejection branch'):
                module.rejected_backup('synthetic', 'challenge', 'reject', 401, 'invalid')

    def test_replay_marker_cannot_be_permanent(self):
        for ttl in (-1, -3):
            with self.subTest(ttl=ttl), mock.patch.object(module, 'resp', return_value=ttl):
                with self.assertRaisesRegex(AssertionError, 'persistent or invalid'):
                    module.replay_marker_ttl('synthetic-challenge')
        for ttl in (-2, 0, 300000):
            with self.subTest(ttl=ttl), mock.patch.object(module, 'resp', return_value=ttl):
                self.assertEqual(module.replay_marker_ttl('synthetic-challenge'), ttl)


if __name__ == '__main__':
    unittest.main()
