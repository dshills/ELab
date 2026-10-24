#!/usr/bin/env python3
"""Replay logic discovery, truth tables, captures and persistence through stdio."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import subprocess
import time

spec = importlib.util.spec_from_file_location('viewport_client', Path(__file__).with_name('viewport-fit-demo.py'))
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
            self.process = subprocess.Popen([str(args.agent.resolve())], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
            self.sequence = 0

    reports = []
    for example in ('logic-inverter', 'logic-switches', 'logic-half-adder'):
        client, fresh = Client(), None
        mutate = lambda command, params=None: client.ask(command, params, True)
        try:
            gates = [p for p in client.ask('parts.list') if p['kind'].startswith('logic-')]
            assert {p['kind'] for p in gates} == {'logic-' + k for k in ('not', 'and', 'or', 'nand', 'nor', 'xor', 'xnor')}
            assert all(p['value_type'] == 'none' and p['model'] == 'elab.generic-cmos-logic.v1' for p in gates)
            assert any(item['id'] == example for item in client.ask('examples.list'))
            mutate('example.load', {'id': example})
            mutate('simulation.pause')
            mutate('simulation.restart')
            initial = client.ask('simulation.wait')
            assert initial['logic_model']['id'] == 'elab.generic-cmos-logic.v1'
            assert not initial['document']['dirty']
            manual = []
            if example == 'logic-switches':
                for a, b in ((False, False), (True, False), (False, True), (True, True)):
                    mutate('switch.set', {'id': 'SA', 'closed': a})
                    mutate('switch.set', {'id': 'SB', 'closed': b})
                    # Existing paused DC edits recompute on restart/resume.
                    mutate('simulation.restart')
                    state = client.ask('simulation.wait')
                    u = state['mixed']['logic_gates'][0]
                    assert u['state'] == ('high' if a and b else 'low'), u
                    manual.append(dict(a=a, b=b, output=u['state']))
                probes = ['v:U1.a', 'v:U1.b', 'v:U1', 'i:U1']
                verify = {'probe_ids': probes}
            else:
                probes = ['v:U1.a', 'v:U1', 'i:D1']
                if example == 'logic-half-adder':
                    probes = ['v:U1.a', 'v:U1.b', 'v:U1', 'v:U2']
                verify = {'duration_s': 1.6, 'sample_step_s': .005, 'probe_ids': probes}
            mutate('simulation.verify', verify)
            operation = client.ask('project.inspect')['last_operation']
            deadline = time.monotonic() + 20
            while True:
                view = client.ask('operation.inspect', {'id': operation})
                if view['operation']['status'] not in ('queued', 'running'):
                    break
                assert time.monotonic() < deadline, 'capture timeout'
                time.sleep(.01)
            assert view['operation']['status'] == 'completed', view
            output = assets / (example + '-capture.json')
            mutate('verification.export', {'id': operation, 'format': 'json', 'path': str(output)})
            capture = json.loads(output.read_text())
            assert capture['metadata']['logic_model']['id'] == 'elab.generic-cmos-logic.v1'
            checked, combinations = 0, set()
            if example != 'logic-switches':
                for row in capture['rows']:
                    v = row['values']
                    if any(1.5 < value < 3.5 for value in v[:2 if example == 'logic-half-adder' else 1]):
                        continue
                    a = v[0] >= 3.5
                    if example == 'logic-half-adder':
                        b = v[1] >= 3.5
                        assert (v[2] >= 3.5) == (a != b), row
                        assert (v[3] >= 3.5) == (a and b), row
                        combinations.add((a, b))
                    else:
                        assert (v[1] >= 3.5) != a, row
                    checked += 1
                assert checked > 200
                if example == 'logic-half-adder':
                    assert len(combinations) == 4, combinations
                mutate('simulation.step', {'duration_s': .3})
            live = client.ask('simulation.wait')
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
            exports = {}
            for fmt in ('svg', 'png', 'pdf'):
                output = assets / (example + '.' + fmt)
                mutate('export.schematic.' + fmt, {'path': str(output)})
                exports[fmt] = hashlib.sha256(output.read_bytes()).hexdigest()
            reports.append(dict(example=example, frames=len(capture['rows']), valid_level_truth_rows=checked,
                                half_adder_combinations=len(combinations), manual_truth_table=manual, exports=exports))
        finally:
            if fresh:
                fresh.close()
            client.close()
    report = dict(passed=True, transport='headless-stdio', examples=reports,
                  scope='Catalog, shared commands, manual AND truth table, inverter and half-adder transient truth, current/power conservation, kept-record save/reopen and SVG/PNG/PDF export.',
                  limitations=['Generic educational gates; no fitted IC, hardware timing or independent SPICE execution.',
                               'Intermediate input voltages do not count as valid digital levels.',
                               'Native pointer and keyboard observations are separate.'])
    with args.output.open('x') as output:
        output.write(json.dumps(report, indent=2) + '\n')
    print(json.dumps(dict(passed=True, examples=[entry['example'] for entry in reports])))


if __name__ == '__main__':
    main()
