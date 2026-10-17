#!/usr/bin/env python3
"""Owned atomic RC edits, no-op selection repair, history, retries and saved data."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import time

from demo_session import connect_demo_client
from recovery_timing import settle_native_view

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def edit(command, **params):
    return dict(command=command, params=params)


def physics(state):
    return {k: state.get(k) for k in ['transient', 'mixed', 'npn', 'result', 'run_id', 'electrical_revision', 'paused']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, mode=0o700, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    fresh = None
    refusals = []

    def inspect():
        return client.ask('project.inspect')

    def refuse(command, params, code):
        before = inspect()
        client.ask(command, params, True, code)
        assert inspect() == before
        refusals.append(dict(command=command, code=code))

    def capture(name):
        admitted = client.ask('simulation.verify', {'duration_s': .003, 'probe_ids': ['v:C1', 'i:C1']}, True)
        identity = admitted['last_operation']
        deadline = time.monotonic() + 10
        while True:
            result = client.ask('operation.inspect', {'id': identity})
            if result['operation']['status'] == 'completed':
                break
            assert result['operation']['status'] not in ('failed', 'canceled') and time.monotonic() < deadline
            time.sleep(.005)
        client.ask('verification.export', {'id': identity, 'format': 'json', 'path': str(assets / name)}, True)
        return identity

    try:
        initial = inspect()
        if initial['recovery'].get('directory'):
            client.ask('recovery.configure', {'directory': initial['recovery']['directory'], 'enabled': False, 'interval_s': 60}, True)
        client.ask('example.rc', {}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': .003}, True)
        if args.connect:
            settle_native_view(client)
            client.ask('view.pan', {'dx': 0, 'dy': 0}, True)
        original_id = capture('original-record.json')
        original = assets / 'original.elab'
        client.ask('project.save', {'path': str(original)}, True)
        original_hash = sha(original)
        before_discovery = inspect()
        contracts = {c: client.ask('capabilities.list', {'command': c})['capabilities'][0]['input_schema'] for c in ['edit.batch', 'edit.undo', 'edit.redo']}
        assert all(s['x-command-contract']['scope'] == 'atomic-edits-and-history' for s in contracts.values())
        assert len(contracts['edit.batch']['x-supported-edit-commands']) == 17 and before_discovery == inspect()
        scalar = edit('component.set', id='R1', value='10k')
        for params in [dict(name=' ', edits=[scalar]), dict(name='µ' * 40 + 'x', edits=[scalar]),
                       dict(name='Empty', edits=[]), dict(name='Null', edits=None), dict(name='Too many', edits=[scalar] * 101),
                       dict(name='Nested', edits=[edit('edit.batch')]), dict(name='File', edits=[edit('project.save', path=str(assets / 'forbidden.elab'))]),
                       dict(name='Null child', edits=[dict(command='component.set', params=None)]),
                       dict(name='Unknown field', edits=[edit('component.set', id='R1', value='20k', extra=True)])]:
            refuse('edit.batch', params, 'invalid-params')
        assert not (assets / 'forbidden.elab').exists()
        client.ask('component.set', {'id': 'R1', 'value': '15k'}, True)
        client.ask('edit.undo', {}, True)
        baseline = inspect()
        temporary = edit('component.place', id='Temporary', kind='resistor', position={'x': 500, 'y': 500}, value='1k')
        refuse('edit.batch', dict(name='Rejected temporary', edits=[temporary, edit('component.set', id='missing', value='1k')]), 'invalid-params')
        no_op = client.ask('edit.batch', dict(name='Temporary construction', edits=[temporary, edit('object.delete', id='Temporary')]), True)
        assert no_op['selected'] == '' and no_op['drawing']['value'] == ''
        assert no_op['components'] == baseline['components'] and no_op['wires'] == baseline['wires']
        assert physics(no_op) == physics(baseline) and no_op['undo_entry'] == baseline['undo_entry'] and no_op['redo_entry'] == baseline['redo_entry']
        assert no_op['document'] == baseline['document'] and no_op['command_revision'] == baseline['command_revision'] + 1
        refuse('component.place', dict(id='Temporary', kind='resistor', position={'x': 500, 'y': 500}), 'invalid-params')
        refuse('edit.redo', {'expected_entry_revision': 0}, 'history-conflict')
        refuse('edit.redo', {'expected_entry_revision': 'x'}, 'invalid-params')
        client.ask('edit.redo', {'expected_entry_revision': None, 'allow_other_actor': None}, True)
        client.ask('edit.undo', {}, True)
        ready = inspect()
        pair = dict(name=' Tune RC pair ', edits=[edit('component.set', id='R1', value='20k'), edit('component.set', id='C1', value='2uF')])
        request = dict(version=1, id='tune-first', command='edit.batch', params=pair, expected_command_revision=ready['command_revision'], key='owned-pair-tuning')

        def raw(request):
            client.process.stdin.write(json.dumps(request) + '\n')
            client.process.stdin.flush()
            line = client.process.stdout.readline()
            assert line
            return json.loads(line)

        response = raw(request)
        assert 'error' not in response
        tuned = inspect()
        assert tuned['undo_entry']['name'] == pair['name'] and tuned['command_revision'] == ready['command_revision'] + 1
        assert tuned['electrical_revision'] == ready['electrical_revision'] + 1 and tuned['run_id'] == ready['run_id'] + 1
        assert tuned['transient']['sample'] == ready['transient']['sample'] and tuned['transient']['stale']
        edited = assets / 'edited.elab'
        client.ask('project.save', {'path': str(edited)}, True)
        edited_hash = sha(edited)
        tip = inspect()['undo_entry']
        undone = client.ask('edit.undo', {'expected_entry_revision': tip['command_revision']}, True)
        assert undone['document']['path'] == str(edited) and undone['document']['dirty'] and undone['document']['recording_count'] == 1
        before_retry = inspect()
        duplicate = raw(dict(request, id='tune-retry'))
        assert duplicate['duplicate'] and duplicate['result']['components'] == tuned['components'] and inspect() == before_retry
        redone = client.ask('edit.redo', {'expected_entry_revision': tip['command_revision']}, True)
        assert not redone['document']['dirty'] and redone['undo_entry'] == tip
        geometry_before = inspect()
        geometry = client.ask('edit.batch', {'name': 'µ' * 40, 'edits': [edit('component.move', id='R1', position={'x': 0, 'y': -90}), edit('component.rotate', id='R1')]}, True)
        assert geometry['undo_entry']['name'] == 'µ' * 40 and physics(geometry) == physics(geometry_before)
        client.ask('edit.undo', {'expected_entry_revision': geometry['undo_entry']['command_revision']}, True)
        assert not inspect()['document']['dirty'] and physics(inspect()) == physics(geometry_before)
        client.ask('verification.export', {'id': original_id, 'format': 'json', 'path': str(assets / 'preserved-record.json')}, True)
        assert (assets / 'preserved-record.json').read_bytes() == (assets / 'original-record.json').read_bytes()
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        live = client.ask('simulation.step', {'duration_s': .003}, True)
        tuned_id = capture('tuned-record.json')
        final_path = assets / 'final.elab'
        client.ask('project.save', {'path': str(final_path)}, True)
        final = inspect()
        assert final['document']['recording_count'] == 2 and not final['document']['dirty']
        assert sha(original) == original_hash and sha(edited) == edited_hash
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': str(final_path)}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['paused'] and not reopened['document']['dirty'] and reopened['document']['recording_count'] == 2 and reopened['transient']['sample']['time_s'] == 0
        entries = {entry['id']: fresh.ask('operation.inspect', {'id': entry['id']})['metadata'] for entry in reopened['operations']}
        for name in ['original', 'tuned']:
            expected = json.loads((assets / (name + '-record.json')).read_bytes())['metadata']
            matches = [identity for identity, metadata in entries.items() if metadata == expected]
            assert len(matches) == 1
            fresh.ask('verification.export', {'id': matches[0], 'format': 'json', 'path': str(assets / ('reopened-' + name + '.json'))}, True)
            assert (assets / ('reopened-' + name + '.json')).read_bytes() == (assets / (name + '-record.json')).read_bytes()
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', contracts=contracts, refusals=refusals,
                      baseline=baseline, no_op=no_op, ready=ready, tuned=tuned, undone=undone, redone=redone, geometry=geometry,
                      live=live, final=final, original_sha256=original_hash, edited_sha256=edited_hash,
                      original_id=original_id, tuned_id=tuned_id, duplicate_returns_original_without_reapply=True,
                      reopened=dict(paused=reopened['paused'], dirty=reopened['document']['dirty'], recording_count=2, time_s=reopened['transient']['sample']['time_s']),
                      limits=['Cross-actor acknowledgment and original attribution use controlled core/native tests; each public transport here has one actor identity',
                              'No new model/numerical/work-limit/file-format or complete physical input/AX/storage/release qualification'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as stream:
            json.dump(report, stream, indent=2)
            stream.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], contracts=len(contracts), refusals=len(refusals))))
    finally:
        try:
            if fresh:
                fresh.close()
            if args.connect:
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()
