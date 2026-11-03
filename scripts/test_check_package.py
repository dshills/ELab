import hashlib
from pathlib import Path
import tempfile
import unittest

from check_package import verify_extracted


class ExtractedPackageTests(unittest.TestCase):
    def test_extracted_package_preserves_exact_inventory_bytes_modes_and_ownership(self):
        for change in ['none', 'bytes', 'mode', 'missing', 'extra', 'symlink']:
            with self.subTest(change=change), tempfile.TemporaryDirectory() as directory:
                app = Path(directory)/'App.app'
                app.mkdir()
                binary = app/'binary'
                binary.write_bytes(b'ELab test executable')
                binary.chmod(0o755)
                files = {'binary': {'sha256': hashlib.sha256(binary.read_bytes()).hexdigest(), 'mode': 0o755}}
                if change == 'bytes':
                    binary.write_bytes(b'changed')
                elif change == 'mode':
                    binary.chmod(0o644)
                elif change == 'missing':
                    binary.unlink()
                elif change == 'extra':
                    (app/'extra').write_text('unexpected')
                elif change == 'symlink':
                    target = Path(directory)/'outside'
                    binary.rename(target)
                    binary.symlink_to(target)
                if change == 'none':
                    verify_extracted(app, files)
                else:
                    with self.assertRaises(ValueError):
                        verify_extracted(app, files)
