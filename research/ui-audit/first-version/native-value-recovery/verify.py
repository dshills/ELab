"""Recheck the retained native/agent evidence without launching ELab."""
import hashlib
import json
from pathlib import Path


def verify():
    base = Path(__file__).resolve().parent
    read = lambda name: json.loads((base / name).read_text())
    session = read('after-native-session.json')
    reference = read('after-agent-reference.json')
    assert session['passed'] and session['normal_closure']
    assert session['host_exit_code'] == session['final_host_exit_code'] == 0
    states = {event['name']: event['state'] for event in session['events']}
    initial, refused, corrected = (states[key] for key in
                                   ('baseline', 'matched-invalid-draft', 'matched-corrected'))
    for key in ('components', 'wires', 'nets', 'result', 'run_id',
                'electrical_revision', 'document', 'undo_entry', 'redo_entry',
                'undo_name', 'redo_name'):
        assert initial.get(key) == refused.get(key), key
    for state, expected in zip((refused, corrected), reference['stages']):
        for key in ('components', 'wires', 'nets', 'selected', 'drawing',
                    'undo_name', 'redo_name', 'status', 'paused'):
            assert state[key] == expected['state'][key], key
        normalize = lambda result: {key: value for key, value in result.items()
                                    if key not in ('run_id', 'electrical_revision')}
        assert normalize(state['result']) == normalize(expected['state']['result'])
    assert corrected['circuit']['resistance_ohm'] == 2200
    assert not corrected['result']['stale']
    assert corrected['run_id'] == initial['run_id'] + 1
    assert corrected['electrical_revision'] == initial['electrical_revision'] + 1
    graph = read('corrected-value.elab')['graph']
    assert graph['components'] == corrected['components']
    assert graph['wires'] == corrected['wires']
    assert not read('before-native-session.json')['passed']
    assert not read('excluded-selection-timeout-session.json')['passed']
    assert read('process-cleanup.json')['passed']
    checkpoint = base / 'checkpoint.json'
    if checkpoint.exists():
        for name, digest in read('checkpoint.json')['artifact_sha256'].items():
            assert hashlib.sha256((base / name).read_bytes()).hexdigest() == digest, name
    print(json.dumps({'passed': True, 'scope': 'Stored evidence integrity and semantic comparison; no runtime replay'}))


if __name__ == '__main__':
    verify()
