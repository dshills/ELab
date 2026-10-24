#!/usr/bin/env python3
"""Exercise Zener/Schottky discovery, simulation and files over public stdio."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import time

spec = importlib.util.spec_from_file_location(
    'viewport_client', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--agent', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)

    class Client(helper.Client):
        def __init__(self):
            self.process = subprocess.Popen([str(args.agent.resolve())],
                stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
            self.sequence = 0

    reports = []
    for example in ('zener', 'schottky'):
        client, fresh = Client(), None
        mutate = lambda command, params=None: client.ask(command, params, True)
        try:
            parts = {part['kind']: part for part in client.ask('parts.list')}
            assert {'zener', 'schottky'} <= parts.keys()
            assert parts['zener']['value_domain']['parameter'] == 'breakdown_voltage'
            assert parts['zener']['terminals'] == ['anode', 'cathode']
            assert parts['schottky']['terminals'] == ['anode', 'cathode']
            assert any(item['id'] == example for item in client.ask('examples.list'))
            mutate('example.load', {'id': example})
            mutate('simulation.pause')
            mutate('simulation.restart')
            initial = client.ask('simulation.wait')
            assert initial['mixed'] and not initial['document']['dirty']
            assert initial[example + '_model']['id'] == parts[example]['model']
            probes = (['v:V1.positive', 'v:RL', 'i:D1'] if example == 'zener'
                else ['v:C1', 'v:C2', 'i:D1', 'i:D2'])
            mutate('simulation.verify', {'duration_s': .1,
                'sample_step_s': .001, 'probe_ids': probes})
            operation = client.ask('project.inspect')['last_operation']
            deadline = time.monotonic() + 10
            while True:
                view = client.ask('operation.inspect', {'id': operation})
                if view['operation']['status'] not in ('queued', 'running'):
                    break
                assert time.monotonic() < deadline, 'capture timeout'
                time.sleep(.01)
            assert view['operation']['status'] == 'completed', view
            captured = assets / (example + '-capture.json')
            mutate('verification.export', {'id': operation,
                'format': 'json', 'path': str(captured)})
            capture = json.loads(captured.read_text())
            assert capture['metadata'][example + '_model'] == initial[example + '_model']
            errors = []
            if example == 'zener':
                for row in capture['rows']:
                    vin, output, current = row['values']
                    expected = (vin/470 + 5.1/10 - 1e-9)/(1/470 + 1/4700 + .1)
                    errors.append(abs(output - expected))
                    assert abs(output - expected) < 2e-7
                    assert current < 0
                    assert abs(current + (output-5.1)/10 + 1e-9) < 1e-10
            else:
                peak = min(capture['rows'], key=lambda row: abs(row['time_s']-.025))
                assert peak['values'][0] > peak['values'][1]+.2
                reverse = min(capture['rows'], key=lambda row: abs(row['time_s']-.075))
                assert abs(reverse['values'][2]+2e-7) < 1e-12
                assert abs(reverse['values'][3]+1e-9) < 1e-12
            mutate('simulation.step', {'duration_s': .075})
            live = client.ask('project.inspect')
            assert abs(live['transient']['sample']['time_s']-.075) < 1e-12
            assert live['mixed']['kcl_residual_a'] < 1e-9
            assert abs(live['mixed']['power_residual_w']) < 1e-8
            path = assets / (example + '.elab')
            mutate('project.save', {'path': str(path)})
            fresh = Client()
            fresh.ask('project.load', {'path': str(path)}, True)
            reopened = fresh.ask('simulation.wait')
            assert reopened['components'] == live['components']
            assert reopened['wires'] == live['wires']
            assert reopened[example + '_model'] == live[example + '_model']
            assert reopened['document']['recording_count'] == 1
            assert not reopened['document']['dirty']
            exports = {}
            for fmt in ('svg', 'png', 'pdf'):
                output = assets / (example + '.' + fmt)
                mutate('export.schematic.' + fmt, {'path': str(output)})
                exports[fmt] = hashlib.sha256(output.read_bytes()).hexdigest()
            if example == 'zener':
                mutate('component.set', {'id': 'D1', 'value': '3.3 V'})
                mutate('simulation.restart')
                edited = client.ask('simulation.wait')
                assert abs(edited['components'][2]['value']-3.3) < 1e-12
                assert -3.5 < edited['mixed']['diodes'][0]['voltage_v'] < -3.3
            reports.append(dict(example=example, captured_rows=len(capture['rows']),
                maximum_regulator_reference_error_v=max(errors, default=0),
                final_diodes=live['mixed']['diodes'], exports=exports))
        finally:
            if fresh:
                fresh.close()
            client.close()
    report = dict(passed=True, transport='headless-stdio', examples=reports,
        scope='Public discovery, protected example loading, live step, capture, Zener line law, Schottky comparison/leakage, value edit, save/reopen and SVG/PNG/PDF exports.',
        limitations=['Educational equation references, not measured hardware or a separate SPICE execution.',
            'Native pointer and keyboard checks are separate.'])
    with args.output.open('x') as output:
        output.write(json.dumps(report, indent=2) + '\n')
    print(json.dumps(dict(passed=True, examples=[entry['example'] for entry in reports])))


if __name__ == '__main__':
    main()
