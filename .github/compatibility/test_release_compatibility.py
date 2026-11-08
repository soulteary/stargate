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


if __name__ == '__main__':
    unittest.main()
