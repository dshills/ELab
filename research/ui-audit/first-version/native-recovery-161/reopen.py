import hashlib,importlib.util,json,subprocess,sys,time
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab');base=Path(__file__).resolve().parent;out=base/'observations';folder=out/'reopen';folder.mkdir()
sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
release=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json');spec=importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
source=out/'restored/Recovered LED.elab';manual=out/'first/Manual LED.elab';first=json.loads((out/'report.json').read_text());source_bytes=source.read_bytes();manual_bytes=manual.read_bytes();assert hashlib.sha256(manual_bytes).hexdigest()==first['manual_save_sha256'];assert json.loads((out/'corrected-close.json').read_text())['process_absent']
assert not [line for line in subprocess.check_output(['/bin/ps','-axo','pid=,comm='],text=True).splitlines() if 'ELab-value-recovery' in line]
binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';assert hashlib.sha256(binary.read_bytes()).hexdigest()==first['identity']['native_sha256']
r=dict(passed=False,release=release,events=[],scope='Fresh owned native app opens the separately recovered file through the real Open panel and closes through Cmd+W; records its child exit code. Original controller failure remains preserved.',source_sha256=hashlib.sha256(source_bytes).hexdigest());host=client=reference=None
def save():
    pending=folder/'report.pending';pending.write_text(json.dumps(r,indent=2)+'\n');pending.replace(folder/'report.json')
def wait(name):
    deadline=time.monotonic()+900
    while not (folder/name).exists():assert host.poll() is None and time.monotonic()<deadline;time.sleep(.05)
    state=client.ask('simulation.wait');r['events'].append(dict(name=name,state=state));save();return state
def ack(name):(folder/(name+'-verified.json')).write_text('{}\n')
try:
    reference=p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']));reference.ask('project.load',dict(path=str(source)),True);expected=reference.ask('simulation.wait');r['reference']=expected
    endpoint=folder/'endpoint.json'
    with (folder/'native-host.txt').open('x') as log:host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--recovery-directory',str(base/'recovery'),'--no-appearance'],stdout=log,stderr=-2)
    r['pid']=host.pid;save();deadline=time.monotonic()+20
    while not endpoint.exists():assert host.poll() is None and time.monotonic()<deadline;time.sleep(.05)
    client=p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=Path(release['agent']));blank=client.ask('simulation.wait');assert not blank['components'] and not blank['wires'];(folder/'ready.json').write_text('{}\n')
    loaded=wait('reopen-loaded');assert loaded['document']['path']==str(source) and not loaded['document']['dirty'];assert loaded['project_work']['status']=='completed' and loaded['project_work']['actor']=='user'
    for key in ['components','wires','nets','selected','drawing','paused','transient_settings','undo_name','redo_name']:assert loaded.get(key)==expected.get(key),key
    for key in ['result','scope','transient','npn','mixed']:p.compare(loaded.get(key),expected.get(key),key)
    assert source.read_bytes()==source_bytes and manual.read_bytes()==manual_bytes;ack('reopen-loaded')
    before=wait('reopen-close');assert not before['document']['dirty'];ack('reopen-close');r['native_close_ready']=True;save()
    code=host.wait(timeout=30);r['native_exit_code']=code;assert code==0;client.close();client=None
    assert source.read_bytes()==source_bytes and manual.read_bytes()==manual_bytes;r['passed']=True;r['normal_closure']=True;save()
except BaseException as error:
    r['error']=type(error).__name__+': '+str(error);save()
    if host and host.poll() is None:
        while not (folder/'allow-error-cleanup').exists() and host.poll() is None:time.sleep(.1)
    raise
finally:
    if client:client.close()
    if reference:reference.close()
    if host and host.poll() is None:
        host.terminate()
        try:host.wait(timeout=10)
        except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
    if host:r['final_native_exit_code']=host.returncode
    save()
