#!/usr/bin/env python3
"""Crash an owned process after record-only edits and restore its complete experiment."""
import argparse
import importlib.util
import json
from pathlib import Path
import tempfile
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--output', type=Path)
    args = parser.parse_args()
    first = helper.Client()
    second = None
    try:
        with tempfile.TemporaryDirectory(prefix='elab-record-only-crash-') as temporary:
            directory = Path(temporary)
            first.ask('example.rc', {}, True)
            first.ask('simulation.pause', {}, True)
            source_path = directory / 'saved.elab'
            first.ask('project.save', {'path': str(source_path)}, True)
            original = source_path.read_bytes()
            first.ask('recovery.configure', {'directory': str(directory / 'recovery'), 'enabled': True, 'interval_s': 1}, True)
            accepted = first.ask('simulation.verify', {'duration_s': .02, 'probe_ids': ['v:C1']}, True)
            deadline = time.monotonic() + 5
            candidate = None
            while time.monotonic() < deadline:
                state = first.ask('project.inspect')
                if state['recovery']['candidates']:
                    candidate = state['recovery']['candidates'][0]
                    break
                time.sleep(.05)
            assert candidate and candidate['recording_count'] == 1
            assert state['document']['recordings_dirty'] and not state['document']['schematic_dirty']
            captured = first.ask('operation.inspect', {'id': accepted['last_operation']})['metadata']
            first.process.kill()
            first.process.wait(timeout=5)
            first.process.stdin.close()
            assert first.process.returncode != 0
            second = helper.Client()
            second.ask('recovery.configure', {'directory': str(directory / 'recovery'), 'enabled': False, 'interval_s': 1}, True)
            candidate = second.ask('project.inspect')['recovery']['candidates'][0]
            assert not candidate['active_session'] and candidate['recording_count'] == 1
            choice = {'id': candidate['id'], 'sha256': candidate['sha256']}
            inspection = second.ask('recovery.inspect', choice)
            assert inspection['document']['recordings'][0] == captured
            page = second.ask('recovery.samples', dict(**choice, recording_index=0, offset=10, limit=1))
            assert page['total'] == 21 and page['rows'][0]['time_s'] == .01
            second.ask('recovery.restore', choice, True)
            restored = second.ask('simulation.wait')
            assert restored['document']['dirty'] and not restored['document']['path']
            assert restored['document']['recording_count'] == 1 and restored['paused']
            assert restored['transient']['sample']['time_s'] == 0
            assert second.ask('operation.inspect', {'id': restored['last_operation']})['metadata'] == captured
            second.ask('project.save', {'path': str(directory / 'recovered.elab')}, True)
            assert source_path.read_bytes() == original
            report = dict(passed=True, crash_exit_code=first.process.returncode, embedded_samples=21,
                          live_restart_time_s=0, explicit_saved_file_preserved=True,
                          checks=['record-only dirty autosave', 'actual SIGKILL', 'inactive crashed lease',
                                  'bounded recovery metadata/rows', 'complete captured asset restore',
                                  'fresh paused live startup', 'unassociated copy / Save As', 'original file unchanged'])
            if args.output:
                args.output.parent.mkdir(parents=True, exist_ok=True)
                args.output.write_text(json.dumps(report, indent=2) + '\n')
            print(json.dumps(report, indent=2))
    finally:
        first.close()
        if second:
            second.close()


if __name__ == '__main__':
    main()
