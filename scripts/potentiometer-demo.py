#!/usr/bin/env python3
"""Public-command potentiometer divider, live RC adjustment and portability checks."""
import argparse
import csv
import importlib.util
import json
import math
from pathlib import Path
import tempfile
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path)
    args = parser.parse_args()
    client = helper.Client(args.connect)
    fresh = None
    try:
        catalog = client.ask('capabilities.list')
        kinds = client.ask('parts.list')
        client.ask('project.clear', {'discard_changes': True}, True)
        client.ask('simulation.resume', {}, True)
        client.ask('edit.batch', {'name': 'Unloaded pot divider', 'edits': [
            {'command': 'component.place', 'params': {'id': 'V1', 'kind': 'dc-source', 'position': {'x': -200, 'y': 0}, 'value': '5V'}},
            {'command': 'component.place', 'params': {'id': 'P1', 'kind': 'potentiometer', 'position': {'x': 0, 'y': -100}, 'value': '10k'}},
            {'command': 'component.place', 'params': {'id': 'GND', 'kind': 'ground', 'position': {'x': 0, 'y': 120}}},
            {'command': 'wire.connect', 'params': {'from': 'V1.positive', 'to': 'P1.1'}},
            {'command': 'wire.connect', 'params': {'from': 'P1.2', 'to': 'GND.reference'}},
            {'command': 'wire.connect', 'params': {'from': 'V1.negative', 'to': 'GND.reference'}},
        ]}, True)
        unloaded = client.ask('simulation.wait')
        assert abs(unloaded['mixed']['potentiometers'][0]['wiper_voltage_v']-2.5) < 1e-9
        endpoints = []
        for w in [0, 1, .5]:
            client.ask('potentiometer.wiper.set', {'id': 'P1', 'position': w}, True)
            reading = client.ask('simulation.wait')['mixed']['potentiometers'][0]
            assert abs(reading['wiper_voltage_v']-5*(1+9998*w)/10000) < 1e-9
            assert reading['upper_resistance_ohm']+reading['lower_resistance_ohm'] == 10000
            endpoints.append(reading['wiper_voltage_v'])
        client.ask('edit.batch', {'name': 'Load divider', 'edits': [
            {'command': 'component.place', 'params': {'id': 'R1', 'kind': 'resistor', 'position': {'x': 200, 'y': 0}, 'value': '10k'}},
            {'command': 'wire.connect', 'params': {'from': 'P1.wiper', 'to': 'R1.1'}},
            {'command': 'wire.connect', 'params': {'from': 'R1.2', 'to': 'GND.reference'}},
        ]}, True)
        loaded = client.ask('simulation.wait')
        assert abs(loaded['mixed']['potentiometers'][0]['wiper_voltage_v']-2) < 1e-9
        assert abs(loaded['mixed']['power_residual_w']) < 1e-8
        client.ask('potentiometer.wiper.set', {'id': 'P1', 'position': 1.1}, True, expect='invalid-params')
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('edit.batch', {'name': 'Pot capacitor experiment', 'edits': [
            {'command': 'object.delete', 'params': {'id': 'R1'}},
            {'command': 'component.place', 'params': {'id': 'Timing', 'kind': 'potentiometer', 'position': {'x': 0, 'y': -110}, 'value': '10k'}},
            {'command': 'wire.connect', 'params': {'from': 'V1.positive', 'to': 'Timing.1'}},
            {'command': 'wire.connect', 'params': {'from': 'Timing.2', 'to': 'GND.reference'}},
            {'command': 'wire.connect', 'params': {'from': 'Timing.wiper', 'to': 'C1.1'}},
        ]}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.step', {'duration_s': .005}, True)
        before = client.ask('project.inspect')
        voltage = before['mixed']['capacitors'][0]['voltage_v']
        ref = 2.5*(1-math.exp(-.005/.0025))
        assert abs(voltage-ref) < .002
        client.ask('potentiometer.wiper.set', {'id': 'Timing', 'position': .75}, True)
        moved = client.ask('project.inspect')
        assert moved['run_id'] == before['run_id']
        assert moved['transient']['sample']['time_s'] == before['transient']['sample']['time_s']
        assert abs(moved['mixed']['capacitors'][0]['voltage_v']-voltage) < 1e-10
        assert moved['transient']['events'][-1]['mode'] == 'wiper:Timing:0.75'
        client.ask('simulation.step', {'duration_s': .005}, True)
        after = client.ask('project.inspect')
        target = 5*7499.5/10000
        tau = 2500.5*7499.5/10000*1e-6
        reference_after = target+(voltage-target)*math.exp(-.005/tau)
        assert abs(after['mixed']['capacitors'][0]['voltage_v']-reference_after) < .002
        accepted = client.ask('simulation.verify', {'duration_s': .01, 'probe_ids': ['v:Timing.wiper', 'i:Timing.wiper', 'v:Timing']}, True)
        op = accepted['last_operation']
        deadline = time.monotonic()+15
        while time.monotonic() < deadline:
            record = client.ask('operation.inspect', {'id': op})
            if record['operation']['status'] not in ['queued', 'running', 'canceling']:
                break
            time.sleep(.02)
        assert record['operation']['status'] == 'completed', record
        assert record['metadata']['potentiometer_model']['id'] == 'elab.linear-potentiometer.v1'
        with tempfile.TemporaryDirectory(prefix='elab-pot-') as temporary:
            directory = args.artifacts.resolve() if args.artifacts else Path(temporary)
            directory.mkdir(parents=True, exist_ok=True)
            for kind in ['json', 'csv']:
                client.ask('verification.export', {'id': op, 'path': str(directory/('pot-record.'+kind)), 'format': kind, 'overwrite': True}, True)
            with (directory/'pot-record.csv').open() as source:
                rows = list(csv.DictReader(source))
            assert rows[-1]['potentiometer_model'] == 'elab.linear-potentiometer.v1'
            client.ask('verification.import', {'path': str(directory/'pot-record.json')}, True)
            path = directory/'pot-rc.elab'
            client.ask('project.save', {'path': str(path), 'overwrite': True}, True)
            client.ask('export.schematic.svg', {'path': str(directory/'pot-rc.svg'), 'overwrite': True}, True)
            fresh = helper.Client()
            fresh.ask('project.load', {'path': str(path)}, True)
            reopened = fresh.ask('project.inspect')
            assert reopened['document']['recording_count'] == 2 and not reopened['document']['dirty'] and reopened['paused']
            assert reopened['potentiometer_model'] == before['potentiometer_model']
            assert next(p for p in reopened['components'] if p['id'] == 'Timing')['wiper_position'] == .75
            report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                          capabilities=len(catalog['capabilities']), library_kinds=len(kinds),
                          unloaded_voltage_v=2.5, loaded_voltage_v=loaded['mixed']['potentiometers'][0]['wiper_voltage_v'],
                          endpoint_voltages_v=endpoints[:2], capacitor_voltage_v=voltage, analytical_voltage_v=ref,
                          changed_capacitor_voltage_v=after['mixed']['capacitors'][0]['voltage_v'], changed_analytical_voltage_v=reference_after,
                          checks=['blank unloaded divider construction', 'finite endpoints preserve total resistance', 'loaded divider/conservation',
                                  'invalid wiper refused', 'Thevenin RC references before/after movement', 'charge/time/run continuity and event',
                                  'Verify model and pin probes', 'JSON import and CSV attribution', 'two embedded records and saved wiper round trip', 'schematic SVG'])
            if args.output:
                args.output.parent.mkdir(parents=True, exist_ok=True)
                args.output.write_text(json.dumps(report, indent=2)+'\n')
            print(json.dumps(report, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()
