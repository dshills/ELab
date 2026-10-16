#!/usr/bin/env python3
"""Background recovery scan/checkpoint and preserved saved/edited project data.
--connect replaces only the designated owned development session.
"""
import argparse
import importlib.util
import json
from pathlib import Path
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def protected(state):
    return {key: state.get(key) for key in ['components', 'wires', 'document', 'view', 'transient',
        'mixed', 'npn', 'undo_entry', 'redo_entry', 'run_id', 'electrical_revision']}


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=True)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        saved_path = str(directory/'saved-source.elab')
        client.ask('project.save', {'path': saved_path, 'overwrite': True}, True)
        saved_bytes = Path(saved_path).read_bytes()
        # Keep its isolated default directory for native sessions; headless
        # clients opt into a private owned directory through the same command.
        recovery = client.ask('project.inspect')['recovery']
        recovery_directory = recovery.get('directory') or str(directory/'recovery')
        client.ask('recovery.configure', {'directory': recovery_directory, 'enabled': False, 'interval_s': 1}, True)
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        before = client.ask('project.inspect')
        started = time.perf_counter()
        admitted = client.ask('recovery.work.start', {'kind': 'checkpoint'}, True)
        admission_ms = (time.perf_counter()-started)*1000
        work_id = admitted['project_work']['id']
        assert admitted['project_work']['status'] == 'queued' and admitted['project_work']['cancelable']
        assert protected(admitted) == protected(before)
        client.ask('component.set', {'id': 'R1', 'value': '30k'}, True)
        client.ask('view.pan', {'dx': 20, 'dy': 10}, True)
        edited = client.ask('project.inspect')
        completed = client.ask('simulation.wait')
        work = client.ask('project.work.inspect', {'id': work_id})
        assert work['status'] == 'completed' and not work['cancelable'] and work['result_recovery_id']
        assert protected(completed) == protected(edited) and Path(saved_path).read_bytes() == saved_bytes
        candidates = client.ask('recovery.scan', {}, True)['recovery']['candidates']
        candidate = next(c for c in candidates if c['id'] == work['result_recovery_id'])
        inspection = client.ask('recovery.inspect', {'id': candidate['id'], 'sha256': candidate['sha256']})
        assert next(p for p in inspection['document']['graph']['components'] if p['id'] == 'R1')['value'] == 20000
        assert client.ask('project.inspect')['document']['dirty']
        before_scan = client.ask('project.inspect')
        scan = client.ask('recovery.work.start', {'kind': 'scan'}, True)
        assert scan['project_work']['status'] == 'queued'
        client.ask('recovery.select', {'id': ''}, True)
        scanned = client.ask('simulation.wait')
        assert scanned['project_work']['status'] == 'completed' and scanned['recovery']['selected'] == ''
        assert protected(scanned) == protected(before_scan)
        client.ask('recovery.checkpoint', {}, True)
        current = client.ask('project.inspect')
        newest = next(c for c in current['recovery']['candidates'] if c['current_session'])
        new_inspection = client.ask('recovery.inspect', {'id': newest['id'], 'sha256': newest['sha256']})
        assert next(p for p in new_inspection['document']['graph']['components'] if p['id'] == 'R1')['value'] == 30000
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        final = client.ask('project.save', {'path': str(directory/'current-source.elab'), 'overwrite': True}, True)
        assert final['paused'] and not final['document']['dirty']
        assert Path(saved_path).read_bytes() == saved_bytes
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
            checkpoint_admission_ms=admission_ms, captured_recovery_id=work['result_recovery_id'],
            checks=['immediate checkpoint admission preserves editor', 'original captured value versus later editable value',
                'explicit saved file never overwritten', 'shared stable job inspection and capsule identity',
                'background scan preserves explicit cleared selection', 'legacy scan/checkpoint complete through same workers',
                'current capture includes later edits; final project is durable/clean/paused'],
            limitations=['blocked race tests separately prove cancellation, publication fencing and newer-list preservation',
                'OS calls can outlast cooperative cancellation/deadline', 'backend timing excludes visible latency',
                'recovery configuration/restore/dismiss/inspection and some shutdown paths remain synchronous'])
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()


if __name__ == '__main__':
    main()
