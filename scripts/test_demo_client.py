"""Exercise actual helper request/retry behavior without a live mutation target."""
import importlib.util
import io
import json
from pathlib import Path
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('persistence_client', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def response(code=None, revision=1, command_revision=1):
    result = dict(version=1, id='reply', revision=revision)
    if code:
        result['error'] = dict(code=code, message='controlled response')
    else:
        result['result'] = dict(components=[], document=dict(generation=1), command_revision=command_revision)
    return json.dumps(result)+'\n'


def client_with_responses(responses, command_preconditions=True):
    client = object.__new__(helper.Client)
    client.revision = client.command_revision = client.sequence = 0
    client.command_preconditions = command_preconditions
    client.process = type('ProcessFixture', (), {})()
    client.process.stdin = io.StringIO()
    client.process.stdout = io.StringIO(''.join(responses))
    return client


class ClientRetryTests(unittest.TestCase):
    def test_explicit_agent_binary_is_used_without_changing_the_default(self):
        for binary in (None, Path('/private/tmp/qualified-agent')):
            process = type('Fixture', (), dict(stdin=io.StringIO(), stdout=io.StringIO(response())))()
            with patch.object(helper.subprocess, 'Popen', return_value=process) as launch:
                helper.Client(agent_binary=binary)
                self.assertEqual(launch.call_args.args[0], [str(binary if binary is not None else helper.ROOT/'build/elab-agent')])

    def test_conflict_refreshes_only_guard_and_retains_exact_request_intent(self):
        for command_guard in (False, True):
            with self.subTest(command_guard=command_guard):
                client = client_with_responses([response('conflict', revision=7),
                    response(revision=8, command_revision=4), response(revision=9, command_revision=5)], command_guard)
                result = client.ask('component.set', {'id': 'R1', 'value': '2k'}, True)
                requests = [json.loads(line) for line in client.process.stdin.getvalue().splitlines()]
                first, query, retried = requests
                self.assertEqual(query['command'], 'project.inspect')
                guard = 'expected_command_revision' if command_guard else 'expected_revision'
                self.assertEqual(first.pop(guard), 0)
                self.assertEqual(retried.pop(guard), 4 if command_guard else 8)
                self.assertEqual(first, retried)
                self.assertEqual(result['command_revision'], 5)

    def test_expected_conflict_is_not_retried_or_refreshed(self):
        client = client_with_responses([response('conflict')])
        self.assertIsNone(client.ask('component.set', {'id': 'R1', 'value': '2k'}, True, 'conflict'))
        self.assertEqual(len(client.process.stdin.getvalue().splitlines()), 1)

    def test_nonconflict_failure_is_not_retried(self):
        client = client_with_responses([response('invalid-params')])
        with self.assertRaises(AssertionError):
            client.ask('component.set', {'id': 'R1', 'value': 'bad'}, True)
        self.assertEqual(len(client.process.stdin.getvalue().splitlines()), 1)

    def test_repeated_conflicts_stop_after_eight_attempts(self):
        responses = []
        for _ in range(7):
            responses += [response('conflict'), response()]
        responses += [response('conflict')]
        client = client_with_responses(responses)
        with self.assertRaises(AssertionError):
            client.ask('component.set', {'id': 'R1', 'value': '2k'}, True)
        requests = [json.loads(line) for line in client.process.stdin.getvalue().splitlines()]
        self.assertEqual(sum(r['command'] == 'component.set' for r in requests), 8)
        self.assertEqual(sum(r['command'] == 'project.inspect' for r in requests), 7)
        self.assertEqual(len({r['key'] for r in requests if r['command'] == 'component.set'}), 1)

    def test_transport_eof_is_not_retried_as_a_conflict(self):
        client = client_with_responses([])
        with self.assertRaisesRegex(AssertionError, 'agent terminated'):
            client.ask('project.inspect')
        self.assertEqual(len(client.process.stdin.getvalue().splitlines()), 1)


if __name__ == '__main__':
    unittest.main()
