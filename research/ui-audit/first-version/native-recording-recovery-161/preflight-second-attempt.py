import importlib.util,json,math,sys,time
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab');base=Path(__file__).resolve().parent
sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
release=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
spec=importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
directory=base/'preflight-recovery';a=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']));b=None;fresh=None

def operation(c):
    deadline=time.monotonic()+10
    while True:
        state=c.ask('project.inspect');view=c.ask('operation.inspect',dict(id=state['last_operation']))
        if view['operation']['status']=='completed':return state,view
        assert view['operation']['status'] not in ['failed','canceled'] and time.monotonic()<deadline;time.sleep(.02)

try:
    a.ask('recovery.configure',dict(directory=str(directory),enabled=False,interval_s=2),True);a.ask('example.rc',{},True);a.ask('simulation.pause',{},True);a.ask('simulation.restart',{},True)
    path=base/'Preflight baseline.elab';a.ask('project.save',dict(path=str(path),overwrite=True),True);baseline=path.read_bytes();before=a.ask('simulation.wait')
    cfg=before['verification_settings'].copy();cfg['duration_s']=.02;a.ask('verification.settings.set',cfg,True);a.ask('simulation.verify',dict(duration_s=.02),True);state,capture=operation(a)
    page=a.ask('operation.samples',dict(id=state['last_operation'],offset=0,limit=100));assert page['total']==21 and len(page['rows'])==21
    assert state['document']['recordings_dirty'] and not state['document']['schematic_dirty'];assert state['transient']==before['transient']
    a.ask('recovery.checkpoint',{},True);state=a.ask('simulation.wait');choice=next(row for row in state['recovery']['candidates'] if row['recording_count']==1);inspection=a.ask('recovery.inspect',dict(id=choice['id'],sha256=choice['sha256']));assert inspection['document']['recordings'][0]==capture['metadata'];a.close();a=None
    b=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']));b.ask('recovery.configure',dict(directory=str(directory),enabled=False,interval_s=2),True)
    for offset in [0,20]:
        b.ask('recovery.read.start',dict(kind='samples',id=choice['id'],sha256=choice['sha256'],recording_index=0,offset=offset,limit=20),True);deadline=time.monotonic()+10
        while True:
            work=b.ask('project.inspect')['recovery']['reads'][-1]
            if work['status']=='completed':break
            assert time.monotonic()<deadline;time.sleep(.02)
        result=b.ask('recovery.read.result',dict(id=work['id']))['samples'];assert result['rows']==page['rows'][offset:offset+20] and result['probes']==page['probes'] and result['total']==21
    b.ask('recovery.restore',dict(id=choice['id'],sha256=choice['sha256']),True);restored=b.ask('simulation.wait');assert restored['paused'] and restored['transient']['sample']['time_s']==0 and restored['document']['recording_count']==1
    restored_capture=b.ask('operation.inspect',dict(id=restored['last_operation']));assert restored_capture['metadata']==capture['metadata']
    b.ask('recording.view.set',dict(mode='recorded'),True);b.ask('recording.step',dict(direction='next'),True);assert b.ask('project.inspect')['recording']['index']==1
    b.ask('recording.seek',dict(time_s='20 ms'),True);assert b.ask('project.inspect')['recording']['index']==20;b.ask('recording.view.set',dict(mode='live'),True)
    saved=base/'Preflight recovered.elab';b.ask('project.save',dict(path=str(saved)),True);assert path.read_bytes()==baseline
    fresh=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']));fresh.ask('project.load',dict(path=str(saved)),True);reopened=fresh.ask('simulation.wait');assert reopened['document']['recording_count']==1 and not reopened['document']['dirty']
    assert fresh.ask('operation.inspect',dict(id=reopened['last_operation']))['metadata']==capture['metadata']
    report=dict(passed=True,scope='Public schema/reference preflight with explicit recovery checkpoint and normal owner close; native automatic checkpoint/SIGKILL remain separate.',initial_verification_settings=before['verification_settings'],configured_verification_settings=cfg,dirty_flags=state['document'],capture=capture,page=page,restored=restored,reopened=reopened)
    (base/'preflight.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(dict(passed=True,probes=[x['id'] for x in page['probes']],sample_count=page['total'],metadata_keys=list(capture['metadata']),verification_settings=cfg,dirty_flags=state['document'])))
finally:
    if a:a.close()
    if b:b.close()
    if fresh:fresh.close()
