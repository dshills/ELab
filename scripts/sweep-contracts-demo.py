#!/usr/bin/env python3
"""Use discovered DC sweep inputs and verify cursor clear through public commands."""
import argparse
import copy
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
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    try:
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        before = client.ask('simulation.wait')
        full = client.ask('capabilities.list')
        ids = [c['id'] for c in full['capabilities'] if c['id'].startswith('sweep.')]
        details = {}
        for command in ids:
            found = client.ask('capabilities.list', {'command': command})
            assert found['schema_version'] == full['schema_version']
            cap = found['capabilities'][0]
            assert len(found['capabilities']) == 1 and cap['id'] == command
            assert cap['input_schema']['x-command-contract']['version'] == 1
            assert cap['output_schema'] == next(c['output_schema'] for c in full['capabilities'] if c['id'] == command)
            details[command] = cap['input_schema']
        assert before == client.ask('project.inspect')
        schema = details['sweep.settings.set']
        assert details['sweep.start']['properties']['settings']['anyOf'][0] == {
            k: v for k, v in schema.items() if k not in ('x-command-contract', 'x-details-scope', 'x-max-retained-operations', 'examples')}
        domains = {d['parameter']: d for d in schema['x-parameter-domains']}
        assert domains['voltage']['minimum'] == -12 and domains['resistance']['maximum'] == 100000
        settings = copy.deepcopy(schema['examples'][0])
        client.ask('sweep.settings.set', settings, True)
        for name, value in [('points', 1), ('points', schema['properties']['points']['maximum'] + 1),
                            ('probe_ids', []), ('probe_ids', ['i:D1', 'i:D1']), ('start', '13 V')]:
            bad = copy.deepcopy(settings)
            bad[name] = value
            state = client.ask('project.inspect')
            client.ask('sweep.settings.set', bad, True, 'invalid-params')
            assert state == client.ask('project.inspect')
        client.ask('sweep.start', details['sweep.start']['examples'][0], True)
        deadline = time.monotonic() + 15
        while True:
            state = client.ask('project.inspect')
            sweep = state['dc_sweep']
            op = next(o for o in sweep['operations'] if o['id'] == sweep['selected_operation'])
            assert op['status'] not in ('failed', 'canceled'), op
            if op['status'] == 'completed':
                break
            assert time.monotonic() < deadline
            time.sleep(.01)
        captured = copy.deepcopy(sweep['result'])
        for example in details['sweep.cursor.set']['examples']:
            client.ask('sweep.cursor.set', example, True)
        assert client.ask('project.inspect')['dc_sweep'].get('cursor_index') is None
        client.ask('sweep.cursor.set', {'index': 5, 'operation_id': op['id']}, True)
        selected = client.ask('project.inspect')
        for params, code in [({}, 'invalid-params'), ({'index': 5.5}, 'invalid-params'),
                             ({'index': None, 'operation_id': 'retired'}, 'conflict')]:
            client.ask('sweep.cursor.set', params, True, code)
            assert selected == client.ask('project.inspect')
        client.ask('sweep.cursor.set', {'index': None, 'operation_id': op['id']}, True)
        after = client.ask('project.inspect')
        assert after['dc_sweep'].get('cursor_index') is None and after['dc_sweep']['result'] == captured
        for key in ('components', 'wires', 'run_id', 'electrical_revision', 'result', 'undo_entry'):
            assert before.get(key) == after.get(key), key
        cursor_contract = details['sweep.cursor.set']['x-command-contract']
        assert 'conflict' in [e['code'] for e in cursor_contract['errors']]
        client.ask('project.clear', {'discard_changes': True}, True)
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'sweep_commands': ids, 'detailed_schemas': details,
                  'captured_curve': captured, 'clear_result': {'cursor_index': None, 'captured_data_preserved': True},
                  'checks': ['paused discovery purity and unchanged output schemas', 'complete sweep family annotations',
                             'nested start settings match standalone schema', 'discover-and-execute settings/start/cursor examples',
                             'published limits and invalid-request purity', 'explicit-null clear, omitted-index refusal and stale-ID guard',
                             'live circuit/time/history and captured data preserved'],
                  'limits': ['Sweep-specific contract coverage; other command semantic details remain incomplete',
                             'Common protocol errors and filesystem/OS outcomes remain separate; no full native action-coverage claim']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as target:
            target.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps({'passed': True, 'sweep_commands': len(ids), 'transport': report['transport']}))
    finally:
        client.close()


if __name__ == '__main__':
    main()
