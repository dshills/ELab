#!/usr/bin/env python3
"""Exercise shared object navigation and Fit on an owned test session."""
import argparse
import json
from pathlib import Path
import importlib.util


spec = importlib.util.spec_from_file_location('elab_persistence', Path(__file__).with_name('persistence-demo.py'))
persistence = importlib.util.module_from_spec(spec)
spec.loader.exec_module(persistence)
Client = persistence.Client


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--close', action='store_true', help='close this owned native host after the workflow')
    args = parser.parse_args()
    client = connect_demo_client(Client, args.connect, command_preconditions=True)
    try:
        client.ask('project.clear', {'discard_changes': True}, True)
        client.ask('component.place', {'kind': 'net-label', 'position': {'x': 0, 'y': 0}, 'value': 'N' * 32}, True)
        fitted = client.ask('view.fit', {'width_px': 650, 'height_px': 400}, True)
        part = fitted['components'][0]
        right = part['position']['x'] + 50 + (len(part['id']) + 35) * 7.5
        assert abs((right - fitted['view']['center']['x']) * fitted['view']['zoom']) <= 325
        client.ask('example.reset', {'discard_changes': True}, True)
        before = client.ask('simulation.pause', {}, True)
        state = client.ask('selection.navigate', {'direction': 'first', 'center': True}, True)
        assert state['selected'] == before['components'][0]['id']
        assert state['view']['center'] == before['components'][0]['position']
        state = client.ask('selection.navigate', {'direction': 'previous'}, True)
        assert state['selected'] == before['wires'][-1]['id']
        state = client.ask('selection.navigate', {'direction': 'last', 'kind': 'components', 'center': True}, True)
        last = state['selected']
        client.ask('view.pan', {'dx': 100, 'dy': 100}, True)
        after = client.ask('view.selection.center', {'id': last}, True)
        assert after['view'] == state['view']
        client.ask('view.selection.center', {'id': 'not-selected'}, True, 'conflict')
        for field in ['components', 'wires', 'electrical_revision', 'run_id', 'undo_name', 'transient']:
            assert before.get(field) == after.get(field), field
        assert before['document']['dirty'] == after['document']['dirty']
        detailed = client.ask('capabilities.list', {'command': 'selection.navigate'})
        schema = detailed['capabilities'][0]['input_schema']
        assert schema['properties']['direction']['enum'] == ['next', 'previous', 'first', 'last']
        report = {'passed': True, 'transport': 'desktop-socket' if args.connect else 'headless-stdio',
                  'first_component': before['components'][0]['id'], 'last_component': last,
                  'wrap_wire': before['wires'][-1]['id'], 'long_label_fit': fitted['view'],
                  'source_physics_history_preserved': True, 'selection_conflict_refused': True,
                  'scope': 'public commands; no physical keyboard/focus/screen-reader claim'}
        args.output.write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps(report))
    finally:
        try:
            if args.close:
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()
