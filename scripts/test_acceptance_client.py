"""Exercise transport failure boundaries of the release acceptance controller."""
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



if __name__ == '__main__':
    unittest.main()
