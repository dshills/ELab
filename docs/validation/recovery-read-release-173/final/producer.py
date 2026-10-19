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
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--agent-binary', type=Path)
    parser.add_argument('--agent-sha256')
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--artifacts', required=True, type=Path)
    args = parser.parse_args()
    if not __debug__:
        parser.error('validation requires Python assertions; run without -O/PYTHONOPTIMIZE')
    if bool(args.agent_binary) != bool(args.agent_sha256):
        parser.error('--agent-binary and --agent-sha256 must be supplied together')
    binary = args.agent_binary.resolve() if args.agent_binary else helper.ROOT / 'build/elab-agent'
    binary_sha256 = hashlib.sha256(binary.read_bytes()).hexdigest()
    if args.agent_sha256 and binary_sha256 != args.agent_sha256:
        parser.error('agent SHA-256 differs; no session started')
    root = args.artifacts.resolve()
    root.mkdir(mode=0o700, parents=True, exist_ok=False)
    directory = root / 'recovery'
    directory.mkdir(mode=0o700)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True, agent_binary=binary)
    observations = []

    def wait(identity):
        deadline = time.monotonic() + 15
        while True:
            work = client.ask('recovery.read.inspect', {'id': identity})
            if work['status'] not in ('queued', 'running', 'canceling'):
                return work
            assert time.monotonic() < deadline, work
            time.sleep(.01)

    def read(params, status='completed', expected_directory=None):
        admitted = client.ask('recovery.read.start', params, True)
        identity = admitted['recovery']['reads'][-1]['id']
        work = wait(identity)
        assert work['status'] == status and work['input'] == params, work
        assert work['actor'] and work['directory'] == str(expected_directory or directory)
        if status != 'completed':
            assert work['result_bytes'] == 0 and 'result_sha256' not in work
            client.ask('recovery.read.result', {'id': identity}, expect='not-ready')
            observations.append(dict(work=copy.deepcopy(work), structured_result_refusal='not-ready'))
            return work, None
        before = client.ask('project.inspect')
        data = bytearray()
        ranges = []
        while len(data) < work['result_bytes']:
            offset = len(data)
            chunk = client.ask('recovery.read.chunk', {'id': identity, 'offset': offset, 'limit': 512})
            assert chunk['id'] == identity and chunk['offset'] == offset
            assert chunk['sha256'] == work['result_sha256'] and chunk['total'] == work['result_bytes']
            data.extend(base64.b64decode(chunk['json_base64'], validate=True))
            assert offset < len(data) <= min(offset + 512, work['result_bytes'])
            assert chunk['next_offset'] == len(data)
            ranges.append([offset, len(data)])
        assert hashlib.sha256(data).hexdigest() == work['result_sha256']
        eof = client.ask('recovery.read.chunk', {'id': identity, 'offset': len(data), 'limit': 512})
        assert eof == dict(id=identity, offset=len(data), next_offset=len(data), total=len(data),
                           sha256=work['result_sha256'], json_base64='')
        for bounds in ({'offset': -1}, {'offset': len(data) + 1}, {'limit': -1}, {'limit': 65537}):
            client.ask('recovery.read.chunk', dict(id=identity, **bounds), expect='invalid-params')
        result = client.ask('recovery.read.result', {'id': identity})
        assert json.loads(data) == result and result['id'] == identity and result['kind'] == params['kind']
        assert before == client.ask('project.inspect')
        client.ask('recovery.read.cancel', {'id': identity}, True, 'unavailable')
        assert before == client.ask('project.inspect')
        observations.append(dict(work=copy.deepcopy(work), chunk_ranges=ranges, eof=eof,
                                 reconstructed_sha256=hashlib.sha256(data).hexdigest(),
                                 result_json_base64=base64.b64encode(data).decode('ascii'),
                                 structured_result=copy.deepcopy(result), terminal_cancel_refusal='unavailable',
                                 invalid_chunk_bounds_refusal='invalid-params'))
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
        client.ask('recovery.read.chunk', {'id': failed['id']}, expect='not-ready')
        for params in (dict(kind='inspection', offset=0, **choice), dict(kind='samples', **choice)):
            prior = client.ask('project.inspect')
            client.ask('recovery.read.start', params, True, 'invalid-params')
            assert client.ask('project.inspect') == prior
        after = client.ask('project.inspect')
        for key in ('components', 'wires', 'document', 'run_id', 'electrical_revision', 'transient', 'undo_entry'):
            assert before.get(key) == after.get(key), key
        assert before['recovery']['selected'] == after['recovery']['selected']
        pure_before = client.ask('project.inspect')
        for command in ('inspect', 'result', 'chunk'):
            client.ask('recovery.read.' + command, {'id': 'unknown-read'}, expect='not-found')
        client.ask('recovery.read.cancel', {'id': 'unknown-read'}, True, 'not-found')
        assert client.ask('project.inspect') == pure_before
        client.ask('recovery.configure', {'directory': str(root / 'other'), 'enabled': False, 'interval_s': 2}, True)
        assert client.ask('recovery.read.result', {'id': work['id']}) == inspection
        assert hashlib.sha256(source.read_bytes()).hexdigest() == source_digest
        assert hashlib.sha256(capsule.read_bytes()).hexdigest() == choice['sha256']
        retained = []
        for _ in range(schemas['start']['x-max-retained-jobs']):
            # The new private directory has no old candidate: each job fails
            # independently and cannot expose a partial result. Terminal jobs
            # count toward the same bounded retention inventory.
            expired, _ = read(dict(kind='inspection', **choice), 'failed', root / 'other')
            assert expired['error']['code'] == 'invalid-recovery'
            retained.append(expired['id'])
        for command in ('inspect', 'result', 'chunk'):
            client.ask('recovery.read.' + command, {'id': work['id']}, expect='not-found')
        inventory = client.ask('project.inspect')['recovery']['reads']
        assert [entry['id'] for entry in inventory] == retained and len(inventory) == 8
        assert all(entry['status'] == 'failed' and entry['result_bytes'] == 0 for entry in inventory)
        assert hashlib.sha256(source.read_bytes()).hexdigest() == source_digest
        assert hashlib.sha256(capsule.read_bytes()).hexdigest() == choice['sha256']
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      agent_binary_sha256=binary_sha256,
                      observed_reads=observations, retained_jobs=copy.deepcopy(inventory),
                      evicted_read_id=work['id'], source_sha256=source_digest, capsule_sha256=choice['sha256'],
                      detailed_schemas=schemas, captured_graph=pure['document']['graph'],
                      sample_page={key: copy.deepcopy(samples[key]) for key in ('rows', 'probes', 'offset', 'next_offset', 'total')},
                      checks=['stable captured read IDs and progress/terminal metadata', 'exact pure-query inspection and original Verify sample parity',
                              'digest-checked 512-byte result chunks equal structured result', 'no partial result on hash failure',
                              'query/refusal/live/history/selection/file purity', 'retained result survives directory reconfiguration',
                              'inclusive empty EOF and rejected chunk bounds preserve source', 'unknown job query/cancel refusal purity',
                              'failed jobs expose no structured or chunked partial result', 'eight retained terminal jobs evict old result access'],
                      limits=['Controlled tests qualify blocked reads, cancellation, four-reader capacity, eviction and Close joining',
                              'No complete physical typing/accessibility or general latency qualification'])
        if args.connect:
            client.ask('app.close', {'discard_changes': True}, True)
    finally:
        client.close()
        assert client.process.returncode == 0, 'owned agent/forwarder exited unsuccessfully'
    report['client_exit_code'] = client.process.returncode
    with args.output.open('x') as target:
        json.dump(report, target, indent=2)
        target.write('\n')
    print(json.dumps(dict(passed=True, transport=report['transport'], checks=len(report['checks']))))


if __name__ == '__main__':
    main()
