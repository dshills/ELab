import importlib.util,json,sys
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab');base=Path(__file__).parent
sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
release=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
spec=importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
directory=base/'preflight-recovery';a=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']));b=None
try:
    a.ask('recovery.configure',dict(directory=str(directory),enabled=False,interval_s=2),True)
    a.ask('example.reset',{},True);a.ask('simulation.wait');a.ask('simulation.pause',{},True)
    baseline=base/'Preflight baseline.elab';a.ask('project.save',dict(path=str(baseline)),True);original=baseline.read_bytes()
    a.ask('component.select',dict(id='R1'),True);a.ask('editor.value.set',dict(value='2.2k'),True);a.ask('component.set',dict(id='R1',value='2.2k'),True)
    a.ask('simulation.resume',{},True);a.ask('simulation.wait');a.ask('simulation.pause',{},True);edited=a.ask('simulation.wait')
    a.ask('recovery.checkpoint',{},True);candidates=a.ask('simulation.wait')['recovery']['candidates'];choice=candidates[0]
    inspected=a.ask('recovery.inspect',dict(id=choice['id'],sha256=choice['sha256']));assert inspected['document']['paused']
    assert next(x for x in inspected['document']['graph']['components'] if x['id']=='R1')['value']==2200
    a.close();a=None
    b=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']));b.ask('recovery.configure',dict(directory=str(directory),enabled=False,interval_s=2),True)
    ready=b.ask('simulation.wait');choice=next(x for x in ready['recovery']['candidates'] if x['id']==choice['id']);assert not choice['active_session'] and not choice['current_session']
    b.ask('recovery.select',dict(id=choice['id']),True);b.ask('recovery.restore',dict(id=choice['id'],sha256=choice['sha256'],expected_document_generation=ready['document']['generation']),True)
    restored=b.ask('simulation.wait');assert restored['components']==edited['components'] and restored['wires']==edited['wires']
    assert restored['document']['dirty'] and not restored['document']['path'] and restored['document']['recovered_from']==str(baseline)
    assert restored['paused'];p.compare(restored['result'],edited['result'])
    b.ask('project.save',dict(path=str(base/'Preflight recovered.elab')),True);assert baseline.read_bytes()==original
    result=dict(passed=True,scope='Public reference schema/control preflight with an explicit checkpoint and normal owner closure; native automatic checkpoint and SIGKILL are separate upcoming checks.',candidate=choice,edited=edited,restored=restored)
    (base/'preflight.json').write_text(json.dumps(result,indent=2)+'\n')
    print('Public recovery preflight passes: inactive owner, exact edited graph/result, dirty unassociated copy, separate Save and unchanged baseline')
finally:
    if a:a.close()
    if b:b.close()
