#!/usr/bin/env python3
"""Build, measure, capture, sweep and reopen floating DC supplies via public commands."""
import argparse
import importlib.util
import json
import math
from pathlib import Path
import time

from validation_artifacts import discovery_summary

spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native endpoint; closes afterward')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect)
    mutate = lambda name, params=None: client.ask(name, params, True)

    def restart():
        mutate('simulation.restart')
        return client.ask('simulation.wait')

    def build(parts, wires):
        for identity, kind, value, x, y, rotation in parts:
            p = dict(id=identity, kind=kind, position=dict(x=x, y=y))
            if value:
                p['value'] = value
            mutate('component.place', p)
            for _ in range(rotation//90):
                mutate('component.rotate', {'id': identity})
        for source, target in wires:
            mutate('wire.connect', {'from': source, 'to': target})

    def read(probe):
        result = client.ask('measurement.inspect', {'target': {'kind': 'probe', 'id': probe}})
        assert result['available'], result
        reading = next(r for r in result['readings'] if r['probe']['id'] == probe)
        assert reading['state'] != 'stale' and 'value' in reading, reading
        return reading['value']

    def complete(command, identity):
        deadline = time.monotonic()+20
        while time.monotonic() < deadline:
            result = client.ask(command, {'id': identity})
            assert result['operation']['status'] not in ('failed', 'canceled'), result
            if result['operation']['status'] == 'completed':
                return result
            time.sleep(.01)
        raise AssertionError(f'{command} did not complete')

    try:
        assert not client.ask('project.inspect')['components'], 'requires fresh owned blank session'
        discovery = discovery_summary(client.ask('capabilities.list', {'command': 'component.place'}), assets/'discovery.json')
        assert 'x-source-reference-rules' in discovery['capabilities'][0]['input_schema']
        part = next(p for p in client.ask('parts.list') if p['kind'] == 'dc-source')
        mutate('simulation.pause')
        build([('V1', 'dc-source', '3 V', -180, -100, 0), ('V2', 'dc-source', '3 V', -180, 100, 0),
               ('R1', 'resistor', '1k', 180, 0, 90), ('J1', 'junction', '', -180, 0, 0),
               ('GND1', 'ground', '', 0, 60, 0)],
              [('V1.positive', 'R1.1'), ('R1.2', 'V2.negative'), ('V1.negative', 'J1.node'),
               ('V2.positive', 'J1.node'), ('J1.node', 'GND1.reference')])
        manual = restart()
        mutate('example.load', {'id': 'split-supply', 'discard_changes': True})
        initial = client.ask('simulation.wait')
        assert manual['components'] == initial['components'] and manual['wires'] == initial['wires']
        split = {probe: read(probe) for probe in ('v:V1.positive', 'v:V2.negative', 'v:V2', 'v:R1', 'i:R1', 'i:V2')}
        for probe, expected in {'v:V1.positive': 3, 'v:V2.negative': -3, 'v:V2': 3,
                                'v:R1': 6, 'i:R1': .006, 'i:V2': .006}.items():
            assert abs(split[probe]-expected) < 1e-12, (probe, split)
        for fmt in ('svg', 'png'):
            mutate(f'export.schematic.{fmt}', {'path': str(assets/f'split.{fmt}')})
        admitted = mutate('simulation.verify', {'duration_s': 0, 'probe_ids': list(split)})
        captured = complete('operation.inspect', admitted['last_operation'])
        for fmt in ('json', 'csv'):
            mutate('verification.export', {'id': admitted['last_operation'], 'format': fmt, 'path': str(assets/f'split.{fmt}')})
        mutate('project.save', {'path': str(assets/'split.elab')})
        mutate('project.clear', {'discard_changes': True})
        mutate('project.load', {'path': str(assets/'split.elab')})
        reopened = client.ask('simulation.wait')
        assert abs(read('v:V2.negative')+3) < 1e-12 and reopened['document']['recording_count'] == 1
        mutate('simulation.pause')
        # Shift the same series circuit's reference from midpoint to bottom rail.
        mutate('object.delete', {'id': 'W5'})
        mutate('wire.connect', {'from': 'V2.negative', 'to': 'GND1.reference'})
        mutate('component.set', {'id': 'V1', 'value': '5 V'})
        series = restart()
        series_values = {p: read(p) for p in ('v:V1.positive', 'v:V1.negative', 'v:V1', 'i:R1')}
        assert abs(series_values['v:V1.positive']-8) < 1e-12
        assert abs(series_values['v:V1.negative']-3) < 1e-12
        assert abs(series_values['v:V1']-5) < 1e-12 and abs(series_values['i:R1']-.008) < 1e-12
        swept = mutate('sweep.start', {'settings': {'component_id': 'V1', 'parameter': 'voltage',
            'start': '-5 V', 'stop': '5 V', 'points': 11, 'probe_ids': ['v:V1.positive', 'v:V1', 'i:R1']}})
        sweep = complete('sweep.inspect', swept['dc_sweep']['selected_operation'])
        for point in sweep['result']['points']:
            assert abs(point['values'][0]-(3+point['input'])) < 1e-12
            assert abs(point['values'][1]-point['input']) < 1e-12
            assert abs(point['values'][2]-(3+point['input'])/1000) < 1e-12
            assert point['kcl_residual_a'] < 1e-9 and abs(point['power_residual_w']) < 1e-8
        assert client.ask('project.inspect')['mixed'] == series['mixed']
        mutate('project.clear', {'discard_changes': True})
        # Return-path resistor: source positive connects directly to capacitor,
        # but it is not a source/capacitor ideal constraint loop.
        build([('VR', 'dc-source', '5 V', -200, 0, 0), ('RR', 'resistor', '10k', 0, -110, 0),
               ('CR', 'capacitor', '1uF', 200, 0, 0), ('GR', 'ground', '', 0, 110, 0)],
              [('VR.positive', 'CR.1'), ('CR.2', 'GR.reference'), ('VR.negative', 'RR.1'), ('RR.2', 'GR.reference')])
        restart()
        rc = []
        for sample in range(1, 6):
            mutate('simulation.step', {'duration_s': '10 ms'})
            state = client.ask('simulation.wait')
            got, expected = read('v:CR'), 5 * -math.expm1(-sample)
            assert abs(got-expected) < .003 and state['mixed']['kcl_residual_a'] < 1e-9
            assert abs(read('v:VR.negative')-(got-5)) < 1e-9
            rc.append({'time_s': sample*.01, 'voltage_v': got, 'analytic_v': expected})
        charge_end = got
        mutate('simulation.input.set', {'mode': 'low'})
        switched = client.ask('simulation.wait')
        assert abs(switched['transient']['sample']['time_s']-.05) < 1e-12
        assert abs(read('v:CR')-charge_end) < 1e-12
        discharge = []
        for sample in range(1, 6):
            mutate('simulation.step', {'duration_s': '10 ms'})
            client.ask('simulation.wait')
            got, expected = read('v:CR'), charge_end*math.exp(-sample)
            assert abs(got-expected) < .003
            discharge.append({'time_s': .05+sample*.01, 'voltage_v': got, 'analytic_v': expected})
        report = dict(passed=True, transport='desktop-socket' if args.connect else 'headless-stdio',
                      discovery=discovery, dc_part=part,
                      observations=dict(split=split, source_results=initial['mixed']['sources'], series=series_values,
                          sweep=sweep['result'], rc_return=rc, rc_discharge=discharge,
                          captured_current_a=captured['metadata']['mixed']['resistors'][0]['current_a']),
                      checks=['blank placement/wiring matches guided split supply', 'signed rail and differential source readings',
                              'Kirchhoff/Ohm current and power signs', 'capture JSON/CSV and saved recording reopen',
                              'ground reference moved to bottom; source values add', 'independent DC sweep preserves live result',
                              'return-path RC matches analytic 10 ms charge/discharge within 3 mV; switching preserves time and voltage'],
                      limits=['DC sources only may float; waveform negatives and NPN emitters remain grounded',
                              'Existing 3-source, node-voltage, integration and Newton limits remain; no physical battery or rating claim'])
        with args.output.open('x') as out:
            out.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({'passed': True, 'transport': report['transport'], 'observations': report['observations']}, indent=2))
    finally:
        if args.connect:
            client.ask('app.close', {'discard_changes': True}, True)
        client.close()


if __name__ == '__main__':
    main()
