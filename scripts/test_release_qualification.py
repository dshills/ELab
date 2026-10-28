import argparse
import json
from pathlib import Path
import stat
import sys
import tempfile
import unittest
from unittest.mock import patch

import release_qualification as release


class ReleaseQualificationTests(unittest.TestCase):
    def package(self, root):
        package = root/'package'
        package.mkdir()
        (package/'elab-agent').write_bytes(b'owned executable')
        (package/'ELab.app').mkdir()
        (package/'ELab.app/native').write_bytes(b'native')
        data = dict(schema=1, source_commit='a'*40, app='ELab.app', agent='elab-agent', files=release.tree(package))
        (package/release.MANIFEST).write_text(json.dumps(data))
        return package, data

    def test_manifest_rejects_mutation_missing_extra_mode_and_link(self):
        for mutation in ['bytes', 'missing', 'extra', 'mode', 'link', 'path']:
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as tmp:
                package, data = self.package(Path(tmp))
                self.assertEqual(release.read_package(package), data)
                file = package/'elab-agent'
                if mutation == 'bytes':
                    file.write_bytes(b'changed')
                elif mutation == 'missing':
                    file.unlink()
                elif mutation == 'extra':
                    (package/'extra').write_text('unqualified')
                elif mutation == 'mode':
                    file.chmod(stat.S_IMODE(file.stat().st_mode) ^ stat.S_IXUSR)
                elif mutation == 'link':
                    (package/'link').symlink_to(file)
                else:
                    data['agent'] = '../owner'
                    (package/release.MANIFEST).write_text(json.dumps(data))
                with self.assertRaises(ValueError):
                    release.read_package(package)

    def test_outputs_never_overlap_or_replace_inputs(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for paths in [(root, root), (root, root/'nested')]:
                with self.assertRaises(ValueError):
                    release.separate(*paths)
            file = root/'valuable'
            file.write_text('keep')
            with self.assertRaises(ValueError):
                release.exclusive(file)
            self.assertEqual(file.read_text(), 'keep')
            broken = root/'broken'
            broken.symlink_to(root/'missing')
            with self.assertRaises(ValueError):
                release.exclusive(broken)

    def test_archive_refuses_unqualified_and_changed_package_before_writing(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            package, data = self.package(root)
            qualification = root/'qualification.json'
            qualification.write_text(json.dumps(dict(passed=True)))
            args = argparse.Namespace(package=package, artifacts=root/'owned', archive=root/'release.zip', qualification=qualification)
            with self.assertRaises(ValueError):
                release.archive(args)
            self.assertFalse(args.archive.exists())
            self.assertFalse(args.artifacts.exists())

    def test_clean_source_required_before_creating_assembly_outputs(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            args = argparse.Namespace(output=root/'out', work=root/'work', examples=root/'examples', licenses=root/'licenses', version='1.2.3', build=1)
            with patch.object(release.subprocess, 'check_output', return_value=' M workbench/model.go\n'):
                with self.assertRaises(ValueError):
                    release.assemble(args)
            self.assertFalse(args.output.exists())
            self.assertFalse(args.work.exists())

    def test_native_cli_refuses_other_platforms(self):
        with patch.object(release.sys, 'platform', 'linux'), self.assertRaises(SystemExit) as caught:
            release.main(['check', '--package', '/tmp/unopened', '--artifacts', '/tmp/uncreated'])
        self.assertEqual(caught.exception.code, 2)

    def test_valid_receipt_does_not_authorize_a_changed_package(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            package, data = self.package(root)
            qualification = root/'qualification.json'
            qualification.write_text(json.dumps(dict(passed=True, manifest_sha256=release.sha(package/release.MANIFEST), source_commit=data['source_commit'])))
            (package/'elab-agent').write_bytes(b'changed after qualification')
            args = argparse.Namespace(package=package, artifacts=root/'owned', archive=root/'release.zip', qualification=qualification)
            with self.assertRaises(ValueError):
                release.archive(args)
            self.assertFalse(args.archive.exists())
            self.assertFalse(args.artifacts.exists())

    def test_unexpected_release_error_is_recorded_without_replacing_failure_file(self):
        for preexisting in [False, True]:
            with self.subTest(preexisting=preexisting), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                out = root/'owned'
                def fail(args):
                    release.exclusive(args.artifacts)
                    args._owned = args.artifacts
                    if preexisting:
                        (out/'failure.json').write_text('preserve')
                    raise KeyError('controller field')
                with patch.object(release.sys, 'platform', 'darwin'), patch.object(release, 'qualify', side_effect=fail), self.assertRaises(KeyError):
                    release.main(['check', '--package', str(root/'package'), '--artifacts', str(out)])
                if preexisting:
                    self.assertEqual((out/'failure.json').read_text(), 'preserve')
                else:
                    self.assertEqual(json.loads((out/'failure.json').read_text())['error_type'], 'KeyError')
