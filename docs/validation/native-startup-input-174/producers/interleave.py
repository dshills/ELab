"""One strict agent edit during the declared CUA held-Reset call."""
from datetime import datetime, timezone
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import time

root = Path('/Users/dshills/Development/projects/ELab')
out = Path('/private/tmp/elab-startup-input-174')
runtime = Path('/private/tmp/elab-first-version-174-signed-runtime')
baseline = json.loads((out/'draft-rl-agent-held-reset.json').read_text())
record = {'case': 'rl-agent-held-reset', 'status': 'waiting_for_cua_signal',
          'scope': 'Controller call order; no exact OS mouse-down/up timestamps'}
client = None


def write():
    (out/'agent-interruption-result.json').write_text(json.dumps(record, indent=2)+'\n')


try:
    write()
    deadline = time.monotonic()+90
    while not (out/'agent-hold-start.json').exists():
        assert time.monotonic() < deadline, 'CUA signal deadline'
        time.sleep(.003)
    record['signal'] = json.loads((out/'agent-hold-start.json').read_text())
    assert record['signal']['case'] == record['case']
    time.sleep(.5)
    report = json.loads((out/'report.json').read_text())
    command = subprocess.check_output(['/bin/ps', '-p', str(report['owned_pid']), '-o', 'command='], text=True).strip()
    assert report['status'] == 'ready_for_cua' and command == report['observed_command']
    assert hashlib.sha256((runtime/'ELab.app/Contents/MacOS/ELab').read_bytes()).hexdigest() == report['native_sha256']
    spec = importlib.util.spec_from_file_location('startup_interleave_helper', root/'scripts/persistence-demo.py')
    helper = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(helper)
    client = helper.Client(str(out/'endpoint.json'), command_preconditions=True, agent_binary=runtime/'agent')
    before = client.ask('project.inspect')
    record['before'] = before
    assert before['document']['generation'] == baseline['document']['generation']
    assert before['selected'] == baseline['selected'] == 'L1' and before['paused']
    assert before['wires'] == baseline['wires']
    delta = before['command_revision']-baseline['command_revision']
    # Focus loss can publish only this user's private startup draft before the
    # held button releases. Do not adopt an unrelated edit or retry a conflict.
    assert delta in [0, 1], 'Ordering unavailable: Reset or other command already completed'
    expected = json.loads(json.dumps(baseline['components']))
    part = next(p for p in expected if p['id'] == 'L1')
    if delta == 1:
        part['initial_current_a'] = .03
        assert before['undo_entry']['actor'] == 'user'
        assert before['undo_entry']['name'] == 'Change inductor startup'
    assert before['components'] == expected, 'Unexpected graph before agent edit'
    request = {'version': 1, 'id': 'startup-hold-01', 'key': 'startup-hold-01-single-attempt',
               'command': 'inductor.initial.set', 'params': {'id': 'L1', 'value': '40 mA'},
               'expected_command_revision': before['command_revision'],
               'expected_document_generation': before['document']['generation']}
    record.update(status='sending_exact_guarded_edit', request=request,
                  invoked_utc=datetime.now(timezone.utc).isoformat())
    write()
    client.process.stdin.write(json.dumps(request)+'\n')
    client.process.stdin.flush()
    response = json.loads(client.process.stdout.readline())
    record.update(response=response, response_utc=datetime.now(timezone.utc).isoformat())
    assert 'error' not in response, response
    after = client.ask('simulation.wait')
    record['after'] = after
    assert after['command_revision'] == before['command_revision']+1
    assert next(p for p in after['components'] if p['id'] == 'L1')['initial_current_a'] == .04
    record['status'] = 'agent_edit_completed'
    write()
    print(json.dumps({'status': record['status'], 'before_command_revision': before['command_revision'],
                      'after_command_revision': after['command_revision']}))
except BaseException as error:
    record.update(status='unavailable', error=type(error).__name__+': '+str(error))
    write()
    raise
finally:
    if client is not None:
        client.close()
