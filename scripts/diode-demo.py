#!/usr/bin/env python3
"""Public-command silicon diode DC, pulse rectification, export and reopen fixture."""
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


def reference_current(voltage, resistance):
    nv = 1.7*1.380649e-23*300.15/1.602176634e-19
    lo, hi = 0., voltage/(resistance+1)
    for _ in range(65):
        i = (lo+hi)/2
        if i*(resistance+1)+nv*math.log1p(i/1e-9) > voltage:
            hi = i
        else:
            lo = i
    return (lo+hi)/2


def reference_capacitor():
    voltage, dt = 0., 2e-6
    def slope(v):
        return reference_current(5-v, 10000)/1e-6
    for _ in range(5000):
        a = slope(voltage)
        b = slope(voltage+dt*a/2)
        c = slope(voltage+dt*b/2)
        d = slope(voltage+dt*c)
        voltage += dt*(a+2*b+2*c+d)/6
    return voltage


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
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.resume', {}, True)
        client.ask('edit.batch', {'name': 'Silicon conduction', 'edits': [
            {'command': 'object.delete', 'params': {'id': 'D1'}},
            {'command': 'component.place', 'params': {'kind': 'diode', 'id': 'Silicon', 'position': {'x': 200, 'y': 0}}},
            {'command': 'wire.connect', 'params': {'from': 'R1.2', 'to': 'Silicon.anode'}},
            {'command': 'wire.connect', 'params': {'from': 'Silicon.cathode', 'to': 'GND.reference'}},
        ]}, True)
        forward = client.ask('simulation.wait')
        current = forward['mixed']['diodes'][0]['current_a']
        ref_current = reference_current(5, 1000)
        assert abs(current-ref_current) < 1e-11
        connected = [w['id'] for w in forward['wires'] if w['from'].startswith('Silicon.') or w['to'].startswith('Silicon.')]
        client.ask('edit.batch', {'name': 'Reverse bias', 'edits': [
            *[{'command': 'object.delete', 'params': {'id': i}} for i in connected],
            {'command': 'wire.connect', 'params': {'from': 'R1.2', 'to': 'Silicon.cathode'}},
            {'command': 'wire.connect', 'params': {'from': 'Silicon.anode', 'to': 'GND.reference'}},
        ]}, True)
        reverse = client.ask('simulation.wait')
        reverse_current = reverse['mixed']['diodes'][0]['current_a']
        assert abs(reverse_current+1e-9) < 1e-12
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('edit.batch', {'name': 'Diode capacitor hold', 'edits': [
            {'command': 'object.delete', 'params': {'id': 'W2'}},
            {'command': 'component.place', 'params': {'kind': 'diode', 'id': 'Rectifier', 'position': {'x': 100, 'y': -60}}},
            {'command': 'wire.connect', 'params': {'from': 'R1.2', 'to': 'Rectifier.anode'}},
            {'command': 'wire.connect', 'params': {'from': 'Rectifier.cathode', 'to': 'C1.1'}},
        ]}, True)
        client.ask('simulation.transient.set', {'step_s': .001, 'high_s': .01, 'low_s': .01, 'initial_voltage_v': 0, 'seconds_per_wall_second': .02}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.step', {'duration_s': .01}, True)
        charge = client.ask('project.inspect')
        cap_voltage = charge['mixed']['capacitors'][0]['voltage_v']
        reference_cap = reference_capacitor()
        assert abs(cap_voltage-reference_cap) < .003
        client.ask('simulation.step', {'duration_s': .009}, True)
        held = client.ask('project.inspect')
        assert abs(held['mixed']['capacitors'][0]['voltage_v']-(cap_voltage-9e-6)) < 2e-8
        assert abs(held['mixed']['diodes'][0]['current_a']+1e-9) < 1e-12
        assert any(e['mode'] == 'pulse-low' for e in held['transient']['events'])
        accepted = client.ask('simulation.verify', {'duration_s': .019, 'probe_ids': ['i:Rectifier', 'v:C1']}, True)
        op = accepted['last_operation']
        deadline = time.monotonic()+15
        while time.monotonic() < deadline:
            record = client.ask('operation.inspect', {'id': op})
            if record['operation']['status'] not in ['queued', 'running', 'canceling']:
                break
            time.sleep(.02)
        assert record['operation']['status'] == 'completed', record
        assert abs(record['metadata']['mixed']['capacitors'][0]['voltage_v']-held['mixed']['capacitors'][0]['voltage_v']) < 1e-10
        with tempfile.TemporaryDirectory(prefix='elab-diode-') as temporary:
            directory = args.artifacts.resolve() if args.artifacts else Path(temporary)
            directory.mkdir(parents=True, exist_ok=True)
            for kind in ['json', 'csv']:
                client.ask('verification.export', {'id': op, 'path': str(directory/('diode-record.'+kind)), 'format': kind, 'overwrite': True}, True)
            with (directory/'diode-record.csv').open() as source:
                rows = list(csv.DictReader(source))
            assert rows[-1]['diode_model'] == 'elab.generic-silicon-diode.v1'
            assert float(rows[-1]['i:Rectifier [A]']) < 0
            client.ask('verification.import', {'path': str(directory/'diode-record.json')}, True)
            path = directory/'diode-rc.elab'
            client.ask('project.save', {'path': str(path), 'overwrite': True}, True)
            client.ask('export.schematic.svg', {'path': str(directory/'diode-rc.svg'), 'overwrite': True}, True)
            fresh = helper.Client()
            fresh.ask('project.load', {'path': str(path)}, True)
            loaded = fresh.ask('project.inspect')
            assert loaded['document']['recording_count'] == 2 and not loaded['document']['dirty']
            assert loaded['diode_model'] == forward['diode_model'] and loaded['paused']
            report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                          capabilities=len(catalog['capabilities']), library_kinds=len(kinds),
                          forward_current_a=current, independent_current_a=ref_current, reverse_current_a=reverse_current,
                          capacitor_voltage_v=cap_voltage, independent_capacitor_voltage_v=reference_cap, held_voltage_v=held['mixed']['capacitors'][0]['voltage_v'],
                          checks=['forward inverse-Shockley reference', 'signed reverse leakage', 'independent nonlinear RC reference', 'automatic pulse rectification/hold',
                                  'exact source edge event', 'live/Verify equivalence', 'record JSON import and CSV model/units',
                                  'two kept records and complete model save/reopen', 'native schematic SVG'])
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
