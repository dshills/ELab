#!/usr/bin/env python3
"""Owned recovery configuration, manual work and cached-selection contracts."""
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
COMMANDS = ['recovery.configure', 'recovery.configure.start', 'recovery.work.start',
            'recovery.scan', 'recovery.checkpoint', 'recovery.select']


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def source_state(state):
    return {key: state.get(key) for key in ['components', 'wires', 'view', 'transient', 'mixed', 'npn',
            'scope', 'paused', 'undo_entry', 'redo_entry', 'run_id', 'electrical_revision', 'document',
            'operations', 'recording', 'selected_operation', 'last_operation']}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, mode=0o700, exist_ok=False)
    target = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    directory, moved = assets / 'recovery', assets / 'recovery-preserved'
    directory.mkdir(mode=0o700)
    refusals, work = [], []

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

    def raw(request):
        target.process.stdin.write(json.dumps(request) + '\n')
        target.process.stdin.flush()
        line = target.process.stdout.readline()
        assert line
        return json.loads(line)

    try:
        initial = inspect()
        if initial['recovery'].get('directory'):
            target.ask('recovery.configure', {'directory': initial['recovery']['directory'], 'enabled': False, 'interval_s': 60}, True)
        target.ask('example.rc', {}, True)
        target.ask('simulation.pause', {}, True)
        target.ask('simulation.restart', {}, True)
        target.ask('simulation.wait')
        target.ask('simulation.step', {'duration_s': .003}, True)
        capture = target.ask('simulation.verify', {'duration_s': .003, 'probe_ids': ['v:C1', 'i:C1']}, True)
        identity = capture['last_operation']
        deadline = time.monotonic() + 10
        while True:
            operation = target.ask('operation.inspect', {'id': identity})
            if operation['operation']['status'] == 'completed':
                break
            assert operation['operation']['status'] not in ('failed', 'canceled') and time.monotonic() < deadline
            time.sleep(.005)
        target.ask('simulation.wait')
        if args.connect:
            settle_native_view(target)
            target.ask('view.pan', {'dx': 0, 'dy': 0}, True)
        manual = assets / 'manual.elab'
        target.ask('project.save', {'path': str(manual)}, True)
        manual_hash = digest(manual)
        baseline = inspect()
        catalog = target.ask('capabilities.list')
        contracts = {c: target.ask('capabilities.list', {'command': c})['capabilities'][0]['input_schema'] for c in COMMANDS}
        assert all(s['x-command-contract']['scope'] == 'recovery-controls-and-work' for s in contracts.values())
        assert baseline == inspect()
        for command, params in [
                ('recovery.select', {'id': None}), ('recovery.select', {'id': 'unknown'}),
                ('recovery.configure.start', {'directory': str(directory), 'enabled': False, 'interval_s': 0}),
                ('recovery.configure', {'directory': str(directory), 'enabled': None, 'interval_s': 2}),
                ('recovery.work.start', {'kind': 'restore'}), ('recovery.work.start', {'kind': None}),
                ('recovery.scan', {'extra': True}), ('recovery.checkpoint', {'extra': True})]:
            before = inspect()
            target.ask(command, params, True, 'invalid-params')
            assert before == inspect()
            refusals.append({'command': command, 'params': params, 'code': 'invalid-params'})
        admitted = target.ask('recovery.configure.start', {'directory': str(directory), 'enabled': False, 'interval_s': '60 s'}, True)
        assert admitted['project_work']['status'] == 'queued' and not admitted['project_work']['recovery_configuration_applied']
        configured = terminal(admitted['project_work']['id'])
        assert configured['status'] == 'completed' and configured['recovery_configuration_applied'] and configured['recovery_catalog_applied']
        assert source_state(inspect()) == source_state(baseline)
        target.ask('component.set', {'id': 'R1', 'value': '20k'}, True)
        dirty = inspect()
        assert dirty['document']['dirty'] and not dirty['recovery']['enabled']
        admitted = target.ask('recovery.work.start', {'kind': 'checkpoint'}, True)
        assert admitted['project_work']['status'] == 'queued'
        checkpoint = terminal(admitted['project_work']['id'])
        assert checkpoint['status'] == 'completed' and checkpoint['result_recovery_id'] and checkpoint['recovery_catalog_applied']
        state = inspect()
        assert source_state(state) == source_state(dirty)
        candidate = state['recovery']['candidates'][0]
        capsule = directory / (candidate['id'] + '.elab-recovery')
        capsule_raw = capsule.read_bytes()
        assert digest(capsule) == candidate['sha256']
        # Preserve exact capsule bytes under a tracked extension before retirement.
        (assets / 'checkpoint-capsule.json').write_bytes(capsule_raw)
        target.ask('recovery.select', {'id': ''}, True)
        target.ask('recovery.checkpoint', {}, True)
        dedup = terminal(inspect()['project_work']['id'])
        assert dedup['status'] == 'completed' and not dedup.get('result_recovery_id')
        assert inspect()['recovery']['selected'] == '' and len(inspect()['recovery']['candidates']) == 1
        assert capsule.read_bytes() == capsule_raw
        admitted = target.ask('recovery.work.start', {'kind': 'scan'}, True)
        scanned = terminal(admitted['project_work']['id'])
        assert scanned['status'] == 'completed' and scanned['command'] == 'recovery.scan' and scanned['recovery_catalog_applied']
        target.ask('recovery.scan', {}, True)
        assert terminal(inspect()['project_work']['id'])['status'] == 'completed'
        assert inspect()['recovery']['selected'] == ''
        healthy = inspect()
        directory.rename(moved)
        directory.write_text('Owned unavailable directory fixture\n')
        try:
            target.ask('recovery.select', {'id': candidate['id']}, True)
            target.ask('recovery.select', {'id': ''}, True)
            before = inspect()
            for command in ['recovery.configure', 'recovery.configure.start']:
                result = target.ask(command, {'directory': str(directory), 'enabled': False, 'interval_s': '1500 ms'}, True)
                installed = terminal(result['project_work']['id'])
                assert installed['status'] == 'completed' and installed['recovery_configuration_applied'] and not installed['recovery_catalog_applied']
                state = inspect()
                assert state['recovery']['candidates'] == before['recovery']['candidates'] and state['recovery']['selected'] == ''
                assert not state['recovery']['enabled'] and state['recovery']['interval_s'] == 1.5
                assert source_state(state) == source_state(dirty)
            before = inspect()
            request = dict(version=1, id='scan-first', command='recovery.scan', params={},
                           expected_command_revision=before['command_revision'], key='owned-recovery-scan-failure')
            failed = raw(request)
            assert failed['error']['code'] == 'file-error'
            failed_id = failed['result']['project_work']['id']
            failed_work = terminal(failed_id)
            assert failed_work['status'] == 'failed' and source_state(inspect()) == source_state(dirty)
            assert inspect()['recovery'] == before['recovery']
        finally:
            directory.unlink()
            moved.rename(directory)
        retry = raw(dict(request, id='scan-retry'))
        assert retry['duplicate'] and retry['error']['code'] == 'file-error' and retry['result']['project_work']['id'] == failed_id
        corrected = inspect()
        repaired = raw(dict(request, id='scan-repaired', key='owned-recovery-scan-corrected', expected_command_revision=corrected['command_revision']))
        assert 'error' not in repaired and not repaired.get('duplicate')
        assert terminal(inspect()['project_work']['id'])['status'] == 'completed'
        assert inspect()['recovery']['selected'] == '' and capsule.read_bytes() == capsule_raw
        assert healthy['recovery']['candidates'] == inspect()['recovery']['candidates']
        target.ask('component.set', {'id': 'R1', 'value': '10k'}, True)
        assert not inspect()['document']['dirty']
        restored = inspect()
        target.ask('recovery.checkpoint', {}, True)
        retired = terminal(inspect()['project_work']['id'])
        final = inspect()
        assert retired['status'] == 'completed' and not retired.get('result_recovery_id')
        assert not final['recovery']['candidates'] and final['recovery']['selected'] == '' and not capsule.exists()
        assert final['recovery']['last_snapshot'] == candidate['id']  # Historical, not file presence.
        assert digest(manual) == manual_hash and final['document']['recording_count'] == 1
        assert source_state(final) == source_state(restored)
        # Electrical edits advance the requested run and mark accepted data stale;
        # recovery must preserve that state, while the accepted sample stays put.
        assert final['transient']['sample'] == baseline['transient']['sample']
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      commands=len(catalog['capabilities']), contracts=contracts, admission_refusals=refusals,
                      work=work, baseline=baseline, dirty=dirty, restored=restored, final=final, capsule_sha256=hashlib.sha256(capsule_raw).hexdigest(),
                      manual_sha256=manual_hash, metadata=operation['metadata'],
                      rows=target.ask('operation.samples', {'id': identity, 'limit': 100})['rows'],
                      retry=dict(failed_code='file-error', same_original_job=True, duplicate_failure_after_repair=True, new_key_succeeded=True),
                      checks=['Immediate and waiting configuration/scan/checkpoint share work metadata',
                              'Manual checkpoint while disabled; unchanged capture deduplicates',
                              'Cached select and both same-store disable aliases work without folder access',
                              'Explicit clear selection survives scan, dedup and same-store settings',
                              'Failed waiting scan retry replays original outcome after storage repair',
                              'Restored saved content retires owned recovery without touching manual file',
                              'Historical last_snapshot survives retirement; graph/live/capture state preserved'],
                      limits=['Publication/cancel/Close/replacement and blocked-worker interleavings use controlled core tests',
                              'No runtime/model/format change, real-device storage, complete physical input/AX or release qualification'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as stream:
            json.dump(report, stream, indent=2)
            stream.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], contracts=len(contracts),
                             admission_refusals=len(refusals), terminal_jobs=len(work), rows=len(report['rows']))))
    finally:
        try:
            if args.connect:
                target.ask('app.close', {'discard_changes': True}, True)
        finally:
            target.close()


if __name__ == '__main__':
    main()
