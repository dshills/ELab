#!/usr/bin/env python3
"""Exercise coils through the public API in a fresh owned headless/native session."""
import argparse
import csv
import hashlib
import importlib.util
import io
import json
import math
from pathlib import Path
import time

from validation_artifacts import discovery_bundle

spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)

ENGINE = 'elab.rlc-mna-radau5.v1'


def physics(state):
    return {key: state.get(key) for key in ('run_id', 'electrical_revision', 'mixed', 'transient')}


def fresh(state):
    assert state['analysis'] == 'RLC network transient' and state['mixed']['engine'] == ENGINE
    assert not state['mixed']['stale'] and state['mixed']['kcl_residual_a'] < 1e-9
    assert abs(state['mixed']['power_residual_w']) < 1e-8
    return state['mixed']['inductors'][0]


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native endpoint; closed afterward unless --keep-open')
    parser.add_argument('--keep-open', action='store_true')
    parser.add_argument('--artifacts', required=True, type=Path)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    assert not args.keep_open or args.connect
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client, reopened = connect_demo_client(helper.Client, args.connect), None
    mutate = lambda command, params=None: client.ask(command, params, True)
    try:
        assert not client.ask('project.inspect')['components'], 'requires a fresh owned blank session'
        part = next(p for p in client.ask('parts.list') if p['kind'] == 'inductor')
        assert part['model'] == 'ideal-inductor.v1' and part['terminals'] == ['1', '2'] and part['unit'] == 'H'
        assert part['value_domain']['minimum'] == 1e-6 and part['value_domain']['maximum'] == 100
        queries = {cmd: client.ask('capabilities.list', {'command': cmd}) for cmd in
                   ('component.place', 'inductor.initial.set', 'simulation.verify')}
        contracts = discovery_bundle(queries, assets/'discovery.json')
        numeric = queries['component.place']['capabilities'][0]['input_schema']['x-network-numerics']['inductor_numerics']
        assert numeric['engine'] == ENGINE
        startup = queries['inductor.initial.set']['capabilities'][0]['input_schema']['properties']['value']
        assert startup['x-numeric-domain'] == {'unit': 'A', 'minimum': -1, 'maximum': 1}

        # Construct the first circuit without templates or private graph APIs.
        for identity, kind, position, value in [
            ('VBUILD', 'dc-source', (-200, 0), '5 V'),
            ('RBUILD', 'resistor', (0, -100), '100 Ω'),
            ('LBUILD', 'inductor', (200, 0), '1 H'),
            ('GBUILD', 'ground', (0, 160), ''),
        ]:
            mutate('component.place', {'id': identity, 'kind': kind,
                                      'position': dict(zip(('x', 'y'), position)), 'value': value})
        mutate('component.rotate', {'id': 'LBUILD'})
        for one, two in [('VBUILD.positive', 'RBUILD.1'), ('RBUILD.2', 'LBUILD.1'),
                         ('LBUILD.2', 'GBUILD.reference'), ('VBUILD.negative', 'GBUILD.reference')]:
            mutate('wire.connect', {'from': one, 'to': two})
        client.ask('simulation.wait')
        mutate('simulation.pause')
        mutate('inductor.initial.set', {'id': 'LBUILD', 'value': '-20 mA'})
        mutate('simulation.restart')
        initial = client.ask('simulation.wait')
        assert fresh(initial)['current_a'] == -.02
        probes = ['i:LBUILD', 'v:LBUILD']
        mutate('scope.probes.set', {'ids': probes})
        mutate('simulation.step', {'duration_s': '5 ms'})
        before = client.ask('project.inspect')
        expected = .05 - .07*math.exp(-.5)
        assert abs(fresh(before)['current_a']-expected) < 1e-5
        mutate('component.move', {'id': 'LBUILD', 'position': {'x': 240, 'y': 20}})
        mutate('view.fit')
        assert physics(client.ask('project.inspect')) == physics(before)
        client.ask('inductor.initial.set', {'id': 'LBUILD', 'value': '1001 mA'}, True, 'invalid-params')
        assert physics(client.ask('project.inspect')) == physics(before)

        operation = mutate('simulation.verify', {'duration_s': '5 ms', 'sample_step_s': '1 ms',
                           'inductor_initials': [{'id': 'LBUILD', 'current_a': .03}],
                           'probe_ids': probes})['last_operation']
        deadline = time.monotonic()+20
        while True:
            view = client.ask('operation.inspect', {'id': operation})
            status = view['operation']['status']
            if status == 'completed':
                break
            assert status not in ('failed', 'canceled') and time.monotonic() < deadline, view
            time.sleep(.01)
        for fmt in ('json', 'csv'):
            mutate('verification.export', {'id': operation, 'format': fmt, 'path': str(assets/f'rl-capture.{fmt}')})
        capture = json.loads((assets/'rl-capture.json').read_bytes())
        assert capture['metadata']['engine'] == ENGINE and len(capture['rows']) == 6
        current_errors, voltage_errors = [], []
        for row in capture['rows']:
            expected = .05 - .02*math.exp(-row['time_s']/.01)
            current_errors.append(abs(row['values'][0]-expected))
            voltage_errors.append(abs(row['values'][1]-(5-100*expected)))
        assert max(current_errors) < 1e-5 and max(voltage_errors) < .001
        csv_rows = list(csv.DictReader(io.StringIO((assets/'rl-capture.csv').read_text())))
        assert len(csv_rows) == 6 and all(row['engine'] == ENGINE for row in csv_rows)
        assert 'i:LBUILD [A]' in csv_rows[0] and 'v:LBUILD [V]' in csv_rows[0]
        assert physics(client.ask('project.inspect')) == physics(before)

        path = assets/'saved-rl.elab'
        mutate('project.save', {'path': str(path)})
        saved_bytes = path.read_bytes()
        reopened = helper.Client(None)
        reopened.ask('project.load', {'path': str(path)}, True)
        loaded = reopened.ask('simulation.wait')
        assert loaded['paused'] and loaded['transient']['sample']['time_s'] == 0
        assert fresh(loaded)['current_a'] == -.02 and loaded['document']['recording_count'] == 1
        assert loaded['scope']['selected'] == probes and path.read_bytes() == saved_bytes
        reopened.close()
        reopened = None

        exports = {}
        for fmt in ('svg', 'png'):
            destination = assets/f'built-rl.{fmt}'
            mutate('export.schematic.'+fmt, {'path': str(destination)})
            exports[fmt] = hashlib.sha256(destination.read_bytes()).hexdigest()
        assert physics(client.ask('project.inspect')) == physics(before)

        examples = []
        for identity, duration in [('rl', .05), ('rlc', .005), ('flyback', .05)]:
            mutate('example.load', {'id': identity, 'discard_changes': True})
            client.ask('simulation.wait')
            mutate('simulation.pause')
            mutate('simulation.restart')
            client.ask('simulation.wait')
            mutate('simulation.step', {'duration_s': duration})
            state = client.ask('project.inspect')
            reading = fresh(state)
            if identity == 'flyback':
                assert state['mixed']['mosfets'][0]['state'] == 'cutoff' and state['mixed']['diodes'][0]['current_a'] > .045
            examples.append(dict(id=identity, time_s=state['transient']['sample']['time_s'], reading=reading))
            for fmt in ('svg', 'png'):
                mutate('export.schematic.'+fmt, {'path': str(assets/f'{identity}.{fmt}')})

        # Reproduce the original mixed-device failure with ordinary edits.
        mutate('object.delete', {'id': 'M1'})
        mutate('component.place', {'id': 'QCOIL', 'kind': 'npn', 'position': {'x': 200, 'y': 200}})
        for one, two in [('JDR.node', 'QCOIL.collector'), ('JG.node', 'QCOIL.base'), ('QCOIL.emitter', 'GND1.reference')]:
            mutate('wire.connect', {'from': one, 'to': two})
        mutate('component.set', {'id': 'RG', 'value': '10k Ω'})
        mutate('component.place', {'id': 'CB', 'kind': 'capacitor', 'position': {'x': 700, 'y': 200}, 'value': '1 µF'})
        mutate('wire.connect', {'from': 'CB.1', 'to': 'QCOIL.base'})
        mutate('wire.connect', {'from': 'CB.2', 'to': 'GND1.reference'})
        mutate('scope.probes.set', {'ids': ['i:L1', 'v:QCOIL.collector', 'v:CB', 'v:VIN.positive']})
        mutate('simulation.restart')
        client.ask('simulation.wait')
        mutate('simulation.step', {'duration_s': '120 ms'})
        mixed = client.ask('project.inspect')
        fresh(mixed)
        assert abs(mixed['transient']['sample']['time_s']-.12) < 1e-12
        assert len(mixed['mixed']['transistors']) == 1 and len(mixed['mixed']['capacitors']) == 1
        mutate('project.save', {'path': str(assets/'mixed-npn-coil.elab')})
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      engine=ENGINE, part=part, discovery=contracts, exports=exports, examples=examples,
                      maximum_rl_capture_current_error_a=max(current_errors),
                      maximum_rl_capture_voltage_error_v=max(voltage_errors),
                      captured_rows=len(capture['rows']), mixed_time_s=mixed['transient']['sample']['time_s'],
                      mixed_current_a=mixed['mixed']['inductors'][0]['current_a'],
                      checks=['blank place/wire/value/rotation', 'signed startup and domain refusal',
                              'live RL exponential and geometry isolation', 'independent Verify startup override',
                              'unit-bearing CSV and complete JSON', 'saved capture/startup fresh-process reopen',
                              'SVG/PNG output without physical mutation', 'all three guided examples',
                              'coupled RC/NPN/L full-period public workflow'],
                      limitations=['generic ideal coil and educational semiconductor models',
                                   'public/native API evidence is separate from physical input/accessibility and physical printing'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as out:
            out.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({key: report[key] for key in ('passed', 'transport', 'engine', 'captured_rows', 'mixed_time_s')}, indent=2))
    finally:
        if args.connect and not args.keep_open:
            client.ask('app.close', {'discard_changes': True}, True)
        client.close()
        if reopened:
            reopened.close()


if __name__ == '__main__':
    main()
