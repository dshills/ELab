"""Read-only verification of retained native draft/replacement observations."""
from pathlib import Path
import gzip
import hashlib
import json
import math

HERE = Path(__file__).resolve().parent
checkpoint = json.loads((HERE / 'checkpoint.json').read_text())
for name, digest in checkpoint['evidence_sha256'].items():
    assert hashlib.sha256((HERE / name).read_bytes()).hexdigest() == digest, name
states = json.loads(gzip.decompress((HERE / 'states.json.gz').read_bytes()))
before = states['04-selected.json']
typed = states['05-typed.json']
old = states['09-before-replacement.json']
loaded = states['10-replacement.json']
selected = states['11-reselected.json']
applied = states['12-fresh-apply.json']
saved = states['13-saved.json']
agent = states['15-agent-replay.json']


def resistor(state):
    return next(c['value'] for c in state['components'] if c['id'] == 'R2')


def compare(left, right):
    if isinstance(left, dict):
        ignored = {'run_id', 'electrical_revision'}
        assert left.keys() - ignored == right.keys() - ignored
        for key in left.keys() - ignored:
            compare(left[key], right[key])
    elif isinstance(left, list):
        assert len(left) == len(right)
        for a, b in zip(left, right):
            compare(a, b)
    elif isinstance(left, (int, float)) and not isinstance(left, bool):
        assert math.isclose(left, right, rel_tol=1e-9, abs_tol=1e-12)
    else:
        assert left == right, (left, right)


assert before['command_revision'] == typed['command_revision'] == old['command_revision']
assert before['components'] == typed['components'] == old['components']
assert resistor(typed) == 1000 and typed['drawing']['value'] == '1 kΩ'
assert 'Value: 2.2k' in (HERE / '05-typed.ax.txt').read_text()
assert not old['document']['dirty']
assert loaded['document']['generation'] > old['document']['generation']
assert loaded['selected'] == '' and loaded['drawing']['value'] == ''
assert 'Component value' not in (HERE / '10-replacement.ax.txt').read_text()
assert 'Value: 3.3 kΩ' in (HERE / '11-reselected.ax.txt').read_text()
assert resistor(loaded) == resistor(selected) == resistor(applied) == resistor(saved) == 3300
assert selected['components'] == applied['components'] == saved['components']
assert selected['wires'] == applied['wires'] == saved['wires']
assert selected.get('undo_entry') == applied.get('undo_entry')
assert not applied['document']['dirty'] and not saved['document']['dirty']
assert not applied['result']['stale'] and states['14-closed.json']['closed']
assert agent['components'] == applied['components'] and agent['wires'] == applied['wires']
compare(applied['result'], agent['result'])
for name, value in [('Baseline.elab', 1000), ('Replacement.elab', 3300), ('Final.elab', 3300)]:
    graph = json.loads((HERE / name).read_text())['graph']
    assert next(c['value'] for c in graph['components'] if c['id'] == 'R2') == value
assert json.loads((HERE / 'Final.elab').read_text())['graph'] == json.loads((HERE / 'Replacement.elab').read_text())['graph']
assert all(not app['isRunning'] for app in json.loads((HERE / '16-app-cleanup.json').read_text()))
print(json.dumps({'passed': True, 'scope': 'physical native unfinished-value/document replacement and fresh Apply; OS deactivation unqualified', 'resistance_ohm': 3300, 'current_a': agent['result']['current_a']}))
