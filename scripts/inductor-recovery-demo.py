#!/usr/bin/env python3
"""Crash an owned coil session, restore its capture, and refuse corrupt imports.

The producer is always a new headless process. --connect selects a fresh owned
native restore target, which is closed afterward. Existing artifacts refuse.
"""
import argparse
import copy
import csv
import hashlib
import importlib.util
import io
import json
import math
from pathlib import Path
import time


spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)

ENGINE = 'elab.rlc-mna-radau5.v1'


def physics(state):
    return {key: state.get(key) for key in ('run_id', 'electrical_revision', 'mixed', 'transient')}


def import_invariants(state):
    return {key: state.get(key) for key in (
        'components', 'wires', 'document', 'undo_name', 'redo_name',
        'paused', 'run_id', 'electrical_revision', 'mixed', 'transient',
        'recording', 'last_operation', 'operations')}


def wait_for(client, predicate, timeout=20):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        state = client.ask('project.inspect')
        if predicate(state):
            return state
        time.sleep(.02)
    raise AssertionError('public state did not reach the expected condition')


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native restore target; closed afterward')
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert not args.output.exists(), 'report already exists'
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    target = connect_demo_client(helper.Client, args.connect)
    producer = helper.Client(None)
    killed = False
    try:
        producer.ask('example.load', {'id': 'rl'}, True)
        producer.ask('simulation.pause', {}, True)
        producer.ask('inductor.initial.set', {'id': 'L1', 'value': '-20 mA'}, True)
        producer.ask('simulation.restart', {}, True)
        initial = producer.ask('simulation.wait')
        assert initial['mixed']['inductors'][0]['current_a'] == -.02
        producer.ask('scope.probes.set', {'ids': ['i:L1', 'v:L1']}, True)
        source = assets / 'original.elab'
        producer.ask('project.save', {'path': str(source)}, True)
        original = source.read_bytes()
        producer.ask('simulation.step', {'duration_s': '3 ms'}, True)
        live = producer.ask('project.inspect')
        assert live['transient']['sample']['time_s'] == .003
        assert abs(live['mixed']['inductors'][0]['current_a'] - (.05-.07*math.exp(-.3))) < 1e-5
        directory = str(assets / 'recovery')
        producer.ask('recovery.configure', {'directory': directory, 'enabled': True, 'interval_s': 1}, True)
        accepted = producer.ask('simulation.verify', {
            'duration_s': '5 ms', 'sample_step_s': '1 ms',
            'inductor_initials': [{'id': 'L1', 'current_a': .03}],
            'probe_ids': ['i:L1', 'v:L1']}, True)
        operation = accepted['last_operation']
        wait_for(producer, lambda state: any(
            item['id'] == operation and item['status'] == 'completed' for item in state['operations']))
        for fmt in ('json', 'csv'):
            producer.ask('verification.export', {'id': operation, 'format': fmt,
                'path': str(assets / f'capture.{fmt}')}, True)
        captured = json.loads((assets / 'capture.json').read_bytes())
        metadata, rows = captured['metadata'], captured['rows']
        assert metadata['engine'] == ENGINE and len(rows) == 6
        # Captured scenarios materialize overrides in their immutable graph;
        # transient settings retain their established portable shape.
        assert next(p for p in metadata['graph']['components'] if p['id'] == 'L1')['initial_current_a'] == .03
        errors = []
        for row in rows:
            expected = .05-.02*math.exp(-row['time_s']/.01)
            errors.append((abs(row['values'][0]-expected), abs(row['values'][1]-(5-100*expected))))
        assert max(v[0] for v in errors) < 1e-5 and max(v[1] for v in errors) < .001
        csv_rows = list(csv.DictReader(io.StringIO((assets / 'capture.csv').read_text())))
        assert len(csv_rows) == 6 and 'i:L1 [A]' in csv_rows[0] and 'v:L1 [V]' in csv_rows[0]
        assert physics(producer.ask('project.inspect')) == physics(live)
        saved = wait_for(producer, lambda state: any(
            item['recording_count'] == 1 for item in state['recovery']['candidates']))
        assert saved['document']['recordings_dirty'] and not saved['document']['schematic_dirty']
        candidate = next(item for item in saved['recovery']['candidates'] if item['recording_count'] == 1)
        candidate_sha = candidate['sha256']
        producer.process.kill()
        assert producer.process.wait(timeout=5) == -9
        killed = True
        producer.process.stdin.close()
        producer.process.stdout.close()

        assert not target.ask('project.inspect')['components'], 'restore target must be fresh and blank'
        target.ask('recovery.configure', {'directory': directory, 'enabled': False, 'interval_s': 1}, True)
        scanned = wait_for(target, lambda state: any(
            item['sha256'] == candidate_sha for item in state['recovery']['candidates']))
        candidate = next(item for item in scanned['recovery']['candidates'] if item['sha256'] == candidate_sha)
        assert not candidate['active_session'] and not candidate['current_session'] and candidate['recording_count'] == 1
        choice = {'id': candidate['id'], 'sha256': candidate_sha}
        inspection = target.ask('recovery.inspect', choice)
        assert inspection['document']['recordings'] == [metadata]
        page = target.ask('recovery.samples', dict(choice, recording_index=0, limit=100))
        assert page['total'] == len(rows) and page['rows'] == rows
        target.ask('recovery.restore', choice, True)
        restored = target.ask('simulation.wait')
        assert restored['paused'] and restored['transient']['sample']['time_s'] == 0
        assert restored['mixed']['inductors'][0]['current_a'] == -.02
        assert next(p for p in restored['components'] if p['id'] == 'L1')['initial_current_a'] == -.02
        assert restored['document']['dirty'] and not restored['document']['path']
        assert restored['document']['recording_count'] == 1
        assert target.ask('operation.inspect', {'id': restored['last_operation']})['metadata'] == metadata
        target.ask('project.save', {'path': str(assets / 'recovered.elab')}, True)
        before = target.ask('project.inspect')
        imported = target.ask('verification.import', {'path': str(assets / 'capture.json')}, True)
        assert physics(imported) == physics(before)
        assert imported['recording']['imported'] and imported['recording']['viewing']
        imported_id = imported['last_operation']
        view = target.ask('operation.inspect', {'id': imported_id})
        assert view['metadata'] == metadata and not view['operation']['source_unchanged']
        assert imported_id != metadata['operation_id']
        target.ask('verification.export', {'id': imported_id, 'format': 'json',
            'path': str(assets / 'reexport.json')}, True)
        assert (assets / 'reexport.json').read_bytes() == (assets / 'capture.json').read_bytes()

        corruptions = {}
        def variant(name):
            value = copy.deepcopy(captured)
            corruptions[name] = value
            return value['metadata']

        variant('missing-coil-result')['mixed'].pop('inductors')
        m = variant('duplicate-coil-result')
        m['mixed']['inductors'].append(copy.deepcopy(m['mixed']['inductors'][0]))
        variant('foreign-coil-result')['mixed']['inductors'][0]['id'] = 'FOREIGN'
        variant('wrong-energy')['mixed']['inductors'][0]['energy_j'] = .5
        variant('overflow-current')['mixed']['inductors'][0]['current_a'] = 1e308
        m = variant('wrong-coil-model')
        next(p for p in m['graph']['components'] if p['id'] == 'L1')['model'] = 'foreign-inductor.v1'
        m = variant('startup-on-resistor')
        next(p for p in m['graph']['components'] if p['id'] == 'R1')['initial_current_a'] = .01
        variant('missing-final-row')
        corruptions['missing-final-row']['rows'].pop()
        corruptions['missing-final-row']['metadata']['sample_count'] -= 1
        refused = []
        for name, value in corruptions.items():
            path = assets / f'bad-{name}.json'
            path.write_text(json.dumps(value, indent=2)+'\n')
            before = import_invariants(target.ask('project.inspect'))
            target.ask('verification.import', {'path': str(path)}, True, 'invalid-recording')
            assert import_invariants(target.ask('project.inspect')) == before, name
            refused.append(name)
        link = assets / 'linked-capture.json'
        link.symlink_to(assets / 'capture.json')
        before = import_invariants(target.ask('project.inspect'))
        target.ask('verification.import', {'path': str(link)}, True, 'file-error')
        assert import_invariants(target.ask('project.inspect')) == before
        assert source.read_bytes() == original
        report = dict(passed=True, transport='desktop-socket' if args.connect else 'headless-stdio',
            producer='fresh-owned-headless', crash_exit_code=producer.process.returncode,
            candidate_sha256=candidate_sha, original_sha256=hashlib.sha256(original).hexdigest(),
            live_before_crash=live['mixed']['inductors'][0], live_after_restore=restored['mixed']['inductors'][0],
            captured_initial_current_a=rows[0]['values'][0], captured_samples=len(rows),
            max_reference_error_a=max(v[0] for v in errors), max_reference_error_v=max(v[1] for v in errors),
            refused_corruptions=refused, refused_symlink=True,
            checks=['record-only dirty autosave', 'actual producer SIGKILL and inactive lease',
                'exact recovery metadata and sample rows', 'paused t=0 restore of saved -20 mA intent',
                'retained independent +30 mA scenario capture', 'fresh local import identity',
                'byte-identical portable re-export', 'atomic corrupt import refusals', 'original saved file unchanged'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as out:
            out.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        if not killed:
            producer.close()
        if target:
            if args.connect:
                target.ask('app.close', {'discard_changes': True}, True)
            target.close()


if __name__ == '__main__':
    main()
