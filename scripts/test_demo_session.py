"""Protect connected circuits from destructive demonstration setup/cleanup."""
import copy
from pathlib import Path
import subprocess
import sys
import unittest

from demo_session import DemoSessionRefused, connect_demo_client, require_fresh_demo_session


FRESH = dict(closed=False, components=[], wires=[], operations=[], document=dict(
    generation=1, path='', dirty=False, schematic_dirty=False,
    preferences_dirty=False, recordings_dirty=False, recording_count=0))


class ClientFixture:
    def __init__(self, snapshot=None, read_error=None, close_error=None):
        self.snapshot = snapshot
        self.read_error = read_error
        self.close_error = close_error
        self.queries = []
        self.closed = False

    def ask(self, command):
        self.queries.append(command)
        if self.read_error:
            raise self.read_error
        return self.snapshot

    def close(self):
        self.closed = True
        if self.close_error:
            raise self.close_error


class DemoSessionTests(unittest.TestCase):
    def test_fresh_blank_is_read_only_and_admissible(self):
        snapshot = copy.deepcopy(FRESH)
        require_fresh_demo_session(snapshot)
        self.assertEqual(snapshot, FRESH)

    def test_blank_graph_does_not_imply_disposable_work(self):
        changes = [
            ('document', 'path', '/private/valuable-empty.elab'),
            ('document', 'generation', 2),
            ('document', 'generation', True),
            ('document', 'dirty', True),
            ('document', 'schematic_dirty', True),
            ('document', 'preferences_dirty', True),
            ('document', 'recordings_dirty', True),
            ('document', 'recording_count', 1),
            (None, 'components', [{'id': 'private-part'}]),
            (None, 'wires', [{'id': 'private-wire'}]),
            (None, 'undo_entry', {'name': 'Delete last component'}),
            (None, 'redo_entry', {'name': 'Place component'}),
            (None, 'operations', [{'id': 'kept-capture'}]),
            (None, 'project_work', {'status': 'running'}),
            (None, 'step_work', {'status': 'running'}),
            (None, 'input_work', {'status': 'running'}),
            (None, 'actuation_work', {'status': 'running'}),
            (None, 'print_preview', {'status': 'completed'}),
            (None, 'shutdown', {'status': 'running'}),
            (None, 'last_operation', 'retained-result'),
            (None, 'closed', True),
        ]
        for parent, key, value in changes:
            with self.subTest(parent=parent, key=key):
                snapshot = copy.deepcopy(FRESH)
                (snapshot[parent] if parent else snapshot)[key] = value
                before = copy.deepcopy(snapshot)
                with self.assertRaises(DemoSessionRefused) as refusal:
                    require_fresh_demo_session(snapshot)
                self.assertEqual(snapshot, before)
                self.assertNotIn('valuable-empty.elab', str(refusal.exception))
                self.assertNotIn('private-part', str(refusal.exception))

    def test_completed_native_startup_setup_is_not_an_experiment(self):
        setup = dict(command='recovery.configure', status='completed', phase='completed',
            recovery_configuration_applied=True, document_generation=1,
            result_document_generation=1, bytes=0)
        require_fresh_demo_session(dict(FRESH, project_work=setup))
        for key, value in [('command', 'project.load'), ('command', 'project.save'),
                ('status', 'running'), ('phase', 'installing'),
                ('recovery_configuration_applied', False), ('document_generation', 2),
                ('result_document_generation', 2), ('bytes', 1), ('error', {'code': 'failed'})]:
            with self.subTest(key=key, value=value), self.assertRaises(DemoSessionRefused):
                require_fresh_demo_session(dict(FRESH, project_work=dict(setup, **{key: value})))

    def test_unknown_snapshots_refuse_instead_of_assuming_blank(self):
        for snapshot in (None, [], {}, dict(components=[], wires=[], document={})):
            with self.subTest(snapshot=snapshot), self.assertRaises(DemoSessionRefused):
                require_fresh_demo_session(snapshot)

    def test_refusal_closes_only_forwarding_client(self):
        client = ClientFixture(dict(FRESH, components=[{'id': 'R1'}]))
        with self.assertRaises(DemoSessionRefused):
            connect_demo_client(lambda endpoint: client, 'owned-endpoint')
        self.assertTrue(client.closed)
        self.assertEqual(client.queries, ['project.inspect'])

    def test_failed_inspection_preserves_original_error_and_closes_proxy(self):
        error = OSError('read failed')
        client = ClientFixture(read_error=error)
        with self.assertRaises(OSError) as raised:
            connect_demo_client(lambda endpoint: client, 'owned-endpoint')
        self.assertIs(raised.exception, error)
        self.assertTrue(client.closed)
        self.assertEqual(client.queries, ['project.inspect'])

    def test_cleanup_failure_does_not_mask_admission_failure(self):
        cleanup = OSError('proxy cleanup failed')
        client = ClientFixture(None, close_error=cleanup)
        with self.assertRaises(DemoSessionRefused) as raised:
            connect_demo_client(lambda endpoint: client, 'owned-endpoint')
        self.assertIs(raised.exception.__cause__, cleanup)
        self.assertTrue(client.closed)

    def test_success_preserves_client_and_factory_options(self):
        client = ClientFixture(copy.deepcopy(FRESH))
        calls = []

        def factory(endpoint, **kwargs):
            calls.append((endpoint, kwargs))
            return client

        admitted = connect_demo_client(factory, 'owned-endpoint', command_preconditions=True)
        self.assertIs(admitted, client)
        self.assertEqual(calls, [('owned-endpoint', {'command_preconditions': True})])
        self.assertEqual(client.queries, ['project.inspect'])
        self.assertFalse(client.closed)

    def test_general_client_library_import_needs_no_demo_module_path(self):
        root = Path(__file__).resolve().parents[1]
        code = ("import importlib.util,sys; "
                "spec=importlib.util.spec_from_file_location('client_library',sys.argv[1]); "
                "module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module); "
                "assert callable(module.Client)")
        imported = subprocess.run([sys.executable, '-I', '-c', code,
            str(root/'scripts/persistence-demo.py')], cwd=root, capture_output=True,
            text=True, timeout=10)
        self.assertEqual(imported.returncode, 0, imported.stderr)

    def test_new_headless_client_needs_no_external_admission(self):
        client = ClientFixture()
        self.assertIs(connect_demo_client(lambda endpoint: client), client)
        self.assertEqual(client.queries, [])
        self.assertFalse(client.closed)


if __name__ == '__main__':
    unittest.main()
