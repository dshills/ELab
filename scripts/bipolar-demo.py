#!/usr/bin/env python3
"""Check bipolar examples, electrical references and file round trips over stdio."""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import subprocess
import time

spec = importlib.util.spec_from_file_location(
    'viewport_client', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def follower_reference(input_voltage):
    """Independent current-domain load line; reverse transport is <20 fA."""
    vt = 1.380649e-23 * 300.15 / 1.602176634e-19
    low, high = 0., input_voltage / 1000
    for _ in range(90):
        current = (low + high) / 2
        drop = 1000 * current + vt * math.log1p(current / 1.01e-14)
        drop += 1000 * current / 101
        if drop > input_voltage:
            high = current
        else:
            low = current
    return 1000 * (low + high) / 2


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--agent', type=Path, default=helper.ROOT / 'build/elab-agent')
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
    for example in ('pnp', 'emitter-follower', 'bipolar-push-pull'):
        client, fresh = Client(), None
        mutate = lambda command, params=None: client.ask(command, params, True)
        try:
            parts = client.ask('parts.list')
            assert any(part['kind'] == 'pnp' for part in parts)
            catalog = client.ask('examples.list')
            assert any(item['id'] == example for item in catalog)
            mutate('example.load', {'id': example})
            mutate('simulation.pause')
            mutate('simulation.restart')
            initial = client.ask('simulation.wait')
            assert initial['mixed'] and not initial['document']['dirty']
            assert initial['example']['info']['id'] == example
            probes = (['v:VIN.positive', 'i:D1', 'v:Q1.base-emitter', 'i:Q1.collector']
                if example == 'pnp' else
                ['v:VIN.positive', 'v:Q1.emitter', 'i:Q1.base', 'i:Q1.collector'])
            if example == 'bipolar-push-pull':
                probes = ['v:VIN.positive', 'v:Q1.emitter', 'i:Q1.collector', 'i:Q2.collector']
            mutate('simulation.verify', {'duration_s': .1, 'sample_step_s': .001, 'probe_ids': probes})
            operation = client.ask('project.inspect')['last_operation']
            deadline = time.monotonic() + 10
            while True:
                view = client.ask('operation.inspect', {'id': operation})
                if view['operation']['status'] not in ('queued', 'running'):
                    break
                assert time.monotonic() < deadline, 'capture timeout'
                time.sleep(.01)
            assert view['operation']['status'] == 'completed', view
            capture_path = assets / (example + '-capture.json')
            mutate('verification.export', {'id': operation, 'format': 'json', 'path': str(capture_path)})
            capture = json.loads(capture_path.read_text())
            errors = []
            for row in capture['rows']:
                vin, output = row['values'][:2]
                if example == 'pnp':
                    if vin == 0:
                        assert output > .0028 and row['values'][3] < -.0028
                    else:
                        assert output < 1e-9 and abs(row['values'][3]) < 1e-9
                else:
                    expected = math.copysign(follower_reference(abs(vin)), vin)
                    errors.append(abs(output - expected))
            if errors:
                assert max(errors) < 2e-6
            mutate('simulation.step', {'duration_s': .075})
            live = client.ask('project.inspect')
            assert abs(live['transient']['sample']['time_s'] - .075) < 1e-12
            assert live['mixed']['kcl_residual_a'] < 1e-9
            assert abs(live['mixed']['power_residual_w']) < 1e-8
            path = assets / (example + '.elab')
            mutate('project.save', {'path': str(path)})
            fresh = Client()
            fresh.ask('project.load', {'path': str(path)}, True)
            reopened = fresh.ask('simulation.wait')
            assert reopened['components'] == live['components']
            assert reopened['wires'] == live['wires']
            assert reopened['document']['recording_count'] == 1
            assert not reopened['document']['dirty']
            assert reopened['transient']['sample']['time_s'] == 0
            exports = {}
            for fmt in ('svg', 'png', 'pdf'):
                output = assets / (example + '.' + fmt)
                mutate('export.schematic.' + fmt, {'path': str(output)})
                exports[fmt] = hashlib.sha256(output.read_bytes()).hexdigest()
            reports.append(dict(example=example, captured_rows=len(capture['rows']),
                maximum_follower_reference_error_v=max(errors, default=0),
                final_transistors=live['mixed']['transistors'], exports=exports))
        finally:
            if fresh:
                fresh.close()
            client.close()
    report = dict(passed=True, transport='headless-stdio', examples=reports,
        scope='Public command discovery, guided loading, live stepping, capture, independent follower load lines, signed PNP switching, save/reopen and SVG/PNG/PDF exports.',
        limitations=['Educational model references, not measured hardware or a separate SPICE execution.',
            'Native pointer/keyboard acceptance is checked separately.'])
    with args.output.open('x') as output:
        output.write(json.dumps(report, indent=2) + '\n')
    print(json.dumps(dict(passed=True, examples=[entry['example'] for entry in reports])))


if __name__ == '__main__':
    main()
