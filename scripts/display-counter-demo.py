#!/usr/bin/env python3
"""Exercise a real 555/counter/decoder/display through the shared public API."""
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

    client, fresh = Client(), None
    mutate = lambda command, params=None: client.ask(command, params, True)
    try:
        catalog = client.ask('parts.list')
        assert all(any(p['kind'] == kind for p in catalog) for kind in ('bcd-decoder', 'seven-segment'))
        mutate('example.load', {'id': 'display-counter'})
        mutate('simulation.pause')
        mutate('simulation.restart')
        state = client.ask('simulation.wait')
        assert not state['document']['dirty'] and state['display_model']['id'] == 'elab.generic-seven-segment.v1'
        seen = set()
        previous, changes = 0, 0
        expected = ('abcdef', 'bc', 'abdeg', 'abcdg', 'bcfg', 'acdfg', 'acdefg', 'abc', 'abcdefg', 'abcdfg')
        for _ in range(100):
            mutate('simulation.step', {'duration_s': .1})
            state = client.ask('simulation.wait')
            display = next(d for d in state['mixed']['displays'] if d['id'] == 'DS1')
            count = state['mixed']['sequential'][0]['memory']['value']
            assert display['digit'] == count
            assert ''.join(c for c, lit in zip('abcdefg', display['lit']) if lit) == expected[count]
            assert abs(sum(display['segment_currents_a']) + display['pin_currents_a'][12]) < 1e-9
            assert state['mixed']['kcl_residual_a'] < 1e-8 and abs(state['mixed']['power_residual_w']) < 1e-7
            seen.add(count)
            if count != previous:
                assert count == (previous + 1) % 10
                changes += 1
                previous = count
        assert len(seen) == 10 and changes >= 10
        mutate('simulation.verify', {'duration_s': .1, 'sample_step_s': .005, 'probe_ids': ['v:C1.1', 'v:U1', 'i:DS1.a', 'i:DS1.g']})
        operation = client.ask('project.inspect')['last_operation']
        deadline = time.monotonic() + 20
        while True:
            view = client.ask('operation.inspect', {'id': operation})
            if view['operation']['status'] not in ('queued', 'running'):
                break
            assert time.monotonic() < deadline
            time.sleep(.01)
        assert view['operation']['status'] == 'completed', view
        mutate('verification.export', {'id': operation, 'format': 'json', 'path': str(assets / 'capture.json')})
        path = assets / 'counter.elab'
        mutate('project.save', {'path': str(path)})
        fresh = Client()
        fresh.ask('project.load', {'path': str(path)}, True)
        restored = fresh.ask('simulation.wait')
        assert restored['document']['recording_count'] == 1 and not restored['document']['dirty']
        assert restored['display_model'] == state['display_model']
        exports = {}
        for fmt in ('svg', 'png', 'pdf'):
            output = assets / ('counter.' + fmt)
            mutate('export.schematic.' + fmt, {'path': str(output)})
            exports[fmt] = hashlib.sha256(output.read_bytes()).hexdigest()
        report = dict(passed=True, transport='headless-stdio', example='display-counter', digits=sorted(seen),
                      observed_increments=changes, sampled_live_seconds=state['mixed']['time_s'],
                      verified=['Discovery', 'real 555 clock', 'all digits and segment currents', 'counter wrap',
                                'KCL/power', 'capture', 'kept-record save/reopen', 'SVG/PNG/PDF'], exports=exports)
        args.output.write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps(report))
    finally:
        if fresh:
            fresh.close()
        client.close()


if __name__ == '__main__':
    main()
