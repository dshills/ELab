import datetime,hashlib,importlib.util,json,shutil,subprocess,sys,time
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab');base=Path(__file__).resolve().parent;out=base/'observations'
first=out/'first';second=out/'restored';first.mkdir();second.mkdir();directory=base/'recovery';directory.mkdir(mode=0o700)
sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
release=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json');identity=json.loads((base/'identity.json').read_text())
binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';assert hashlib.sha256(binary.read_bytes()).hexdigest()==identity['native_sha256']
spec=importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
manual=first/'Manual LED.elab';recovered=second/'Recovered LED.elab'
report=dict(passed=False,release=release,identity=identity,events=[],reference={},hosts=[],
    scope='Actual native LED example, Save dialog, private value edit, automatic private recovery, owned SIGKILL, fresh native recovery selection/inspection/restore and separate native Save dialog. No public native graph or trace seeding. Last manual save and original capsule must remain unchanged.',
    started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
host=client=reference=None;phase='first'

def save():
    path=out/'report.pending';path.write_text(json.dumps(report,indent=2)+'\n');path.replace(out/'report.json')
def ack(name):(out/(name+'-verified.json')).write_text('{}\n')
def wait(name):
    deadline=time.monotonic()+900
    while not (out/name).exists():
        assert host.poll() is None,'host exited before '+name
        assert time.monotonic()<deadline,'UI lease expired '+name
        time.sleep(.05)
    state=client.ask('simulation.wait');report['events'].append(dict(name=name,phase=phase,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=state));save();return state
def equivalent(a,b):
    for key in ['components','wires','nets','selected','drawing','paused','transient_settings','undo_name','redo_name']:assert a.get(key)==b.get(key),key
    for key in ['result','npn','mixed','transient','scope','trace_focus']:p.compare(a.get(key),b.get(key),key)
def check(name):
    expected=reference.ask('simulation.wait');report['reference'][name]=expected;state=wait(name);equivalent(state,expected);ack(name);return state
def protected(before,after):
    for key in ['components','wires','nets','selected','result','npn','mixed','transient','scope','document','paused','run_id','electrical_revision','undo_entry','redo_entry']:assert before.get(key)==after.get(key),key
def stable_file():
    assert manual.read_bytes()==baseline_bytes,'last manual save changed'
def launch(folder):
    endpoint=folder/'endpoint.json';assert not endpoint.exists()
    log=(folder/'native-host.txt').open('x');process=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--recovery-directory',str(directory),'--no-appearance'],stdout=log,stderr=-2);log.close();deadline=time.monotonic()+20
    while not endpoint.exists():assert process.poll() is None and time.monotonic()<deadline;time.sleep(.05)
    connection=p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=Path(release['agent']));state=connection.ask('simulation.wait')
    assert not state['components'] and not state['wires'] and state['recovery']['directory']==str(directory) and state['recovery']['enabled']
    return process,connection,state
def source_choice(state):
    return next((row for row in state['recovery']['candidates'] if row['id']==choice['id']),None)

try:
    reference=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']));reference.ask('example.reset',{},True);reference.ask('simulation.wait');reference.ask('simulation.pause',{},True)
    host,client,blank=launch(first);report['hosts'].append(dict(phase=phase,pid=host.pid));save();(out/'first-ready.json').write_text('{}\n')
    loaded=check('loaded-paused');assert loaded['paused'] and not loaded['result']['stale']
    baseline=wait('baseline-saved');assert baseline['document']['path']==str(manual) and not baseline['document']['dirty'];assert baseline['project_work']['status']=='completed' and baseline['project_work']['actor']=='user'
    baseline_bytes=manual.read_bytes();report['manual_save_sha256']=hashlib.sha256(baseline_bytes).hexdigest();report['manual_save_project_work']=baseline['project_work'];reference.ask('project.save',dict(path=str(out/'Agent baseline.elab')),True)
    a=json.loads(baseline_bytes);b=json.loads((out/'Agent baseline.elab').read_text());assert {k:v for k,v in a.items() if k!='view'}=={k:v for k,v in b.items() if k!='view'};ack('baseline-saved')
    reference.ask('component.select',dict(id='R1'),True);selected=check('selected-R1');private=wait('private-value');protected(selected,private);ack('private-value')
    reference.ask('editor.value.set',dict(value='2.2k'),True);reference.ask('component.set',dict(id='R1',value='2.2k'),True);stale=check('edited-stale');assert stale['result']['stale'];stable_file()
    reference.ask('simulation.resume',{},True);reference.ask('simulation.wait');reference.ask('simulation.pause',{},True);edited=check('edited-fresh');assert not edited['result']['stale'] and edited['document']['dirty'];stable_file()
    wait('await-automatic');deadline=time.monotonic()+12;choice=None;inspected=None
    while time.monotonic()<deadline and choice is None:
        state=client.ask('project.inspect');protected(edited,state);stable_file()
        for row in state['recovery']['candidates']:
            if row.get('source_path')!=str(manual):continue
            inspection=client.ask('recovery.inspect',dict(id=row['id'],sha256=row['sha256']))
            doc=inspection['document']
            if doc['graph']['components']==edited['components'] and doc['graph']['wires']==edited['wires'] and doc['paused']:
                choice=row;inspected=inspection;break
        if choice is None:time.sleep(.05)
    assert choice is not None,'automatic complete edited snapshot not observed'
    snapshot=directory/(choice['id']+'.elab-recovery');capsule_bytes=snapshot.read_bytes();assert hashlib.sha256(capsule_bytes).hexdigest()==choice['sha256']
    capsule=json.loads(capsule_bytes)
    # Public recovery.inspect above validates the original encoded document hash;
    # preserve the raw capsule bytes rather than re-encoding their Go JSON.
    shutil.copyfile(snapshot,out/'Original automatic snapshot.elab-recovery');report['automatic_candidate']=choice;report['automatic_inspection']=inspected;report['automatic_capsule_sha256']=choice['sha256'];save();(out/'choice.json').write_text(json.dumps(choice,indent=2)+'\n');ack('await-automatic')
    wait('allow-crash');stable_file();assert snapshot.read_bytes()==capsule_bytes
    first_pid=host.pid;host.kill();code=host.wait(timeout=10);assert code==-9;client.close();client=None
    status=subprocess.run(['/bin/ps','-p',str(first_pid),'-o','pid=,comm='],text=True,capture_output=True);assert status.returncode==1 and not status.stdout.strip();report['hosts'][0].update(exit_code=code,controlled_sigkill=True,process_absent=True);save();(out/'first-stopped.json').write_text('{}\n')
    deadline=time.monotonic()+900
    while not (out/'allow-relaunch').exists():assert time.monotonic()<deadline;time.sleep(.05)
    reference.close();reference=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']));reference.ask('recovery.configure',dict(directory=str(directory),enabled=False,interval_s=2),True);reference.ask('simulation.wait')
    phase='restored';host,client,blank=launch(second);report['hosts'].append(dict(phase=phase,pid=host.pid));candidate=source_choice(blank);assert candidate and candidate['sha256']==choice['sha256'] and not candidate['active_session'] and not candidate['current_session'];report['inactive_candidate']=candidate;save();(out/'restored-ready.json').write_text('{}\n')
    reference.ask('recovery.select',dict(id=choice['id']),True);selected=wait('recovery-selected');assert selected['recovery']['selected']==choice['id'];protected(blank,selected);ack('recovery-selected')
    reference.ask('recovery.read.start',dict(kind='inspection',id=choice['id'],sha256=choice['sha256']),True);wait('recovery-inspected');deadline=time.monotonic()+10
    while True:
        state=client.ask('project.inspect');protected(selected,state);works=[w for w in state['recovery'].get('reads',[]) if w['input']['id']==choice['id'] and w['input']['kind']=='inspection']
        if works and works[-1]['status']=='completed':break
        assert time.monotonic()<deadline;time.sleep(.05)
    report['inspection_read_work']=works[-1];save();stable_file();ack('recovery-inspected')
    reference.ask('recovery.restore',dict(id=choice['id'],sha256=choice['sha256'],expected_document_generation=blank['document']['generation']),True);restored=check('recovery-restored')
    assert restored['components']==edited['components'] and restored['wires']==edited['wires'];assert restored['document']['dirty'] and not restored['document']['path'] and restored['document']['recovered_from']==str(manual);assert restored['paused'] and not restored['result']['stale'];p.compare(restored['result'],edited['result']);stable_file();assert snapshot.read_bytes()==capsule_bytes
    saved=wait('recovered-saved');assert saved['document']['path']==str(recovered) and not saved['document']['dirty'];assert saved['project_work']['status']=='completed' and saved['project_work']['actor']=='user';report['recovered_save_project_work']=saved['project_work'];stable_file();assert snapshot.read_bytes()==capsule_bytes
    reference.ask('project.save',dict(path=str(out/'Agent recovered.elab')),True);a=json.loads(recovered.read_bytes());b=json.loads((out/'Agent recovered.elab').read_text());assert {k:v for k,v in a.items() if k!='view'}=={k:v for k,v in b.items() if k!='view'};equivalent(saved,reference.ask('simulation.wait'));report['saved_fields_except_view_match']=True;save();ack('recovered-saved')
    wait('allow-native-close');(out/'close-ready.json').write_text('{}\n');code=host.wait(timeout=30);assert code==0;client.close();client=None;report['hosts'][1].update(exit_code=code,normal_closure=True);stable_file();assert snapshot.read_bytes()==capsule_bytes;report['passed']=True;save()
except BaseException as error:
    report['error']=type(error).__name__+': '+str(error);save()
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
    report['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
