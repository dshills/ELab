#!/usr/bin/env python3
"""Verify untouched editable values preserve source, physics and saved history."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)

    def component(state, identifier):
        return next(c for c in state['components'] if c['id'] == identifier)

    def unchanged_apply(state):
        after = client.ask('component.set', {'id': 'C1', 'value': state['drawing']['value']}, True)
        for key in ('components', 'wires', 'run_id', 'electrical_revision', 'transient', 'document', 'undo_entry', 'redo_entry'):
            assert after.get(key) == state.get(key), key
        assert after['command_revision'] > state['command_revision']
        return after

    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        client.ask('component.set', dict(id='C1', value='1.234567890123456 µF'), True)
        client.ask('component.select', dict(id='C1'), True)
        client.ask('component.set', dict(id='R1', value='20k'), True)
        client.ask('edit.undo', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', dict(duration_s='1 ms'), True)
        project = directory / 'exact-value.elab'
        client.ask('project.save', dict(path=str(project)), True)
        digest = hashlib.sha256(project.read_bytes()).hexdigest()
        before = client.ask('project.inspect')
        original = component(before, 'C1')['value']
        assert before.get('redo_entry') and not before['document']['dirty']
        state = before
        for _ in range(5):
            state = unchanged_apply(state)
            assert component(state, 'C1')['value'].hex() == original.hex()
        client.ask('edit.redo', {}, True)
        assert component(client.ask('project.inspect'), 'R1')['value'] == 20000
        client.ask('edit.undo', {}, True)
        restored = client.ask('project.inspect')
        assert component(restored, 'C1')['value'].hex() == original.hex() and not restored['document']['dirty']
        client.ask('project.load', dict(path=str(project)), True)
        client.ask('simulation.wait')
        client.ask('component.select', dict(id='C1'), True)
        reopened = client.ask('project.inspect')
        for _ in range(3):
            reopened = unchanged_apply(reopened)
            assert component(reopened, 'C1')['value'].hex() == original.hex()
        assert hashlib.sha256(project.read_bytes()).hexdigest() == digest
        readable = {}
        client.ask('component.set', dict(id='C1', value='1uF'), True)
        readable['capacitor'] = client.ask('project.inspect')['drawing']['value']
        client.ask('component.set', dict(id='R1', value='1k'), True)
        client.ask('component.select', dict(id='R1'), True)
        readable['resistor'] = client.ask('project.inspect')['drawing']['value']
        assert readable == dict(capacitor='1 µF', resistor='1 kΩ')
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      exact_value_hex=original.hex(), editable_text=before['drawing']['value'], readable=readable,
                      source_graph=dict(components=before['components'], wires=before['wires']),
                      sample=before['transient']['sample'], checks=['Published editable text preserves stored float bits',
                      'Five untouched applies preserve graph/physics/save/undo/redo while command revisions advance',
                      'Redo remains usable and returns to the clean saved graph', 'Reopened project survives three untouched applies',
                      'Saved file bytes unchanged', 'Common values retain compact engineering units'],
                      limits=['Editable draft formatting only; plot/measurement captions retain their display precision',
                              'No new solver/model/transport or project-format claim'])
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], editable_text=report['editable_text'])))
    finally:
        client.close()


if __name__ == '__main__':
    main()
