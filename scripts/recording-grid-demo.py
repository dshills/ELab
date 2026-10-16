#!/usr/bin/env python3
"""Qualify complete recording grids through shared import and project commands."""
import argparse
import copy
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
    observations = []

    def source(state):
        return {key: state.get(key) for key in ('document', 'components', 'wires', 'transient', 'run_id',
                                               'electrical_revision', 'undo_entry', 'recording', 'operations')}

    def refuse_file(command, path, code, extra=None):
        before = client.ask('project.inspect')
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        client.ask(command, dict(path=str(path), **(extra or {})), True, code)
        after = client.ask('project.inspect')
        assert source(before) == source(after), command
        assert after['project_work']['status'] == 'failed'
        assert 'missing a requested output step' in after['project_work']['error']['message']
        assert hashlib.sha256(path.read_bytes()).hexdigest() == digest

    try:
        contract = client.ask('capabilities.list', {'command': 'scope.window.set'})
        # The visible-window contract distinguishes plot membership, requested
        # time admission and the stricter portable recording-grid check.
        assert 'required output spacing' in json.dumps(contract)
        for i, duration in enumerate((1e-20, 1e-12, .001 + .5e-12, math.nextafter(.001, math.inf), .002)):
            client.ask('example.rc', {'discard_changes': True}, True)
            client.ask('simulation.pause', {}, True)
            client.ask('simulation.wait')
            started = client.ask('simulation.verify', dict(duration_s=duration, input_mode='charge', probe_ids=['v:C1']), True)
            identity = started['last_operation']
            deadline = time.monotonic() + 10
            while True:
                operation = client.ask('operation.inspect', {'id': identity})['operation']
                assert operation['status'] not in ('failed', 'canceled'), operation
                if operation['status'] == 'completed':
                    break
                assert time.monotonic() < deadline
                time.sleep(.01)
            complete_path = assets / f'complete-{i}.json'
            client.ask('verification.export', dict(id=identity, path=str(complete_path), format='json'), True)
            complete = json.loads(complete_path.read_text())
            assert complete['rows'][-1]['time_s'] == duration
            client.ask('verification.import', {'path': str(complete_path)}, True)
            assert client.ask('project.inspect')['recording']['last_sample_time_s'] == duration
            incomplete = copy.deepcopy(complete)
            incomplete['rows'].pop()
            metadata = incomplete['metadata']
            metadata['sample_count'] = len(incomplete['rows'])
            for channel, statistic in enumerate(metadata['statistics']):
                values = [row['values'][channel] for row in incomplete['rows']]
                statistic.update(count=len(values), min=min(values), max=max(values),
                                 sample_mean=sum(values) / len(values), sample_rms=math.sqrt(sum(v*v for v in values) / len(values)))
            bad_path = assets / f'missing-final-{i}.json'
            bad_path.write_text(json.dumps(incomplete) + '\n')
            refuse_file('verification.import', bad_path, 'invalid-recording')
            project_path = assets / f'embedded-{i}.elab'
            client.ask('project.save', {'path': str(project_path)}, True)
            project = json.loads(project_path.read_text())
            for entry in project['recordings']['entries']:
                entry['data'] = incomplete
            bad_project = assets / f'missing-final-{i}.elab'
            bad_project.write_text(json.dumps(project) + '\n')
            refuse_file('project.load', bad_project, 'invalid-document', {'discard_changes': True})
            observations.append(dict(duration_s=duration, complete_times_s=[row['time_s'] for row in complete['rows']],
                                     incomplete_times_s=[row['time_s'] for row in incomplete['rows']]))
        rounded = copy.deepcopy(complete)
        rounded['rows'][-1]['time_s'] -= 5e-13
        rounded_path = assets / 'ordinary-rounded.json'
        rounded_path.write_text(json.dumps(rounded) + '\n')
        client.ask('verification.import', {'path': str(rounded_path)}, True)
        view = client.ask('project.inspect')['recording']
        assert view['last_sample_time_s'] == rounded['rows'][-1]['time_s']
        whole = client.ask('recording.window.set', dict(start_s=0, end_s=view['last_sample_time_s']), True)['recording']
        assert whole['window_sample_count'] == whole['sample_count'] == len(rounded['rows'])
        reexported = assets / 'ordinary-rounded-copy.json'
        client.ask('verification.export', dict(id=view['id'], path=str(reexported), format='json'), True)
        assert json.loads(reexported.read_text()) == rounded
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', observations=observations,
                      rounded_endpoint_s=view['last_sample_time_s'],
                      checks=['Complete generated tiny/final-partial grids import successfully',
                              'Missing final row refuses despite consistent reduced count/statistics',
                              'One-ULP final partial step remains distinct from preceding sample',
                              'Both standalone recording import and embedded project load use the same validation',
                              'Ordinary rounded timestamps import/re-export unchanged and complete-window controls retain every row',
                              'Failed loads preserve source, viewer, retained operations and file bytes'],
                      limits=['Recorded-file structure/grid validation does not establish physical accuracy',
                              'Ordinary decimal timestamp tolerance is retained subject to local grid spacing',
                              'No interpolation, re-solve, saved format or numerical kernel change'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as output:
            json.dump(report, output, indent=2)
            output.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], cases=len(observations))))
    finally:
        try:
            if args.connect:
                client.ask('project.inspect')
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()
