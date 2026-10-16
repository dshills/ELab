#!/usr/bin/env python3
"""Background recorded import, provenance, strict failure and kept portability.
--connect replaces only the designated owned development session.
"""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def live(state):
    return {key: state.get(key) for key in ['components', 'wires', 'view', 'transient', 'mixed',
        'npn', 'undo_entry', 'redo_entry', 'run_id', 'electrical_revision']}


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=True)
    source = helper.Client(command_preconditions=True)
    target = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    fresh = None
    try:
        source.ask('example.rc', {}, True)
        source.ask('simulation.pause', {}, True)
        accepted = source.ask('simulation.verify', {'duration_s': .01, 'probe_ids': ['v:C1', 'i:C1']}, True)
        original_id = accepted['last_operation']
        deadline = time.monotonic()+10
        while True:
            original = source.ask('operation.inspect', {'id': original_id})
            if original['operation']['status'] not in ['queued', 'running', 'canceling']:
                break
            assert time.monotonic() < deadline
            time.sleep(.01)
        assert original['operation']['status'] == 'completed'
        path = str(directory/'source-record.json')
        source.ask('verification.export', {'id': original_id, 'path': path, 'format': 'json', 'overwrite': True}, True)
        target.ask('example.rc', {'discard_changes': True}, True)
        target.ask('simulation.pause', {}, True)
        target.ask('simulation.restart', {}, True)
        target.ask('simulation.wait')
        target.ask('simulation.step', {'duration_s': .005}, True)
        before = target.ask('project.inspect')
        began = time.perf_counter()
        admitted = target.ask('verification.import.start', {'path': path,
            'expected_document_generation': before['document']['generation']}, True)
        admission_ms = (time.perf_counter()-began)*1000
        work_id = admitted['project_work']['id']
        assert admitted['project_work']['status'] == 'queued' and live(admitted) == live(before)
        assert admitted['document']['recording_count'] == 0
        target.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        target.ask('view.pan', {'dx': 20, 'dy': 10}, True)
        edited = target.ask('project.inspect')
        completed = target.ask('simulation.wait')
        work = target.ask('project.work.inspect', {'id': work_id})
        assert work['status'] == 'completed' and not work['cancelable'] and work['viewer_selected']
        assert live(completed) == live(edited)
        assert completed['recording']['id'] == work['result_operation_id'] and completed['recording']['imported']
        assert completed['document']['recording_count'] == 1 and completed['document']['recordings_dirty']
        imported = target.ask('operation.inspect', {'id': work['result_operation_id']})
        assert imported['metadata'] == original['metadata'] and not imported['operation']['source_unchanged']
        digest = hashlib.sha256(Path(path).read_bytes()).hexdigest()
        assert work['source_sha256'] == digest and imported['operation']['source_sha256'] == digest
        copy_path = str(directory/'reexported-record.json')
        target.ask('verification.export', {'id': work['result_operation_id'], 'path': copy_path, 'format': 'json', 'overwrite': True}, True)
        assert Path(path).read_bytes() == Path(copy_path).read_bytes()
        protected = target.ask('project.inspect')
        bad = directory/'invalid-record.json'
        bad.write_text('invalid recorded data\n')
        target.ask('verification.import.start', {'path': str(bad)}, True)
        failed = target.ask('simulation.wait')
        assert failed['project_work']['status'] == 'failed' and failed['project_work']['error']['code'] == 'invalid-recording'
        assert live(failed) == live(protected) and failed['document'] == protected['document']
        assert failed['recording'] == protected['recording'] and failed['operations'] == protected['operations']
        target.ask('simulation.restart', {}, True)
        target.ask('simulation.wait')
        saved_path = str(directory/'imported-project.elab')
        target.ask('project.save', {'path': saved_path, 'overwrite': True}, True)
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': saved_path}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['paused'] and not reopened['document']['dirty'] and reopened['document']['recording_count'] == 1
        assert reopened['transient']['sample']['time_s'] == 0 and reopened['recording']['viewing']
        assert fresh.ask('operation.inspect', {'id': reopened['recording']['id']})['metadata'] == original['metadata']
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
            import_admission_ms=admission_ms, source_sha256=digest, result_operation_id=work['result_operation_id'],
            checks=['separate process produces immutable recorded source', 'queued import preserves live source and publishes no staged record',
                'later value/navigation edits preserved on complete attachment', 'source hash and original metadata/rows retained',
                'import never claims current live source association', 'byte-identical reexport',
                'invalid file records only terminal job bookkeeping', 'kept project round trip with recorded viewer and fresh paused live startup'],
            limitations=['blocked race tests separately prove cancellation, capacity races, replacement and later-viewer ownership',
                'OS calls may outlast cooperative cancellation/deadline', 'backend timing excludes visible latency',
                'manual recovery and some shutdown paths remain synchronous'])
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        source.close()
        target.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()
