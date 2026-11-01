"""Portable checks for the web launcher's release/fixture separation."""
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import package_web


class WebPackageTests(unittest.TestCase):
    def test_shipping_metadata_has_no_test_storage_or_native_identifier(self):
        info = package_web.bundle_info('0.1.0-alpha.1', '12')
        self.assertEqual(info['CFBundleIdentifier'], 'com.dshills.elab.web')
        self.assertEqual(info['CFBundleShortVersionString'], '0.1.0')
        self.assertNotIn('ELabWebTestDataDirectory', info)
        self.assertEqual(info['CFBundleExecutable'], 'ELabWebLauncher')

    def test_fixture_storage_is_explicit_and_cannot_be_distributed(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            info = package_web.bundle_info('0.1.0-alpha.1', '1', root / 'data')
            self.assertEqual(info['ELabWebTestDataDirectory'], str((root / 'data').resolve()))
            self.assertTrue(info['CFBundleIdentifier'].endswith('.test'))
            with patch('platform.system', return_value='Darwin'):
                with self.assertRaisesRegex(ValueError, 'cannot be archived'):
                    package_web.package(root / 'Test.app', '0.1.0-alpha.1', '1',
                                        archive=root / 'Test.zip', test_data=root / 'data')

    def test_existing_outputs_and_invalid_versions_are_preserved(self):
        for version, build in [('bad', '1'), ('0.1.0', 'one')]:
            with self.assertRaises(ValueError):
                package_web.bundle_info(version, build)
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory) / 'Existing.app'
            target.write_text('preserve')
            with patch('platform.system', return_value='Darwin'):
                with self.assertRaisesRegex(ValueError, 'new .app'):
                    package_web.package(target, '0.1.0-alpha.1', '1')
            self.assertEqual(target.read_text(), 'preserve')


if __name__ == '__main__':
    unittest.main()
