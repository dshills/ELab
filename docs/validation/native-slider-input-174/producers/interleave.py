"""One strictly guarded public edit during an outstanding CUA hold attempt."""
from pathlib import Path
import datetime
import hashlib
import importlib.util
import json
import subprocess
import time

root = Path('/Users/dshills/Development/projects/ELab')
out = Path('/private/tmp/elab-slider-input-174')
runtime = Path('/private/tmp/elab-first-version-174-signed-runtime')
baseline = json.loads((out / 'before-agent-hold-rendered.json').read_text())
record = dict(case='agent-hold-01', status='waiting_for_cua_signal',
              scope='Controller/agent call order only; no exact OS mouse-down timestamp')
def write():
    (out / 'agent-interruption-result.json').write_text(json.dumps(record, indent=2) + '\n')
write()
client = None
try:
    deadline = time.monotonic() + 90
    signal = out / 'agent-hold-start.json'
    while not signal.exists():
        assert time.monotonic() < deadline, 'CUA signal deadline'
        time.sleep(.003)
    marker = json.loads(signal.read_text())
    assert marker['case'] == record['case']
    record['cua_signal'] = marker
    time.sleep(.5)
    report = json.loads((out / 'report.json').read_text())
    assert report['status'] == 'ready_for_cua'
    command = subprocess.check_output(['/bin/ps', '-p', str(report['owned_pid']), '-o', 'command='], text=True).strip()
    assert command == report['observed_command']
    assert hashlib.sha256((runtime / 'ELab.app/Contents/MacOS/ELab').read_bytes()).hexdigest() == report['native_sha256']
    spec = importlib.util.spec_from_file_location('strict_hold_client', root / 'scripts/persistence-demo.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    client = module.Client(str(out / 'endpoint.json'), command_preconditions=True, agent_binary=runtime / 'agent')
    before = client.ask('project.inspect')
    record['before'] = before
    assert before['command_revision'] == baseline['command_revision'], 'Hold ordering unavailable: command already changed'
    assert before['document']['generation'] == baseline['document']['generation']
    assert next(p['wiper_position'] for p in before['components'] if p['id'] == 'Timing') == .5
    request = dict(version=1, id='agent-hold-01', key='agent-hold-01-single-attempt',
                   command='potentiometer.wiper.set', params=dict(id='Timing', position=.8),
                   expected_command_revision=baseline['command_revision'],
                   expected_document_generation=baseline['document']['generation'])
    record.update(status='sending_exact_guarded_edit', request=request,
                  invoked_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    write()
    client.process.stdin.write(json.dumps(request) + '\n')
    client.process.stdin.flush()
    response = json.loads(client.process.stdout.readline())
    record.update(response=response, response_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
    assert 'error' not in response, response
    state = client.ask('simulation.wait')
    record['after'] = state
    assert state['command_revision'] == baseline['command_revision'] + 1
    assert next(p['wiper_position'] for p in state['components'] if p['id'] == 'Timing') == .8
    record['status'] = 'agent_edit_completed'
    write()
    print(json.dumps(dict(status=record['status'], command_revision=state['command_revision'])))
except BaseException as error:
    record.update(status='unavailable', error=type(error).__name__ + ': ' + str(error))
    write()
    raise
finally:
    if client is not None:
        client.close()
