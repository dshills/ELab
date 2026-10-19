"""Refuse changed or unsafe review archives before invoking macOS extraction."""
import hashlib
from pathlib import Path
import stat
import tempfile
import unittest
import warnings
import zipfile

import package_review


class ReviewArchiveTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix='elab-archive-unit-')
        self.addCleanup(temporary.cleanup)
        self.archive = Path(temporary.name)/'review.zip'
        self.expected = {'ELab.app/Contents/MacOS/ELab': dict(
            sha256=hashlib.sha256(b'qualified executable').hexdigest(), mode='0o755')}

    def write(self, entries):
        with zipfile.ZipFile(self.archive, 'w') as archive:
            for name, mode, data in entries:
                info = zipfile.ZipInfo(name)
                info.create_system = 3
                info.external_attr = mode << 16
                archive.writestr(info, data)

    def test_archive_must_keep_exact_bytes_and_executable_mode(self):
        name = 'review/ELab.app/Contents/MacOS/ELab'
        for mode, data in [(stat.S_IFREG | 0o755, b'changed executable'),
                           (stat.S_IFREG | 0o644, b'qualified executable'),
                           (stat.S_IFLNK | 0o755, b'qualified executable')]:
            with self.subTest(mode=mode, data=data):
                self.write([(name, mode, data)])
                with self.assertRaises(ValueError):
                    package_review.verify_archive(self.archive, 'review', self.expected)
        self.write([(name, stat.S_IFREG | 0o755, b'qualified executable')])
        package_review.verify_archive(self.archive, 'review', self.expected)

    def test_duplicate_and_extra_entries_cannot_reach_extraction(self):
        entry = ('review/ELab.app/Contents/MacOS/ELab', stat.S_IFREG | 0o755, b'qualified executable')
        for extra in [entry, ('review/extra', stat.S_IFREG | 0o644, b'extra'),
                      ('../outside', stat.S_IFREG | 0o644, b'outside')]:
            with self.subTest(extra=extra), warnings.catch_warnings():
                warnings.simplefilter('ignore', UserWarning)
                self.write([entry, extra])
                with self.assertRaises(ValueError):
                    package_review.verify_archive(self.archive, 'review', self.expected)

    def test_even_matching_manifest_paths_cannot_escape_or_be_links(self):
        for relative in ['../outside', '/outside', 'bad\\path']:
            with self.subTest(relative=relative):
                expected = {relative: dict(sha256=hashlib.sha256(b'payload').hexdigest(), mode='0o644')}
                self.write([('review/'+relative, stat.S_IFREG | 0o644, b'payload')])
                with self.assertRaises(ValueError):
                    package_review.verify_archive(self.archive, 'review', expected)


if __name__ == '__main__':
    unittest.main()
