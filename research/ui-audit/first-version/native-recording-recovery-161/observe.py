import datetime,hashlib,importlib.util,json,math,shutil,subprocess,sys,time
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab');base=Path(__file__).resolve().parent;out=base/'observations'
first=out/'first';second=out/'restored';first.mkdir();second.mkdir();directory=base/'recovery';directory.mkdir(mode=0o700)
protocol=dict(barriers=['loaded-paused','restarted','prepare-baseline','private-duration','duration-committed','capture-completed','automatic-snapshot','recovery-selected','recovery-inspected','recovery-page-first','recovery-page-last','recovery-restored','recorded-view','recorded-next','private-seek','recorded-end','view-live','save-recovered'],signals=['allow-crash','allow-relaunch','finish'])
(out/'protocol.json').write_text(json.dumps(protocol,indent=2)+'\n')
sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
release=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json');identity=json.loads((base/'identity.json').read_text())
binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';assert hashlib.sha256(binary.read_bytes()).hexdigest()==identity['native_sha256']
spec=importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
manual=out/'Manual RC.elab';recovered=out/'Recovered RC.elab';r=dict(passed=False,release=release,identity=identity,events=[],reference={},hosts=[],checks={},scope='Native RC example and Verify, automatic recording-only recovery, owned SIGKILL, native inspection/paged sample reads/Restore and recorded view/Next/seek/live controls. Public saves prepare the baseline and persist final files; native file dialogs are not repeated.',started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
host=client=reference=None;phase='first'
def save():
    pending=out/'report.pending';pending.write_text(json.dumps(r,indent=2)+'\n');pending.replace(out/'report.json')
def record(name,state):r['events'].append(dict(name=name,phase=phase,state=state));save()
def wait(name,signal=False):
    assert name in protocol['signals' if signal else 'barriers'],name
    deadline=time.monotonic()+900
    while not (out/name).exists():assert host.poll() is None and time.monotonic()<deadline;time.sleep(.05)
    state=client.ask('simulation.wait');record(name,state);return state
def barrier(name,check):
    state=wait(name);check(state);(out/(name+'-verified.json')).write_text('{}\n');return state
def same(a,b):
    for key in ['components','wires','nets','selected','drawing','paused','transient_settings','undo_name','redo_name']:assert a.get(key)==b.get(key),key
    for key in ['result','npn','mixed','transient','scope','trace_focus']:p.compare(a.get(key),b.get(key),key)
def compare_state(name,state):
    expected=reference.ask('simulation.wait');r['reference'][name]=expected;same(state,expected);save()
def live_unchanged(before,after,document=False):
    keys=['components','wires','nets','selected','paused','run_id','electrical_revision','transient','scope','result','mixed','npn']
    if document:keys+=['document','undo_entry','redo_entry']
    for key in keys:assert before.get(key)==after.get(key),key
def launch(folder):
    endpoint=folder/'endpoint.json';assert not endpoint.exists()
    with (folder/'native-host.txt').open('x') as log:process=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--recovery-directory',str(directory),'--no-appearance'],stdout=log,stderr=-2)
    deadline=time.monotonic()+20
    while not endpoint.exists():assert process.poll() is None and time.monotonic()<deadline;time.sleep(.05)
    connection=p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=Path(release['agent']));state=connection.ask('simulation.wait');assert not state['components'] and not state['wires'];assert state['recovery']['enabled'] and state['recovery']['directory']==str(directory)
    return process,connection,state
def capture(c):
    deadline=time.monotonic()+10
    while True:
        state=c.ask('project.inspect');view=c.ask('operation.inspect',dict(id=state['last_operation']))
        if view['operation']['status']=='completed':return state,view,c.ask('operation.samples',dict(id=state['last_operation'],offset=0,limit=100))
        assert view['operation']['status'] not in ['failed','canceled'] and time.monotonic()<deadline;time.sleep(.02)
def sample_law(page):
    assert page['total']==21
    maximum=0.
    for row in page['rows']:
        for probe,value in zip(page['probes'],row['values']):
            expected=5*(1-math.exp(-row['time_s']/.01)) if probe['id']=='v:C1.1' else 5
            assert probe['id'] in ['v:C1.1','v:V1.positive'];error=abs(value-expected);assert error<=max(.001,.001*abs(expected));maximum=max(maximum,error)
    return dict(passed=True,rows=len(page['rows']),probes=len(page['probes']),maximum_error_v=maximum)
def read_complete(c,kind,offset=None):
    deadline=time.monotonic()+10
    while True:
        works=[w for w in c.ask('project.inspect')['recovery'].get('reads',[]) if w['input']['kind']==kind and w['input']['id']==choice['id'] and (offset is None or w['input'].get('offset',0)==offset)]
        if works and works[-1]['status']=='completed':return works[-1],c.ask('recovery.read.result',dict(id=works[-1]['id']))
        assert time.monotonic()<deadline;time.sleep(.02)
def reference_read(kind,offset=None):
    params=dict(kind=kind,id=choice['id'],sha256=choice['sha256'])
    if offset is not None:params.update(recording_index=0,offset=offset,limit=20)
    reference.ask('recovery.read.start',params,True);return read_complete(reference,kind,offset)
def capsule_unchanged():
    assert manual.read_bytes()==manual_bytes and snapshot.read_bytes()==capsule_bytes

try:
    reference=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']));reference.ask('example.rc',{},True);reference.ask('simulation.pause',{},True);reference.ask('simulation.wait')
    host,client,blank=launch(first);r['hosts'].append(dict(phase=phase,pid=host.pid));save();(out/'first-ready.json').write_text('{}\n')
    def loaded(s):
        assert s['paused']
        expected=reference.ask('simulation.wait')
        for key in ['components','wires','nets','transient_settings']:assert s[key]==expected[key],key
    barrier('loaded-paused',loaded);reference.ask('simulation.restart',{},True);barrier('restarted',lambda s:compare_state('restarted',s))
    def baseline(s):
        client.ask('project.save',dict(path=str(manual)),True);reference.ask('project.save',dict(path=str(out/'Agent baseline.elab')),True)
        a=json.loads(manual.read_text());b=json.loads((out/'Agent baseline.elab').read_text());assert {k:v for k,v in a.items() if k!='view'}=={k:v for k,v in b.items() if k!='view'};record('baseline-ready',client.ask('simulation.wait'))
    barrier('prepare-baseline',baseline);manual_bytes=manual.read_bytes();r['manual_sha256']=hashlib.sha256(manual_bytes).hexdigest();before=client.ask('simulation.wait')
    barrier('private-duration',lambda s:live_unchanged(before,s,True));cfg=before['verification_settings'].copy();cfg['duration_s']=.02;reference.ask('verification.settings.set',cfg,True)
    def duration(s):assert s['verification_settings']==cfg;live_unchanged(before,s,True)
    barrier('duration-committed',duration)
    reference.ask('simulation.verify',dict(duration_s=.02),True);ref_state,ref_capture,ref_page=capture(reference)
    def captured(s):
        global original_capture,original_page,captured_state
        captured_state,original_capture,original_page=capture(client);live_unchanged(before,captured_state);assert original_capture['operation']['actor']=='user';assert captured_state['document']['recordings_dirty'] and not captured_state['document']['schematic_dirty'] and not captured_state['document']['preferences_dirty']
        assert original_page['rows']==ref_page['rows'] and original_page['probes']==ref_page['probes'];assert original_page['total']==21
        provenance={'operation_id','created_at','input_revision','electrical_revision','document_generation'}
        p.compare({k:v for k,v in original_capture['metadata'].items() if k not in provenance},{k:v for k,v in ref_capture['metadata'].items() if k not in provenance})
        r['original_capture']=original_capture;r['original_page']=original_page;r['reference_capture']=ref_capture;r['capture_provenance_fields_excluded_only_for_independent_first_capture']=sorted(provenance);r['checks']['independent_rc']=sample_law(original_page);record('capture-ready',captured_state);assert manual.read_bytes()==manual_bytes
    barrier('capture-completed',captured)
    def automatic(s):
        global choice,snapshot,capsule_bytes
        deadline=time.monotonic()+12;choice=None
        while time.monotonic()<deadline and choice is None:
            state=client.ask('project.inspect');live_unchanged(captured_state,state);assert manual.read_bytes()==manual_bytes
            for row in state['recovery']['candidates']:
                if row['recording_count']!=1 or row.get('source_path')!=str(manual):continue
                inspected=client.ask('recovery.inspect',dict(id=row['id'],sha256=row['sha256']))
                if inspected['document']['recordings'][0]==original_capture['metadata']:choice=row;break
            if choice is None:time.sleep(.05)
        assert choice is not None;snapshot=directory/(choice['id']+'.elab-recovery');capsule_bytes=snapshot.read_bytes();assert hashlib.sha256(capsule_bytes).hexdigest()==choice['sha256'];shutil.copyfile(snapshot,out/'Original recorded snapshot.elab-recovery');r['choice']=choice;r['capsule_sha256']=choice['sha256'];save()
    barrier('automatic-snapshot',automatic);wait('allow-crash',True);capsule_unchanged();pid=host.pid;host.kill();code=host.wait(timeout=10);assert code==-9;client.close();client=None
    status=subprocess.run(['/bin/ps','-p',str(pid),'-o','pid=,comm='],capture_output=True,text=True);assert status.returncode==1 and not status.stdout.strip();r['hosts'][0].update(exit_code=-9,process_absent=True);save();(out/'first-stopped.json').write_text('{}\n')
    deadline=time.monotonic()+900
    while not (out/'allow-relaunch').exists():assert time.monotonic()<deadline;time.sleep(.05)
    reference.close();reference=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']));reference.ask('recovery.configure',dict(directory=str(directory),enabled=False,interval_s=2),True);reference.ask('simulation.wait')
    phase='restored';host,client,blank=launch(second);r['hosts'].append(dict(phase=phase,pid=host.pid));candidate=next(row for row in blank['recovery']['candidates'] if row['id']==choice['id']);assert candidate['owner_status']=='inactive' and not candidate['active_session'];r['inactive_choice']=candidate;save();(out/'restored-ready.json').write_text('{}\n')
    reference.ask('recovery.select',dict(id=choice['id']),True)
    def selected(s):assert s['recovery']['selected']==choice['id'];live_unchanged(blank,s,True)
    barrier('recovery-selected',selected);reference_read('inspection')
    def inspected(s):
        work,result=read_complete(client,'inspection');assert result['inspection']['document']['recordings'][0]==original_capture['metadata'];live_unchanged(blank,client.ask('project.inspect'),True);r['inspection_work']=work;save()
    barrier('recovery-inspected',inspected)
    for label,offset in [('recovery-page-first',0),('recovery-page-last',20)]:
        _,ref_result=reference_read('samples',offset)
        def page_check(s,offset=offset,label=label):
            work,result=read_complete(client,'samples',offset);page=result['samples'];assert page==ref_result['samples'];assert page['rows']==original_page['rows'][offset:offset+20] and page['probes']==original_page['probes'] and page['total']==21;live_unchanged(blank,client.ask('project.inspect'),True);r['checks'][label]=dict(work=work,page=page);save();capsule_unchanged()
        barrier(label,page_check)
    reference.ask('recovery.restore',dict(id=choice['id'],sha256=choice['sha256'],expected_document_generation=blank['document']['generation']),True)
    def restored_check(s):
        compare_state('recovery-restored',s);assert s['paused'] and s['transient']['sample']['time_s']==0 and s['document']['recording_count']==1 and s['document']['dirty'] and not s['document']['path'];assert client.ask('operation.inspect',dict(id=s['last_operation']))['metadata']==original_capture['metadata'];capsule_unchanged()
    restored=barrier('recovery-restored',restored_check)
    for label,command,params in [('recorded-view','recording.view.set',dict(mode='recorded')),('recorded-next','recording.step',dict(direction='next'))]:
        reference.ask(command,params,True)
        def view_check(s,label=label):compare_state(label,s);p.compare(s['recording'],reference.ask('simulation.wait')['recording']);live_unchanged(restored,s)
        barrier(label,view_check)
    seek_before=client.ask('simulation.wait')
    def private_seek(s):live_unchanged(seek_before,s,True);assert s['recording']==seek_before['recording']
    barrier('private-seek',private_seek);reference.ask('recording.seek',dict(time_s='20 ms'),True)
    def end(s):compare_state('recorded-end',s);p.compare(s['recording'],reference.ask('simulation.wait')['recording']);assert s['recording']['sample_index']==20 and s['recording']['time_s']==.02;live_unchanged(restored,s);capsule_unchanged()
    barrier('recorded-end',end);reference.ask('recording.view.set',dict(mode='live'),True)
    def live(s):compare_state('view-live',s);assert not s['recording']['viewing'];live_unchanged(restored,s)
    barrier('view-live',live)
    def saved(s):
        client.ask('project.save',dict(path=str(recovered)),True);reference.ask('project.save',dict(path=str(out/'Agent recovered RC.elab')),True);a=json.loads(recovered.read_text());b=json.loads((out/'Agent recovered RC.elab').read_text());assert {k:v for k,v in a.items() if k!='view'}=={k:v for k,v in b.items() if k!='view'};capsule=json.loads(capsule_bytes);assert a['recordings']['entries']==capsule['document']['recordings']['entries'];capsule_unchanged();record('saved-clean',client.ask('simulation.wait'));r['saved_fields_except_view_match']=True;save()
    barrier('save-recovered',saved);wait('finish',True);client.ask('app.close',{},True);client.close();client=None;code=host.wait(timeout=15);assert code==0;r['hosts'][1].update(exit_code=0,normal_closure=True);capsule_unchanged();r['passed']=True;save()
except BaseException as error:
    r['error']=type(error).__name__+': '+str(error);save()
    if host and host.poll() is None:
        while not (out/'allow-error-cleanup').exists() and host.poll() is None:time.sleep(.1)
    raise
finally:
    if client:client.close()
    if reference:reference.close()
    if host and host.poll() is None:
        host.terminate()
        try:host.wait(timeout=10)
        except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
    r['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
