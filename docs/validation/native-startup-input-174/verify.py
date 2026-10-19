#!/usr/bin/env python3
"""Replay archived startup qualification evidence without application access."""
from datetime import datetime
import gzip
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import sys

HERE = Path(__file__).resolve().parent
CASES = ['rc-restart', 'rc-default', 'rl-restart', 'rl-default', 'rl-agent-held-reset']


def read(name):
    data = (HERE/name).read_bytes()
    return gzip.decompress(data) if name.endswith('.gz') else data


def load(name):
    return json.loads(read(name))


def state(name, prefix='native'):
    return load(prefix+'/'+name+'.json.gz')


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
    # Reuse the exact archived comparison policy from the preceding proof.
    spec = importlib.util.spec_from_file_location('startup_comparison', HERE/'producers/comparison-helper.py')
    comparator = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(comparator)
    same = comparator.same
    report, receipt = state('report'), load('release-receipt.json.gz')
    for key in ['native_sha256', 'agent_sha256']:
        assert report[key] == receipt[key]
    assert report['same_pid_after_exec'] and report['open_exit_code'] == 0
    assert report['owned_process_absent'] and report['endpoint_removed']
    assert not report['forced_cleanup'] and report['native_exit_status'] is None
    assert not any(k in report for k in ['error', 'cleanup_error'])
    for key in ['shared_geometry_unchanged', 'review_unchanged', 'copied_bundle_unchanged',
                'canonical_unchanged', 'baseline_file_unchanged']:
        assert report[key], key
    assert report['shared_geometry_before'] == report['shared_geometry_after']
    assert all(x['denied'] and x['errno'] in [1, 13] for x in report['controls'].values())
    for name, pin in report['baseline_files'].items():
        assert hashlib.sha256(read('native/'+name)).hexdigest() == pin
    closed = state('close-response')
    assert closed['closed'] and not closed['document']['dirty']
    assert closed['shutdown']['status'] == 'completed'
    assert all(closed['shutdown'][k] for k in ['workers_joined', 'exit_ready', 'recovery_finalized'])
    reference_report = state('report', 'reference')
    assert reference_report['passed'] and reference_report['cases'] == CASES
    assert reference_report['process_exit_code'] == 0 and reference_report['agent_sha256'] == receipt['agent_sha256']

    checks, reference_errors = [], {}
    for case in CASES:
        before, draft, after = [state(stage+'-'+case) for stage in ['before', 'draft', 'after']]
        # Private typing does not mutate even observation-level snapshot state.
        assert before == draft, case+' private typing changed accepted state'
        startup = 'initial_voltage_v' if case.startswith('rc-') else 'initial_current_a'
        part_id = 'C1' if case.startswith('rc-') else 'L1'
        part = next(p for p in after['components'] if p['id'] == part_id)
        assert after['command_revision']-before['command_revision'] == (1 if case.endswith('held-reset') else 2)
        assert after['undo_entry']['name'] == ('Change capacitor startup' if part_id == 'C1' else 'Change inductor startup')
        assert after['undo_entry']['actor'] == ('agent.socket' if case.endswith('held-reset') else 'user')
        assert after['paused']
        if case.endswith('restart'):
            assert part[startup] == (3 if part_id == 'C1' else .03)
            assert after['transient']['sample']['time_s'] == 0 and not after['transient']['stale']
        elif case.endswith('default'):
            assert startup not in part
            assert after['transient']['stale']
            assert after['transient']['sample'] == before['transient']['sample']
            assert 'stale' in read('native/after-'+case+'.ax.txt.gz').decode().lower()
        else:
            assert part[startup] == .04 and after['transient']['stale']
            assert after['transient']['sample'] == before['transient']['sample']
        for stage in ['before', 'after']:
            name = stage+'-'+case
            native, reference = state(name), state(name, 'reference')
            for key in ['components', 'wires', 'transient', 'scope', 'mixed']:
                assert (key in native) == (key in reference), (name, key)
                if key in native:
                    same(native[key], reference[key], name+'.'+key)
            # Stale samples retain their original accepted startup, not the new
            # uninitialized graph setting. Both fixtures stay in the high pulse.
            initial = (3 if part_id == 'C1' else .03) if stage == 'after' and case.endswith('restart') else (2 if part_id == 'C1' else -.02)
            errors = {k: 0. for k in ['voltage_v', 'current_a', 'energy_j']}
            rows = native['transient']['samples']
            assert rows
            for row in rows:
                t = row['time_s']
                assert 0 <= t <= .005 and row['input_v'] == 5
                if part_id == 'C1':
                    voltage = 5+(initial-5)*math.exp(-t/.01)
                    current, energy = (5-voltage)/10000, .5e-6*voltage*voltage
                else:
                    current = .05+(initial-.05)*math.exp(-t/.01)
                    voltage, energy = 5-100*current, .5*current*current
                for key, expected in [('voltage_v', voltage), ('current_a', current), ('energy_j', energy)]:
                    assert math.isclose(row[key], expected, rel_tol=1e-9, abs_tol=1e-12), (name, key, row[key], expected)
                    errors[key] = max(errors[key], abs(row[key]-expected))
            reference_errors[name] = {'samples': len(rows), 'maximum_absolute_errors': errors}
        checks.append(case)

    controller, agent = state('agent-hold-controller'), state('agent-interruption-result')
    assert controller['completed'] and controller['requested_duration_ms'] == 2000
    assert agent['status'] == 'agent_edit_completed' and 'error' not in agent['response']
    utc = lambda s: datetime.fromisoformat(s.replace('Z', '+00:00'))
    assert utc(controller['invocation_utc']) < utc(agent['invoked_utc'])
    assert utc(agent['invoked_utc']) <= utc(agent['response_utc']) < utc(controller['return_utc'])
    request, final = agent['request'], state('after-rl-agent-held-reset')
    assert request['expected_command_revision'] == 38 and final['command_revision'] == 39
    assert request['expected_document_generation'] == final['document']['generation']
    assert request['params'] == {'id': 'L1', 'value': '40 mA'}
    assert 'Value: 40 mA' in read('native/after-rl-agent-held-reset.ax.txt.gz').decode()
    project = load('native/Final-Startup-174.elab')
    assert project['graph']['components'] == final['components'] and project['graph']['wires'] == final['wires']
    hygiene = load('repo-hygiene/rename-receipt.json.gz')
    preceding = HERE.parent/'native-slider-input-174'
    assert not (preceding/hygiene['old_path']).exists()
    data = (preceding/hygiene['new_path']).read_bytes()
    assert hashlib.sha256(data).hexdigest() == hygiene['original_sha256'] == hygiene['renamed_sha256']
    original = load('repo-hygiene/original-slider-manifest.json.gz')
    current = json.loads((preceding/'manifest.json').read_text())
    expected = json.loads(json.dumps(original))
    expected['files'][hygiene['new_path']] = expected['files'].pop(hygiene['old_path'])
    assert current == expected
    quality_checks = load('repo-hygiene/checks.json.gz')
    assert all(quality_checks[k] == 0 for k in ['root_test_exit_code', 'root_vet_exit_code', 'root_package_discovery_exit_code'])
    assert not read('repo-hygiene/root-package-list-stderr.txt.gz')
    assert not read('repo-hygiene/root-vet.txt.gz')
    packages = read('repo-hygiene/root-package-list.txt.gz').decode().splitlines()
    assert packages and not any('/docs/validation/' in p for p in packages)
    assert 'FAIL' not in read('repo-hygiene/root-test.txt.gz').decode()
    return {'passed': True, 'evidence_files': len(manifest), 'cases': checks,
            'native_agent_stages': 10, 'private_draft_snapshots_unchanged': 5,
            'archive_source_bytes_preserved': True, 'root_package_checks_passed': True,
            'independent_references': reference_errors,
            'comparison': 'Full graph/transient/scope and mixed where present; independent run/electrical IDs excluded; rel 1e-9 abs 1e-12',
            'scope': 'Four native one-click startup cases; one held-click call containing a guarded agent edit. No exact OS input ordering or latency claim.'}


if __name__ == '__main__':
    print(json.dumps(verify(), indent=2))
