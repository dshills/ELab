#!/usr/bin/env python3
"""Inspect live components, named terminals, wires, probes and anchored nets."""
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
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    client = helper.Client(args.connect, command_preconditions=True)
    fresh = None
    try:
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.resume', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.pause', {}, True)
        client.ask('component.select', {'id': 'R1'}, True)
        before = client.ask('project.inspect')
        component = client.ask('measurement.inspect')
        readings = {r['probe']['id']: r for r in component['readings']}
        assert component['available'] and component['result_kind'] == 'dc' and len(readings) == 4
        assert abs(readings['v:R1']['value']-1000*readings['i:R1']['value']) < 1e-12
        assert readings['v:R1']['reference_display'] == 'Voltage: R1.1 - R1.2'
        assert readings['i:R1']['reference_display'] == 'Current: R1.1 -> R1.2'
        ground = client.ask('measurement.inspect', {'target': {'kind': 'terminal', 'id': 'GND.reference'}})
        assert ground['readings'][0]['value'] == 0
        assert client.ask('project.inspect') == before
        client.ask('measurement.pin.add', {'id': 'i:R1', 'position': {'x': 300, 'y': 180}}, True)
        assert client.ask('project.inspect')['pinned_readings'][0]['value'] == readings['i:R1']['value']
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        before = client.ask('simulation.step', {'duration_s': .01}, True)
        selected = client.ask('measurement.select', {'kind': 'terminal', 'id': 'C1.1'}, True)
        terminal = selected['measurement_inspection']
        voltage = terminal['readings'][0]['value']
        assert abs(voltage-5*(1-math.exp(-1))) < 1e-12
        assert terminal['wire_ids'] == ['W2'] and set(terminal['terminal_refs']) == {'C1.1', 'R1.2'}
        for field in ['transient', 'run_id', 'electrical_revision', 'selected', 'document', 'undo_entry']:
            assert selected[field] == before[field], field
        net = client.ask('measurement.select', {'kind': 'net', 'id': 'net:C1.1'}, True)['measurement_inspection']
        assert net['target']['anchor'] == 'C1.1' and net['readings'][0]['value'] == voltage
        client.ask('component.select', {'id': 'W1'}, True)
        wire = client.ask('measurement.inspect')
        assert len(wire['readings']) == 1 and wire['readings'][0]['probe']['unit'] == 'V' and wire['readings'][0]['value'] == 5
        client.ask('measurement.select', {'kind': 'probe', 'id': 'i:C1'}, True)
        client.ask('simulation.input.set', {'mode': 'discharge'}, True)
        client.ask('simulation.step', {'duration_s': .001}, True)
        assert client.ask('measurement.inspect')['readings'][0]['value'] < 0
        client.ask('scope.probes.set', {'ids': ['v:C1.1', 'i:C1']}, True)
        client.ask('measurement.pin.add', {'id': 'v:C1.1', 'position': {'x': 350, 'y': 200}}, True)
        accepted = client.ask('simulation.verify', {'duration_s': .01, 'probe_ids': ['v:C1.1', 'i:C1']}, True)
        deadline = time.monotonic()+15
        while True:
            record = client.ask('operation.inspect', {'id': accepted['last_operation']})
            if record['operation']['status'] not in ['queued', 'running', 'canceling']:
                break
            assert time.monotonic() < deadline
            time.sleep(.005)
        assert record['operation']['status'] == 'completed'
        client.ask('operation.select', {'id': accepted['last_operation']}, True)
        live = client.ask('measurement.inspect')
        client.ask('recording.view.set', {'mode': 'recorded'}, True)
        assert client.ask('measurement.inspect') == live
        client.ask('recording.view.set', {'mode': 'live'}, True)
        client.ask('measurement.select', {'kind': 'net', 'id': 'net:C1.1'}, True)
        directory = args.artifacts.resolve()
        directory.mkdir(parents=True, exist_ok=True)
        path = str(directory/'inspected-rc.elab')
        client.ask('project.save', {'path': path, 'overwrite': True}, True)
        anchor_part = 'A'+str(time.time_ns())
        client.ask('component.place', {'id': anchor_part, 'kind': 'resistor', 'position': {'x': -500, 'y': -400}}, True)
        client.ask('wire.connect', {'from': anchor_part+'.1', 'to': 'C1.1'}, True)
        renamed = client.ask('measurement.inspect')
        assert renamed['target'] == {'kind': 'net', 'id': 'net:'+anchor_part+'.1', 'anchor': 'C1.1'}
        assert renamed['readings'][0]['state'] == 'stale' and abs(renamed['readings'][0]['value']-voltage*math.exp(-.1)) < 1e-12
        client.ask('object.delete', {'id': 'W2'}, True)
        assert 'R1.2' not in client.ask('measurement.inspect')['terminal_refs']
        client.ask('object.delete', {'id': 'C1'}, True)
        missing = client.ask('measurement.inspect')
        assert not missing['available'] and not missing['readings']
        client.ask('edit.undo', {}, True)
        assert client.ask('measurement.inspect')['available']
        client.ask('project.load', {'path': path, 'discard_changes': True}, True)
        reopened = client.ask('simulation.wait')
        assert 'target' not in reopened['measurement_inspection'] and reopened['paused'] and not reopened['document']['dirty']
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': path}, True)
        reopened = fresh.ask('simulation.wait')
        assert 'target' not in reopened['measurement_inspection'] and reopened['pinned_readings'][0]['value'] == 0
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                      rc_10ms_voltage_v=voltage,
                      checks=['read-only DC Ohm/reference-ground query', 'accepted DC pin value',
                              'independent RC terminal voltage', 'selection preserves physics/history/document',
                              'wire voltage without invented ideal-wire current', 'signed current and named references',
                              'same inspection probe IDs plot and pin', 'recorded view does not replace live inspection',
                              'net selection follows stable anchor through rename/split', 'deleted anchor unavailable and undo restores it',
                              'fresh paused reopen clears ephemeral inspection and restores pin/record', 'owned project ends clean and paused'],
                      limitations=['net selection follows its captured terminal; removing the anchor requires undo or reselection',
                                   'inspection is live, not a captured-record circuit selection mode',
                                   'semantic controls and pointer tests do not qualify complete keyboard focus or screen-reader behavior'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()
