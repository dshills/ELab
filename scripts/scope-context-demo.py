#!/usr/bin/env python3
"""Check pinned recorded cursor targets through either public transport."""
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
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--artifacts', required=True, type=Path)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)

    def capture():
        state = client.ask('simulation.verify', {'duration_s': '20 ms', 'input_mode': 'charge',
                                               'probe_ids': ['v:C1']}, True)
        identity = state['last_operation']
        deadline = time.monotonic() + 15
        while True:
            op = client.ask('operation.inspect', {'id': identity})['operation']
            assert op['status'] not in ('failed', 'canceled'), op
            if op['status'] == 'completed':
                return identity
            assert time.monotonic() < deadline
            time.sleep(.01)

    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        schema = client.ask('capabilities.list', {'command': 'recording.cursor.set'})['capabilities'][0]['input_schema']
        assert schema['properties']['operation_id']['description']
        assert 'conflict' in [e['code'] for e in schema['x-command-contract']['errors']]
        first = capture()
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        second = capture()
        client.ask('operation.select', {'id': first}, True)
        client.ask('recording.view.set', {'mode': 'recorded'}, True)
        client.ask('recording.cursor.set', {'operation_id': first, 'cursor': 'a', 'time_s': '10 ms'}, True)
        client.ask('operation.select', {'id': second}, True)
        before = client.ask('project.inspect')
        client.ask('recording.cursor.set', {'operation_id': first, 'cursor': 'a', 'time_s': '12 ms'}, True, 'conflict')
        assert before == client.ask('project.inspect')
        pinned = client.ask('recording.cursor.set', {'operation_id': second, 'cursor': 'a', 'time_s': '12 ms'}, True)
        assert pinned['recording']['scope']['a']['requested_s'] == .012
        legacy = client.ask('recording.cursor.set', {'cursor': 'b', 'time_s': '20 ms'}, True)
        assert legacy['recording']['scope']['b']['requested_s'] == .02
        client.ask('recording.window.set', {'start_s': '4 ms', 'end_s': '16 ms'}, True)
        interval = client.ask('project.inspect')['recording']['scope']
        assert interval['a']['valid'] and not interval['b']['valid']
        for key in ('components', 'wires', 'document', 'run_id', 'electrical_revision', 'transient', 'undo_entry'):
            assert before.get(key) == legacy.get(key), key
        project = directory / 'cursor-captures.elab'
        client.ask('project.save', {'path': str(project)}, True)
        digest = hashlib.sha256(project.read_bytes()).hexdigest()
        client.ask('project.load', {'path': str(project)}, True)
        reopened = client.ask('simulation.wait')
        current = reopened['recording']['id']
        assert current not in (first, second)
        before = client.ask('project.inspect')
        client.ask('recording.cursor.set', {'operation_id': second, 'cursor': 'a', 'time_s': 0}, True, 'conflict')
        assert before == client.ask('project.inspect')
        fresh = client.ask('recording.cursor.set', {'operation_id': current, 'cursor': 'a', 'time_s': '10 ms'}, True)
        assert not fresh['document']['dirty'] and fresh['recording']['scope']['a']['requested_s'] == .01
        assert hashlib.sha256(project.read_bytes()).hexdigest() == digest
        client.ask('project.clear', {'discard_changes': True}, True)
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      detailed_schema=schema, pinned_cursor=pinned['recording']['scope']['a'],
                      legacy_cursor=legacy['recording']['scope']['b'], interval_cursors={k: interval[k] for k in ('a', 'b')},
                      reopened_cursor=fresh['recording']['scope']['a'],
                      checks=['explicit capture target refused with a current revision after selection change',
                              'pinned and legacy unit-time cursor inputs', 'full-duration admission and visible-interval validity stay distinct',
                              'reopened local identities require fresh pinned IDs', 'refusal/query/live/history/dirty/file-byte preservation'],
                      limits=['Public command contract; native queued-frame/drag and GPU behavior are qualified separately',
                              'No OS display-completion, complete physical gesture/focus/accessibility or hardware qualification claim'])
        with args.output.open('x') as target:
            json.dump(report, target, indent=2)
            target.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'])))
    finally:
        client.close()


if __name__ == '__main__':
    main()
