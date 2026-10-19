import datetime, hashlib, importlib.util, json, subprocess, sys, time
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab')
base=Path(__file__).resolve().parent
out=base/'observations'
requests=out/'requests'
requests.mkdir()
sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
release=validate_release(root,root/'research/ui-audit/first-version/recorded-timing-review-checkpoint.json')
identity=json.loads((base/'identity.json').read_text())
binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery'
assert hashlib.sha256(binary.read_bytes()).hexdigest()==identity['native_sha256']
assert not [row for row in subprocess.check_output(['/bin/ps','-axo','pid=,comm='],text=True).splitlines() if 'ELab-value-recovery' in row]
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py')
p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
host=client=reference=None
r=dict(passed=False,release=release,identity=identity,stages=[],started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),scope='Exact /163 executable in a disposable custom bundle. Public commands seed one pulse-NPN recorded waveform; native CUA alone drives the threshold field and direction menu. No from-blank construction or whole-bundle claim.')
protected=['components','wires','nets','document','run_id','electrical_revision','paused','transient','mixed','npn','result','scope','undo_entry','redo_entry']
def save():
    pending=out/'report.pending';pending.write_text(json.dumps(r,indent=2)+'\n');pending.replace(out/'report.json')
def capture(c,operation):
    deadline=time.monotonic()+15
    while True:
        result=c.ask('operation.inspect',dict(id=operation))
        if result['operation']['status']=='completed':
            rows=[];pages=[];offset=0
            while True:
                page=c.ask('operation.samples',dict(id=operation,offset=offset,limit=200));pages.append(page);rows+=page['rows'];offset=page['next_offset']
                if offset==page['total']:return result,pages,rows
        assert result['operation']['status'] not in ['failed','canceled'] and time.monotonic()<deadline
        time.sleep(.02)
def prepare(c):
    assert not c.ask('simulation.wait')['components']
    c.ask('example.npn-pulse',{},True);c.ask('simulation.pause',{},True);c.ask('simulation.restart',{},True);c.ask('simulation.wait')
    c.ask('simulation.verify',dict(duration_s=.5,probe_ids=['v:VIN.positive']),True)
    operation=c.ask('project.inspect')['last_operation'];data=capture(c,operation)
    c.ask('recording.view.set',dict(mode='recorded'),True)
    c.ask('recording.timing.set',dict(id=operation,probe_id='v:VIN.positive',level='2.5 V',direction='both',start_s=0,end_s=.5,limit=8,offset=0),True)
    state=c.ask('simulation.wait');assert state['recording']['timing']['total_crossings']>8
    return state,data
try:
    endpoint=out/'endpoint.json'
    with (out/'native-host.txt').open('x') as log:
        host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2)
    r['pid']=host.pid;save();deadline=time.monotonic()+25
    while not endpoint.exists():assert host.poll() is None and time.monotonic()<deadline;time.sleep(.05)
    client=p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=Path(release['agent']))
    reference=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']))
    baseline,original=prepare(client);agent_baseline,agent_original=prepare(reference)
    p.compare(baseline['recording']['timing'],agent_baseline['recording']['timing'])
    r.update(baseline=baseline,reference_baseline=agent_baseline,original_capture=original[0],original_pages=original[1]);save();(out/'ready.json').write_text(json.dumps(dict(pid=host.pid,command_revision=baseline['command_revision']))+'\n')
    sequence=1;deadline=time.monotonic()+1800
    while True:
        request=requests/(str(sequence)+'.json')
        while not request.exists():assert host.poll() is None and time.monotonic()<deadline;time.sleep(.05)
        work=json.loads(request.read_text());row=dict(name=work['name'],passed=False)
        try:
            state=client.ask('simulation.wait');row['state']=state
            for key in protected:assert state.get(key)==baseline.get(key),key
            for command in work.get('agent_commands',[]):reference.ask(command['id'],command['params'],True)
            expected=reference.ask('simulation.wait');row['reference']=expected
            p.compare(state['recording']['timing'],expected['recording']['timing'])
            if 'command_delta' in work:assert state['command_revision']==baseline['command_revision']+work['command_delta'],'command_delta'
            row['passed']=True
        except BaseException as error:row['error']=type(error).__name__+': '+str(error)
        r['stages'].append(row);save();(out/(str(sequence)+'-response.json')).write_text(json.dumps(row,indent=2)+'\n')
        if work.get('finish'):break
        sequence+=1
    final=capture(client,baseline['last_operation']);assert final==original
    r['capture_metadata_pages_unchanged']=True
    for c,label in [(client,'Native'),(reference,'Agent')]:
        c.ask('project.save',dict(path=str(out/(label+' timing.elab'))),True)
    a=json.loads((out/'Native timing.elab').read_text());b=json.loads((out/'Agent timing.elab').read_text())
    assert {k:v for k,v in a.items() if k!='view'}=={k:v for k,v in b.items() if k!='view'}
    r['saved_fields_except_view_match']=True
    client.ask('app.close',{},True);client.close();client=None
    r['exit_code']=host.wait(timeout=15);assert r['exit_code']==0
    r['normal_closure']=True;r['passed']=all(s['passed'] for s in r['stages'])
except BaseException as error:
    r['error']=type(error).__name__+': '+str(error);save()
    if host and host.poll() is None:
        deadline=time.monotonic()+1800
        while not (out/'allow-error-cleanup').exists() and host.poll() is None and time.monotonic()<deadline:time.sleep(.1)
    raise
finally:
    if client:client.close()
    if reference:reference.close()
    if host and host.poll() is None:
        host.terminate()
        try:host.wait(timeout=10)
        except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
    if host:r['final_host_exit_code']=host.returncode
    r['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
