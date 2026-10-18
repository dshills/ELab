import datetime,hashlib,importlib.util,json,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-untitled-156';out=base/'observations'
sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/recording-owner-review-checkpoint.json')
spec=importlib.util.spec_from_file_location('untitled_parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';endpoint=out/'endpoint.json'
assert hashlib.sha256(binary.read_bytes()).hexdigest()=='24e98f76934fbc2956474ef0db5bd10ed7d71ef6adf1758039f6222110d56cdf'
for line in subprocess.check_output(['/bin/ps','-axo','command='],text=True).splitlines():
 assert 'ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery' not in line,'existing harness is running'
target=out/'Target.elab';kept=out/'Preserved untitled.elab';assert not target.exists() and not kept.exists() and not endpoint.exists()
result=dict(passed=False,scope='Actual native untitled Save-and-continue picker cancellation and retry; isolated /156',release=proof,native_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),events=[])
client=None;reference=None;host=None

def save():
 pending=out/'report.pending';pending.write_text(json.dumps(result,indent=2)+'\n');pending.replace(out/'report.json')
def event(name,state):
 result['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=state));save()
def marker(name,data=None):
 (out/name).write_text(json.dumps(data or dict(pid=host.pid),indent=2)+'\n')
def wait_for(predicate,seconds=300):
 deadline=time.monotonic()+seconds
 while time.monotonic()<deadline:
  assert host.poll() is None,'original native host exited'
  state=client.ask('project.inspect')
  if predicate(state):return state
  time.sleep(.1)
 raise TimeoutError('UI observation lease ended without required untitled stage')
def seed(c,value):
 c.ask('example.reset',dict(discard_changes=True),True);c.ask('simulation.wait');c.ask('component.set',dict(id='R1',value=value),True);return c.ask('simulation.wait')
def compare(left,right,identities=False):
 for key in ['components','wires','nets','selected','paused','transient_settings','undo_name','redo_name']:
  assert (key in left)==(key in right) and left.get(key)==right.get(key),key
 parity.compare(left['result'],right['result']);parity.compare(left['scope'],right['scope'])
 if identities:
  for key in ['result','scope','run_id','electrical_revision','undo_entry','redo_entry','document','status']:
   assert (key in left)==(key in right) and left.get(key)==right.get(key),key
with (out/'host.txt').open('x') as log:
 try:
  reference=parity.helper.Client(command_preconditions=True,agent_binary=agent)
  seed(reference,'2k');reference.ask('project.save',dict(path=str(target)),True);reference.ask('simulation.wait');reference.ask('project.load',dict(path=str(target)),True);target_state=reference.ask('simulation.wait');target_bytes=target.read_bytes();(out/'Target-original.elab').write_bytes(target_bytes)
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);result['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():
   assert host.poll() is None;assert time.monotonic()<deadline,'endpoint admission timeout';time.sleep(.05)
  client=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);dirty=seed(client,'2.2k');assert dirty['document']['path']=='' and dirty['document']['dirty'];compare(dirty,dirty,True);event('untitled-dirty',dirty)
  marker('ready.json',dict(pid=host.pid,target=str(target),preserve_path=str(kept)))
  canceled=wait_for(lambda s:(out/'save-picker-cancel-observed').exists());event('save-picker-canceled',canceled);compare(dirty,canceled,True);assert not kept.exists() and target.read_bytes()==target_bytes;marker('cancel-verified.json')
  loaded=wait_for(lambda s:s['document']['path']==str(target) and s['document']['generation']>dirty['document']['generation'] and not s['document']['dirty'] and not s['result']['stale'])
  event('save-and-continued',loaded);assert loaded['project_work']['actor']=='user' and loaded['project_work']['command']=='project.load';assert kept.exists() and target.read_bytes()==target_bytes
  document=json.loads(kept.read_text());assert document['graph']['components']==dirty['components'] and document['graph']['wires']==dirty['wires']
  compare(loaded,target_state)
  seed(reference,'2.2k');reference.ask('project.save',dict(path=str(out/'Agent-preserved.elab')),True);reference.ask('simulation.wait');ref_doc=json.loads((out/'Agent-preserved.elab').read_text())
  assert {k:v for k,v in document.items() if k!='view'}=={k:v for k,v in ref_doc.items() if k!='view'}
  reference.ask('project.load',dict(path=str(kept)),True);reopened=reference.ask('simulation.wait');event('agent-reopened-preserved',reopened)
  for key in ['components','wires','nets','transient_settings']:assert reopened[key]==dirty[key],key
  parity.compare(reopened['result'],dirty['result']);assert not reopened['document']['dirty'] and not reopened['result']['stale']
  reference.ask('project.load',dict(path=str(target)),True);ref_loaded=reference.ask('simulation.wait');event('agent-opened-target',ref_loaded);compare(loaded,ref_loaded)
  result['passed']=True;save();marker('save-observed.json')
  wait_for(lambda s:(out/'finish-observation').exists());client.ask('app.close',{},True);client.close();client=None;result['host_exit_code']=host.wait(timeout=15);result['normal_closure']=result['host_exit_code']==0
 except BaseException as error:
  result['error']=type(error).__name__+': '+str(error);raise
 finally:
  if reference:reference.close()
  if client:client.close()
  if host and host.poll() is None:
   host.terminate()
   try:host.wait(timeout=10)
   except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
  if host:result['final_host_exit_code']=host.returncode
  result['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
