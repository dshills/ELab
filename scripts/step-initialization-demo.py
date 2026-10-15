#!/usr/bin/env python3
"""Check public Step unit normalization, replay and unchanged context guards.

Held-initializer timing/cancellation is qualified by controlled core tests; this
workflow deliberately uses only public commands and no private delay hooks.
"""
import argparse
import hashlib
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
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect, command_preconditions=True)

    def raw_request(params, key, *, command_revision=None, revision=None, generation=None, raw_params=None):
        client.sequence += 1
        request = dict(version=1, id=f'unit-step-{client.sequence}', command='simulation.step', params=params, key=key)
        if command_revision is not None:
            request['expected_command_revision'] = command_revision
        if revision is not None:
            request['expected_revision'] = revision
        if generation is not None:
            request['expected_document_generation'] = generation
        if raw_params is None:
            payload = json.dumps(request)
        else:
            request['params'] = '__RAW_DURATION_TEST__'
            payload = json.dumps(request).replace('"__RAW_DURATION_TEST__"', raw_params)
        client.process.stdin.write(payload + '\n')
        client.process.stdin.flush()
        response = json.loads(client.process.stdout.readline())
        client.revision = response['revision']
        # A replay contains the original snapshot, which may be stale. Refresh
        # project.inspect before depending on a subsequent mutation guard.
        return response

    try:
        schema = client.ask('capabilities.list', {'command': 'simulation.step'})['capabilities'][0]['input_schema']
        assert schema['properties']['duration_s']['$ref'] == 'urn:elab:time:1'
        client.ask('example.npn-pulse', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        before = client.ask('project.inspect')
        guard = before['command_revision']
        first = raw_request({'duration_s': '1 ms'}, 'equivalent-step', command_revision=guard)
        assert 'error' not in first and not first.get('duplicate', False)
        original = first['result']
        assert original['step_work']['status'] == 'completed'
        assert original['step_work']['completed_samples'] == 1
        assert original['transient']['sample']['time_s'] == .001
        for field in ('components', 'wires', 'document', 'undo_entry', 'run_id', 'electrical_revision'):
            assert original.get(field) == before.get(field), field
        client.ask('project.inspect')
        for duration in (0.001, '1000 us', '1000000 ns'):
            retry = raw_request({'duration_s': duration}, 'equivalent-step', command_revision=guard)
            assert retry.get('duplicate') and retry['result'] == original and 'error' not in retry
            client.ask('project.inspect')

        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        path = assets / 'unrelated-led.elab'
        client.ask('project.save', {'path': str(path)}, True)
        saved_digest = hashlib.sha256(path.read_bytes()).hexdigest()
        unrelated = client.ask('project.inspect')
        replay = raw_request({'duration_s': '1 ms'}, 'equivalent-step', command_revision=guard,
                             generation=unrelated['document']['generation'] + 1)
        assert replay.get('duplicate') and replay['result'] == original and 'error' not in replay
        assert client.ask('project.inspect') == unrelated
        reused = raw_request({'duration_s': '2 ms'}, 'equivalent-step', command_revision=guard)
        assert reused.get('error', {}).get('code') == 'key-reused'
        assert client.ask('project.inspect') == unrelated
        malformed = raw_request({}, 'malformed', command_revision=unrelated['command_revision'],
                                raw_params='{"duration_s":"1 ms","duration_s":0.001}')
        # The wire decoder rejects duplicate fields before constructing a Request.
        assert malformed.get('error', {}).get('code') == 'invalid-json', malformed
        assert client.ask('project.inspect') == unrelated
        for name, guards in (
            ('all-state', dict(revision=unrelated['revision'] - 1)),
            ('command', dict(command_revision=unrelated['command_revision'] - 1)),
            ('document', dict(command_revision=unrelated['command_revision'], generation=unrelated['document']['generation'] + 1)),
        ):
            rejected = raw_request({'duration_s': '1 ms'}, 'stale-' + name, **guards)
            assert rejected.get('error', {}).get('code') == 'conflict', rejected
            assert client.ask('project.inspect') == unrelated
        assert hashlib.sha256(path.read_bytes()).hexdigest() == saved_digest
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', detailed_schema=schema,
                      result=dict(analysis=original['analysis'], step_status=original['step_work']['status'],
                                  completed_samples=original['step_work']['completed_samples'], time_s=original['transient']['sample']['time_s']),
                      checks=['Unit-text Step completes once', 'Numeric/us/ns equivalents replay the original result',
                              'Replay precedes newer context guards and preserves an unrelated saved document', 'Changed duration retains key-reused error',
                              'Duplicate JSON fields remain invalid', 'Revision/command/document conflicts preserve state', 'Source/history/saved bytes preserved'],
                      limits=['Held-initializer wait, cancellation and post-wait guard behavior are covered by controlled core tests, not an artificial public delay API',
                              'This workflow does not establish complete mounted physical input, accessibility or visible-latency acceptance'])
        with args.output.open('x') as target:
            json.dump(report, target, indent=2)
            target.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], completed_samples=report['result']['completed_samples'])))
    finally:
        try:
            if args.connect:
                client.ask('project.inspect')
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()
