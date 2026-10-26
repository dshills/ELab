"""Keep the selected upstream packaging tool exact before executing it."""
import json
from pathlib import Path
import shutil
import tempfile
import unittest

import package_app


class PackagerPinTests(unittest.TestCase):
    def fixture(self):
        temporary = tempfile.TemporaryDirectory(prefix='elab-icon-pin-')
        self.addCleanup(temporary.cleanup)
        root = Path(temporary.name)
        pin = package_app.ROOT / 'third_party/vellum/packager.lock.json'
        relative = json.loads(pin.read_text())['root']
        shutil.copytree(package_app.ROOT / relative, root / relative)
        shutil.copyfile(pin, root / 'third_party/vellum/packager.lock.json')
        return root, root / relative

    def test_current_packager_is_complete_and_exact(self):
        self.assertTrue((package_app.verify_packager() / 'scripts/app_icons.py').is_file())

    def test_changed_helper_is_refused(self):
        root, tools = self.fixture()
        with (tools / 'scripts/app_icons.py').open('a') as stream:
            stream.write('\n# changed helper\n')
        with self.assertRaisesRegex(ValueError, 'file changed'):
            package_app.verify_packager(root)

    def test_unpinned_import_and_symlink_are_refused(self):
        root, tools = self.fixture()
        extra = tools / 'scripts/unpinned.py'
        extra.write_text('# unpinned import\n')
        with self.assertRaisesRegex(ValueError, 'inventory changed'):
            package_app.verify_packager(root)
        extra.unlink()
        helper = tools / 'scripts/app_icons.py'
        helper.unlink()
        helper.symlink_to(tools / 'scripts/app_notices.py')
        with self.assertRaisesRegex(ValueError, 'symlink'):
            package_app.verify_packager(root)


if __name__ == '__main__':
    unittest.main()
