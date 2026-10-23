"""Check cursor refusal/correction through a matching candidate or extracted agent."""
from pathlib import Path
import argparse
import hashlib
import importlib.util
import json
import time

root = Path('/Users/dshills/Development/projects/ELab')
stage = Path(__file__).resolve().parent
parser = argparse.ArgumentParser()
parser.add_argument('--connect')
parser.add_argument('--keep-open', action='store_true')
parser.add_argument('--agent-binary', type=Path)
parser.add_argument('--output', type=Path)
args = parser.parse_args()
if args.keep_open and not args.connect:
    parser.error('--keep-open requires an owned native endpoint')
build = json.loads((stage / 'build-result-signed.json').read_text())
binary = args.agent_binary or Path(build['review']) / 'elab-agent'
assert hashlib.sha256(binary.read_bytes()).hexdigest() == build['agent_sha256']
spec = importlib.util.spec_from_file_location('client', root / 'scripts/persistence-demo.py')
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
c = helper.Client(endpoint=args.connect, command_preconditions=True, agent_binary=binary)
report = dict(source_commit=build['source_commit'], agent_sha256=build['agent_sha256'],
              agent_binary=str(binary), transport='native-socket' if args.connect else 'headless-stdio',
              checks=[])


def raw(command, params, revision=None):
    c.sequence += 1
    request = dict(version=1, id=f'cursor-refusal-{c.sequence}', command=command,
                   params=params, expected_command_revision=c.command_revision if revision is None else revision,
                   key=f'cursor-refusal-{c.sequence}')
    c.process.stdin.write(json.dumps(request) + '\n')
    c.process.stdin.flush()
    line = c.process.stdout.readline()
    assert line, 'agent ended during cursor test'
    return json.loads(line)


def settle():
    """Wait for paused calculations/layout publication, without hiding refusals."""
    deadline = time.monotonic() + 5
    before = c.ask('simulation.wait')
    while time.monotonic() < deadline:
        time.sleep(.05)
        after = c.ask('project.inspect')
        if before == after:
            return after
        before = after
    raise AssertionError('paused state did not settle')


def capture():
    accepted = c.ask('simulation.verify', {'duration_s': .02, 'probe_ids': ['v:C1', 'i:C1']}, True)
    identifier = accepted['last_operation']
    deadline = time.monotonic() + 5
    while time.monotonic() < deadline:
        record = c.ask('operation.inspect', {'id': identifier})
        status = record['operation']['status']
        if status == 'completed':
            c.ask('recording.view.set', {'mode': 'recorded'}, True)
            return identifier
        assert status not in ('failed', 'canceled'), record
        time.sleep(.01)
    raise AssertionError('verification did not complete')


try:
    c.ask('example.rc', {'discard_changes': True}, True)
    c.ask('simulation.pause', {}, True)
    c.ask('simulation.restart', {}, True)
    c.ask('simulation.step', {'duration_s': .02}, True)
    for recorded in (False, True):
        identifier = capture() if recorded else None
        command = 'recording.cursor.set' if recorded else 'scope.cursor.set'
        for cursor, corrected in [('a', '4 ms'), ('b', '16 ms')]:
            for bad in ('oops', '10 Hz', '-1 ms', '30 ms', -1, .03):
                before = settle()
                params = dict(cursor=cursor, time_s=bad)
                if recorded:
                    params['operation_id'] = identifier
                response = raw(command, params)
                assert response['error']['code'] == 'invalid-params', response
                help = response['help']
                assert 'cursor ' + cursor.upper() in help['summary'], help
                assert '0 s to 0.02 s' in help['action'] and 'ms' in help['action'], help
                assert help['details'] == response['error']['message'], help
                assert before == c.ask('project.inspect'), 'refusal changed accepted state'
            corrected_params = dict(cursor=cursor, time_s=corrected)
            if recorded:
                corrected_params['operation_id'] = identifier
            accepted = c.ask(command, corrected_params, True)
            scope = accepted['recording']['scope'] if recorded else accepted['scope']
            assert scope[cursor]['valid'] and scope[cursor]['requested_s'] == (.004 if cursor == 'a' else .016), scope
        before = settle()
        scope = before['recording']['scope'] if recorded else before['scope']
        assert abs(scope['delta_s'] - .012) < 1e-12 and abs(scope['interval_hz'] - 1/.012) < 1e-9, scope
        old_revision = before['command_revision']
        c.ask(command, {'cursor': 'a', 'time_s': '8 ms'}, True)
        current = settle()
        stale = raw(command, {'cursor': 'a', 'time_s': '30 ms'}, old_revision)
        assert stale['error']['code'] == 'conflict' and 'help' not in stale, stale
        stale_parser = raw(command, {'cursor': 'a', 'time_s': 'oops'}, old_revision)
        assert stale_parser['error']['code'] == 'invalid-params' and stale_parser['help']['summary'] == 'That request could not be applied.', stale_parser
        malformed = raw(command, {'cursor': 'a', 'time_s': 'oops', 'extra': True})
        assert malformed['error']['code'] == 'invalid-params' and malformed['help']['summary'] == 'That request could not be applied.', malformed
        if recorded:
            wrong_capture = raw(command, {'cursor': 'a', 'time_s': 'oops', 'operation_id': 'missing-recording'})
            assert wrong_capture['error']['code'] == 'invalid-params' and wrong_capture['help']['summary'] == 'That request could not be applied.', wrong_capture
        assert current == c.ask('project.inspect'), 'malformed/stale requests changed accepted state'
        report['checks'].append(dict(mode='recorded' if recorded else 'live', cursors=['a', 'b'], bad_values_per_cursor=6,
                                     refused_state_preserved=True, unit_correction=True, delta_s=.012,
                                     stale_requests_cannot_borrow_guidance=True))
    if args.keep_open:
        report.update(passed=True, kept_running=True, shutdown=None)
    else:
        job = c.ask('app.close', {'discard_changes': True}, True)['shutdown']
        job = c.wait_for_shutdown_outcome(job)
        assert job['status'] == 'completed' and job['exit_ready'], job
        report.update(passed=True, kept_running=False, shutdown=job)
finally:
    c.close()
name = 'native-protocol-report.json' if args.connect else 'protocol-report.json'
(args.output or stage / name).write_text(json.dumps(report, indent=2) + '\n')
print(json.dumps(dict(passed=True, modes=len(report['checks']), transport=report['transport'], source_commit=report['source_commit'])))
