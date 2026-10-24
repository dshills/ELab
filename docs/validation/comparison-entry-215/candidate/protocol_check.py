"""Check exact captured comparisons, request ownership and RC references."""
import hashlib, importlib.util, json, math, time
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab');stage=Path(__file__).resolve().parent
build=json.loads((stage/'build-result-signed.json').read_text());binary=Path(build['review'])/'elab-agent'
assert hashlib.sha256(binary.read_bytes()).hexdigest()==build['agent_sha256']
spec=importlib.util.spec_from_file_location('client',root/'scripts/persistence-demo.py')
helper=importlib.util.module_from_spec(spec);spec.loader.exec_module(helper)
c=helper.Client(command_preconditions=True,agent_binary=binary)
report=dict(source_commit=build['source_commit'],agent_sha256=build['agent_sha256'],transport='headless-stdio',
    queries_checked=0,scalar_readings_checked=0,different_sample_times_queries=0,refusal_cases=0,
    max_voltage_error_v=0.,max_current_error_a=0.)

def capture(duration,step):
    identifier=c.ask('simulation.verify',dict(duration_s=duration,sample_step_s=step,input_mode='charge',probe_ids=['v:C1','i:C1']),True)['last_operation']
    deadline=time.monotonic()+5
    while True:
        operation=c.ask('operation.inspect',{'id':identifier})
        status=operation['operation']['status']
        if status=='completed':break
        assert status not in ['failed','canceled'] and time.monotonic()<deadline,operation
        time.sleep(.01)
    page=c.ask('operation.samples',dict(id=identifier,offset=0,limit=operation['metadata']['sample_count']))
    return dict(id=identifier,metadata=operation['metadata'],page=page)

def refuse(params,code='invalid-params',revision=None):
    before=c.ask('project.inspect');c.sequence+=1
    guard=before['command_revision'] if revision is None else revision
    request=dict(version=1,id=f'refuse-{c.sequence}',key=f'refuse-{c.sequence}',command='recording.compare.set',params=params,expected_command_revision=guard)
    c.process.stdin.write(json.dumps(request)+'\n');c.process.stdin.flush()
    response=json.loads(c.process.stdout.readline());assert response['error']['code']==code,response
    assert before==c.ask('project.inspect'),'comparison refusal changed accepted state'
    report['refusal_cases']+=1

def check(params,a,b):
    before=c.ask('project.inspect');result=c.ask('recording.compare',params)
    assert before==c.ask('project.inspect') and result['available']
    requested=params['time_s'];actual={}
    for label,record in [('a',a),('b',b)]:
        rows=record['page']['rows']
        index=min(range(len(rows)),key=lambda i:(abs(rows[i]['time_s']-requested),-i))
        sample=result[label]
        assert sample['id']==record['id'] and sample['sample_index']==index and sample['time_s']==rows[index]['time_s']
        assert sample['offset_from_requested_s']==rows[index]['time_s']-requested
        actual[label]=sample['time_s']
    assert len(result['readings'])==2
    for reading in result['readings']:
        probe=reading['probe']
        if probe['id']=='v:C1':
            av,bv=5*(1-math.exp(-actual['a']/.02)),5*(1-math.exp(-actual['b']/.01));unit='V'
        else:
            assert probe['id']=='i:C1'
            av,bv=.00025*math.exp(-actual['a']/.02),.0005*math.exp(-actual['b']/.01);unit='A'
        assert probe['unit']==unit
        for field,expected in [('a',av),('b',bv),('b_minus_a',bv-av)]:
            error=abs(reading[field]-expected);key='max_voltage_error_v' if unit=='V' else 'max_current_error_a'
            report[key]=max(report[key],error)
            assert error<(1e-10 if unit=='V' else 1e-14),(field,reading,expected)
            report['scalar_readings_checked']+=1
    report['queries_checked']+=1
    if actual['a']!=actual['b']:report['different_sample_times_queries']+=1

try:
    c.ask('example.rc',{'discard_changes':True},True);c.ask('simulation.pause',{},True)
    b=capture(.02,.002)
    c.ask('component.set',dict(id='R1',value='20k'),True);c.ask('simulation.wait')
    a=capture(.02,.001);short=capture(.01,.001)
    assert len(a['page']['rows'])==21 and len(b['page']['rows'])==11
    for record,resistance in [(a,20000),(b,10000),(short,20000)]:
        assert next(p for p in record['metadata']['graph']['components'] if p['id']=='R1')['value']==resistance
    (stage/'captured-rc-comparisons.json').write_text(json.dumps(dict(a=a,b=b,short=short),indent=2)+'\n')
    c.ask('operation.select',{'id':a['id']},True);c.ask('recording.view.set',{'mode':'recorded'},True)
    c.ask('recording.window.set',dict(start_s='0 s',end_s='20 ms'),True)
    for cursor,at in [('a','5 ms'),('b','15 ms')]:c.ask('recording.cursor.set',dict(cursor=cursor,time_s=at),True)
    params=dict(id=a['id'],reference_id=b['id'],time_s=.01,show_statistics=True,show_overlay=True)
    c.ask('recording.compare.set',params,True)
    scope=c.ask('project.inspect')['recording']['scope']
    for at in [i*.001 for i in range(21)]+[.0005]:check(dict(params,time_s=at),a,b)
    old_revision=c.command_revision
    c.ask('component.set',dict(id='R1',value='30k'),True);c.ask('simulation.wait')
    for record in [a,b,short]:assert record['metadata']==c.ask('operation.inspect',{'id':record['id']})['metadata']
    check(dict(params,time_s=.0095),a,b)
    for bad in [dict(params,time_s='oops'),dict(params,time_s=-.001),dict(params,time_s=.03),
                dict(params,reference_id=a['id']),dict(params,reference_id=short['id'],time_s=.015)]:refuse(bad)
    refuse(dict(params,time_s=.006),'conflict',old_revision)
    corrected=dict(params,time_s='6 ms',show_statistics=False)
    state=c.ask('recording.compare.set',corrected,True)
    assert state['recording']['comparison']==c.ask('recording.compare',corrected)
    assert state['recording']['comparison']['settings']['time_s']==.006
    assert not state['recording']['comparison']['settings'].get('show_statistics',False)
    assert state['recording']['reference_overlay']['id']==b['id']
    changed=dict(corrected,reference_id=short['id'],time_s='2 ms')
    state=c.ask('recording.compare.set',changed,True)
    assert state['recording']['comparison']['b']['id']==short['id']
    assert state['recording']['reference_overlay']['id']==short['id']
    cleared=c.ask('recording.compare.set',dict(id=a['id'],reference_id='',time_s=0),True)
    assert cleared['recording'].get('comparison') is None and cleared['recording'].get('reference_overlay') is None
    assert cleared['recording']['scope']==scope
    assert next(p for p in cleared['components'] if p['id']=='R1')['value']==30000
    report.update(captured_metadata_preserved_after_live_edit=True,live_resistance_ohm=30000,
        captured_a_resistance_ohm=20000,captured_b_resistance_ohm=10000,unit_time_correction_succeeded=True,
        preference_and_reference_changes_succeeded=True,clear_preserves_scope_window_and_cursors=True,
        independent_reference='5 V / 1 uF, zero startup: A R=20 kohm, dt=1 ms; B R=10 kohm, dt=2 ms. Vc=5*(1-exp(-t/RC)); Ic=(5/R)*exp(-t/RC), each at its own saved timestamp; delta B-A.')
    job=c.ask('app.close',{'discard_changes':True},True)['shutdown'];job=c.wait_for_shutdown_outcome(job)
    assert job['status']=='completed' and job['workers_joined'] and job['exit_ready'];report['shutdown']=job
finally:c.close()
assert c.process.returncode==0;report.update(passed=True,agent_exit_code=0)
(stage/'protocol-report.json').write_text(json.dumps(report,indent=2)+'\n')
print(json.dumps({k:report[k] for k in ['passed','queries_checked','scalar_readings_checked','different_sample_times_queries','refusal_cases','max_voltage_error_v','max_current_error_a','agent_exit_code']}))
