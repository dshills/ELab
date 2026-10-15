#!/usr/bin/env python3
"""Qualify logarithmic electronics experiments through either public client."""
import argparse
import importlib.util
import json
import math
from pathlib import Path
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect, command_preconditions=True)
    curves = []
    try:
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        for name, target, parameter, start, stop, points in [
            ('resistance', 'R1', 'resistance', '100 ohm', '100k ohm', 7),
            ('bias', 'V1', 'voltage', '0.05 V', '5 V', 11),
            ('descending', 'R1', 'resistance', '100k ohm', '100 ohm', 7),
        ]:
            settings = dict(component_id=target, parameter=parameter, start=start, stop=stop,
                            spacing='logarithmic', points=points, probe_ids=['i:D1', 'v:D1'])
            before = client.ask('project.inspect')
            client.ask('sweep.settings.set', settings, True)
            client.ask('sweep.start', {}, True)
            deadline = time.monotonic() + 10
            while True:
                state = client.ask('project.inspect')
                op = next(o for o in state['dc_sweep']['operations'] if o['id'] == state['dc_sweep']['selected_operation'])
                assert op['status'] not in ('failed', 'canceled'), op
                if op['status'] == 'completed':
                    break
                assert time.monotonic() < deadline
                time.sleep(.01)
            for key in ('components', 'wires', 'result', 'run_id', 'electrical_revision'):
                assert state[key] == before[key], key
            curve = client.ask('sweep.inspect', {'id': op['id']})
            result = curve['result']
            assert result['spacing'] == curve['metadata']['settings']['spacing'] == 'logarithmic'
            a, b = result['start_si'], result['stop_si']
            for index, point in enumerate(result['points']):
                expected = a * (b / a) ** (index / (points - 1))
                assert math.isclose(point['input'], expected, rel_tol=3e-15)
            client.ask('sweep.cursor.set', {'index': points // 2, 'operation_id': op['id']}, True)
            assert client.ask('project.inspect')['dc_sweep']['cursor_index'] == points // 2
            for fmt in ('json', 'csv'):
                client.ask('sweep.export', {'id': op['id'], 'path': str(directory / (name + '.' + fmt)), 'format': fmt}, True)
            client.ask('sweep.import', {'path': str(directory / (name + '.json'))}, True)
            imported = client.ask('project.inspect')['dc_sweep']['result']
            assert imported == result
            curves.append(result)
        project = directory / 'log-sweeps.elab'
        client.ask('project.save', {'path': str(project)}, True)
        client.ask('project.load', {'path': str(project)}, True)
        client.ask('simulation.wait')
        assert client.ask('project.inspect')['dc_sweep']['result'] == curves[-1]
        before = client.ask('project.inspect')
        bad = dict(settings, component_id='V1', parameter='voltage', start='0 V', stop='5 V')
        client.ask('sweep.settings.set', bad, True, 'invalid-params')
        assert client.ask('project.inspect') == before
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', curves=curves,
                      checks=['exact geometric axes including descending', 'live source and physics unchanged',
                              'captured spacing and measured cursor available through shared commands',
                              'JSON/import/project round trips preserve exact curves', 'invalid settings preserve all public state'],
                      limits=['Generic DC models and solver tolerances unchanged', 'Not AC analysis or real-device certification'])
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps(dict(passed=True, cases=len(curves), transport=report['transport'])))
    finally:
        client.close()


if __name__ == '__main__':
    main()
