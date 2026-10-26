"""Exercise transport failure boundaries of the release acceptance controller."""
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import textwrap
import unittest

from acceptance_client import AcceptanceClient


class AcceptanceTransportTests(unittest.TestCase):
    def fixture(self, body, timeout_s=1):
        directory = tempfile.TemporaryDirectory(prefix='elab-acceptance-transport-')
        self.addCleanup(directory.cleanup)
        path = Path(directory.name)/'agent'
        path.write_text('#!'+sys.executable+'\n'+textwrap.dedent(body))
        path.chmod(0o700)
        client = AcceptanceClient(path, timeout_s=timeout_s)
        self.addCleanup(client.close)
        return client

    def test_fragmented_response_and_own_command_guard(self):
        client = self.fixture('''
            import sys,json,time
            for line in sys.stdin:
                request=json.loads(line)
                if request['command'] != 'project.inspect':
                    assert request['expected_command_revision']==7
                    assert request['expected_document_generation']==3
                response=json.dumps(dict(id=request['id'],result=dict(command_revision=7,document=dict(generation=3))))+'\\n'
                for part in (response[:11],response[11:]):
                    sys.stdout.write(part);sys.stdout.flush();time.sleep(.001)
        ''')
        self.assertEqual(client.ask('view.fit', mutate=True)['command_revision'], 7)

    def test_no_response_times_out_without_retrying_an_ambiguous_command(self):
        client = self.fixture('''
            import sys
            for line in sys.stdin:
                pass
        ''', timeout_s=.05)
        with self.assertRaises(TimeoutError):
            client.ask('project.inspect')
        self.assertEqual(client.sequence, 1)

    def test_wrong_response_identity_refuses(self):
        client = self.fixture('''
            import sys,json
            for line in sys.stdin:
                print(json.dumps(dict(id='wrong',result={})),flush=True)
        ''')
        with self.assertRaisesRegex(RuntimeError, 'identity'):
            client.ask('project.inspect')


class InteractionReceiptTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        spec = importlib.util.spec_from_file_location('interaction_runner', Path(__file__).with_name('v1-interactions.py'))
        cls.runner = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(cls.runner)

    def receipt(self, entries):
        directory = tempfile.TemporaryDirectory(prefix='elab-interaction-receipt-')
        self.addCleanup(directory.cleanup)
        path = Path(directory.name)/'events.jsonl'
        path.write_text(''.join(json.dumps(x)+'\n' for x in entries))
        return path

    def test_empty_or_skipped_run_cannot_qualify_a_matrix(self):
        for events in ([dict(Action='pass')], [dict(Action='skip', Test='TestWanted'), dict(Action='pass')]):
            with self.subTest(events=events), self.assertRaises(ValueError):
                self.runner.summarize_events(self.receipt(events), ['TestWanted'])

    def test_failed_subtest_cannot_hide_behind_a_top_level_pass(self):
        events = [dict(Action='fail', Test='TestWanted/failed'),
                  dict(Action='pass', Test='TestWanted'), dict(Action='pass')]
        with self.assertRaises(ValueError):
            self.runner.summarize_events(self.receipt(events), ['TestWanted'])

    def test_current_matrix_resolves_and_complete_receipt_passes(self):
        matrix, selected, _ = self.runner.load_matrix(self.runner.DEFAULT)
        self.assertEqual(len(matrix['families']), 15)
        result = self.runner.summarize_events(self.receipt(
            [dict(Action='pass', Test=x) for x in selected]+[dict(Action='pass')]), selected)
        self.assertEqual(result['top_level_passed'], len(selected))


if __name__ == '__main__':
    unittest.main()
