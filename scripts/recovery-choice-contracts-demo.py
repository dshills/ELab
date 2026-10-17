#!/usr/bin/env python3
"""Owned recovery reads, foreign ownership, guarded copy restore and dismissal."""
import argparse
import hashlib
import importlib.util
import json
from pathlib import Path
import time

from demo_session import connect_demo_client
from recovery_timing import settle_native_view

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
COMMANDS = ['recovery.restore', 'recovery.restore.start', 'recovery.dismiss', 'recovery.dismiss.start', 'recovery.inspect', 'recovery.samples']


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def circuit(state):
    return {k: state.get(k) for k in ['components', 'wires', 'view', 'transient', 'mixed', 'npn', 'scope',
            'paused', 'undo_entry', 'redo_entry', 'run_id', 'electrical_revision', 'document', 'operations',
            'recording', 'selected_operation', 'last_operation']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, mode=0o700, exist_ok=False)
    target = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    source = fresh = None
    work, refusals, query_errors, observations = [], [], [], []
    directory = assets / 'recovery'
    directory.mkdir(mode=0o700)

    def inspect():
        return target.ask('project.inspect')

    def terminal(identity):
        deadline = time.monotonic() + 10
        while True:
            result = target.ask('project.work.inspect', {'id': identity})
            if result['status'] not in ('queued', 'running', 'canceling'):
                work.append(result)
                return result
            assert time.monotonic() < deadline
            time.sleep(.005)

    def failure(command, params, code, admitted=False):
        before = inspect()
        target.ask(command, params, True, code)
        after = inspect()
        assert circuit(before) == circuit(after)
        if admitted:
            result = terminal(after['project_work']['id'])
            assert result['status'] == 'failed' and result['error']['code'] == code
            if command.startswith('recovery.dismiss'):
                assert not result['recovery_snapshot_removed']
        else:
            assert before == after
        refusals.append(dict(command=command, code=code, admitted=admitted))

    def pure(command, params, code=None):
        before = inspect()
        result = target.ask(command, params, expect=code)
        assert before == inspect()
        if code:
            query_errors.append(dict(command=command, code=code))
        return result

    try:
        initial = inspect()
        if initial['recovery'].get('directory'):
            target.ask('recovery.configure', {'directory': initial['recovery']['directory'], 'enabled': False, 'interval_s': 60}, True)
        source = helper.Client(command_preconditions=True)
        source.ask('example.rc', {}, True)
        source.ask('simulation.pause', {}, True)
        source.ask('simulation.restart', {}, True)
        source.ask('simulation.wait')
        source.ask('simulation.step', {'duration_s': .003}, True)
        verify = source.ask('simulation.verify', {'duration_s': .003, 'probe_ids': ['v:C1', 'i:C1']}, True)
        deadline = time.monotonic() + 10
        while True:
            original = source.ask('operation.inspect', {'id': verify['last_operation']})
            if original['operation']['status'] == 'completed':
                break
            assert original['operation']['status'] not in ('failed', 'canceled') and time.monotonic() < deadline
            time.sleep(.005)
        source.ask('simulation.wait')
        manual_source = assets / 'source.elab'
        source.ask('project.save', {'path': str(manual_source)}, True)
        source_hash = digest(manual_source)
        source.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        source.ask('recovery.configure', {'directory': str(directory), 'enabled': False, 'interval_s': 60}, True)
        source.ask('recovery.checkpoint', {}, True)
        candidate = source.ask('project.inspect')['recovery']['candidates'][0]
        choice = dict(id=candidate['id'], sha256=candidate['sha256'])
        capsule = directory / (candidate['id'] + '.elab-recovery')
        capsule_bytes = capsule.read_bytes()
        (assets / 'source-capsule.json').write_bytes(capsule_bytes)
        source.ask('verification.export', {'id': verify['last_operation'], 'format': 'json', 'path': str(assets / 'source-record.json')}, True)
        record_bytes = (assets / 'source-record.json').read_bytes()
        target.ask('example.rc', {}, True)
        target.ask('simulation.pause', {}, True)
        target.ask('simulation.restart', {}, True)
        target.ask('simulation.wait')
        target.ask('simulation.step', {'duration_s': .002}, True)
        if args.connect:
            settle_native_view(target)
            target.ask('view.pan', {'dx': 0, 'dy': 0}, True)
        manual_target = assets / 'target.elab'
        target.ask('project.save', {'path': str(manual_target)}, True)
        target_hash = digest(manual_target)
        target.ask('component.set', {'id': 'R1', 'value': '30k'}, True)
        target.ask('recovery.configure', {'directory': str(directory), 'enabled': False, 'interval_s': 60}, True)
        baseline = inspect()
        catalog = target.ask('capabilities.list')
        contracts = {c: target.ask('capabilities.list', {'command': c})['capabilities'][0]['input_schema'] for c in COMMANDS}
        assert all(s['x-command-contract']['scope'] == 'recovery-choice-and-query' for s in contracts.values())
        assert baseline == inspect()
        active = pure('recovery.inspect', dict(choice, discard_changes=True, expected_document_generation=baseline['document']['generation'] + 99))
        assert active['candidate']['owner_status'] == 'active' and active['candidate']['active_session']
        assert active['document']['graph']['components'] != baseline['components'] and active['directory'] == str(directory)
        samples = pure('recovery.samples', dict(choice, recording_index=0))
        assert samples == pure('recovery.samples', dict(choice, recording_index=0, offset=None, limit=None))
        assert samples['id'] == choice['id'] and samples['rows'] == json.loads(record_bytes)['rows']
        eof = pure('recovery.samples', dict(choice, recording_index=0, offset=samples['total'], limit=200))
        assert eof['rows'] == [] and eof['next_offset'] == eof['total'] == 4
        for command, params, code in [
                ('recovery.inspect', dict(choice, sha256='old'), 'file-conflict'),
                ('recovery.inspect', dict(choice, id='invalid'), 'invalid-recovery'),
                ('recovery.inspect', dict(choice, sha256=None), 'invalid-params'),
                ('recovery.samples', choice, 'invalid-params'),
                ('recovery.samples', dict(choice, recording_index=1), 'invalid-params'),
                ('recovery.samples', dict(choice, recording_index=0, offset=5), 'invalid-params'),
                ('recovery.samples', dict(choice, recording_index=0, limit=201), 'invalid-params'),
                ('recovery.samples', dict(choice, id='invalid', recording_index=-1, limit=201), 'invalid-recovery'),
                ('recovery.samples', dict(choice, sha256='old', recording_index=-1, limit=201), 'file-conflict'),
                ('recovery.samples', dict(choice, recording_index=None), 'invalid-params')]:
            pure(command, params, code)
        failure('recovery.restore.start', choice, 'unsaved-changes')
        failure('recovery.restore.start', dict(choice, discard_changes=True, expected_document_generation=baseline['document']['generation'] + 1), 'conflict')
        failure('recovery.restore.start', dict(choice, id=None), 'invalid-params')
        failure('recovery.restore.start', dict(choice, sha256=None), 'invalid-params')
        failure('recovery.dismiss.start', dict(choice, id='invalid'), 'invalid-params')
        failure('recovery.restore', dict(choice, id='invalid', discard_changes=True), 'invalid-recovery', True)
        failure('recovery.restore', dict(choice, sha256='old', discard_changes=True), 'file-conflict', True)
        failure('recovery.dismiss', choice, 'unavailable', True)
        failure('recovery.dismiss', dict(choice, sha256='old'), 'file-conflict', True)
        payload = json.loads(capsule_bytes)
        lease = directory / ('session-' + payload['session'] + '.json')
        lease_bytes = lease.read_bytes()
        lease.write_text('Owned damaged lease\n')
        try:
            unknown = pure('recovery.inspect', choice)
            assert unknown['candidate']['owner_status'] == 'unknown' and unknown['candidate']['active_session']
            failure('recovery.dismiss', choice, 'unavailable', True)
            before_restore = inspect()
            admitted = target.ask('recovery.restore.start', dict(choice, discard_changes=True), True)
            assert admitted['project_work']['status'] == 'queued' and circuit(admitted) == circuit(before_restore)
            restored_work = terminal(admitted['project_work']['id'])
            assert restored_work['status'] == 'completed' and restored_work['source_sha256'] == choice['sha256']
            restored = target.ask('simulation.wait')
            assert restored['document']['generation'] == before_restore['document']['generation'] + 1
            assert restored['document']['path'] == '' and restored['document']['dirty'] and restored['document']['recovered_from'] == str(manual_source)
            assert restored['paused'] and restored['transient']['sample']['time_s'] == 0
            assert next(c for c in restored['components'] if c['id'] == 'R1')['value'] == 20000
            assert restored['undo_entry']['actor'] == restored_work['actor'] and restored['undo_entry']['command_revision'] == restored_work['command_revision']
        finally:
            lease.write_bytes(lease_bytes)
        restored_id = restored['operations'][0]['id']
        target.ask('verification.export', {'id': restored_id, 'format': 'json', 'path': str(assets / 'restored-record.json')}, True)
        assert (assets / 'restored-record.json').read_bytes() == record_bytes
        copy_path = assets / 'restored-copy.elab'
        target.ask('project.save', {'path': str(copy_path)}, True)
        source.close()
        source = None
        target.ask('recovery.scan', {}, True)
        inactive = pure('recovery.inspect', choice)
        assert inactive['candidate']['owner_status'] == 'inactive' and not inactive['candidate']['active_session']
        before_second = inspect()
        target.ask('recovery.restore', choice, True)
        second_work = terminal(inspect()['project_work']['id'])
        second = target.ask('simulation.wait')
        assert second_work['status'] == 'completed' and second['document']['generation'] == before_second['document']['generation'] + 1
        assert second['paused'] and second['document']['path'] == '' and second['document']['dirty'] and second['transient']['sample']['time_s'] == 0
        assert capsule.read_bytes() == capsule_bytes
        before_dismiss = inspect()
        admitted = target.ask('recovery.dismiss.start', choice, True)
        assert admitted['project_work']['status'] == 'queued' and not admitted['project_work']['recovery_snapshot_removed']
        removed = terminal(admitted['project_work']['id'])
        assert removed['status'] == 'completed' and removed['recovery_snapshot_removed'] and removed['source_sha256'] == choice['sha256'] and not capsule.exists()
        assert circuit(inspect()) == circuit(before_dismiss)
        damaged_id = 'rec-owned-damaged'
        damaged = directory / (damaged_id + '.elab-recovery')
        damaged.write_text('{"owned_damaged_fixture":true}\n')
        (assets / 'damaged-capsule.json').write_bytes(damaged.read_bytes())
        target.ask('recovery.scan', {}, True)
        problem = next(p for p in inspect()['recovery']['problems'] if p['id'] == damaged_id)
        before_damaged = inspect()
        target.ask('recovery.dismiss', {'id': problem['id'], 'sha256': problem['sha256']}, True)
        damaged_work = terminal(inspect()['project_work']['id'])
        assert damaged_work['status'] == 'completed' and damaged_work['recovery_snapshot_removed'] and not damaged.exists()
        assert circuit(inspect()) == circuit(before_damaged)
        pure('recovery.inspect', choice, 'invalid-recovery')
        assert digest(manual_source) == source_hash and digest(manual_target) == target_hash
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': str(copy_path)}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['paused'] and not reopened['document']['dirty'] and reopened['document']['recording_count'] == 1 and reopened['transient']['sample']['time_s'] == 0
        fresh.ask('verification.export', {'id': reopened['operations'][0]['id'], 'format': 'json', 'path': str(assets / 'reopened-record.json')}, True)
        assert (assets / 'reopened-record.json').read_bytes() == record_bytes
        observations = [dict(owner_status=x['candidate']['owner_status'], active_session=x['candidate']['active_session'], current_session=x['candidate']['current_session']) for x in [active, unknown, inactive]]
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', commands=len(catalog['capabilities']),
                      contracts=contracts, admission_and_work_refusals=refusals, pure_query_errors=query_errors, work=work,
                      baseline=baseline, restored=restored, second_restored=second, final=inspect(), inspection=active,
                      samples=samples, eof=eof, ownership=observations, source_sha256=source_hash, target_sha256=target_hash,
                      capsule_sha256=hashlib.sha256(capsule_bytes).hexdigest(), record_sha256=hashlib.sha256(record_bytes).hexdigest(),
                      reopened=dict(paused=reopened['paused'], dirty=reopened['document']['dirty'], recording_count=reopened['document']['recording_count'], time_s=reopened['transient']['sample']['time_s']),
                      limits=['Controlled core race tests cover blocked workers/cancel/Close/replacement/catalog updates; no nondeterministic public race claim',
                              'No runtime/model/format change, hostile-process conditional unlink, complete physical input/AX, general storage or release qualification'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as stream:
            json.dump(report, stream, indent=2)
            stream.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], contracts=len(contracts), refusals=len(refusals), query_errors=len(query_errors), terminal_jobs=len(work))))
    finally:
        try:
            if fresh:
                fresh.close()
            if source:
                source.close()
            if args.connect:
                target.ask('app.close', {'discard_changes': True}, True)
        finally:
            target.close()


if __name__ == '__main__':
    main()
