import datetime,hashlib,importlib.util,json,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-files-156-ui';out=base/'observations'
sys.path.insert(0,str(root/'scripts'))
spec=importlib.util.spec_from_file_location('file_parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
agent=root/'build/ELab-0.139.0-156-review/elab-agent';binary=base/'ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';endpoint=out/'endpoint.json'
assert hashlib.sha256(binary.read_bytes()).hexdigest()=='24e98f76934fbc2956474ef0db5bd10ed7d71ef6adf1758039f6222110d56cdf'
for line in subprocess.check_output(['/bin/ps','-axo','command='],text=True).splitlines():
 assert 'ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery' not in line,'an existing harness instance is running'
expected=out/'LED UI.elab';changed=out/'LED agent change.elab';assert not expected.exists() and not changed.exists()
result=dict(passed=False,scope='Native macOS Save As/Open through CUA; owned /156 process and matching packaged agent',started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),native_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),agent_sha256=hashlib.sha256(agent.read_bytes()).hexdigest(),events=[],view_history=[])
client=None;host=None

def save():
 pending=out/'report.pending';pending.write_text(json.dumps(result,indent=2)+'\n');pending.replace(out/'report.json')
def event(name,state):
 result['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=state));save()
def wait_for(predicate,seconds=300):
 deadline=time.monotonic()+seconds;previous=None
 while time.monotonic()<deadline:
  assert host.poll() is None,'original native host exited'
  state=client.ask('project.inspect')
  if state['command_revision']!=previous:
   result['view_history'].append(dict(revision=state['command_revision'],viewport=state['viewport'],view=state['view']));previous=state['command_revision'];save()
  if predicate(state):return state
  time.sleep(.1)
 raise TimeoutError('UI observation lease ended without the required stage')
def seed(target):
 target.ask('example.reset',dict(discard_changes=True),True);target.ask('simulation.wait');target.ask('component.select',dict(id='R1'),True);return target.ask('simulation.wait')
with (out/'host.txt').open('x') as log:
 try:
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);result['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():
   assert host.poll() is None;assert time.monotonic()<deadline,'endpoint admission timeout';time.sleep(.05)
  client=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);baseline=seed(client);event('baseline',baseline)
  (out/'ready.json').write_text(json.dumps(dict(pid=host.pid,save_path=str(expected)),indent=2)+'\n')
  saved=wait_for(lambda state:state['document']['path']==str(expected) and not state['document']['dirty'] and expected.exists())
  for key in ['components','wires','nets','result','run_id','electrical_revision','undo_entry','redo_entry']:assert saved.get(key)==baseline.get(key),key
  ui_document=json.loads(expected.read_text());assert ui_document['format']=='elab.schematic' and ui_document['version']==4
  assert ui_document['graph']['components']==baseline['components'] and ui_document['graph']['wires']==baseline['wires']
  event('ui-saved',saved);(out/'save-observed.json').write_text(json.dumps(dict(pid=host.pid,path=str(expected)),indent=2)+'\n')
  wait_for(lambda state:(out/'allow-agent-change').exists())
  client.ask('component.set',dict(id='R1',value='2.2k'),True);client.ask('simulation.wait')
  client.ask('project.save',dict(path=str(changed)),True);changed_state=client.ask('simulation.wait');assert not changed_state['document']['dirty'];event('agent-changed-and-saved',changed_state)
  (out/'open-ready.json').write_text(json.dumps(dict(pid=host.pid,open_path=str(expected),current_path=str(changed)),indent=2)+'\n')
  loaded=wait_for(lambda state:state['document']['path']==str(expected) and state['document']['generation']>baseline['document']['generation'] and not state['document']['dirty'] and state['circuit']['resistance_ohm']==1000 and state.get('result') and not state['result']['stale'])
  for key in ['components','wires','nets','transient_settings']:assert loaded[key]==baseline[key],key
  parity.compare(loaded['result'],baseline['result']);assert loaded['project_work']['command']=='project.load' and loaded['project_work']['actor']=='user'
  event('ui-opened',loaded);(out/'open-observed.json').write_text(json.dumps(dict(pid=host.pid,path=str(expected)),indent=2)+'\n')
  reference=parity.helper.Client(command_preconditions=True,agent_binary=agent)
  try:
   seed(reference)
   matching=next(v for v in result['view_history'] if v['view']==ui_document['view'] and v['viewport']['registered'])
   reference.ask('view.viewport.set',dict(width_px=matching['viewport']['width_px'],height_px=matching['viewport']['height_px']),True)
   agent_path=out/'LED agent.elab';reference.ask('project.save',dict(path=str(agent_path)),True);reference.ask('simulation.wait')
   assert json.loads(agent_path.read_text())==ui_document,'independent agent save differs from native UI document'
   reference.ask('project.load',dict(path=str(expected)),True);agent_loaded=reference.ask('simulation.wait')
   for key in ['components','wires','nets','transient_settings']:assert agent_loaded[key]==loaded[key],key
   parity.compare(agent_loaded['result'],loaded['result']);event('packaged-agent-opened',agent_loaded)
  finally:reference.close()
  result['independent_agent_document_equal']=True;result['passed']=True;save()
  wait_for(lambda state:(out/'finish-observation').exists())
  client.ask('app.close',{},True);client.close();client=None;result['host_exit_code']=host.wait(timeout=15);result['normal_closure']=result['host_exit_code']==0
 except BaseException as error:
  result['error']=type(error).__name__+': '+str(error);raise
 finally:
  if client:client.close()
  if host and host.poll() is None:
   host.terminate()
   try:host.wait(timeout=10)
   except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
  if host:result['final_host_exit_code']=host.returncode
  result['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
