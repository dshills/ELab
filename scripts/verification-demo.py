#!/usr/bin/env python3
"""Verify circuits through public asynchronous operations, independent of live state.
--connect replaces the given owned development session's schematic.
"""
import argparse
import csv
import importlib.util
import json
import math
from pathlib import Path
import tempfile
import time


spec = importlib.util.spec_from_file_location('persistence_demo', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect)
    try:
        discovery = client.ask('capabilities.list')
        assert discovery['schema_version'] == 2
        for capability in discovery['capabilities']:
            ref = capability['output_schema'].get('$ref')
            if ref:
                assert ref in discovery['schemas']

        def wait(operation_id):
            deadline = time.monotonic() + 15
            while time.monotonic() < deadline:
                view = client.ask('operation.inspect', {'id': operation_id})
                if view['operation']['status'] not in ['queued', 'running', 'canceling']:
                    return view
                time.sleep(.02)
            raise AssertionError('verification did not finish')

        def samples(operation_id):
            result, offset = [], 0
            while True:
                page = client.ask('operation.samples', {'id': operation_id, 'offset': offset, 'limit': 100})
                result.extend(page['rows'])
                if page['next_offset'] == page['total']:
                    return page['probes'], result
                offset = page['next_offset']

        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.step', {'duration_s': .03}, True)
        before = client.ask('project.inspect')
        accepted = client.ask('simulation.verify', {'duration_s': .01, 'probe_ids': ['v:C1', 'i:C1']}, True)
        operation_id = accepted['last_operation']
        rc = wait(operation_id)
        assert rc['operation']['status'] == 'completed'
        probes, rows = samples(operation_id)
        assert len(rows) == 11 and abs(rows[-1]['values'][0] - 5*(1-math.exp(-1))) < 1e-12
        after = client.ask('project.inspect')
        assert after['transient']['sample'] == before['transient']['sample']
        assert after['document']['path'] == before['document']['path'] and after['document']['generation'] == before['document']['generation']
        assert after['document']['schematic_dirty'] == before['document']['schematic_dirty'] and after['document']['recordings_dirty']
        assert after['run_id'] == before['run_id']
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        assert not client.ask('operation.inspect', {'id': operation_id})['operation']['source_unchanged']
        client.ask('simulation.verify', {'duration_s': 10}, True, expect='invalid-params')
        with tempfile.TemporaryDirectory(prefix='elab-verification-') as temporary:
            directory = args.artifacts.resolve() if args.artifacts else Path(temporary)
            directory.mkdir(parents=True, exist_ok=True)
            artifact_paths = []
            for kind in ['json', 'csv']:
                path = directory / ('rc-record.' + kind)
                client.ask('verification.export', {'id': operation_id, 'path': str(path), 'format': kind, 'overwrite': True}, True)
                artifact_paths.append(str(path))
            document = json.loads(Path(artifact_paths[0]).read_text())
            assert document['metadata']['format'] == 'elab.verification'
            assert document['metadata']['graph'] == rc['metadata']['graph']
            assert document['rows'] == rows
            with Path(artifact_paths[1]).open() as f:
                csv_rows = list(csv.reader(f))
            assert len(csv_rows) == 12 and csv_rows[0][-1] == 'i:C1 [A]'
            client.ask('example.flasher', {'discard_changes': True}, True)
            client.ask('simulation.pause', {}, True)
            client.ask('simulation.restart', {}, True)
            before = client.ask('project.inspect')
            accepted = client.ask('simulation.verify', {'duration_s': .1, 'probe_ids': ['i:D1', 'i:D2', 'v:C1', 'v:C2']}, True)
            flasher = wait(accepted['last_operation'])
            assert flasher['operation']['status'] == 'completed'
            assert flasher['metadata']['mixed']['kcl_residual_a'] <= 1e-9
            assert client.ask('project.inspect')['transient']['sample'] == before['transient']['sample']
            client.ask('simulation.step', {'duration_s': .1}, True)
            live = client.ask('project.inspect')['mixed']
            for actual, recorded in zip(live['capacitors'], flasher['metadata']['mixed']['capacitors']):
                assert abs(actual['voltage_v'] - recorded['voltage_v']) < 1e-8
            # Real cancellation: the longest admitted flasher verification.
            accepted = client.ask('simulation.verify', {'duration_s': 5}, True)
            client.ask('operation.cancel', {'id': accepted['last_operation']}, True)
            canceled = wait(accepted['last_operation'])
            assert canceled['operation']['status'] == 'canceled'
            client.ask('operation.samples', {'id': accepted['last_operation']}, expect='not-ready')
            # Return the app to the completed flasher's recorded readout.
            client.ask('operation.select', {'id': flasher['operation']['id']}, True)
            report = {'passed': True, 'transport': 'desktop socket' if args.connect else 'headless stdio',
                      'schema_version': discovery['schema_version'], 'capabilities': len(discovery['capabilities']),
                      'rc_voltage_10ms_v': rows[-1]['values'][0], 'rc_samples': len(rows),
                      'flasher_samples': flasher['metadata']['sample_count'], 'cancellation': canceled['operation']['status'],
                      'checks': ['local schema reference resolution', 'accepted operation ID and completion',
                                 'RC analytical reference', 'paged SI samples', 'captured source survives edits',
                                 'live time/state/history preserved', 'admission bounds',
                                 'portable recorded JSON/CSV', 'mixed live/offline numerical parity',
                                 'real cancellation with no partial recorded samples']}
            if args.artifacts:
                report['artifacts'] = artifact_paths
            if args.output:
                args.output.parent.mkdir(parents=True, exist_ok=True)
                args.output.write_text(json.dumps(report, indent=2) + '\n')
            print(json.dumps(report, indent=2))
    finally:
        client.close()


if __name__ == '__main__':
    main()
