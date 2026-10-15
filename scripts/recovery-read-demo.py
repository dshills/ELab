#!/usr/bin/env python3
"""Qualify observable recovery reads against exact pure-query/Verify results."""
import argparse
import base64
import copy
import hashlib
import importlib.util
import json
from pathlib import Path
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--artifacts', required=True, type=Path)
    args = parser.parse_args()
    root = args.artifacts.resolve()
    root.mkdir(mode=0o700, parents=True, exist_ok=False)
    directory = root / 'recovery'
    directory.mkdir(mode=0o700)
    client = helper.Client(args.connect, command_preconditions=True)

    def wait(identity):
        deadline = time.monotonic() + 15
        while True:
            work = client.ask('recovery.read.inspect', {'id': identity})
            if work['status'] not in ('queued', 'running', 'canceling'):
                return work
            assert time.monotonic() < deadline, work
            time.sleep(.01)

    def read(params, status='completed'):
        admitted = client.ask('recovery.read.start', params, True)
        identity = admitted['recovery']['reads'][-1]['id']
        work = wait(identity)
        assert work['status'] == status and work['input'] == params, work
        assert work['actor'] and work['directory'] == str(directory)
        if status != 'completed':
            assert work['result_bytes'] == 0 and 'result_sha256' not in work
            client.ask('recovery.read.result', {'id': identity}, expect='not-ready')
            return work, None
        before = client.ask('project.inspect')
        data = bytearray()
        while len(data) < work['result_bytes']:
            chunk = client.ask('recovery.read.chunk', {'id': identity, 'offset': len(data), 'limit': 512})
            assert chunk['id'] == identity and chunk['offset'] == len(data)
            assert chunk['sha256'] == work['result_sha256'] and chunk['total'] == work['result_bytes']
            data.extend(base64.b64decode(chunk['json_base64'], validate=True))
            assert chunk['next_offset'] == len(data)
        assert hashlib.sha256(data).hexdigest() == work['result_sha256']
        result = client.ask('recovery.read.result', {'id': identity})
        assert json.loads(data) == result and result['id'] == identity and result['kind'] == params['kind']
        assert before == client.ask('project.inspect')
        client.ask('recovery.read.cancel', {'id': identity}, True, 'unavailable')
        assert before == client.ask('project.inspect')
        return work, result

    try:
        schemas = {name: client.ask('capabilities.list', {'command': 'recovery.read.' + name})['capabilities'][0]['input_schema']
                   for name in ('start', 'cancel', 'inspect', 'result', 'chunk')}
        assert schemas['start']['x-max-concurrent-reads'] == 4
        assert schemas['start']['x-max-retained-jobs'] == 8
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        source = root / 'source.elab'
        client.ask('project.save', {'path': str(source)}, True)
        source_digest = hashlib.sha256(source.read_bytes()).hexdigest()
        state = client.ask('simulation.verify', {'duration_s': '20 ms'}, True)
        operation = state['last_operation']
        deadline = time.monotonic() + 15
        while True:
            recorded = client.ask('operation.inspect', {'id': operation})
            assert recorded['operation']['status'] not in ('failed', 'canceled'), recorded
            if recorded['operation']['status'] == 'completed':
                break
            assert time.monotonic() < deadline
            time.sleep(.01)
        expected = client.ask('operation.samples', {'id': operation, 'offset': 10, 'limit': 2})
        client.ask('recovery.configure', {'directory': str(directory), 'enabled': False, 'interval_s': 2}, True)
        client.ask('recovery.checkpoint', {}, True)
        candidate = client.ask('project.inspect')['recovery']['candidates'][0]
        choice = {key: candidate[key] for key in ('id', 'sha256')}
        capsule = directory / (choice['id'] + '.elab-recovery')
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        client.ask('simulation.wait')
        before = client.ask('project.inspect')
        work, inspection = read(dict(kind='inspection', **choice))
        pure = client.ask('recovery.inspect', choice)
        assert inspection['inspection'] == pure
        assert pure['document']['recordings'][0] == recorded['metadata']
        assert next(c for c in pure['document']['graph']['components'] if c['id'] == 'R1')['value'] == 10000
        _, samples = read(dict(kind='samples', recording_index=0, offset=10, limit=2, **choice))
        samples = samples['samples']
        assert samples == client.ask('recovery.samples', dict(recording_index=0, offset=10, limit=2, **choice))
        for key in ('rows', 'probes', 'offset', 'next_offset', 'total'):
            assert samples[key] == expected[key], key
        failed, _ = read(dict(kind='inspection', id=choice['id'], sha256='0' * 64), 'failed')
        assert failed['error']['code'] == 'file-conflict'
        for params in (dict(kind='inspection', offset=0, **choice), dict(kind='samples', **choice)):
            prior = client.ask('project.inspect')
            client.ask('recovery.read.start', params, True, 'invalid-params')
            assert client.ask('project.inspect') == prior
        after = client.ask('project.inspect')
        for key in ('components', 'wires', 'document', 'run_id', 'electrical_revision', 'transient', 'undo_entry'):
            assert before.get(key) == after.get(key), key
        assert before['recovery']['selected'] == after['recovery']['selected']
        client.ask('recovery.configure', {'directory': str(root / 'other'), 'enabled': False, 'interval_s': 2}, True)
        assert client.ask('recovery.read.result', {'id': work['id']}) == inspection
        assert hashlib.sha256(source.read_bytes()).hexdigest() == source_digest
        assert hashlib.sha256(capsule.read_bytes()).hexdigest() == choice['sha256']
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      detailed_schemas=schemas, captured_graph=pure['document']['graph'],
                      sample_page={key: copy.deepcopy(samples[key]) for key in ('rows', 'probes', 'offset', 'next_offset', 'total')},
                      checks=['stable captured read IDs and progress/terminal metadata', 'exact pure-query inspection and original Verify sample parity',
                              'digest-checked 512-byte result chunks equal structured result', 'no partial result on hash failure',
                              'query/refusal/live/history/selection/file purity', 'retained result survives directory reconfiguration'],
                      limits=['Controlled tests qualify blocked reads, cancellation, four-reader capacity, eviction and Close joining',
                              'No complete physical typing/accessibility or general latency qualification'])
        with args.output.open('x') as target:
            json.dump(report, target, indent=2)
            target.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'])))
        if args.connect:
            client.ask('app.close', {'discard_changes': True}, True)
    finally:
        client.close()


if __name__ == '__main__':
    main()
