import datetime,hashlib,importlib.util,json,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-decisions-156-final';out=base/'observations'
sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/recording-owner-review-checkpoint.json')
spec=importlib.util.spec_from_file_location('decision_parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';endpoint=out/'endpoint.json'
assert hashlib.sha256(binary.read_bytes()).hexdigest()=='24e98f76934fbc2956474ef0db5bd10ed7d71ef6adf1758039f6222110d56cdf'
for line in subprocess.check_output(['/bin/ps','-axo','command='],text=True).splitlines():
 assert 'ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery' not in line,'an existing harness instance is running'
working=out/'Working.elab';target=out/'Target.elab'
assert not working.exists() and not target.exists() and not endpoint.exists()
result=dict(passed=False,scope='Actual native unsaved Cancel/Discard/Save-and-continue and picker Cancel; isolated /156',release=proof,native_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),events=[])
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
 raise TimeoutError('UI observation lease ended without required decision stage')
def seed(c):
 c.ask('example.reset',dict(discard_changes=True),True);return c.ask('simulation.wait')
def value(c,text):
 c.ask('component.set',dict(id='R1',value=text),True);return c.ask('simulation.wait')
def compare(left,right,identities=False):
 for key in ['components','wires','nets','selected','paused','transient_settings','undo_name','redo_name']:
  assert (key in left)==(key in right) and left.get(key)==right.get(key),key
 parity.compare(left['result'],right['result']);parity.compare(left['scope'],right['scope'])
 if identities:
  for key in ['result','run_id','electrical_revision','undo_entry','redo_entry','document','status']:
   assert (key in left)==(key in right) and left.get(key)==right.get(key),key
with (out/'host.txt').open('x') as log:
 try:
  reference=parity.helper.Client(command_preconditions=True,agent_binary=agent)
  seed(reference);value(reference,'2k');reference.ask('project.save',dict(path=str(target)),True);reference.ask('simulation.wait')
  original_target=target.read_bytes();(out/'Target-original.elab').write_bytes(original_target)
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);result['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():
   assert host.poll() is None;assert time.monotonic()<deadline,'endpoint admission timeout';time.sleep(.05)
  client=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent)
  seed(client);client.ask('project.save',dict(path=str(working)),True);client.ask('simulation.wait')
  working_bytes=working.read_bytes();(out/'Working-original.elab').write_bytes(working_bytes)
  dirty=value(client,'2.2k');assert dirty['document']['dirty'];event('dirty-before-cancel',dirty)
  marker('ready.json',dict(pid=host.pid,working=str(working),target=str(target)))
  canceled=wait_for(lambda s:(out/'cancel-observed').exists());event('cancel-observed-before-check',canceled);compare(dirty,canceled,True);assert working.read_bytes()==working_bytes and target.read_bytes()==original_target
  event('unsaved-cancel',canceled);marker('cancel-verified.json')
  picker_canceled=wait_for(lambda s:(out/'picker-cancel-observed').exists());compare(canceled,picker_canceled,True);assert working.read_bytes()==working_bytes and target.read_bytes()==original_target
  event('picker-cancel',picker_canceled);marker('discard-ready.json')
  discarded=wait_for(lambda s:s['document']['path']==str(target) and s['document']['generation']>dirty['document']['generation'] and not s['document']['dirty'] and not s['result']['stale'])
  assert discarded['project_work']['actor']=='user' and discarded['project_work']['command']=='project.load'
  assert discarded['circuit']['resistance_ohm']==2000 and working.read_bytes()==working_bytes and target.read_bytes()==original_target
  reference.ask('project.load',dict(path=str(working),discard_changes=True),True);reference.ask('simulation.wait');value(reference,'2.2k');reference.ask('project.load',dict(path=str(target),discard_changes=True),True);ref_discarded=reference.ask('simulation.wait');compare(discarded,ref_discarded)
  event('discard-loaded',discarded);event('agent-discard-reference',ref_discarded);marker('discard-observed.json')
  wait_for(lambda s:(out/'allow-save-prepare').exists())
  tosave=value(client,'2.2k');assert tosave['document']['dirty'];event('dirty-before-save',tosave);marker('save-ready.json')
  saved_loaded=wait_for(lambda s:s['document']['path']==str(working) and s['document']['generation']>discarded['document']['generation'] and not s['document']['dirty'] and not s['result']['stale'])
  assert saved_loaded['project_work']['actor']=='user' and saved_loaded['project_work']['command']=='project.load'
  assert saved_loaded['circuit']['resistance_ohm']==1000 and working.read_bytes()==working_bytes
  target_doc=json.loads(target.read_text());assert target_doc['graph']['components']==tosave['components'] and target_doc['graph']['wires']==tosave['wires']
  assert next(c for c in target_doc['graph']['components'] if c['id']=='R1')['value']==2200
  value(reference,'2.2k');reference.ask('project.save',dict(path=str(out/'Agent-preserved.elab')),True);reference.ask('simulation.wait')
  ref_doc=json.loads((out/'Agent-preserved.elab').read_text())
  for key in target_doc:
   if key!='view':assert target_doc[key]==ref_doc[key],key
  reference.ask('project.load',dict(path=str(working)),True);ref_saved=reference.ask('simulation.wait');compare(saved_loaded,ref_saved)
  event('save-then-loaded',saved_loaded);event('agent-save-reference',ref_saved);result['passed']=True;marker('save-observed.json');save()
  wait_for(lambda s:(out/'finish-observation').exists())
  client.ask('app.close',{},True);client.close();client=None;result['host_exit_code']=host.wait(timeout=15);result['normal_closure']=result['host_exit_code']==0
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
