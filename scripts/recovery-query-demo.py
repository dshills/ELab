#!/usr/bin/env python3
"""Qualify pure captured recovery inspection/sample queries over either public transport."""
import argparse
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
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    root = args.artifacts.resolve()
    root.mkdir(mode=0o700, parents=True, exist_ok=False)
    directory = root / 'recovery'
    directory.mkdir(mode=0o700)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        source = root / 'source.elab'
        client.ask('project.save', {'path': str(source)}, True)
        source_digest = hashlib.sha256(source.read_bytes()).hexdigest()
        state = client.ask('simulation.verify', {'duration_s': '20 ms'}, True)
        operation_id = state['last_operation']
        deadline = time.monotonic() + 10
        while True:
            recorded = client.ask('operation.inspect', {'id': operation_id})
            assert recorded['operation']['status'] not in ('failed', 'canceled'), recorded
            if recorded['operation']['status'] == 'completed':
                break
            assert time.monotonic() < deadline
            time.sleep(.01)
        expected = client.ask('operation.samples', {'id': operation_id, 'offset': 10, 'limit': 2})
        client.ask('recovery.configure', {'directory': str(directory), 'enabled': False, 'interval_s': 2}, True)
        client.ask('recovery.checkpoint', {}, True)
        candidate = client.ask('project.inspect')['recovery']['candidates'][0]
        choice = {'id': candidate['id'], 'sha256': candidate['sha256']}
        capsule = directory / (candidate['id'] + '.elab-recovery')
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': '1 ms'}, True)
        before = client.ask('project.inspect')
        inspection = client.ask('recovery.inspect', choice)
        assert inspection['directory'] == str(directory)
        assert inspection['candidate']['sha256'] == candidate['sha256'] and inspection['candidate']['recording_count'] == 1
        assert inspection['document']['recordings'][0] == recorded['metadata']
        old_resistor = next(c for c in inspection['document']['graph']['components'] if c['id'] == 'R1')
        live_resistor = next(c for c in before['components'] if c['id'] == 'R1')
        assert old_resistor['value'] == 10000 and live_resistor['value'] == 20000
        samples = client.ask('recovery.samples', dict(choice, recording_index=0, offset=10, limit=2))
        for key in ('rows', 'probes', 'offset', 'next_offset', 'total'):
            assert samples[key] == expected[key], key
        assert samples['id'] == candidate['id']
        assert client.ask('project.inspect') == before
        for command, params, code in [
            ('recovery.inspect', dict(choice, sha256='old'), 'file-conflict'),
            ('recovery.samples', dict(choice, recording_index=0, limit=201), 'invalid-params'),
            ('recovery.samples', dict(choice, recording_index=99), 'invalid-params'),
            ('recovery.samples', dict(choice, recording_index=0, offset=-1), 'invalid-params'),
            ('recovery.samples', choice, 'invalid-params'),
        ]:
            client.ask(command, params, expect=code)
            assert client.ask('project.inspect') == before
        assert hashlib.sha256(source.read_bytes()).hexdigest() == source_digest
        assert hashlib.sha256(capsule.read_bytes()).hexdigest() == candidate['sha256']
        (root / 'captured-capsule.elab-recovery').write_bytes(capsule.read_bytes())
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'captured_graph': inspection['document']['graph'], 'sample_page': {k: copy.deepcopy(samples[k]) for k in
                    ('rows', 'probes', 'offset', 'next_offset', 'total')},
                  'live_semantic': {k: copy.deepcopy(before.get(k)) for k in ('components', 'wires', 'run_id',
                    'electrical_revision', 'result', 'transient', 'mixed', 'npn', 'view')},
                  'checks': ['captured directory/hash identified', 'inspection contains original graph and exact recorded metadata',
                             'sample rows/units match retained verification, not later live inputs',
                             'successful and rejected reads preserve all public state and revisions',
                             'saved schematic and capsule bytes unchanged'],
                  'limits': ['Blocked-read edit/step, pool saturation, cancellation/Close and reconfiguration are covered by controlled core tests',
                             'Legacy queries remain request-scoped and pure; explicit observable read-job aliases remain an async-coverage task',
                             'No complete physical-input/accessibility or general latency claim']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as target:
            target.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps({'passed': True, 'transport': report['transport']}))
    finally:
        client.close()


if __name__ == '__main__':
    main()
