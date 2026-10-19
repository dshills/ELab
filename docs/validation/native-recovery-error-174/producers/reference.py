"""One fresh packaged replay and two independent saved-project reopens."""
import hashlib
import importlib.util
import json
from pathlib import Path
import sys

root = Path('/Users/dshills/Development/projects/ELab')
out = Path('/private/tmp/elab-recovery-error-174')
runtime = Path('/private/tmp/elab-first-version-174-signed-runtime')
pin = 'a8636912f15394e647a773d36395fd7f0380d9315f4c16a3e599a51390e6aa54'
assert sys.flags.optimize == 0
assert hashlib.sha256((runtime/'elab-agent').read_bytes()).hexdigest() == pin
dest = out/'reference'
dest.mkdir(exist_ok=False)
spec = importlib.util.spec_from_file_location('recovery_reference_helper', root/'scripts/persistence-demo.py')
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
report = {'passed': False, 'agent_sha256': pin, 'transport': 'fresh packaged headless stdio', 'process_exits': [], 'reopens': []}


def save(name, value):
    with (dest/(name+'.json')).open('x') as stream:
        json.dump(value, stream, indent=2)
        stream.write('\n')


def capture(client, name):
    state = client.ask('simulation.wait')
    save(name, state)
    return state


def finish(client, name):
    try:
        closed = client.ask('app.close', {}, True)
        save(name+'-close', closed)
        assert closed['closed'] and closed['shutdown']['status'] == 'completed' and closed['shutdown']['exit_ready']
    finally:
        client.close()
        report['process_exits'].append({'name': name, 'exit_code': client.process.returncode})
        assert client.process.returncode == 0


client = helper.Client(command_preconditions=True, agent_binary=runtime/'agent')
try:
    client.ask('project.load', {'path': str(out/'Manual-RC.elab')}, True)
    client.ask('simulation.wait')
    client.ask('recording.view.set', {'mode': 'live'}, True)
    client.ask('simulation.step', {'duration_s': .01}, True)
    client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
    client.ask('simulation.restart', {}, True)
    client.ask('simulation.wait')
    client.ask('simulation.step', {'duration_s': .01}, True)
    capture(client, 'before-native-edit')
    client.ask('editor.value.set', {'value': '30k'}, True)
    client.ask('component.set', {'id': 'R1', 'value': '30k'}, True)
    capture(client, 'after-native-edit')
    client.ask('project.save', {'path': str(out/'Agent-Final-Replay.elab')}, True)
finally:
    finish(client, 'replay')

try:
    for name, filename, resistance in [('manual', 'Manual-RC.elab', 10000), ('final', 'Final-Recovery-174.elab', 30000)]:
        client = helper.Client(command_preconditions=True, agent_binary=runtime/'agent')
        try:
            client.ask('project.load', {'path': str(out/filename)}, True)
            state = capture(client, 'reopen-'+name)
            assert state['paused'] and not state['document']['dirty'] and state['document']['recording_count'] == 1
            assert state['transient']['sample']['time_s'] == 0
            assert next(p['value'] for p in state['components'] if p['id'] == 'R1') == resistance
            identifier = state['recording']['id']
            operation = client.ask('operation.inspect', {'id': identifier})
            save('reopen-'+name+'-operation', operation)
            expected = json.loads((out/filename).read_text())['recordings']['entries'][0]['data']
            assert operation['metadata'] == expected['metadata']
            offset, rows, pages = 0, [], []
            while offset < len(expected['rows']):
                page = client.ask('operation.samples', {'id': identifier, 'offset': offset, 'limit': 200})
                assert page['rows'] and len(page['rows']) <= 200
                pages.append(page)
                rows.extend(page['rows'])
                offset += len(page['rows'])
            save('reopen-'+name+'-pages', pages)
            assert rows == expected['rows']
            report['reopens'].append({'name': name, 'resistance_ohm': resistance, 'recorded_rows': len(rows), 'recording_exact': True})
        finally:
            finish(client, 'reopen-'+name)
    assert (out/'Agent-Final-Replay.elab').read_bytes() == (out/'Final-Recovery-174.elab').read_bytes()
    report.update(passed=True, native_agent_saved_project_bytes_identical=True)
finally:
    save('report', report)
print(json.dumps(report))
