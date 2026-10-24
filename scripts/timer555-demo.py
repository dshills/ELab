#!/usr/bin/env python3
"""Replay the 555 examples and RC timing references through public agent commands."""
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


def astable_reference(rb):
    phases = []
    for discharge in (1e-9, .1):
        ga, gb = 1/10000, 1/rb
        conductance = gb*(ga+discharge)/(ga+gb+discharge)+2e-9
        target = (5*ga*gb/(ga+gb+discharge))/conductance
        phases.append((target, 10e-6/conductance))
    (up, tu), (down, td) = phases
    first = tu*math.log(up/(up-10/3))
    high = tu*math.log((up-5/3)/(up-10/3))
    low = td*math.log((10/3-down)/(5/3-down))
    return first, high+low


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
    for example in ('555-flasher', '555-oscillator', '555-one-shot'):
        client, fresh = Client(), None
        mutate = lambda command, params=None: client.ask(command, params, True)
        try:
            part = next(p for p in client.ask('parts.list') if p['kind'] == 'timer555')
            assert part['terminals'] == ['gnd', 'trigger', 'out', 'reset',
                'control', 'threshold', 'discharge', 'vcc']
            assert any(item['id'] == example for item in client.ask('examples.list'))
            mutate('example.load', {'id': example})
            mutate('simulation.pause')
            mutate('simulation.restart')
            initial = client.ask('simulation.wait')
            assert not initial['document']['dirty']
            assert initial['timer555_model']['id'] == part['model']
            assert initial['mixed']['timers555'][0]['high']
            probes = ['v:U1', 'v:C1', 'i:D1', 'i:U1']
            mutate('simulation.verify', {'duration_s': 1.6,
                'sample_step_s': .005, 'probe_ids': probes})
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
            assert capture['metadata']['engine'] == 'elab.timer555-hybrid.v1'
            falls = [right['time_s'] for left, right in zip(capture['rows'], capture['rows'][1:])
                if left['time_s'] == right['time_s'] and left['values'][0] > 2.5
                and right['values'][0] < 2.5]
            if example == '555-one-shot':
                g = 1/47000+2e-9
                target, tau = (5/47000)/g, 10e-6/g
                expected = tau*math.log(target/(target-10/3))
                errors = [abs(falls[0]-expected)]
            else:
                first, period = astable_reference(47000 if example == '555-flasher' else 50000)
                assert len(falls) >= 2
                errors = [abs(at-first-index*period) for index, at in enumerate(falls)]
            assert max(errors) < 1e-5, (falls, errors)
            mutate('simulation.step', {'duration_s': .3})
            live = client.ask('project.inspect')
            assert live['mixed']['timers555'][0]['high']
            assert live['mixed']['kcl_residual_a'] < 1e-9
            assert abs(live['mixed']['power_residual_w']) < 1e-8
            if example == '555-oscillator':
                charge = live['mixed']['capacitors'][0]['voltage_v']
                mutate('potentiometer.wiper.set', {'id': 'P1', 'position': .8})
                adjusted = client.ask('simulation.wait')
                assert adjusted['mixed']['timers555'][0]['high']
                assert abs(adjusted['mixed']['capacitors'][0]['voltage_v']-charge) < 1e-10
                assert adjusted['transient']['sample']['time_s'] == live['transient']['sample']['time_s']
                live = adjusted
            path = assets / (example + '.elab')
            mutate('project.save', {'path': str(path)})
            fresh = Client()
            fresh.ask('project.load', {'path': str(path)}, True)
            reopened = fresh.ask('simulation.wait')
            assert reopened['components'] == live['components']
            assert reopened['wires'] == live['wires']
            assert reopened['document']['recording_count'] == 1
            assert not reopened['document']['dirty']
            assert reopened['mixed']['timers555'][0]['high']
            exports = {}
            for fmt in ('svg', 'png', 'pdf'):
                output = assets / (example + '.' + fmt)
                mutate('export.schematic.' + fmt, {'path': str(output)})
                exports[fmt] = hashlib.sha256(output.read_bytes()).hexdigest()
            reports.append(dict(example=example, frames=len(capture['rows']),
                falling_edges_s=falls, maximum_rc_reference_error_s=max(errors), exports=exports))
        finally:
            if fresh:
                fresh.close()
            client.close()
    report = dict(passed=True, transport='headless-stdio', examples=reports,
        scope='Discovery, native-shared commands, live step, retained timer edge captures, independent RC timing, live wiper charge/latch preservation, saved recordings and schematic exports.',
        limitations=['Generic educational model; no fitted TLC555, measured hardware or independent SPICE execution.',
            'Native pointer and keyboard observations are separate.'])
    with args.output.open('x') as output:
        output.write(json.dumps(report, indent=2)+'\n')
    print(json.dumps(dict(passed=True, examples=[entry['example'] for entry in reports])))


if __name__ == '__main__':
    main()
