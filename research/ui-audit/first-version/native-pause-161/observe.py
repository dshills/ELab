import datetime,hashlib,importlib.util,json,subprocess,sys,time
from pathlib import Path
from references import npn_check,mosfet_check,mosfet_trace_check

root=Path('/Users/dshills/Development/projects/ELab');base=Path(__file__).parent;out=base/'observations'
sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
release=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
identity=json.loads((base/'identity.json').read_text());binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery'
assert hashlib.sha256(binary.read_bytes()).hexdigest()==identity['native_sha256']
endpoint=out/'endpoint.json';assert not endpoint.exists()
spec=importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
independent=json.loads((base/'independent-references.json').read_text())
report=dict(passed=False,release=release,identity=identity,events=[],reference={},cycles=[],
    scope='Native blank → NPN pulse controls/save → NMOS example/probe sequence, then six single-click live Resume/Pause cycles. Require visible completion; no automatic click retry; retain owned host on failed completion for diagnosis.',
    started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
host=client=reference=None

def save():
    pending=out/'report.pending';pending.write_text(json.dumps(report,indent=2)+'\n');pending.replace(out/'report.json')

def ack(name):(out/(name+'-verified.json')).write_text('{}\n')

def wait(name):
    deadline=time.monotonic()+900
    while not (out/name).exists():
        assert host.poll() is None,'host exited before '+name
        assert time.monotonic()<deadline,'observation lease expired '+name
        # A CUA completion failure is a diagnostic state, not permission to stop
        # the native app or retry its click. Preserve subsequent live snapshots.
        if (out/'completion-failed.json').exists() and 'completion_failure' not in report:
            report['completion_failure']=json.loads((out/'completion-failed.json').read_text())
            report['completion_failure_state']=client.ask('simulation.wait');save()
        time.sleep(.05)
    state=client.ask('simulation.wait');report['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=state));save();return state

def equivalent(a,b):
    for key in ['components','wires','nets','selected','drawing','paused','transient_settings','undo_name','redo_name']:
        assert a.get(key)==b.get(key),key
    for key in ['npn','mixed','transient','scope','result','trace_focus']:p.compare(a.get(key),b.get(key),key)

def check(name):
    expected=reference.ask('simulation.wait');report['reference'][name]=expected;state=wait(name);equivalent(state,expected);ack(name);return state

def advance(target):
    at=reference.ask('simulation.wait')['transient']['sample']['time_s']
    while target-at>1e-10:
        reference.ask('simulation.step.start',dict(duration_s=min(1.,target-at)),True);at=reference.ask('simulation.wait')['transient']['sample']['time_s']

def reach(name,target):
    running=wait(name);assert not running['paused'];deadline=time.monotonic()+45
    while running['transient']['sample']['time_s']<target:
        assert host.poll() is None and time.monotonic()<deadline
        running=client.ask('project.inspect');assert not running['paused'] and not running['transient']['stale'] and running['status']!='error';time.sleep(.05)
    ack(name)

def pause_at(name,kind):
    state=wait(name);assert state['paused'];advance(state['transient']['sample']['time_s']);expected=reference.ask('simulation.wait');report['reference'][name]=expected;equivalent(state,expected)
    if kind=='npn':npn_check(state,independent)
    else:mosfet_check(state,'nmos');mosfet_trace_check(state,'nmos')
    sample=state['transient']['sample'];time.sleep(.12);stable=client.ask('project.inspect');assert stable['paused'] and stable['transient']['sample']==sample and stable['command_revision']==state['command_revision'];ack(name);return state

def save_pair(label):
    wait(label+'-allow-save')
    for service,actor in [(client,'Native'),(reference,'Agent')]:service.ask('project.save',dict(path=str(out/(actor+' '+label+'.elab'))),True)
    a=json.loads((out/('Native '+label+'.elab')).read_text());b=json.loads((out/('Agent '+label+'.elab')).read_text());assert {k:v for k,v in a.items() if k!='view'}=={k:v for k,v in b.items() if k!='view'};ack(label+'-allow-save')

with (out/'native-host.txt').open('x') as log:
    try:
        reference=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']))
        host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);report['pid']=host.pid;save();deadline=time.monotonic()+20
        while not endpoint.exists():assert host.poll() is None and time.monotonic()<deadline;time.sleep(.05)
        client=p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=Path(release['agent']));blank=client.ask('simulation.wait');assert not blank['components'] and not blank['wires'];(out/'ready.json').write_text('{}\n')
        reference.ask('example.npn-pulse',{},True);reference.ask('simulation.pause',{},True);reference.ask('simulation.restart',{},True);initial=reference.ask('simulation.wait');loaded=wait('npn-loaded-paused');assert loaded['paused']
        for key in ['components','wires','nets','transient_settings']:assert loaded[key]==initial[key],key
        ack('npn-loaded-paused');check('npn-restart');reach('npn-running',.155);pause_at('npn-paused','npn')
        for mode in ['low','high','auto']:
            reference.ask('simulation.input.set',dict(mode=mode),True);queued=check('npn-queued-'+mode);reference.ask('simulation.step.start',dict(duration_s=queued['transient_settings']['step_s']),True);check('npn-step-'+mode)
        save_pair('npn')
        reference.ask('example.load',dict(id='nmos'),True);reference.ask('simulation.pause',{},True);reference.ask('simulation.restart',{},True);initial=reference.ask('simulation.wait');loaded=wait('nmos-loaded-paused');assert loaded['paused']
        for key in ['components','wires','nets','transient_settings']:assert loaded[key]==initial[key],key
        ack('nmos-loaded-paused');check('nmos-restart');reference.ask('scope.probes.set',dict(ids=initial['scope']['selected']+['i:M1.gate']),True);check('nmos-gate-probe')
        for number in range(1,7):
            name=f'cycle-{number}';before=wait(name+'-before');assert before['paused'];start=before['transient']['sample']['time_s'];ack(name+'-before')
            reach(name+'-running',.555 if number==1 else start+.055)
            paused=pause_at(name+'-paused','nmos');assert paused['transient']['sample']['time_s']>start
            assert paused['command_revision']==before['command_revision']+2
            completion=[json.loads((out/(name+'-'+action+'-completion.json')).read_text()) for action in ['resume','pause']]
            assert all(row['passed'] and row['clicks']==1 for row in completion)
            report['cycles'].append(dict(number=number,passed=True,start_s=start,end_s=paused['transient']['sample']['time_s'],command_revision_before=before['command_revision'],command_revision_after=paused['command_revision'],visible_completion_observations=[len(row['observations']) for row in completion]));save()
        save_pair('nmos');report['passed']=True;save();(out/'all-verified.json').write_text('{}\n');wait('finish');client.ask('app.close',{},True);client.close();client=None;report['host_exit_code']=host.wait(timeout=15);report['normal_closure']=report['host_exit_code']==0
    except BaseException as error:
        report['error']=type(error).__name__+': '+str(error);save()
        # Keep the host for explicit diagnosis; terminal cleanup needs a marker.
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
        if host:report['final_host_exit_code']=host.returncode
        report['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
