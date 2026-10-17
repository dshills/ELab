#!/usr/bin/env python3
"""Validate guarded recovery dismissal through owned public sessions and files."""
import argparse
import copy
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
    root = args.artifacts.resolve()
    root.mkdir(mode=0o700, parents=True, exist_ok=False)
    directory = root / 'recovery'
    directory.mkdir(mode=0o700)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    owner = helper.Client(command_preconditions=True)
    copied = None
    ownership_states = {}

    def live(state):
        return {k: copy.deepcopy(state.get(k)) for k in ('components', 'wires', 'run_id', 'electrical_revision',
                'result', 'transient', 'mixed', 'npn', 'document', 'undo_entry', 'view')}

    def completed(operation_id):
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline:
            state = client.ask('project.inspect')
            work = state['project_work']
            assert work['id'] == operation_id and work['status'] not in ('failed', 'canceled', 'superseded'), work
            if work['status'] == 'completed':
                return state
            time.sleep(.01)
        raise AssertionError('Dismissal did not complete')

    try:
        owner.ask('example.reset', {'discard_changes': True}, True)
        owner.ask('simulation.pause', {}, True)
        owner.ask('simulation.wait')
        saved = root / 'owner-source.elab'
        owner.ask('project.save', {'path': str(saved)}, True)
        saved_digest = hashlib.sha256(saved.read_bytes()).hexdigest()
        owner.ask('component.set', {'id': 'R1', 'value': '2k'}, True)
        owner.ask('recovery.configure', {'directory': str(directory), 'enabled': False, 'interval_s': 2}, True)
        owner.ask('recovery.checkpoint', {}, True)
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': '1 ms'}, True)
        client.ask('recovery.configure', {'directory': str(directory), 'enabled': False, 'interval_s': 2}, True)
        state = client.ask('project.inspect')
        candidate = state['recovery']['candidates'][0]
        assert candidate['active_session'] and not candidate['current_session'] and candidate['owner_status']=='active'
        ownership_states['active']=candidate
        choice = {'id': candidate['id'], 'sha256': candidate['sha256']}
        capsule = directory / (candidate['id'] + '.elab-recovery')
        capsule_digest = hashlib.sha256(capsule.read_bytes()).hexdigest()
        before = client.ask('project.inspect')
        client.ask('recovery.dismiss', choice, True, 'unavailable')
        protected = client.ask('project.inspect')
        assert live(protected) == live(before) and not protected['project_work']['recovery_snapshot_removed']
        assert hashlib.sha256(capsule.read_bytes()).hexdigest() == capsule_digest
        capsule_data = json.loads(capsule.read_text())
        lease = directory / ('session-'+capsule_data['session']+'.json')
        original_lease = lease.read_bytes()
        assert json.loads(original_lease)['pid']==owner.process.pid
        lease.write_text('owned deliberately malformed lease')
        client.ask('recovery.scan', {}, True)
        unknown=client.ask('project.inspect')['recovery']['candidates'][0]
        assert unknown['owner_status']=='unknown' and unknown['active_session'] and not unknown['current_session']
        ownership_states['unknown']=unknown
        inspected=client.ask('recovery.inspect',choice)
        assert inspected['candidate']['owner_status']=='unknown' and inspected['candidate']['active_session']
        before=client.ask('project.inspect')
        client.ask('recovery.dismiss',choice,True,'unavailable')
        refused_unknown=client.ask('project.inspect')
        assert live(refused_unknown)==live(before) and not refused_unknown['project_work']['recovery_snapshot_removed']
        assert hashlib.sha256(capsule.read_bytes()).hexdigest()==capsule_digest
        copied=helper.Client(command_preconditions=True)
        copied.ask('recovery.configure',{'directory':str(directory),'enabled':False,'interval_s':2},True)
        copied.ask('recovery.restore',choice,True)
        copy_state=copied.ask('simulation.wait')
        assert copy_state['components']==inspected['document']['graph']['components']
        assert copy_state['wires']==inspected['document']['graph']['wires']
        assert copy_state['document']['dirty'] and not copy_state['document']['path']
        assert copy_state['document']['recovered_from']==str(saved)
        assert hashlib.sha256(capsule.read_bytes()).hexdigest()==capsule_digest
        assert hashlib.sha256(saved.read_bytes()).hexdigest()==saved_digest
        copied.close();copied=None
        lease.write_bytes(original_lease)
        client.ask('recovery.scan',{},True)
        repaired=client.ask('project.inspect')['recovery']['candidates'][0]
        assert repaired['owner_status']=='active' and repaired['active_session']
        ownership_states['repaired']=repaired
        owner.close()
        owner = None
        client.ask('recovery.scan', {}, True)
        inactive = client.ask('project.inspect')['recovery']['candidates'][0]
        assert not inactive['active_session'] and inactive['sha256'] == capsule_digest and inactive['owner_status']=='inactive'
        ownership_states['inactive']=inactive
        before = client.ask('project.inspect')
        admitted = client.ask('recovery.dismiss.start', choice, True)
        assert admitted['project_work']['status'] == 'queued' and admitted['project_work']['cancelable']
        assert not admitted['project_work']['recovery_snapshot_removed']
        removed = completed(admitted['project_work']['id'])
        assert removed['project_work']['recovery_snapshot_removed'] and removed['project_work']['result_recovery_id'] == choice['id']
        assert removed['project_work']['source_sha256'] == choice['sha256']
        assert not capsule.exists() and live(removed) == live(before)
        assert not any(c['id'] == choice['id'] for c in removed['recovery']['candidates'])
        damaged_id = 'rec-' + 'a'*24 + '-0000000001'
        damaged = directory / (damaged_id + '.elab-recovery')
        damaged.write_text('owned deliberately damaged capsule')
        client.ask('recovery.scan', {}, True)
        problem = next(p for p in client.ask('project.inspect')['recovery']['problems'] if p['id'] == damaged_id)
        choice = {'id': problem['id'], 'sha256': problem['sha256']}
        before = client.ask('project.inspect')
        client.ask('recovery.dismiss', {'id': choice['id'], 'sha256': 'old'}, True, 'file-conflict')
        refused = client.ask('project.inspect')
        assert damaged.exists() and live(refused) == live(before)
        removed = client.ask('recovery.dismiss', choice, True)
        assert removed['project_work']['recovery_snapshot_removed'] and not damaged.exists()
        assert live(removed) == live(before)
        assert hashlib.sha256(saved.read_bytes()).hexdigest() == saved_digest
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'live_semantic': {k: live(removed)[k] for k in ('components', 'wires', 'run_id', 'electrical_revision',
                                   'result', 'transient', 'mixed', 'npn', 'view')},
                  'ownership_states':ownership_states,'unknown_copy_restored':True,'capsule_sha256':capsule_digest,'saved_source_sha256':saved_digest,
                  'unknown_refusal':refused_unknown['project_work']['error'],
                  'checks': ['real separate-process active owner protected','unknown lease remains protected in scan/inspection/dismissal','unknown-owner snapshot restores as a dirty copy','repair and close yield active/inactive states', 'waiting refusal exposes false removal outcome',
                             'exact inactive capsule removed through immediate background command', 'admission differs from removal',
                             'result ID and digest identify removed snapshot', 'stale hash preserves damaged capsule',
                             'waiting damaged-snapshot dismissal succeeds', 'live RC circuit/physics/history/dirty state preserved',
                             'explicit saved schematic bytes unchanged'],
                  'limits': ['Controlled core tests cover cancellation, changed identity/ownership, retries and blocked commit/catalog interleavings',
                             'No hostile-filesystem atomic-unlink, power-loss, full accessibility or general latency qualification']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as target:
            target.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps({'passed': True, 'transport': report['transport']}))
    finally:
        if copied is not None:
            copied.close()
        if owner is not None:
            owner.close()
        client.close()


if __name__ == '__main__':
    main()
