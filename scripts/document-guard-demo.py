#!/usr/bin/env python3
"""Verify optional envelope context guards through public transports."""
import argparse
import importlib.util
import json
from pathlib import Path


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def guarded(client, generation, key):
    client.sequence += 1
    request = {'version': 1, 'id': f'guard-{client.sequence}', 'command': 'component.set',
               'params': {'id': 'R1', 'value': '2k'}, 'expected_command_revision': client.command_revision,
               'expected_document_generation': generation, 'key': key}
    client.process.stdin.write(json.dumps(request) + '\n')
    client.process.stdin.flush()
    return json.loads(client.process.stdout.readline())


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='owned validation session; replaces its circuit')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    try:
        discovery = client.ask('capabilities.list')
        assert 'expected_document_generation' in discovery['mutation_preconditions']
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        first = client.ask('simulation.restart', {}, True)
        first = client.ask('simulation.wait')
        client.ask('project.clear', {'discard_changes': True}, True)
        client.ask('example.reset', {}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        current = client.ask('simulation.wait')
        assert first['document']['generation'] != current['document']['generation']
        result = guarded(client, first['document']['generation'], 'context-test')
        assert result['error']['code'] == 'conflict'
        assert client.ask('project.inspect') == current
        accepted = guarded(client, current['document']['generation'], 'context-test')
        assert 'error' not in accepted and not accepted.get('duplicate')
        edited = client.ask('simulation.wait')
        resistor = next(p for p in edited['components'] if p['id'] == 'R1')
        assert resistor['value'] == 2000
        duplicate = guarded(client, first['document']['generation'], 'context-test')
        assert duplicate['duplicate'] and duplicate['result'] == accepted['result']
        assert client.ask('project.inspect') == edited
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'checks': ['context guard advertised', 'old document generation refuses despite current command guard',
                             'refusal preserves graph/history/physics', 'fresh generation permits same intended action',
                             'successful retry retains original response despite stale context', 'legacy requests remain supported'],
                  'resistance_ohm': resistor['value'],
                  'limits': ['publication-only case is covered by controlled core regression, not this end-to-end replacement workflow',
                             'source inventory is not exhaustive mounted gesture/native capability qualification']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps(report))
    finally:
        client.close()


if __name__ == '__main__':
    main()
