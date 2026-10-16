#!/usr/bin/env python3
"""Qualify the revised template while preserving actual previous-release files."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import time

spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def invariant(state):
    return {key: state.get(key) for key in ('components', 'wires', 'document', 'undo_name',
                                           'redo_name', 'run_id', 'electrical_revision', 'mixed', 'transient')}


def electrical(graph):
    clone = json.loads(json.dumps(graph))
    for part in clone['components']:
        part.pop('position', None)
        part.pop('rotation', None)
    return clone


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native session; closes unless --keep-open succeeds')
    parser.add_argument('--keep-open', action='store_true')
    parser.add_argument('--legacy', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assert not args.keep_open or args.connect
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    original = args.legacy.read_bytes()
    old = json.loads(original)
    client = helper.Client(args.connect)
    mutate = lambda name, params=None: client.ask(name, params, True)
    keep = False
    try:
        assert not client.ask('project.inspect')['components'], 'requires fresh owned blank session'
        mutate('project.load', {'path': str(args.legacy.resolve())})
        reopened = client.ask('simulation.wait')
        assert reopened['components'] == old['graph']['components'] and reopened['wires'] == old['graph']['wires']
        assert reopened['paused'] == old['paused'] and reopened['document']['recording_count'] == 1 and not reopened['mixed']['stale']
        mutate('project.save', {'path': str(assets/'legacy-reopened.elab')})
        saved = json.loads((assets/'legacy-reopened.elab').read_bytes())
        assert saved['graph'] == old['graph'] and saved['recordings'] == old['recordings'] and saved['paused'] == old['paused']
        mutate('simulation.pause')
        reopened = client.ask('project.inspect')
        mutate('component.move', {'id': 'RB1', 'position': {'x': -210, 'y': -100}})
        edited = client.ask('project.inspect')
        assert edited['mixed'] == reopened['mixed'] and edited['run_id'] == reopened['run_id']
        client.ask('example.current.reset', {}, True, 'unsaved-changes')
        assert invariant(client.ask('project.inspect')) == invariant(edited)
        mutate('example.current.reset', {'expected_example_id': 'flasher',
                                        'expected_document_generation': edited['document']['generation'],
                                        'discard_changes': True})
        fresh = client.ask('simulation.wait')
        graph = {k: fresh[k] for k in ('components', 'wires')}
        assert electrical(graph) == electrical(old['graph'])
        positions = {p['id']: p['position'] for p in fresh['components'] if p['id'] in ('V1', 'RB1', 'RB2')}
        assert positions == {'V1': {'x': -400, 'y': 0}, 'RB1': {'x': -330, 'y': -100}, 'RB2': {'x': 280, 'y': -100}}
        assert fresh['paused'] and not fresh['example']['modified'] and fresh['transient']['sample']['time_s'] == 0
        mutate('edit.undo')
        restored = client.ask('simulation.wait')
        assert restored['components'] == edited['components'] and restored['wires'] == edited['wires']
        # Project replacement clears recordings; undo intentionally restores only schematic history.
        assert restored['document']['recording_count'] == 0
        mutate('edit.redo')
        redone = client.ask('simulation.wait')
        assert redone['components'] == graph['components'] and redone['wires'] == graph['wires']
        operation = mutate('simulation.verify', {'duration_s': .2, 'probe_ids': ['i:D1', 'i:D2', 'v:C1', 'v:C2']})['last_operation']
        deadline = time.monotonic()+20
        while True:
            capture = client.ask('operation.inspect', {'id': operation})
            if capture['operation']['status'] == 'completed':
                break
            assert capture['operation']['status'] not in ('failed', 'canceled') and time.monotonic() < deadline
            time.sleep(.01)
        mutate('verification.export', {'id': operation, 'format': 'json', 'path': str(assets/'flasher.json')})
        captured = json.loads((assets/'flasher.json').read_bytes())
        assert captured['rows'] == old['recordings']['entries'][0]['data']['rows']
        assert captured['metadata']['graph'] == graph
        mutate('project.save', {'path': str(assets/'flasher.elab')})
        before_export = invariant(client.ask('project.inspect'))
        exports = {}
        for fmt in ('svg', 'png', 'pdf'):
            path = assets/f'flasher.{fmt}'
            mutate('export.schematic.'+fmt, {'path': str(path)})
            if fmt != 'pdf':
                exports[fmt] = hashlib.sha256(path.read_bytes()).hexdigest()
            else:
                assert path.read_bytes().startswith(b'%PDF-')
        assert invariant(client.ask('project.inspect')) == before_export
        mutate('project.load', {'path': str(assets/'flasher.elab')})
        final = client.ask('simulation.wait')
        assert final['components'] == graph['components'] and final['document']['recording_count'] == 1
        assert args.legacy.read_bytes() == original
        report = dict(passed=True, transport='desktop-socket' if args.connect else 'headless-stdio',
                      old_project_sha256=hashlib.sha256(original).hexdigest(), revised_positions=positions,
                      captured_rows=len(captured['rows']), rows_sha256=hashlib.sha256(json.dumps(captured['rows'], sort_keys=True).encode()).hexdigest(),
                      exports=exports, checks=['old saved geometry and kept recording reopen/resave unchanged',
                                              'dirty current reset refuses; guarded discard resets only on request',
                                              'undo restores old geometry; redo restores new template; replacement clears session recordings',
                                              'captured 200 ms values exactly match previous-release recording',
                                              'exports preserve circuit/history/document; new saved template reopens'])
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
        if args.keep_open:
            mutate('scope.visibility.set', {'visible': False})
            mutate('view.fit', {'follow_viewport': True})
            keep = True
    finally:
        if args.connect and not keep:
            client.ask('app.close', {'discard_changes': True}, True)
        client.close()


if __name__ == '__main__':
    main()
