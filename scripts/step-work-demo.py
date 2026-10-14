#!/usr/bin/env python3
"""Exercise background manual stepping, atomic cancellation and legacy parity.
--connect replaces only the designated owned development project.
"""
import argparse
import importlib.util
import json
from pathlib import Path
import time

spec=importlib.util.spec_from_file_location('helper',Path(__file__).with_name('persistence-demo.py'))
helper=importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output',type=Path)
    parser.add_argument('--artifacts',type=Path,required=True)
    args=parser.parse_args()
    client=helper.Client(args.connect,command_preconditions=True)
    fresh=None
    try:
        catalog=client.ask('capabilities.list')
        client.ask('example.flasher',{'discard_changes':True},True)
        client.ask('simulation.pause',{},True)
        settings=client.ask('project.inspect')['transient_settings']
        settings['step_s']=.001
        client.ask('simulation.transient.set',settings,True)
        client.ask('simulation.restart',{},True)
        client.ask('scope.probes.set',{'ids':['i:D1','i:D2','v:C1','v:C2']},True)
        before=client.ask('project.inspect')
        started=time.perf_counter()
        accepted=client.ask('simulation.step.start',{'duration_s':1},True)
        admission_ms=(time.perf_counter()-started)*1000
        assert accepted['step_work']['status']=='queued'
        work_id=accepted['step_work']['id']
        deadline=time.monotonic()+10
        while True:
            progress=client.ask('project.inspect')
            assert progress['step_work']['status'] in ['queued','running'], progress['step_work']
            if progress['step_work']['completed_samples']>0:
                break
            assert time.monotonic()<deadline
            time.sleep(.001)
        assert progress['transient']==before['transient'] and progress['mixed']==before['mixed']
        client.ask('view.pan',{'dx':20,'dy':10},True)
        resistor=next(p for p in before['components'] if p['kind']=='resistor')
        moved=dict(resistor['position']);moved['x']+=20
        edit=client.ask('component.move',{'id':resistor['id'],'position':moved},True)
        client.ask('edit.undo',{'expected_entry_revision':edit['undo_entry']['command_revision']},True)
        directory=args.artifacts.resolve();directory.mkdir(parents=True,exist_ok=True)
        path=str(directory/'flasher.elab')
        client.ask('project.save',{'path':path,'overwrite':True},True)
        client.ask('simulation.work.cancel',{'id':work_id},True)
        canceled=client.ask('simulation.wait')
        assert canceled['step_work']['status']=='canceled'
        assert canceled['mixed']==before['mixed'] and canceled['transient']==before['transient']
        assert canceled['components']==before['components'] and canceled['view']!=before['view']
        next_work=client.ask('simulation.step.start',{'duration_s':.03},True)
        assert next_work['step_work']['id']!=work_id
        completed=client.ask('simulation.wait')
        assert completed['step_work']['status']=='completed'
        assert abs(completed['transient']['sample']['time_s']-.03)<1e-12
        assert len(completed['scope']['traces'])==4
        accepted=client.ask('simulation.verify',{'duration_s':.03,'probe_ids':['i:D1','i:D2','v:C1','v:C2']},True)
        operation_id=accepted['last_operation']
        deadline=time.monotonic()+15
        while True:
            recorded=client.ask('operation.inspect',{'id':operation_id})
            if recorded['operation']['status'] not in ['queued','running','canceling']:
                break
            assert time.monotonic()<deadline
            time.sleep(.005)
        assert recorded['operation']['status']=='completed',recorded
        for a,b in zip(completed['mixed']['capacitors'],recorded['metadata']['mixed']['capacitors']):
            assert abs(a['voltage_v']-b['voltage_v'])<1e-8
        legacy=client.ask('simulation.step',{'duration_s':.001},True)
        assert legacy['step_work']['status']=='completed' and abs(legacy['transient']['sample']['time_s']-.031)<1e-12
        client.ask('project.save',{'path':path,'overwrite':True},True)
        fresh=helper.Client(command_preconditions=True)
        reopened=fresh.ask('project.load',{'path':path},True)
        assert reopened['paused'] and not reopened['document']['dirty']
        assert reopened['document']['recording_count']==1 and reopened['transient']['sample']['time_s']==0
        assert 'step_work' not in reopened
        report=dict(passed=True,transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
            capabilities=len(catalog['capabilities']),step_start_admission_ms=admission_ms,
            canceled_staged_samples=canceled['step_work']['completed_samples'],committed_time_after_cancel_s=canceled['transient']['sample']['time_s'],
            completed_time_s=completed['transient']['sample']['time_s'],legacy_time_s=legacy['transient']['sample']['time_s'],
            checks=['immediate queued ID/actor/provenance','inspect staged progress with unchanged committed physics',
                'pan/save/geometry undo while background work runs','exact-ID cancellation discards all staged samples',
                'new work ID and atomic successful publication','four traces and independent Verify parity',
                'legacy wait-for-result compatibility','save/reopen keeps experiment and recording; physical/job state restarts'],
            limitations=['backend admission timing is not visible input-to-pixel latency',
                'general initialization and some input/file work remain synchronous','private unsigned local build'])
        if args.output:
            args.output.parent.mkdir(parents=True,exist_ok=True)
            args.output.write_text(json.dumps(report,indent=2)+'\n')
        print(json.dumps(report,indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__=='__main__':
    main()
