#!/usr/bin/env python3
"""Interrupt owned automatic storage, observe failure while editing, then retry."""
import argparse
import copy
import importlib.util
import json
from pathlib import Path
import time

from demo_session import connect_demo_client
from recovery_timing import assert_live_preserved, digest, settle_native_view

spec = importlib.util.spec_from_file_location('client_helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--exercise-disable', action='store_true', help='disable during the fault, prove stopped scheduling, then re-enable')
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(mode=0o700, parents=True, exist_ok=False)
    recovery = directory/'recovery'
    moved = directory/'recovery-preserved'
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    broken = False

    def await_capsule(expected_value):
        deadline = time.monotonic()+5
        while True:
            state = client.ask('project.inspect')
            if not state['recovery'].get('error') and state['recovery']['candidates']:
                candidate = state['recovery']['candidates'][0]
                path = recovery/(candidate['id']+'.elab-recovery')
                raw = path.read_bytes();capsule = json.loads(raw)
                part = next(p for p in capsule['document']['graph']['components'] if p['id']=='R1')
                if part['value']==expected_value:
                    assert digest(raw)==candidate['sha256']
                    inspection = client.ask('recovery.inspect', {key:candidate[key] for key in ('id','sha256')})
                    assert inspection['candidate']['sha256']==candidate['sha256']
                    return state, candidate, raw, capsule['document']
            assert time.monotonic()<deadline, 'automatic complete retry did not appear'
            time.sleep(.025)

    try:
        client.ask('example.rc', {}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        accepted = client.ask('simulation.verify', {'duration_s':.2}, True)
        deadline = time.monotonic()+10
        while True:
            recorded = client.ask('operation.inspect', {'id':accepted['last_operation']})
            assert recorded['operation']['status'] not in ('failed','canceled')
            if recorded['operation']['status']=='completed':break
            assert time.monotonic()<deadline
            time.sleep(.01)
        if args.connect:
            settle_native_view(client)
            client.ask('view.pan', {'dx':0,'dy':0}, True)
        source = directory/'manual-source.elab'
        client.ask('project.save', {'path':str(source)}, True)
        manual = source.read_bytes();baseline = json.loads(manual)
        client.ask('recovery.configure', {'directory':str(recovery),'enabled':True,'interval_s':2}, True)
        client.ask('component.set', {'id':'R1','value':'20k'}, True)
        initial, initial_candidate, initial_raw, _ = await_capsule(20000)
        (directory/'last-good.elab-recovery').write_bytes(initial_raw)
        recovery.rename(moved)
        recovery.write_text('owned interrupted recovery storage')
        broken = True
        edits = 0;started = time.monotonic()
        while True:
            value = '30k' if edits%2==0 else '40k'
            before = client.ask('component.set', {'id':'R1','value':value}, True)
            before = client.ask('simulation.wait')
            edits += 1
            failed = client.ask('project.inspect')
            if failed['recovery'].get('error'):break
            assert time.monotonic()-started<5, 'automatic storage failure was not observable while editing'
            time.sleep(.05)
        failure_elapsed = time.monotonic()-started
        assert_live_preserved(before,failed,'paused',bool(args.connect))
        old_path = moved/(initial_candidate['id']+'.elab-recovery')
        assert old_path.read_bytes()==initial_raw and source.read_bytes()==manual
        (directory/'failure-state.json').write_text(json.dumps(failed,indent=2)+'\n')
        disable_report = {}
        if args.exercise_disable:
            started = time.monotonic()
            disabled = client.ask('recovery.configure', {'directory':str(recovery),'enabled':False,'interval_s':2}, True)
            disable_elapsed = time.monotonic()-started
            expected_recovery = copy.deepcopy(failed['recovery']);expected_recovery['enabled'] = False
            assert disabled['recovery']==expected_recovery
            assert disabled['project_work']['recovery_configuration_applied'] and disabled['project_work']['recovery_catalog_applied'] is False
            (directory/'disabled-state.json').write_text(json.dumps(disabled,indent=2)+'\n')
            admitted = client.ask('recovery.configure.start', {'directory':str(recovery),'enabled':False,'interval_s':'2 s'}, True)
            deadline = time.monotonic()+5
            while True:
                repeated = client.ask('project.inspect')
                assert repeated['project_work']['id']==admitted['project_work']['id']
                assert repeated['project_work']['status'] not in ('failed','canceled','superseded')
                if repeated['project_work']['status']=='completed':break
                assert time.monotonic()<deadline
                time.sleep(.005)
            assert repeated['recovery']==disabled['recovery'] and repeated['project_work']['recovery_catalog_applied'] is False
            client.ask('recovery.configure', {'directory':str(recovery),'enabled':True,'interval_s':2}, True, 'file-error')
            denied = client.ask('project.inspect')
            assert denied['recovery']==disabled['recovery'] and not denied['project_work']['recovery_configuration_applied']
            (directory/'reenable-denied-state.json').write_text(json.dumps(denied,indent=2)+'\n')
            disable_report = {'disable_s':disable_elapsed,'both_disable_aliases_completed':True,'catalog_applied_on_disable':False,
                              'last_error_and_catalog_preserved':True,'reenable_refused_while_unavailable':True}
        committed = client.ask('component.set', {'id':'R1','value':'50k'}, True)
        committed = client.ask('simulation.wait')
        started = time.monotonic()
        recovery.unlink();moved.rename(recovery);broken = False
        if args.exercise_disable:
            held_before = client.ask('project.inspect')
            files_before = {p.name:digest(p.read_bytes()) for p in recovery.iterdir()}
            held_started = time.monotonic()
            time.sleep(3)
            held_after = client.ask('project.inspect')
            held_elapsed = time.monotonic()-held_started
            assert held_after['recovery']==held_before['recovery'] and held_after['recovery']['enabled'] is False
            assert files_before=={p.name:digest(p.read_bytes()) for p in recovery.iterdir()}
            assert_live_preserved(held_before,held_after,'paused',bool(args.connect))
            (directory/'disabled-after-storage-restored.json').write_text(json.dumps(held_after,indent=2)+'\n')
            started = time.monotonic()
            enabled = client.ask('recovery.configure', {'directory':str(recovery),'enabled':True,'interval_s':2}, True)
            assert enabled['recovery']['enabled'] and enabled['project_work']['recovery_catalog_applied']
            committed = client.ask('simulation.wait')
            disable_report.update(available_storage_held_disabled_s=held_elapsed,
                                  disabled_file_set_unchanged=True,reenable_catalog_applied=True)
        ready, candidate, raw, recovered = await_capsule(50000)
        retry_elapsed = time.monotonic()-started
        assert_live_preserved(committed,ready,'paused',bool(args.connect))
        expected = copy.deepcopy(baseline)
        expected['graph'] = {'components':committed['components'],'wires':committed['wires']}
        expected['view'] = committed['view']
        assert recovered==expected and recovered['recordings']==baseline['recordings']
        assert source.read_bytes()==manual and (recovery/(initial_candidate['id']+'.elab-recovery')).read_bytes()==initial_raw
        (directory/'retry.elab-recovery').write_bytes(raw)
        (directory/'retry-state.json').write_text(json.dumps(ready,indent=2)+'\n')
        report = {'passed':True,'transport':'native-socket' if args.connect else 'headless-stdio',
                  'ordinary_interval_s':2,'failure_visible_s':failure_elapsed,'automatic_retry_s':retry_elapsed,
                  'edits_during_fault':edits,'error':failed['recovery']['error'],
                  'last_good_capsule_sha256':digest(initial_raw),'retry_capsule_sha256':digest(raw),
                  'manual_source_sha256':digest(manual),'recorded_rows':sum(len(e['data']['rows']) for e in recovered['recordings']['entries']),
                  'manual_source_and_previous_capsule_unchanged':True,'retry_document_exact':True,
                  'disable_control':disable_report,'retry_timing_origin':'reenable dispatch' if args.exercise_disable else 'storage repair',
                  'checks':['actual interrupted private storage','automatic error during repeated editing',
                            'no manual checkpoint/scan for failure or retry','latest dirty graph and full kept record saved automatically',
                            'manual source and last complete capsule unchanged','editor/history/physics preserved by status publication'],
                  'limits':['Owned cooperating filesystem fault on this workstation; not power-loss/hostile-filesystem qualification',
                            'Controlled core tests separately force an edit exactly between filesystem preparation and publication']}
        args.output.write_text(json.dumps(report,indent=2)+'\n')
        print(json.dumps({k:report[k] for k in ['passed','transport','failure_visible_s','automatic_retry_s','edits_during_fault']}))
        if args.connect:client.ask('app.close', {'discard_changes':True}, True)
    finally:
        if broken:
            recovery.unlink();moved.rename(recovery)
        client.close()


if __name__=='__main__':main()
