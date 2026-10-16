#!/usr/bin/env python3
"""Qualify two captured RC experiments through either public client."""
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
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)

    def capture(step):
        state = client.ask('simulation.verify', dict(duration_s='20 ms', sample_step_s=step,
                           input_mode='charge', probe_ids=['v:C1', 'i:R1']), True)
        identifier = state['last_operation']
        deadline = time.monotonic() + 10
        while True:
            view = client.ask('operation.inspect', {'id': identifier})
            assert view['operation']['status'] not in ('failed', 'canceled'), view
            if view['operation']['status'] == 'completed':
                return identifier
            assert time.monotonic() < deadline
            time.sleep(.01)

    def semantic(comparison):
        result = json.loads(json.dumps(comparison))
        for side in ('a', 'b'):
            for key in ('id', 'origin_id'):
                result[side].pop(key)
        result['settings'].pop('id')
        result['settings'].pop('reference_id')
        return result

    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        first = capture('1 ms')
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        second = capture('2 ms')
        params = dict(id=second, reference_id=first, time_s='11.4 ms')
        before = client.ask('project.inspect')
        result = client.ask('recording.compare', params)
        assert client.ask('project.inspect') == before
        assert result['available'] and result['a']['sample_index'] == 6 and result['b']['sample_index'] == 11
        voltage = next(r for r in result['readings'] if r['probe']['id'] == 'v:C1')
        assert math.isclose(voltage['a'], 5 * (1 - math.exp(-result['a']['time_s'] / .02)), abs_tol=1e-12)
        assert math.isclose(voltage['b'], 5 * (1 - math.exp(-result['b']['time_s'] / .01)), abs_tol=1e-12)
        assert voltage['b_minus_a'] == voltage['b'] - voltage['a']
        configured = client.ask('recording.compare.set', params, True)
        assert configured['recording']['comparison'] == result
        for key in ('components', 'wires', 'document', 'run_id', 'electrical_revision', 'transient', 'undo_entry'):
            assert configured.get(key) == before.get(key), key
        for bad, code in [(dict(params, time_s='21 ms'), 'invalid-params'),
                          (dict(params, reference_id=second), 'invalid-params'),
                          (dict(params, reference_id='missing'), 'not-ready')]:
            client.ask('recording.compare', bad, expect=code)
            assert client.ask('project.inspect') == configured
        client.ask('component.set', {'id': 'R1', 'value': '30k'}, True)
        assert client.ask('recording.compare', params) == result
        for identifier, name in ((first, '10k'), (second, '20k')):
            client.ask('verification.export', {'id': identifier, 'path': str(directory / (name + '.json')), 'format': 'json'}, True)
        client.ask('verification.import', {'path': str(directory / '10k.json')}, True)
        imported = client.ask('project.inspect')['recording']['id']
        imported_result = client.ask('recording.compare', dict(id=second, reference_id=imported, time_s='11.4 ms'))
        assert semantic(result) == semantic(imported_result)
        project = directory / 'compared-captures.elab'
        client.ask('project.save', {'path': str(project)}, True)
        client.ask('project.load', {'path': str(project)}, True)
        client.ask('simulation.wait')
        state = client.ask('project.inspect')
        assert 'comparison' not in state['recording']
        captures = []
        for operation in state['operations']:
            view = client.ask('operation.inspect', {'id': operation['id']})
            resistor = next(c for c in view['metadata']['graph']['components'] if c['id'] == 'R1')
            captures.append((resistor['value'], operation['id']))
        a = next(identifier for value, identifier in captures if value == 20000)
        b = next(identifier for value, identifier in captures if value == 10000)
        reopened = client.ask('recording.compare', dict(id=a, reference_id=b, time_s='11.4 ms'))
        assert semantic(reopened) == semantic(result)
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      comparison=semantic(result), checks=['Analytic RC results at exact independent sample times',
                      'Pure query and rejected requests preserve all public state', 'Configured view equals pure query without live/history/dirty changes',
                      'Later live edits do not change captured readings', 'Imported and reopened captures compare identically'],
                      limits=['This table workflow does not test overlay or time alignment', 'Sample statistics cover whole captures and depend on resolution'])
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps(dict(passed=True, transport=report['transport'])))
    finally:
        client.close()


if __name__ == '__main__':
    main()
