import json
import hashlib
from pathlib import Path
import plistlib
import stat
import subprocess
import tempfile
import unittest
import warnings
from unittest.mock import patch
import zipfile

import check_installation
import distribution
import package_web


class DistributionTests(unittest.TestCase):
    def test_identity_requires_exact_installed_developer_id(self):
        output = ('  1) '+ 'A'*40+' "Apple Development: Owner (ABCDEFGHIJ)"\n'
                  '  2) '+ 'B'*40+' "Developer ID Application: Owner (ABCDEFGHIJ)"\n')
        with patch.object(distribution, 'run', return_value=subprocess.CompletedProcess([], 0, output, '')):
            self.assertEqual(distribution.developer_identity('b'*40), 'B'*40)
            self.assertEqual(distribution.developer_identity('Developer ID Application: Owner (ABCDEFGHIJ)'), 'B'*40)
            for value in ['-', 'A'*40, 'Owner', 'Developer ID Application']:
                with self.assertRaises(ValueError):
                    distribution.developer_identity(value)

    def test_signature_rejects_adhoc_missing_runtime_timestamp_or_debugging(self):
        good = ('Authority=Developer ID Application: Owner (ABCDEFGHIJ)\n'
                'TeamIdentifier=ABCDEFGHIJ\nCodeDirectory v=20500 flags=0x10000(runtime)\n'
                'Timestamp=Nov 4, 2026\n')
        cases = [(good, {}, True), (good.replace('Developer ID Application:', 'Apple Development:'), {}, False),
                 (good.replace('0x10000(runtime)', '0x2(adhoc)'), {}, False),
                 (good.replace('Timestamp=Nov 4, 2026\n', ''), {}, False),
                 (good, {'com.apple.security.get-task-allow': True}, False)]
        for details, entitlements, accepted in cases:
            with self.subTest(details=details, entitlements=entitlements):
                def run(command, **kwargs):
                    output = plistlib.dumps(entitlements).decode() if '--entitlements' in command else ''
                    return subprocess.CompletedProcess(command, 0, output, details)
                with patch.object(distribution, 'run', side_effect=run):
                    if accepted:
                        self.assertEqual(distribution.signature(Path('app'))['team'], 'ABCDEFGHIJ')
                    else:
                        with self.assertRaises(ValueError):
                            distribution.signature(Path('app'))

    def test_notary_acceptance_staples_then_checks_distribution(self):
        calls = []
        def run(command, **kwargs):
            calls.append([str(v) for v in command])
            return subprocess.CompletedProcess(command, 0, json.dumps({'status':'Accepted', 'id':'12345678-1234-1234-1234-123456789abc'}), '')
        def verify(app):
            self.assertEqual(calls[-1][:3], ['xcrun', 'stapler', 'staple'])
            return dict(stapled_ticket_valid=True, gatekeeper_accepted=True)
        with patch.object(distribution, 'run', side_effect=run), patch.object(distribution, 'verify_distribution', side_effect=verify):
            result = distribution.notarize(Path('App.app'), Path('work'), 'ELab-notary')
        self.assertEqual(result['status'], 'Accepted')
        self.assertTrue(result['gatekeeper_accepted'])
        self.assertIn('--wait', calls[1])
        self.assertIn('--keychain-profile', calls[1])
        self.assertNotIn('--password', calls[1])

    def test_pending_rejected_and_malformed_notary_replies_never_staple(self):
        for reply in [{'status':'In Progress'}, {'status':'Invalid'}, {'status':'Accepted'},
                      {'status':'Accepted', 'id':'bad'}]:
            with self.subTest(reply=reply), patch.object(distribution, 'run', return_value=subprocess.CompletedProcess([], 0, json.dumps(reply), '')) as run:
                with self.assertRaises(ValueError):
                    distribution.notarize(Path('App.app'), Path('work'), 'profile')
                self.assertFalse(any('stapler' in call.args[0] for call in run.call_args_list))

    def test_accepted_submission_cannot_hide_failed_gatekeeper_check(self):
        with patch.object(distribution, 'run', return_value=subprocess.CompletedProcess([], 0, json.dumps({'status':'Accepted','id':'12345678-1234-1234-1234-123456789abc'}), '')), \
             patch.object(distribution, 'verify_distribution', side_effect=subprocess.CalledProcessError(1, ['spctl'])):
            with self.assertRaises(subprocess.CalledProcessError):
                distribution.notarize(Path('App.app'), Path('work'), 'profile')

    def test_fixture_and_dirty_source_cannot_be_notarized(self):
        with tempfile.TemporaryDirectory() as directory, patch.object(package_web.platform, 'system', return_value='Darwin'):
            app = Path(directory)/'App.app'
            with self.assertRaisesRegex(ValueError, 'test-configured'):
                package_web.package(app, '1.0.0', '1', test_data=directory, notary_profile='profile')
            with self.assertRaisesRegex(ValueError, 'Keychain profile'):
                package_web.package(app, '1.0.0', '1', notary_profile='')
            with patch.object(package_web, 'developer_identity', return_value='B'*40), \
                 patch.object(package_web, 'source_snapshot', return_value={}), \
                 patch.object(package_web, 'run', side_effect=['a'*40, ' M source.go']):
                with self.assertRaisesRegex(ValueError, 'clean committed source'):
                    package_web.package(app, '1.0.0', '1', identity='B'*40, notary_profile='profile')
            self.assertFalse(app.exists())

    def test_distribution_requires_one_team_and_shipping_bundle(self):
        with tempfile.TemporaryDirectory() as directory:
            app=Path(directory)/'App.app';(app/'Contents').mkdir(parents=True)
            info=app/'Contents/Info.plist'
            info.write_bytes(plistlib.dumps({'CFBundleIdentifier':'com.dshills.elab.web.test'}))
            with self.assertRaises(ValueError):
                distribution.verify_distribution(app)
            info.write_bytes(plistlib.dumps({'CFBundleIdentifier':'com.dshills.elab.web'}))
            with patch.object(distribution,'signature',side_effect=[{'team':'ABCDEFGHIJ'},{'team':'KLMNOPQRST'}]), \
                 patch.object(distribution,'run') as run:
                with self.assertRaisesRegex(ValueError,'teams differ'):
                    distribution.verify_distribution(app)
                run.assert_not_called()

    def test_install_archive_refuses_traversal_links_duplicates_and_size(self):
        for name, mode in [('../outside',stat.S_IFREG),('/outside',stat.S_IFREG),
                           ('App.app/link',stat.S_IFLNK),('App.app/../outside',stat.S_IFREG),
                           ('App.app\\outside',stat.S_IFREG)]:
            with self.subTest(name=name),tempfile.TemporaryDirectory() as directory:
                path=Path(directory)/'bad.zip'
                with zipfile.ZipFile(path,'w') as archive:
                    entry=zipfile.ZipInfo(name);entry.external_attr=(mode|0o644)<<16
                    archive.writestr(entry,'data')
                with self.assertRaises(ValueError):
                    check_installation.safe_archive(path)
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'good.zip'
            with zipfile.ZipFile(path,'w') as archive:
                archive.writestr('App.app/Contents/file','data')
            self.assertEqual(check_installation.safe_archive(path),'App.app')
            with self.assertRaises(ValueError):
                check_installation.safe_archive(path,byte_limit=1)
            with self.assertRaisesRegex(ValueError,'SHA-256'):
                check_installation.check(path,'0'*64,Path(directory)/'report')
            self.assertFalse((Path(directory)/'report').exists())
            with warnings.catch_warnings():
                warnings.simplefilter('ignore', UserWarning)
                with zipfile.ZipFile(path,'a') as archive:
                    archive.writestr('App.app/Contents/file','replacement')
            with self.assertRaisesRegex(ValueError,'unsafe'):
                check_installation.safe_archive(path)

    def test_failed_gatekeeper_check_never_runs_the_artifact(self):
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory);archive=root/'app.zip'
            with zipfile.ZipFile(archive,'w') as zipped:
                zipped.writestr('App.app/Contents/file','data')
            checksum=hashlib.sha256(archive.read_bytes()).hexdigest()
            with patch.object(check_installation.subprocess,'run'), \
                 patch.object(check_installation,'verify_bundle'), \
                 patch.object(check_installation,'verify_distribution',side_effect=subprocess.CalledProcessError(1,['spctl'])), \
                 patch.object(check_installation,'qualify') as qualify:
                with self.assertRaises(subprocess.CalledProcessError):
                    check_installation.check(archive,checksum,root/'report')
                qualify.assert_not_called()
            self.assertFalse((root/'report/installation.json').exists())


if __name__ == '__main__':
    unittest.main()
