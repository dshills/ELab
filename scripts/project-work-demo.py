#!/usr/bin/env python3
"""Background project-open admission, failure purity and legacy wait compatibility.
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
    return {key: state.get(key) for key in ['components', 'wires', 'document', 'view', 'result',
        'transient', 'mixed', 'npn', 'undo_entry', 'redo_entry', 'run_id', 'electrical_revision']}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    client = helper.Client(args.connect, command_preconditions=True)
    fresh = None
    try:
        directory = args.artifacts.resolve()
        directory.mkdir(parents=True, exist_ok=True)
        path = str(directory/'opened-led.elab')
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        original = client.ask('simulation.wait')
        client.ask('project.save', {'path': path, 'overwrite': True}, True)
        client.ask('component.set', {'id': 'R1', 'value': '2k'}, True)
        client.ask('simulation.restart', {}, True)
        edited = client.ask('simulation.wait')
        client.ask('project.load.start', {'path': path}, True, expect='unsaved-changes')
        started = time.perf_counter()
        admitted = client.ask('project.load.start', {'path': path, 'discard_changes': True,
            'expected_document_generation': edited['document']['generation']}, True)
        admission_ms = (time.perf_counter()-started)*1000
        work = admitted['project_work']
        assert work['status'] == 'queued' and work['document_generation'] == edited['document']['generation']
        assert protected(admitted) == protected(edited)
        opened = client.ask('simulation.wait')
        assert opened['project_work']['status'] == 'completed' and opened['project_work']['id'] == work['id']
        assert opened['project_work']['bytes'] == Path(path).stat().st_size
        assert opened['project_work']['result_document_generation'] == opened['document']['generation']
        assert opened['project_work']['result_run_id'] == opened['run_id']
        assert opened['components'] == original['components'] and opened['result']['current_a'] == original['result']['current_a']
        assert opened['paused'] and not opened['document']['dirty']
        assert opened['undo_entry']['actor'] == work['actor'] and opened['undo_entry']['command_revision'] == work['command_revision']
        before = client.ask('project.inspect')
        bad = directory/'invalid.elab'
        bad.write_text('invalid local project fixture\n')
        client.ask('project.load.start', {'path': str(bad)}, True)
        failed = client.ask('simulation.wait')
        assert failed['project_work']['status'] == 'failed' and failed['project_work']['error']['code'] == 'invalid-document'
        assert protected(failed) == protected(before)
        legacy = client.ask('project.load', {'path': path}, True)
        assert legacy['project_work']['status'] == 'completed'
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': path}, True)
        restored = fresh.ask('simulation.wait')
        assert restored['components'] == original['components'] and restored['paused'] and not restored['document']['dirty']
        assert restored['result']['current_a'] == original['result']['current_a']
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
            open_admission_ms=admission_ms, work=work, bytes=opened['project_work']['bytes'],
            checks=['unsaved-change preflight refusal', 'queued admission preserves current project',
                'bounded background reading/validation with bytes and source/result identities',
                'atomic clean fresh paused publication and original actor undo',
                'invalid file terminal outcome preserves drawing/history/readings/save association',
                'legacy waiting command and fresh-process portability'],
            limitations=['blocked-reader navigation/save/cancel/supersession are separately proven by controlled race tests',
                'OS filesystem calls may finish after cancellation; canceled work cannot install a document',
                'backend admission timing excludes visible latency', 'save/export/manual recovery still include synchronous work'])
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()
