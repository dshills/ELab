#!/usr/bin/env python3
"""Verify captured-failure guidance and preserved live/kept state via public commands."""
import argparse
import hashlib
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
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)

    def inspect():
        return client.ask('project.inspect')

    def finish(identity):
        deadline = time.monotonic() + 10
        while True:
            view = client.ask('operation.inspect', {'id': identity})
            if view['operation']['status'] in ('completed', 'failed', 'canceled'):
                return view
            assert time.monotonic() < deadline
            time.sleep(.01)

    def preserved(before, after):
        for key in ('components', 'wires', 'transient', 'mixed', 'run_id', 'electrical_revision',
                    'document', 'drawing', 'selected', 'undo_entry'):
            assert before.get(key) == after.get(key), key

    try:
        client.ask('example.rectifier', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        discovery = client.ask('capabilities.list', {'command': 'operation.inspect'})
        cap = discovery['capabilities'][0]
        operation_ref = cap['output_schema']['properties']['operation']['$ref']
        assert operation_ref == 'urn:elab:verification-operation:1'
        help_ref = discovery['schemas'][operation_ref]['properties']['failure_help']['anyOf'][0]['$ref']
        assert help_ref == 'urn:elab:diagnostic-help:1'
        assert discovery['schemas'][help_ref]['properties']['action']['type'] == 'string'
        catalog_ids = {item['id'] for item in client.ask('capabilities.list')['capabilities']}
        assert {'operation.inspect', 'simulation.verify', 'diagnostics.details.set'} <= catalog_ids

        accepted = client.ask('simulation.verify', dict(duration_s='1 ms', probe_ids=['v:C1']), True)
        kept_id = accepted['last_operation']
        kept = finish(kept_id)
        assert kept['operation']['status'] == 'completed' and 'failure_help' not in kept['operation']
        asset = assets / 'kept-before-failure.json'
        client.ask('verification.export', dict(id=kept_id, path=str(asset), format='json'), True)
        digest = hashlib.sha256(asset.read_bytes()).hexdigest()
        before = inspect()
        duration = math.nextafter(.001, math.inf)
        accepted = client.ask('simulation.verify', {'duration_s': duration}, True)
        failed_id = accepted['last_operation']
        failed = finish(failed_id)
        operation = failed['operation']
        explanation = operation['failure_help']
        assert operation['status'] == 'failed' and operation['error']['code'] == 'solve-failed'
        assert 'mixed timestep too small' in operation['error']['message']
        assert explanation['source'] == 'verification' and explanation['code'] == operation['error']['code']
        assert explanation['details'] == operation['error']['message']
        assert explanation['object_ids'] == explanation['terminal_refs'] == explanation['wire_ids'] == []
        assert 'No partial recording' in explanation['action'] and 'earlier' not in explanation['summary']
        assert operation['source_unchanged'] and not operation['kept_in_project']
        assert failed['metadata']['duration_s'] == duration and failed['metadata']['sample_count'] == 0
        client.ask('operation.samples', {'id': failed_id}, True, 'not-ready')
        after = inspect()
        preserved(before, after)
        assert 'verification' not in after and 'recording' not in after
        assert client.ask('operation.inspect', {'id': kept_id})['metadata'] == kept['metadata']
        before_query = inspect()
        assert client.ask('operation.inspect', {'id': failed_id}) == failed
        assert inspect() == before_query
        disclosed = client.ask('diagnostics.details.set', {'visible': True}, True)
        assert disclosed['diagnostic_presentation']['details_visible']
        preserved(before_query, disclosed)
        client.ask('diagnostics.details.set', {'visible': False}, True)

        client.ask('component.set', {'id': 'R1', 'value': '200 ohm'}, True)
        before_query = inspect()
        earlier = client.ask('operation.inspect', {'id': failed_id})
        assert not earlier['operation']['source_unchanged']
        assert 'earlier captured' in earlier['operation']['failure_help']['summary']
        assert earlier['operation']['error'] == operation['error'] and earlier['metadata'] == failed['metadata']
        assert earlier['operation']['failure_help']['object_ids'] == [] and inspect() == before_query
        assert all(item['source'] != 'verification' for item in before_query['diagnostic_presentation']['items'])
        selected = client.ask('operation.select', {'id': kept_id}, True)
        assert selected['verification']['id'] == kept_id
        assert next(op for op in selected['operations'] if op['id'] == kept_id).get('failure_help') is None
        copy = assets / 'kept-after-failure.json'
        client.ask('verification.export', dict(id=kept_id, path=str(copy), format='json'), True)
        assert hashlib.sha256(copy.read_bytes()).hexdigest() == digest
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      observations=dict(error_code=operation['error']['code'], requested_duration_s=duration,
                                        help=explanation, changed_source_help=earlier['operation']['failure_help'],
                                        kept_sample_count=kept['metadata']['sample_count']),
                      schemas=dict(operation=discovery['schemas'][operation_ref], diagnostic_help=discovery['schemas'][help_ref],
                                   inspect_input=cap['input_schema']),
                      checks=['Real captured numerical failure supplies summary/action/technical reason without changing error identity',
                              'Local registry exposes typed failure guidance and ordinary catalog remains available',
                              'No partial samples, live/source/history change or kept recording loss',
                              'Read-only failure inspection is pure; technical disclosure uses the existing shared command',
                              'Later circuit edits retain original captured input/error and label the earlier experiment without guessed markers',
                              'Prior kept recording remains selectable and byte-identical on export'],
                      limits=['Real failure exercises existing rectifier minimum integration-step refusal; equations/limits are unchanged',
                              'Work-attempt/time/overall-timeout help branches use controlled core tests, not artificially delayed native jobs',
                              'No complete physical keyboard/focus/accessibility/visible-latency or device qualification'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as output:
            json.dump(report, output, indent=2)
            output.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], error_code=operation['error']['code'])))
    finally:
        try:
            if args.connect:
                client.ask('project.inspect')
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()
