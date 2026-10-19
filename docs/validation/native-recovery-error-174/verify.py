#!/usr/bin/env python3
"""Check preserved native recovery evidence offline; never launch an app."""
import gzip
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import sys

HERE = Path(__file__).resolve().parent


def read(name):
    data = (HERE/name).read_bytes()
    return gzip.decompress(data) if name.endswith('.gz') else data


def load(name):
    return json.loads(read(name))


def state(name, prefix='native'):
    return load(prefix+'/'+name+'.json.gz')


def resistor(s):
    return next(p['value'] for p in s['components'] if p['id'] == 'R1')


def recorded_voltage(at):
    """Independent exact 10 kohm / 1 uF, 0/5 V, 50 ms phase solution."""
    voltage, source, previous = 0., 5., 0.
    for i in range(1, 5):
        edge = i*.05
        if edge >= at-1e-12:
            return source+(voltage-source)*math.exp(-(at-previous)/.01)
        voltage = source+(voltage-source)*math.exp(-(edge-previous)/.01)
        previous, source = edge, 5-source
    raise AssertionError('Recorded reference horizon exceeded')


def verify():
    assert sys.flags.optimize == 0, 'Run without Python optimization'
    manifest = load('manifest.json')['files']
    for name, entry in manifest.items():
        relative = Path(name)
        assert not relative.is_absolute() and '..' not in relative.parts
        p = HERE/relative
        assert p.is_file() and not p.is_symlink(), name
        data = p.read_bytes()
        assert len(data) == entry['bytes'] and hashlib.sha256(data).hexdigest() == entry['sha256'], name
        original = gzip.decompress(data) if entry['gzip'] else data
        assert len(original) == entry['original_bytes'], name
        assert hashlib.sha256(original).hexdigest() == entry['original_sha256'], name
    spec = importlib.util.spec_from_file_location('recovery_comparison', HERE/'producers/comparison-helper.py')
    comparison = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(comparison)
    same = comparison.same
    report, receipt = state('report'), load('release-receipt.json.gz')
    assert report['status'] == 'completed' and 'error' not in report and 'cleanup_error' not in report
    for key in ['native_sha256', 'agent_sha256']:
        assert report[key] == receipt[key]
    assert report['same_pid_after_exec'] and report['open_exit_code'] == 0
    assert report['owned_process_absent'] and report['endpoint_removed']
    assert report['forced_cleanup'] is False and report['native_exit_status'] is None
    for key in ['shared_geometry_unchanged', 'review_unchanged', 'copied_bundle_unchanged',
                'canonical_unchanged', 'baseline_file_unchanged']:
        assert report[key], key
    assert report['shared_geometry_before'] == report['shared_geometry_after']
    assert all(c['denied'] and c['errno'] in [1, 13] for c in report['controls'].values())
    assert hashlib.sha256(read('native/Manual-RC.elab')).hexdigest() == report['baseline_sha256']
    closed = state('close-response')
    assert closed['closed'] and not closed['document']['dirty']
    assert closed['shutdown']['status'] == 'completed'
    assert all(closed['shutdown'][k] for k in ['workers_joined', 'exit_ready', 'recovery_finalized'])

    before = state('before-storage-break')
    failure = state('automatic-failure')
    protected = ['components', 'wires', 'transient', 'scope', 'document', 'run_id',
                 'electrical_revision', 'undo_entry', 'redo_entry', 'paused']
    for key in protected:
        assert before.get(key) == failure.get(key), key
    assert not before['recovery'].get('error') and failure['recovery']['error']
    assert failure['recovery']['enabled'] and failure['recovery']['interval_s'] == 2
    error = failure['recovery']['error']
    assert error.endswith('recovery: not a directory')
    for name in ['automatic-failure-visible', 'failure-detail-complete', 'failure-native-draft', 'failure-after-native-edit']:
        ax = read('native/'+name+'.ax.txt.gz').decode()
        assert 'Autosave needs attention' in ax and error in ax
        assert 'ID: node:cmVjb3ZlcnktZXJyb3I' in ax
    original, draft, edited = [state(n) for n in ['before-native-edit', 'private-native-draft', 'after-native-edit']]
    assert original == draft
    assert resistor(original) == 20000 and resistor(edited) == 30000
    assert edited['command_revision'] == original['command_revision']+2
    assert edited['undo_entry'] == {'name': 'Change value', 'actor': 'user', 'command_revision': 17}
    assert edited['paused'] and edited['transient']['stale'] and edited['document']['dirty']
    assert edited['transient']['sample'] == original['transient']['sample']
    assert edited['recovery']['error'] == error
    retry = state('after-automatic-retry')
    for key in protected:
        assert retry.get(key) == edited.get(key), key
    assert not retry['recovery'].get('error') and retry['document']['dirty']
    assert retry['command_revision'] == edited['command_revision']+1
    assert retry['viewport'] != edited['viewport']
    assert {k for k, v in edited.items() if retry.get(k) != v} == {'viewport', 'recovery', 'command_revision', 'revision'}
    for name in ['automatic-retry-visible', 'automatic-retry-detail']:
        ax = read('native/'+name+'.ax.txt.gz').decode()
        assert 'Autosave needs attention' not in ax and 'ID: node:cmVjb3ZlcnktZXJyb3I' not in ax
        assert '17:28:48' in ax and 'Last snapshot' in ax
    for mode in ['break', 'repair']:
        result = state('storage-'+mode+'-result')
        assert result['passed'] and result['last_good_preserved'] and result['manual_source_preserved']
        assert result['forwarder_exit_code'] == 0
    assert state('storage-repair-result')['full_recording_preserved']

    manual, final = load('native/Manual-RC.elab'), load('native/Final-Recovery-174.elab')
    good, recovered = load('native/last-good.elab-recovery'), load('native/retry.elab-recovery')
    assert good['document']['recordings'] == recovered['document']['recordings'] == manual['recordings'] == final['recordings']
    assert good['document']['graph'] == {'components': original['components'], 'wires': original['wires']}
    expected = json.loads(json.dumps(manual))
    expected['graph'] = {'components': edited['components'], 'wires': edited['wires']}
    assert recovered['document'] == expected == final
    store = state('store-before-close')
    assert len(store) == 2
    for name, digest_key in [('last-good.elab-recovery', 'good_capsule_sha256'), ('retry.elab-recovery', 'retry_sha256')]:
        digest = hashlib.sha256(read('native/'+name)).hexdigest()
        expected_digest = report[digest_key] if digest_key in report else state('storage-repair-result')[digest_key]
        assert digest == expected_digest and digest in [v['sha256'] for v in store.values()]
    for name in ['before-native-edit', 'after-native-edit']:
        native, reference = state(name), state(name, 'reference')
        for key in ['components', 'wires', 'transient', 'scope']:
            same(native[key], reference[key], name+'.'+key)
    assert read('native/Agent-Final-Replay.elab') == read('native/Final-Recovery-174.elab')
    fresh = state('report', 'reference')
    assert fresh['passed'] and fresh['agent_sha256'] == receipt['agent_sha256']
    assert len(fresh['process_exits']) == 3 and all(p['exit_code'] == 0 for p in fresh['process_exits'])

    capture = manual['recordings']['entries'][0]['data']
    assert capture['metadata'] == state('recording-baseline')['metadata']
    assert capture['metadata']['duration_s'] == .2 and capture['metadata']['sample_count'] == len(capture['rows']) == 209
    assert [p['id'] for p in capture['metadata']['probes']] == ['v:C1', 'i:R1']
    for name, resistance in [('manual', 10000), ('final', 30000)]:
        restored = state('reopen-'+name, 'reference')
        assert resistor(restored) == resistance and restored['paused'] and not restored['document']['dirty']
        assert restored['transient']['sample']['time_s'] == 0 and restored['document']['recording_count'] == 1
        assert state('reopen-'+name+'-operation', 'reference')['metadata'] == capture['metadata']
        pages = state('reopen-'+name+'-pages', 'reference')
        assert [len(p['rows']) for p in pages] == [200, 9]
        assert [row for page in pages for row in page['rows']] == capture['rows']
    edge_counts, max_v, max_i = {}, 0., 0.
    for row in capture['rows']:
        at = row['time_s']
        assert 0 <= at <= .2 and len(row['values']) == 2
        edge = round(at/.05)
        if edge > 0 and math.isclose(at, edge*.05, rel_tol=0, abs_tol=1e-12):
            seen = edge_counts.get(edge, 0)
            source = (5. if edge % 2 else 0.) if seen == 0 else (0. if edge % 2 else 5.)
            edge_counts[edge] = seen+1
        else:
            source = 5. if int(math.floor((at+1e-14)/.05)) % 2 == 0 else 0.
        voltage = recorded_voltage(at)
        current = (source-voltage)/10000
        max_v = max(max_v, abs(row['values'][0]-voltage))
        max_i = max(max_i, abs(row['values'][1]-current))
        assert math.isclose(row['values'][0], voltage, rel_tol=1e-9, abs_tol=1e-12)
        assert math.isclose(row['values'][1], current, rel_tol=1e-9, abs_tol=1e-12)
    assert edge_counts == {1: 3, 2: 3, 3: 3, 4: 3}
    live_error = max(abs(r['voltage_v']-5*-math.expm1(-r['time_s']/.02)) for r in original['transient']['samples'])
    assert live_error < 1e-12
    return {'passed': True, 'evidence_files': len(manifest), 'native_warning_observed_and_cleared': True,
            'ordinary_native_edit_accepted_with_error_retained': True, 'native_agent_physical_stages': 2,
            'fresh_reopens': 2, 'recorded_rows_exact_per_reopen': 209,
            'manual_source_and_previous_capsule_preserved': True, 'native_agent_saved_bytes_identical': True,
            'independent_recording_max_voltage_error_v': max_v, 'independent_recording_max_current_error_a': max_i,
            'independent_live_max_voltage_error_v': live_error,
            'scope': 'One owned-directory fault/native edit/automatic retry; no timing, power-loss, reader or all-filesystem acceptance claim'}


if __name__ == '__main__':
    print(json.dumps(verify(), indent=2))
