#!/usr/bin/env python3
"""Check sequential state, captures and persistence through public agent commands."""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import subprocess
import time

spec = importlib.util.spec_from_file_location('viewport_client', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
KINDS = ('d-flipflop', 'jk-flipflop', 't-flipflop', 'binary-counter', 'decade-counter')


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
            self.process = subprocess.Popen([str(args.agent.resolve())], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
            self.sequence = 0

    reports = []
    for kind in KINDS:
        client, fresh = Client(), None
        mutate = lambda command, params=None: client.ask(command, params, True)
        try:
            part = next(p for p in client.ask('parts.list') if p['kind'] == kind)
            assert part['model'] == 'elab.generic-sequential-cmos.v1' and part['value_type'] == 'none'
            mutate('example.load', {'id': kind})
            mutate('simulation.pause')
            mutate('simulation.restart')
            initial = client.ask('simulation.wait')
            assert initial['mixed']['sequential'][0]['memory']['value'] == 0
            assert not initial['document']['dirty']
            counter = kind.endswith('counter')
            probes = ['v:U1.clk', 'v:U1.q', 'v:U1.nq']
            if counter:
                probes = ['v:U1.clk'] + ['v:U1.q' + str(k) for k in range(4)]
            duration, sample_step = (3.3, .004) if counter else (1.8, .002)
            mutate('simulation.verify', {'duration_s': duration, 'sample_step_s': sample_step, 'probe_ids': probes})
            operation = client.ask('project.inspect')['last_operation']
            deadline = time.monotonic() + 30
            while True:
                view = client.ask('operation.inspect', {'id': operation})
                if view['operation']['status'] not in ('queued', 'running'):
                    break
                assert time.monotonic() < deadline, 'capture timeout'
                time.sleep(.01)
            assert view['operation']['status'] == 'completed', view
            output = assets / (kind + '-capture.json')
            mutate('verification.export', {'id': operation, 'format': 'json', 'path': str(output)})
            capture = json.loads(output.read_text())
            assert capture['metadata']['engine'] == 'elab.sequential-hybrid.v1'
            assert capture['metadata']['sequential_model']['id'] == part['model']
            seen, checked = set(), 0
            for row in capture['rows']:
                at, values = row['time_s'], row['values']
                near_edge = abs((at - .014) / .2 - round((at - .014) / .2)) < 1e-6
                if near_edge:
                    continue  # Distinct left/right event rows share this time.
                edges = 0 if at < .014 else math.floor((at - .014) / .2) + 1
                if counter:
                    value = sum((1 << k) for k, v in enumerate(values[1:]) if v > 3.5)
                    expected = edges % (16 if kind == 'binary-counter' else 10)
                else:
                    value = int(values[1] > 3.5)
                    assert int(values[2] > 3.5) == 1 - value
                    expected = edges % 2
                    if kind == 'd-flipflop':
                        edge_time = .014 + (edges - 1) * .2
                        expected = int(edges > 0 and edge_time % .6 < .3)
                assert value == expected, (kind, at, value, expected)
                checked += 1
                seen.add(value)
            assert checked > 800
            assert len(seen) == (16 if kind == 'binary-counter' else 10 if counter else 2)
            mutate('simulation.step', {'duration_s': .05})
            live = client.ask('simulation.wait')
            assert live['mixed']['sequential'][0]['memory']['value'] == 1
            if counter:
                mutate('switch.set', {'id': 'SENABLE', 'closed': False})
                mutate('simulation.step', {'duration_s': .2})
                assert client.ask('simulation.wait')['mixed']['sequential'][0]['memory']['value'] == 1
            mutate('switch.set', {'id': 'SRESET', 'closed': True})
            mutate('simulation.step', {'duration_s': .002})
            reset = client.ask('simulation.wait')
            assert reset['mixed']['sequential'][0]['memory']['value'] == 0
            assert reset['mixed']['kcl_residual_a'] < 1e-8
            assert abs(reset['mixed']['power_residual_w']) < 1e-8
            path = assets / (kind + '.elab')
            mutate('project.save', {'path': str(path)})
            fresh = Client()
            fresh.ask('project.load', {'path': str(path)}, True)
            reopened = fresh.ask('simulation.wait')
            assert reopened['components'] == reset['components'] and reopened['wires'] == reset['wires']
            assert reopened['document']['recording_count'] == 1 and not reopened['document']['dirty']
            assert reopened['mixed']['sequential'][0]['memory']['value'] == 0
            exports = {}
            for fmt in ('svg', 'png', 'pdf'):
                output = assets / (kind + '.' + fmt)
                mutate('export.schematic.' + fmt, {'path': str(output)})
                exports[fmt] = hashlib.sha256(output.read_bytes()).hexdigest()
            reports.append(dict(example=kind, captured_rows=len(capture['rows']), checked_state_rows=checked,
                                distinct_values=sorted(seen), live_enable_hold=counter, live_reset=True, exports=exports))
        finally:
            if fresh:
                fresh.close()
            client.close()
    report = dict(passed=True, transport='headless-stdio', examples=reports,
                  scope='Discovery, independent rising-edge sequences and wrap, complementary outputs, captures, enable/reset, power/current balance, kept-record save/reopen and schematic exports.',
                  limitations=['Functional educational devices; no measured hardware timing, setup/hold or metastability qualification.',
                               'Native input and common-clock/feedback/stored-energy tests are recorded separately.'])
    with args.output.open('x') as output:
        output.write(json.dumps(report, indent=2) + '\n')
    print(json.dumps(dict(passed=True, examples=[entry['example'] for entry in reports])))


if __name__ == '__main__':
    main()
