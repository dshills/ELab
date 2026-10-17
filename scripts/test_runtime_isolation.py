"""Prevent false portability evidence and accidental reuse of user files."""
import copy
import json
from pathlib import Path
import plistlib
import subprocess
import sys
import tempfile
import unittest

import runtime_isolation as isolation


class RuntimeIsolationTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix='elab-runtime-unit-')
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name).resolve()

    def test_failed_and_interrupted_stages_record_terminal_state(self):
        for index, error in enumerate([subprocess.TimeoutExpired(['owned'], 1), KeyboardInterrupt(), subprocess.CalledProcessError(3, ['owned'])]):
            directory = self.root/str(index)
            directory.mkdir()
            result = dict(stages=[])
            journal = directory/'journal.json'
            def save():
                journal.write_text(json.dumps(result))
            def failed(*args, **kwargs):
                raise error
            with self.subTest(error=type(error).__name__), self.assertRaises(type(error)):
                isolation.execute_stage(result, directory, directory, 'owned', ['owned'], 1, failed, save)
            saved = json.loads(journal.read_text())['stages'][0]
            self.assertEqual(saved['status'], 'interrupted' if isinstance(error, KeyboardInterrupt) else 'failed')
            self.assertEqual(saved['error_type'], type(error).__name__)
            self.assertEqual(saved['exit_code'], getattr(error, 'returncode', None))

    def test_loading_the_pinned_runner_preserves_verified_sdk_and_bytecode_preference(self):
        previous = sys.dont_write_bytecode
        sdk, runner = isolation.pinned_child_runner()
        self.assertIs(sys.dont_write_bytecode, previous)
        self.assertTrue(callable(runner))
        again, _ = isolation.pinned_child_runner()  # Re-verifies every SDK file/inventory.
        self.assertEqual(sdk, again)
        self.assertIs(sys.dont_write_bytecode, previous)

    def test_permission_denial_must_be_distinguished_from_a_missing_file(self):
        result = dict(allowed_read=True, files={'source': dict(denied=True, errno=1)},
                      network={name: dict(denied=True, errno=1) for name in
                               ['ipv4_tcp', 'ipv4_udp', 'ipv6_tcp', 'ipv6_udp']})
        isolation.validate_controls(result, ['source'])
        changes = [
            lambda r: r.update(allowed_read=False),
            lambda r: r['files'].clear(),
            lambda r: r['files']['source'].update(errno=2),
            lambda r: r['files']['source'].update(errno=True),
            lambda r: r['files']['source'].update(denied=False),
            lambda r: r['network'].pop('ipv6_udp'),
            lambda r: r['network']['ipv4_tcp'].update(errno=110, denied=False),
        ]
        for change in changes:
            damaged = copy.deepcopy(result)
            change(damaged)
            with self.subTest(damaged=damaged), self.assertRaises(ValueError):
                isolation.validate_controls(damaged, ['source'])
        isolation.validate_controls(result, ['source'])

    def test_dependency_closure_rejects_relative_brew_and_traversal_paths(self):
        known = ['\t/System/Library/Frameworks/AppKit.framework/Versions/C/AppKit (compatibility version 45.0.0)',
                 '\t/usr/lib/libSystem.B.dylib (compatibility version 1.0.0)']
        self.assertEqual(len(isolation.native_dependencies('ELab:\n'+'\n'.join(known))), 2)
        for dependency in ['@rpath/libmissing.dylib', '/opt/homebrew/lib/libhidden.dylib',
                           '/System/Library/../../Users/hidden.dylib', '/usr/libevil/hidden.dylib']:
            with self.subTest(dependency=dependency), self.assertRaises(ValueError):
                isolation.native_dependencies('ELab:\n\t'+dependency+' (compatibility version 1.0.0)')
        with self.assertRaises(ValueError):
            isolation.native_dependencies('ELab:\n')

    def test_profile_quotes_paths_and_requires_specific_absolute_directories(self):
        profile = isolation.sandbox_profile([Path('/private/tmp/a "quote" and \\ slash/电路')])
        self.assertIn('\\"quote\\"', profile)
        self.assertIn('\\\\ slash', profile)
        self.assertIn('电路', profile)
        self.assertIn('(deny network-outbound (remote ip "*:*"))', profile)
        for paths in [[], [Path('/')], [Path('relative')], [Path('/tmp/new\nrule')]]:
            with self.subTest(paths=paths), self.assertRaises(ValueError):
                isolation.sandbox_profile(paths)

    def test_existing_and_blocked_destinations_preserve_contents(self):
        valuable = self.root/'existing'
        valuable.mkdir()
        notes = valuable/'notes.txt'
        notes.write_text('keep this circuit')
        with self.assertRaises(FileExistsError):
            isolation.create_output(valuable, [self.root/'development'])
        development = self.root/'development'
        development.mkdir()
        for path in [development, development/'new']:
            with self.subTest(path=path), self.assertRaises(ValueError):
                isolation.create_output(path, [development])
        self.assertEqual(notes.read_text(), 'keep this circuit')
        self.assertFalse((development/'new').exists())

    def bundle(self):
        app = self.root/'ELab.app'
        (app/'Contents/MacOS').mkdir(parents=True)
        (app/'Contents/Resources').mkdir()
        executable = app/'Contents/MacOS/ELab'
        executable.write_bytes(b'owned executable fixture')
        executable.chmod(0o755)
        (app/'Contents/Info.plist').write_bytes(plistlib.dumps(dict(CFBundleExecutable='ELab')))
        (app/'Contents/Resources/APP_MANIFEST.json').write_text(json.dumps(dict(
            private_build=True, source_binary_sha256=isolation.digest(executable))))
        return app, executable

    def test_modified_or_linked_bundle_inputs_refuse(self):
        app, executable = self.bundle()
        self.assertEqual(isolation.bundle_executable(app), executable)
        executable.write_bytes(b'changed executable')
        with self.assertRaisesRegex(ValueError, 'mismatch'):
            isolation.bundle_executable(app)
        target = self.root/'precious'
        target.write_text('original')
        (app/'Contents/Resources/link').symlink_to(target)
        with self.assertRaisesRegex(ValueError, 'link'):
            isolation.bundle_tree(app)
        self.assertEqual(target.read_text(), 'original')

    def test_executable_cannot_escape_bundle_and_copy_modes_are_evidence(self):
        app, executable = self.bundle()
        before = isolation.bundle_tree(app)
        executable.chmod(0o700)
        self.assertNotEqual(before, isolation.bundle_tree(app))
        for name in ['../outside', '/tmp/outside', '..', 'dir\\outside']:
            (app/'Contents/Info.plist').write_bytes(plistlib.dumps(dict(CFBundleExecutable=name)))
            with self.subTest(name=name), self.assertRaises(ValueError):
                isolation.bundle_executable(app)

    @unittest.skipUnless(Path('/bin/sh').is_file(), 'POSIX launcher syntax')
    def test_launcher_paths_with_quotes_spaces_and_unicode_are_shell_data(self):
        command = self.root/'launcher'
        isolation.launcher(command, self.root/'p "quoted".sb', self.root/"a 'quoted' 电路", self.root/'empty dir')
        subprocess.run(['/bin/sh', '-n', str(command)], check=True, capture_output=True)
        first = command.read_bytes()
        with self.assertRaises(FileExistsError):
            isolation.launcher(command, self.root/'other', self.root/'other', self.root)
        self.assertEqual(command.read_bytes(), first)
        self.assertIn('"$@"', command.read_text())


if __name__ == '__main__':
    unittest.main()
