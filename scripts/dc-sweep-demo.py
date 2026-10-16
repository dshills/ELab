#!/usr/bin/env python3
"""Exercise DC sweeps through an owned stdio or native-socket session."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    cases = []

    def completed():
        deadline = time.monotonic() + 15
        while time.monotonic() < deadline:
            state = client.ask('project.inspect')
            sweep = state['dc_sweep']
            op = next(op for op in sweep['operations'] if op['id'] == sweep['selected_operation'])
            assert op['status'] not in ('failed', 'canceled'), op
            if op['status'] == 'completed':
                return state
            time.sleep(.01)
        raise AssertionError('DC sweep did not complete')

    def run(name, settings):
        before = client.ask('project.inspect')
        client.ask('sweep.start', {'settings': settings}, True)
        state = completed()
        for key in ('components', 'wires', 'run_id', 'electrical_revision', 'result', 'npn', 'mixed', 'transient', 'undo_entry'):
            assert before.get(key) == state.get(key), key
        sweep = state['dc_sweep']
        record = client.ask('sweep.inspect', {'id': sweep['selected_operation']})
        for point in record['result']['points']:
            assert point['kcl_residual_a'] < 1e-9 and abs(point['power_residual_w']) < 1e-8
        for format in ('json', 'csv'):
            client.ask('sweep.export', {'id': sweep['selected_operation'], 'path': str(directory / (name + '.' + format)), 'format': format}, True)
        client.ask('sweep.cursor.set', {'operation_id': sweep['selected_operation'], 'index': len(record['result']['points']) // 2}, True)
        project = directory / (name + '.elab')
        client.ask('project.save', {'path': str(project)}, True)
        assert json.loads(project.read_text())['version'] == 5
        client.ask('project.clear', {'discard_changes': True}, True)
        client.ask('project.load', {'path': str(project)}, True)
        restored = client.ask('project.inspect')
        assert restored['dc_sweep']['result'] == record['result'] and not restored['document']['dirty']
        assert restored['dc_sweep']['cursor_index'] == len(record['result']['points']) // 2
        cases.append({'case': name, 'settings': settings, 'result': record['result'],
                      'dc_sources': record['metadata']['dc_sources'],
                      'json_sha256': hashlib.sha256((directory / (name + '.json')).read_bytes()).hexdigest(),
                      'csv_sha256': hashlib.sha256((directory / (name + '.csv')).read_bytes()).hexdigest()})
        return record

    try:
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        record = run('led-voltage', {'component_id': 'V1', 'parameter': 'voltage', 'start': '0 V', 'stop': '5 V', 'points': 11, 'probe_ids': ['i:D1', 'v:D1']})
        currents = [p['values'][0] for p in record['result']['points']]
        assert currents[0] == 0 and all(b >= a for a, b in zip(currents, currents[1:]))
        assert abs(currents[-1] - .00291679) < 2e-8

        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        record = run('rc-steady-state', {'component_id': 'V1', 'parameter': 'voltage', 'start': '-5 V', 'stop': '5 V', 'points': 11, 'probe_ids': ['v:C1', 'i:C1']})
        for point in record['result']['points']:
            assert abs(point['values'][0] - point['input']) < 1e-10 and point['values'][1] == 0

        client.ask('example.npn', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        state = client.ask('simulation.wait')
        source = next(p for p in state['components'] if p['kind'] == 'dc-source' and p['id'] != 'VCC')
        run('npn-drive', {'component_id': source['id'], 'parameter': 'voltage', 'start': '0 V', 'stop': '5 V', 'points': 11, 'probe_ids': ['i:Q1.base', 'i:Q1.collector', 'v:Q1.collector']})
        client.ask('project.clear', {'discard_changes': True}, True)
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio', 'cases': cases,
                  'checks': ['independent captured sweeps', 'unchanged live physics/history', 'signed DC/open capacitor semantics',
                             'KCL/power residuals', 'complete JSON/SI CSV', 'version-5 save/reopen and exact point cursor'],
                  'limits': ['Configured source DC biases, not transient phases or held inputs', 'Existing generic models', '2–201 linear independent points; no nested/log/continuation sweep']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as target:
            target.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps({'passed': True, 'transport': report['transport'], 'cases': len(cases)}))
    finally:
        client.close()


if __name__ == '__main__':
    main()
