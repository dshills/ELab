#!/usr/bin/env python3
"""Validate actionable diagnostic associations and pure rejected-request context."""
import argparse
import importlib.util
import json
from pathlib import Path

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    client = helper.Client(args.connect, command_preconditions=True)
    fresh = None
    try:
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.resume', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.pause', {}, True)
        client.ask('object.delete', {'id': 'W1'}, True)
        opened = client.ask('project.inspect')
        item = opened['diagnostic_presentation']['items'][0]
        assert item['code'] == 'open-terminal' and item['object_ids'] == ['V1'] and item['terminal_refs'] == ['V1.positive']
        assert item['action'] and item['details'] and not opened['diagnostic_presentation']['details_visible']
        client.ask('drawing.tool.set', {'tool': 'wire'}, True)
        client.ask('wire.begin', {'terminal': 'V1.positive'}, True)
        before = client.ask('project.inspect')
        shown = client.ask('diagnostics.details.set', {'visible': True}, True)
        for field in ['drawing', 'run_id', 'electrical_revision', 'result', 'document', 'undo_entry']:
            assert shown[field] == before[field], field
        assert shown['diagnostic_presentation']['details_visible']
        client.ask('wire.connect', {'from': 'V1.positive', 'to': 'R1.1'}, True)
        assert not client.ask('project.inspect')['diagnostic_presentation']['items']
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        before = client.ask('project.inspect')
        # Inspect the full envelope: help belongs to the rejected request, not
        # global mutable service state. This also exercises the public guards.
        request = dict(version=1, id='diagnostic-invalid-value', command='component.set',
                       params={'id': 'R1', 'value': 'invalid'},
                       expected_command_revision=client.command_revision, key='diagnostic-invalid-value')
        client.process.stdin.write(json.dumps(request)+'\n')
        client.process.stdin.flush()
        response = json.loads(client.process.stdout.readline())
        assert response['error']['code'] == 'invalid-params'
        assert response['help']['source'] == 'request' and response['help']['object_ids'] == ['R1']
        assert response['help']['details'] == response['error']['message']
        assert client.ask('project.inspect') == before
        client.ask('diagnostics.details.set', {'visible': False}, True)
        directory = args.artifacts.resolve()
        directory.mkdir(parents=True, exist_ok=True)
        path = str(directory/'repaired-led.elab')
        client.ask('project.save', {'path': path, 'overwrite': True}, True)
        client.ask('diagnostics.details.set', {'visible': True}, True)
        assert not client.ask('project.inspect')['document']['dirty']
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': path}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['paused'] and not reopened['document']['dirty']
        assert not reopened['diagnostic_presentation']['details_visible'] and not reopened['diagnostic_presentation']['items']
        client.ask('diagnostics.details.set', {'visible': False}, True)
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                      checks=['known open terminal/object associations and actionable text', 'technical details use the shared command',
                              'disclosure preserves unfinished wire gesture/physics/history/document',
                              'correction removes ghost markers', 'failed value envelope retains known request context',
                              'rejected request does not mutate service state', 'disclosure remains session-only and fresh reopen is clean',
                              'owned project ends clean and paused'],
                      limitations=['unknown solver locations receive no fabricated highlight',
                                   'request targets describe related intent, not a guessed solver cause',
                                   'native marker/disclosure/GPU and accessibility evidence are separate checks'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()
