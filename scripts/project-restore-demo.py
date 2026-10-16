#!/usr/bin/env python3
"""Exercise background recovery Restore through the shared public API.
--connect replaces only a designated owned development session.
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
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=True)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        source = directory/'saved-source.elab'
        client.ask('project.save', {'path': str(source), 'overwrite': True}, True)
        source_bytes = source.read_bytes()
        recovery = client.ask('project.inspect')['recovery']
        client.ask('recovery.configure', {'directory': recovery.get('directory') or str(directory/'recovery'),
            'enabled': False, 'interval_s': 1}, True)
        client.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        captured = client.ask('recovery.checkpoint', {}, True)
        recovery_id = captured['project_work']['result_recovery_id']
        candidate = next(c for c in captured['recovery']['candidates'] if c['id'] == recovery_id)
        client.ask('component.set', {'id': 'R1', 'value': '30k'}, True)
        before = client.ask('project.inspect')
        choice = {'id': candidate['id'], 'sha256': candidate['sha256'],
            'expected_document_generation': before['document']['generation']}
        client.ask('recovery.restore.start', choice, True, expect='unsaved-changes')
        choice['discard_changes'] = True
        started = time.perf_counter()
        admitted = client.ask('recovery.restore.start', choice, True)
        admission_ms = (time.perf_counter()-started)*1000
        job_id = admitted['project_work']['id']
        assert admitted['project_work']['status'] == 'queued'
        assert admitted['document'] == before['document'] and admitted['components'] == before['components']
        restored = client.ask('simulation.wait')
        work = client.ask('project.work.inspect', {'id': job_id})
        assert work['status'] == 'completed' and not work['cancelable']
        assert work['source_sha256'] == candidate['sha256'] and work['result_recovery_id'] == candidate['id']
        assert restored['document']['generation'] == before['document']['generation']+1
        assert restored['document']['dirty'] and restored['document']['path'] == ''
        assert restored['document']['recovered_from'] == str(source)
        assert next(p for p in restored['components'] if p['id'] == 'R1')['value'] == 20000
        assert restored['run_id'] == before['run_id']+1 and restored['paused']
        assert restored['transient']['sample']['time_s'] == 0
        assert restored['undo_entry']['command_revision'] == work['command_revision']
        assert source.read_bytes() == source_bytes
        final = client.ask('project.save', {'path': str(directory/'restored-copy.elab'), 'overwrite': True}, True)
        assert not final['document']['dirty'] and final['paused']
        assert source.read_bytes() == source_bytes
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
            restore_admission_ms=admission_ms, job_id=job_id, source_sha256=hashlib.sha256(source_bytes).hexdigest(),
            checks=['dirty/generation protection before admission', 'immediate admission leaves original state intact',
                'completed stable job/hash/capsule identity', 'fresh paused zero-time unassociated dirty copy',
                'original admitted history attribution', 'separate durable saved copy; original file byte-preserved'],
            limitations=['blocked race tests separately exercise cancellation and intervening edits/replacement',
                'OS reads may outlast cancellation/deadline; no stale result can commit',
                'backend admission timing excludes visible latency',
                'recovery configuration/dismiss/inspection and shutdown remain separate responsiveness work'])
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()


if __name__ == '__main__':
    main()
