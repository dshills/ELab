#!/usr/bin/env python3
"""Verify preserved evidence offline; never launch or drive an application."""
from datetime import datetime
import gzip
import hashlib
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


def same(a, b, path='result'):
    if isinstance(a, dict):
        assert isinstance(b, dict), path
        ignored = {'run_id', 'electrical_revision'}
        a = {k: v for k, v in a.items() if k not in ignored}
        b = {k: v for k, v in b.items() if k not in ignored}
        assert a.keys() == b.keys(), path
        for key in a:
            same(a[key], b[key], path+'.'+key)
    elif isinstance(a, list):
        assert isinstance(b, list) and len(a) == len(b), path
        for i, (left, right) in enumerate(zip(a, b)):
            same(left, right, f'{path}[{i}]')
    elif isinstance(a, (int, float)) and not isinstance(a, bool):
        assert isinstance(b, (int, float)) and not isinstance(b, bool), path
        assert math.isclose(a, b, rel_tol=1e-9, abs_tol=1e-12), (path, a, b)
    else:
        assert a == b, (path, a, b)


def wiper(s):
    return next(p['wiper_position'] for p in s['components'] if p['id'] == 'Timing')


def capacitor(s):
    return s['mixed']['capacitors'][0]['voltage_v']


def continuous(a, b):
    for key in ['run_id', 'electrical_revision']:
        assert a[key] == b[key]
    assert a['mixed']['time_s'] == b['mixed']['time_s']
    assert capacitor(a) == capacitor(b)


def exact_voltage(time_s, wiper_events):
    """Independent piecewise Thevenin solution, not the production integrator.

    10 kohm linear track, two 1 ohm endpoint resistances, 1 uF ideal C;
    0/5 V ideal pulse switches every 50 ms, initially high. Charge is continuous.
    """
    changes = [(i*.05, 'pulse', 0. if i % 2 else 5.) for i in range(1, 15)]
    changes += [(t, 'wiper', w) for t, w in wiper_events]
    changes += [(time_s, 'end', 0.)]
    position, source, voltage, previous = .5, 5., 0., 0.
    for end, kind, value in sorted(changes, key=lambda c: c[0]):
        if end > time_s+1e-14:
            break
        end = min(time_s, end)
        upper = 1+9998*(1-position)
        lower = 1+9998*position
        tau = upper*lower/10000*1e-6
        target = source*lower/10000
        voltage = target+(voltage-target)*math.exp(-(end-previous)/tau)
        previous = end
        if kind == 'pulse':
            source = value
        elif kind == 'wiper':
            position = value
        else:
            return voltage
    raise AssertionError('Missing reference endpoint')


def verify():
    assert sys.flags.optimize == 0, 'Run without Python optimization'
    manifest = load('manifest.json')['files']
    for name, entry in manifest.items():
        relative = Path(name)
        assert not relative.is_absolute() and '..' not in relative.parts
        p = HERE/relative
        assert p.is_file() and not p.is_symlink(), name
        data = p.read_bytes()
        assert len(data) == entry['bytes'], name
        assert hashlib.sha256(data).hexdigest() == entry['sha256'], name
        original = gzip.decompress(data) if entry['gzip'] else data
        assert len(original) == entry['original_bytes'], name
        assert hashlib.sha256(original).hexdigest() == entry['original_sha256'], name

    report = state('report')
    receipt = load('release-receipt.json.gz')
    for key in ['native_sha256', 'agent_sha256']:
        assert report[key] == receipt[key]
    assert report['same_pid_after_exec'] and report['open_exit_code'] == 0
    assert report['owned_process_absent'] and report['endpoint_removed']
    assert report['forced_cleanup'] is False and report['native_exit_status'] is None
    assert 'error' not in report and 'cleanup_error' not in report
    for key in ['shared_geometry_unchanged', 'review_unchanged', 'copied_bundle_unchanged',
                'canonical_unchanged', 'baseline_file_unchanged']:
        assert report[key], key
    assert report['shared_geometry_before'] == report['shared_geometry_after']
    assert all(c['denied'] and c['errno'] in [1, 13] for c in report['controls'].values())
    assert hashlib.sha256(read('native/Baseline-Pot-174.elab')).hexdigest() == report['baseline_sha256']
    closed = state('close-response')
    assert closed['closed'] and not closed['document']['dirty']
    assert all(closed['shutdown'][k] for k in ['workers_joined', 'exit_ready', 'recovery_finalized'])
    assert closed['shutdown']['status'] == 'completed'

    before, after, undo = [state(n) for n in ['before-paused-drag', 'after-paused-drag', 'after-paused-undo']]
    assert wiper(before) == .5 and wiper(after) == .7510121154785157
    continuous(before, after)
    assert after['command_revision'] == before['command_revision']+1
    assert after['undo_entry'] == {'name': 'Adjust wiper', 'actor': 'user', 'command_revision': 8}
    assert after['input_work']['status'] == 'completed' and after['input_work']['actor'] == 'user'
    assert undo['command_revision'] == after['command_revision']+1
    assert undo['components'] == before['components'] and undo['wires'] == before['wires']
    assert undo['redo_entry'] == after['undo_entry']
    assert undo['mixed']['stale'] and undo['transient']['stale']
    assert 'stale' in read('native/after-paused-undo.ax.txt.gz').decode().lower()

    running_before, running = [state(n) for n in ['before-running-drag', 'after-running-pause']]
    assert not running_before['paused'] and running['paused']
    assert running['command_revision'] == running_before['command_revision']+2
    assert running['undo_entry'] == {'name': 'Adjust wiper', 'actor': 'user', 'command_revision': 14}
    for key in ['run_id', 'electrical_revision']:
        assert running[key] == running_before[key]
    assert not running['mixed']['stale'] and not running['transient']['stale']
    assert wiper(running) == wiper(after)
    event = [e for e in running['transient']['events'] if e['mode'].startswith('wiper:')]
    assert len(event) == 1 and event[0]['time_s'] == .68
    rows = [r for r in running['transient']['samples'] if r['time_s'] == .68]
    assert len(rows) == 3 and all(r['voltage_v'] == rows[0]['voltage_v'] for r in rows)
    assert rows[-1]['current_a'] != rows[0]['current_a']
    assert math.isclose(running['mixed']['time_s'], .691, abs_tol=1e-12)

    hold_before, held = state('before-agent-hold-rendered'), state('after-agent-hold')
    controller, agent = state('agent-hold-controller'), state('agent-interruption-result')
    assert controller['completed'] and controller['requested_duration_ms'] == 2000
    assert agent['status'] == 'agent_edit_completed' and 'error' not in agent['response']
    utc = lambda s: datetime.fromisoformat(s.replace('Z', '+00:00'))
    assert utc(controller['invocation_utc']) < utc(agent['invoked_utc'])
    assert utc(agent['invoked_utc']) <= utc(agent['response_utc']) < utc(controller['return_utc'])
    request = agent['request']
    assert request['expected_command_revision'] == hold_before['command_revision'] == 18
    assert request['expected_document_generation'] == hold_before['document']['generation']
    assert request['params'] == {'id': 'Timing', 'position': .8}
    assert held['command_revision'] == 19 and wiper(held) == .8
    assert held['undo_entry'] == {'name': 'Adjust wiper', 'actor': 'agent.socket', 'command_revision': 19}
    continuous(hold_before, held)
    held_ax = read('native/after-agent-hold.ax.txt.gz').decode()
    assert 'Wiper 80%' in held_ax and 'by agent.socket' in held_ax

    cases = ['before-paused-drag', 'after-paused-drag', 'after-paused-undo',
             'after-running-pause', 'after-agent-hold']
    for name in cases:
        native, reference = state(name), state(name, 'reference')
        for key in ['components', 'wires', 'mixed', 'transient', 'scope']:
            same(native[key], reference[key], name+'.'+key)
    reference_report = state('report', 'reference')
    assert reference_report['passed'] and reference_report['process_exit_code'] == 0
    assert reference_report['agent_sha256'] == receipt['agent_sha256']

    reference_errors = {}
    for name in cases+['before-running-drag']:
        s = state(name)
        events = [(e['time_s'], float(e['mode'].split(':')[2])) for e in s['transient']['events']
                  if e['mode'].startswith('wiper:')]
        errors = [abs(row['voltage_v']-exact_voltage(row['time_s'], events))
                  for row in s['transient']['samples']]
        assert errors and max(errors) < .002, name
        reference_errors[name] = {'samples': len(errors), 'maximum_absolute_error_v': max(errors)}
    final_project = load('native/Final-Pot-174.elab')
    assert final_project['graph']['components'] == held['components']
    assert final_project['graph']['wires'] == held['wires']
    return {'passed': True, 'evidence_files': len(manifest), 'native_agent_stages': len(cases),
            'comparison': 'Full graph/mixed/transient/scope; independent run/electrical IDs excluded; rel 1e-9 abs 1e-12',
            'rc_reference': reference_errors, 'rc_tolerance_v': .002,
            'scope': 'One paused drag/Undo, one running drag/Pause, one held-click agent-edit call; no OS timing/blur claim'}


if __name__ == '__main__':
    print(json.dumps(verify(), indent=2))
