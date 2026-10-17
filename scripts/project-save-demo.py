#!/usr/bin/env python3
"""Background captured saves, exact artifacts and legacy persistence protections.
--connect requires a fresh owned blank development session.
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


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    fresh = None
    try:
        directory = args.artifacts.resolve()
        directory.mkdir(parents=True, exist_ok=True)
        path = str(directory/'captured-led.elab')
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        original = client.ask('simulation.wait')
        started = time.perf_counter()
        admitted = client.ask('project.save.start', {'path': path, 'overwrite': True}, True)
        admission_ms = (time.perf_counter()-started)*1000
        work = admitted['project_work']
        assert work['status'] == 'queued' and work['cancelable'] and work['command'] == 'project.save'
        assert admitted['document'] == original['document'] and admitted['components'] == original['components']
        client.ask('component.set', {'id': 'R1', 'value': '2k'}, True)
        client.ask('view.pan', {'dx': 20, 'dy': 10}, True)
        inspected = client.ask('project.work.inspect', {'id': work['id']})
        assert inspected['command_revision'] == work['command_revision']
        # Captured job revisions are not the latest session command precondition.
        client.ask('view.pan', {'dx': 10, 'dy': 0}, True)
        completed = client.ask('simulation.wait')
        assert completed['project_work']['status'] == 'completed' and completed['project_work']['associated_with_current_document']
        assert completed['document']['dirty'] and completed['document']['schematic_dirty']
        data = Path(path).read_bytes()
        document = json.loads(data)
        assert next(p for p in document['graph']['components'] if p['id'] == 'R1')['value'] == 1000
        assert completed['project_work']['artifact_sha256'] == hashlib.sha256(data).hexdigest()
        assert completed['project_work']['bytes'] == len(data) and completed['project_work']['result_document_generation'] == original['document']['generation']
        client.ask('simulation.restart', {}, True)
        current = client.ask('simulation.wait')
        saved = client.ask('project.save', {}, True)
        assert saved['project_work']['status'] == 'completed' and not saved['document']['dirty']
        first_job = client.ask('project.work.inspect', {'id': work['id']})
        assert first_job['artifact_sha256'] == completed['project_work']['artifact_sha256']
        outside = b'outside file edit after completed save\n'
        Path(path).write_bytes(outside)
        client.ask('project.save', {}, True, expect='file-conflict')
        assert Path(path).read_bytes() == outside
        overwritten = client.ask('project.save', {'overwrite': True}, True)
        assert overwritten['project_work']['status'] == 'completed' and not overwritten['document']['dirty']
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': path}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['components'] == current['components'] and reopened['view'] == current['view']
        assert reopened['paused'] and not reopened['document']['dirty']
        assert reopened['result']['current_a'] == current['result']['current_a']
        temps = list(directory.glob('.elab-save-*'))
        assert not temps, temps
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
            save_admission_ms=admission_ms, admission=work, captured_artifact_sha256=completed['project_work']['artifact_sha256'],
            restored_current_a=reopened['result']['current_a'],
            checks=['immediate queued captured save leaves current association unchanged',
                'file contains admission graph while later value edits remain dirty',
                'stable operation ID/byte count/captured generation/exact artifact hash',
                'operation inspection does not regress current command preconditions',
                'legacy waiting save marks the newly captured project clean',
                'earlier operation remains inspectable after a newer save',
                'external conflict preserves outside file; explicit overwrite succeeds',
                'fresh paused offline reopening with exact graph/view/reading', 'temporary files removed'],
            limitations=['controlled race tests separately cover blocked preparation/publication, cancellation and replacement',
                'publishing saves cannot be canceled; replacement/close never associates their file with a new document',
                'OS calls can outlast cancellation/deadline', 'portable final-check/rename has no universal cross-process compare-and-swap guarantee',
                'backend timing excludes visible latency', 'complete event/native input, accessibility and release qualification remain open'])
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
