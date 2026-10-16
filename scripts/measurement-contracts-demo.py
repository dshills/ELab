#!/usr/bin/env python3
"""Exercise discoverable live inspection and saved pin controls through public commands."""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
COMMANDS = ['measurement.inspect', 'measurement.select', 'measurement.pin.add',
            'measurement.pin.move', 'measurement.pin.remove']


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)

    def inspect():
        return client.ask('project.inspect')

    def pin(state, identity):
        return next(p for p in state['pinned_readings'] if p['definition']['id'] == identity)

    def physics(state):
        return {key: state.get(key) for key in ('components', 'wires', 'transient', 'run_id',
                                               'electrical_revision', 'selected')}

    def refuse(command, params, code):
        before = inspect()
        client.ask(command, params, True, code)
        assert inspect() == before

    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': '10 ms'}, True)
        before = inspect()
        schemas = {}
        for command in COMMANDS:
            cap = client.ask('capabilities.list', {'command': command})['capabilities'][0]
            schema = cap['input_schema']
            assert schema['x-command-contract']['scope'] == 'live-measurement-inspection-and-pins'
            schemas[command] = schema
        assert inspect() == before

        client.ask('component.select', {'id': 'R1'}, True)
        client.ask('measurement.select', {'kind': 'net', 'id': 'net:C1.1'}, True)
        before = inspect()
        current = client.ask('measurement.inspect')
        assert current['target']['anchor'] == 'C1.1' and current['target']['kind'] == 'net'
        assert client.ask('measurement.inspect', {'target': None}) == current
        empty = client.ask('measurement.inspect', {'target': {'kind': '', 'id': ''}})
        assert 'target' not in empty and not empty['available'] and empty['readings'] == []
        ground = client.ask('measurement.inspect', {'target': {'kind': 'terminal', 'id': 'GND.reference'}})
        assert ground['readings'][0]['value'] == 0 and ground['readings'][0]['state'] == 'live'
        assert inspect() == before
        cleared = client.ask('measurement.select', {'kind': '', 'id': ''}, True)
        assert cleared['measurement_inspection']['target'] == {'kind': 'object', 'id': 'R1'}
        for params, code in [({'kind': 'current', 'id': 'C1'}, 'invalid-params'),
                             ({'kind': '', 'id': 'C1'}, 'invalid-params'),
                             ({'kind': 'terminal', 'id': 'C1.missing'}, 'not-found')]:
            refuse('measurement.select', params, code)

        client.ask('scope.probes.set', {'ids': []}, True)
        before = inspect()
        client.ask('measurement.pin.add', dict(id='v:C1', position=dict(x=23, y=47)), True)
        state = client.ask('measurement.pin.add', dict(id='i:C1', position=dict(x=320, y=50)), True)
        assert physics(before) == physics(state) and state['document']['preferences_dirty']
        charged = pin(state, 'v:C1')['value']
        assert math.isclose(charged, 5 * -math.expm1(-1), rel_tol=1e-12)
        assert pin(state, 'v:C1')['definition']['position'] == dict(x=20, y=50)
        refuse('measurement.pin.add', dict(id='v:C1', position=dict(x=0, y=0)), 'invalid-params')
        refuse('measurement.pin.move', dict(id='v:C1', position=dict(x=math.nextafter(10000, math.inf), y=0)), 'invalid-params')
        refuse('measurement.pin.move', dict(id='v:C1', position=dict(x=0, y=0), previous_position=dict(x=21, y=50)), 'conflict')
        state = client.ask('measurement.pin.move', dict(id='v:C1', position=dict(x=-5, y=5), previous_position=dict(x=20, y=50)), True)
        assert pin(state, 'v:C1')['definition']['position'] == dict(x=-10, y=10)
        before = inspect()
        same = client.ask('measurement.pin.move', dict(id='v:C1', position=dict(x=-9, y=9), previous_position=dict(x=-10, y=10)), True)
        for key in ('document', 'undo_entry', 'pinned_readings'):
            assert before.get(key) == same.get(key), key
        undo = client.ask('edit.undo', {}, True)
        assert pin(undo, 'v:C1')['definition']['position'] == dict(x=20, y=50)
        redo = client.ask('edit.redo', {}, True)
        assert pin(redo, 'v:C1')['definition']['position'] == dict(x=-10, y=10)
        assert physics(before) == physics(redo)

        fillers = []
        state = inspect()
        available = state['scope']['available']
        for probe in available:
            if len(state['pinned_readings']) == 8:
                break
            if probe['id'] not in ('v:C1', 'i:C1'):
                fillers.append(probe['id'])
                state = client.ask('measurement.pin.add', dict(id=probe['id'], position=dict(x=100*len(fillers), y=400)), True)
        assert len(state['pinned_readings']) == schemas['measurement.pin.add']['x-max-pinned-readings'] == 8
        ninth = next(p['id'] for p in available if p['id'] not in fillers + ['v:C1', 'i:C1'])
        refuse('measurement.pin.add', dict(id=ninth, position=dict(x=0, y=0)), 'invalid-params')
        for identity in fillers:
            client.ask('measurement.pin.remove', {'id': identity}, True)
        refuse('measurement.pin.remove', {'id': 'v:missing'}, 'not-found')
        refuse('measurement.pin.move', dict(id='v:missing', position=dict(x=0, y=0)), 'not-found')

        started = client.ask('simulation.verify', dict(duration_s='10 ms', input_mode='charge', probe_ids=['v:C1']), True)
        identity = started['last_operation']
        deadline = time.monotonic() + 10
        while True:
            operation = client.ask('operation.inspect', {'id': identity})['operation']
            assert operation['status'] not in ('failed', 'canceled'), operation
            if operation['status'] == 'completed':
                break
            assert time.monotonic() < deadline
            time.sleep(.01)
        client.ask('recording.view.set', {'mode': 'recorded'}, True)
        client.ask('simulation.input.set', {'mode': 'discharge'}, True)
        live = client.ask('simulation.step', {'duration_s': '1 ms'}, True)
        discharged = pin(live, 'v:C1')['value']
        current_a = pin(live, 'i:C1')['value']
        assert math.isclose(discharged, charged * math.exp(-.1), rel_tol=1e-12)
        assert math.isclose(current_a, -discharged / 10000, rel_tol=1e-12)
        assert live['recording']['time_s'] == 0 and live['recording']['readings'][0]['value'] == 0
        measured = client.ask('measurement.inspect', {'target': {'kind': 'probe', 'id': 'v:C1'}})
        assert measured['readings'][0]['value'] == discharged

        client.ask('recording.view.set', {'mode': 'live'}, True)
        client.ask('measurement.select', {'kind': 'terminal', 'id': 'C1.1'}, True)
        stale = client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        assert pin(stale, 'v:C1')['state'] == 'stale' and pin(stale, 'v:C1')['value'] == discharged
        missing = client.ask('object.delete', {'id': 'C1'}, True)
        assert not missing['measurement_inspection']['available']
        for identity in ('v:C1', 'i:C1'):
            assert pin(missing, identity)['state'] == 'unavailable' and 'value' not in pin(missing, identity)
        client.ask('measurement.pin.move', dict(id='v:C1', position=dict(x=0, y=600), previous_position=dict(x=-10, y=10)), True)
        removed = client.ask('measurement.pin.remove', {'id': 'i:C1'}, True)
        assert len(removed['pinned_readings']) == 1
        restored = client.ask('edit.undo', {}, True)
        assert pin(restored, 'i:C1')['state'] == 'unavailable'
        project = assets / 'missing-live-pins.elab'
        client.ask('project.save', {'path': str(project)}, True)
        digest = hashlib.sha256(project.read_bytes()).hexdigest()
        client.ask('project.load', {'path': str(project)}, True)
        reopened = client.ask('simulation.wait')
        assert all(p['state'] == 'unavailable' and 'value' not in p for p in reopened['pinned_readings'])
        assert hashlib.sha256(project.read_bytes()).hexdigest() == digest and not reopened['document']['dirty']
        client.ask('component.place', dict(id='C2', kind='capacitor', position=dict(x=600, y=600)), True)
        fresh = client.ask('measurement.pin.add', dict(id='v:C2', position=dict(x=0, y=0)), True)
        assert pin(fresh, 'v:C2')['state'] == 'unavailable' and 'value' not in pin(fresh, 'v:C2')
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', detailed_schemas=schemas,
                      observations=dict(accepted_ground_v=ground['readings'][0]['value'], charged_v=charged,
                                        discharged_v=discharged, discharged_a=current_a, maximum_pins=8,
                                        half_grid_position=dict(x=-10, y=10), reopened_missing_pins=2),
                      checks=['Five pure discoverable measurement contracts with current inventories and signed live provenance',
                              'Omitted/null/empty query targets and empty selection have distinct documented behavior',
                              'Unplotted live pins, snapping, boundary/duplicate/capacity refusal and exact previous-position guards',
                              'Same snapped move creates no history; undo/redo restores a whole move without physical changes',
                              'Captured replay does not substitute values for signed live RC voltage/current pins or inspection',
                              'Paused edits mark old accepted values stale; removed sources are unavailable without numeric zeros',
                              'Missing pins remain movable/removable/undoable and portable through save/load',
                              'New current source without accepted data can be pinned explicitly unavailable'],
                      limits=['No new device models, numerical kernels, saved formats or public command IDs',
                              'Semantic/native endpoint behavior is separate from complete keyboard/focus/IME/accessibility/latency qualification'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as output:
            json.dump(report, output, indent=2)
            output.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], contracts=len(schemas))))
    finally:
        try:
            if args.connect:
                client.ask('project.inspect')
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()
