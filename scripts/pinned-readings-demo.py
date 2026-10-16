#!/usr/bin/env python3
"""Validate live pinned measurements, portable preferences, and SVG parity."""
import argparse
import importlib.util
import json
import math
from pathlib import Path
import time


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
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    fresh = None
    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        directory = args.artifacts.resolve()
        directory.mkdir(parents=True, exist_ok=True)
        path = str(directory/'pinned-rc.elab')
        clean = client.ask('project.save', {'path': path, 'overwrite': True}, True)
        for i, probe in enumerate(['v:C1', 'i:C1']):
            client.ask('measurement.pin.add', {'id': probe, 'position': {'x': 350, 'y': 250+60*i}}, True)
        pinned = client.ask('project.inspect')
        assert pinned['run_id'] == clean['run_id'] and pinned['transient'] == clean['transient']
        assert pinned['document']['preferences_dirty'] and not pinned['document']['schematic_dirty']
        charge = client.ask('simulation.step', {'duration_s': .01}, True)
        voltage = charge['pinned_readings'][0]['value']
        assert abs(voltage-5*(1-math.exp(-1))) < 1e-12
        assert charge['pinned_readings'][1]['value'] > 0
        client.ask('simulation.input.set', {'mode': 'discharge'}, True)
        discharge = client.ask('simulation.step', {'duration_s': .001}, True)
        assert abs(discharge['pinned_readings'][0]['value']-voltage*math.exp(-.1)) < 1e-12
        assert discharge['pinned_readings'][1]['value'] < 0
        client.ask('measurement.pin.move', {'id': 'i:C1', 'position': {'x': -420, 'y': 300},
                                          'previous_position': {'x': 350, 'y': 310}}, True)
        client.ask('measurement.pin.move', {'id': 'i:C1', 'position': {'x': 0, 'y': 0},
                                          'previous_position': {'x': 350, 'y': 310}}, True, 'conflict')
        client.ask('edit.undo', {}, True)
        assert client.ask('project.inspect')['pinned_readings'][1]['definition']['position']['x'] == 350
        client.ask('edit.redo', {}, True)
        client.ask('view.fit', {'width_px': 800, 'height_px': 500}, True)
        client.ask('export.schematic.svg', {'path': str(directory/'pinned-rc.svg'), 'overwrite': True}, True)
        svg = (directory/'pinned-rc.svg').read_text()
        assert 'Live v:C1' in svg and 'Live i:C1' in svg and 'pinned_readings' in svg
        accepted = client.ask('simulation.verify', {'duration_s': .01, 'probe_ids': ['v:C1', 'i:C1']}, True)
        deadline = time.monotonic()+15
        while True:
            record = client.ask('operation.inspect', {'id': accepted['last_operation']})
            if record['operation']['status'] not in ['queued', 'running', 'canceling']:
                break
            assert time.monotonic() < deadline
            time.sleep(.005)
        assert record['operation']['status'] == 'completed'
        client.ask('operation.select', {'id': accepted['last_operation']}, True)
        live = client.ask('project.inspect')['pinned_readings']
        recorded = client.ask('recording.view.set', {'mode': 'recorded'}, True)
        assert recorded['pinned_readings'] == live
        client.ask('recording.view.set', {'mode': 'live'}, True)
        client.ask('project.save', {'path': path, 'overwrite': True}, True)
        document = json.loads(Path(path).read_text())
        assert document['version'] == 4 and len(document['pinned_readings']['items']) == 2
        assert all(set(pin) == {'id', 'position'} for pin in document['pinned_readings']['items'])
        client.ask('object.delete', {'id': 'C1'}, True)
        unavailable = client.ask('project.inspect')['pinned_readings']
        assert all(pin['state'] == 'unavailable' and 'value' not in pin for pin in unavailable)
        missing_path = str(directory/'missing-source.elab')
        client.ask('project.save', {'path': missing_path, 'overwrite': True}, True)
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': missing_path}, True)
        assert all(pin['state'] == 'unavailable' for pin in fresh.ask('simulation.wait')['pinned_readings'])
        fresh.ask('project.load', {'path': path}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['paused'] and not reopened['document']['dirty'] and reopened['document']['recording_count'] == 1
        assert reopened['pinned_readings'][0]['state'] == 'live' and reopened['pinned_readings'][0]['value'] == 0
        assert reopened['pinned_readings'][1]['definition']['position']['x'] == -420
        client.ask('project.load', {'path': path}, True)
        final = client.ask('simulation.wait')
        assert final['paused'] and not final['document']['dirty']
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                      charge_voltage_v=voltage, discharge_voltage_v=discharge['pinned_readings'][0]['value'],
                      discharge_current_a=discharge['pinned_readings'][1]['value'],
                      checks=['pin preferences alone mark document dirty', 'pins preserve initial physics',
                              'independent exact RC charge/discharge reference', 'signed current and units',
                              'stale gesture refusal and move undo/redo', 'SVG retains cards and measurement provenance',
                              'recorded plot leaves explicitly live pins unchanged', 'saved pins contain no trusted measurements',
                              'removed source remains unavailable across save/reopen', 'fresh paused startup with positions and kept record',
                              'owned project ends clean and paused'],
                      limitations=['eight pins and grid positions inside ±10000 units',
                                   'version-4 optional pinned_readings extension: older builds reject it rather than discard it',
                                   'no power/derived-expression pins or recorded-reading pin mode',
                                   'mounted event tests/GPU readback do not establish complete keyboard or screen-reader qualification'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()
