#!/usr/bin/env python3
"""Exercise captured curve sharing on headless stdio or an owned native socket."""
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
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    cases = []
    try:
        for example, source, probes in [('led', 'V1', ['i:D1', 'v:D1']),
                                        ('rc', 'V1', ['v:C1', 'i:C1']),
                                        ('npn', 'VIN', ['i:Q1.base', 'i:Q1.collector'])]:
            command = 'example.reset' if example == 'led' else 'example.' + example
            client.ask(command, {'discard_changes': True}, True)
            client.ask('simulation.pause', {}, True)
            client.ask('simulation.wait')
            client.ask('sweep.start', {'settings': {'component_id': source, 'parameter': 'voltage',
                       'start': '0 V', 'stop': '5 V', 'points': 11, 'probe_ids': probes}}, True)
            deadline = time.monotonic() + 15
            while True:
                state = client.ask('project.inspect')
                sweep = state['dc_sweep']
                op = next(op for op in sweep['operations'] if op['id'] == sweep['selected_operation'])
                assert op['status'] not in ('failed', 'canceled'), op
                if op['status'] == 'completed':
                    break
                assert time.monotonic() < deadline
                time.sleep(.01)
            original = client.ask('sweep.inspect', {'id': op['id']})
            path = directory / (example + '.json')
            client.ask('sweep.export', {'id': op['id'], 'path': str(path), 'format': 'json'}, True)
            client.ask('project.clear', {'discard_changes': True}, True)
            before = client.ask('project.inspect')
            imported = client.ask('sweep.import', {'path': str(path)}, True)
            for key in ('components', 'wires', 'run_id', 'electrical_revision', 'result', 'undo_entry'):
                assert before.get(key) == imported.get(key), key
            work = imported['project_work']
            curve = client.ask('sweep.inspect', {'id': work['result_operation_id']})
            assert curve['result'] == original['result'] and curve['metadata'] == original['metadata']
            assert curve['operation']['kind'] == 'sweep-import' and not curve['operation']['source_unchanged']
            assert curve['operation']['source_sha256'] == hashlib.sha256(path.read_bytes()).hexdigest()
            client.ask('sweep.trace.select', {'id': probes[-1]}, True)
            client.ask('sweep.cursor.set', {'index': 5, 'operation_id': curve['operation']['id']}, True)
            exported = directory / (example + '-copy.json')
            client.ask('sweep.export', {'id': curve['operation']['id'], 'path': str(exported), 'format': 'json'}, True)
            assert json.loads(exported.read_text()) == json.loads(path.read_text())
            project = directory / (example + '.elab')
            client.ask('project.save', {'path': str(project)}, True)
            client.ask('project.clear', {'discard_changes': True}, True)
            client.ask('project.load', {'path': str(project)}, True)
            restored = client.ask('simulation.wait')
            assert not restored['document']['dirty'] and len(restored['components']) == 0
            assert restored['dc_sweep']['result'] == original['result']
            assert restored['dc_sweep']['trace_id'] == probes[-1] and restored['dc_sweep']['cursor_index'] == 5
            client.ask('verification.import', {'path': str(path)}, True, 'invalid-recording')
            assert client.ask('project.inspect')['dc_sweep'] == restored['dc_sweep']
            cases.append({'example': example, 'result': curve['result'], 'metadata': curve['metadata'],
                          'digest': curve['operation']['source_sha256'], 'cursor_index': 5,
                          'source_unchanged': False, 'kept_round_trip': True})
        client.ask('project.clear', {'discard_changes': True}, True)
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'cases': cases, 'checks': ['exact imported metadata/results', 'unchanged blank live circuit/history',
                  'file digest', 'recorded provenance', 'exact reexport', 'trace/cursor save and reopen',
                  'DC data refused by time-recording importer']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as target:
            target.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps({'passed': True, 'transport': report['transport'], 'cases': len(cases)}))
    finally:
        client.close()


if __name__ == '__main__':
    main()
