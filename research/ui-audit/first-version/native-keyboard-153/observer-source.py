import datetime,hashlib,importlib.util,json,subprocess,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/keyboard-native-153-ui';out=base/'retry';out.mkdir()
spec=importlib.util.spec_from_file_location('key_ui_client',root/'scripts/persistence-demo.py');helper=importlib.util.module_from_spec(spec);spec.loader.exec_module(helper)
binary=base/'ELab-keyboard-153.app/Contents/MacOS/ELab-keyboard-153';endpoint=out/'endpoint.json'
result=dict(passed=False,scope='Owned byte-identical /153 executable in separately identified UI harness; fresh pointer-focus and Right Arrow only',binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),events=[])
def save(): (out/'report.json').write_text(json.dumps(result,indent=2)+'\n')
def event(name,state):
 result['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=state));save()
client=None;host=None
with (out/'host.txt').open('x') as log:
 try:
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2)
  result['pid']=host.pid;save()
  deadline=time.monotonic()+20
  while not endpoint.exists():
   assert host.poll() is None,'native host exited before endpoint';assert time.monotonic()<deadline,'endpoint admission timeout';time.sleep(.05)
  client=helper.Client(str(endpoint),command_preconditions=True,agent_binary=root/'build/ELab-0.136.0-153-review/elab-agent')
  client.ask('example.reset',dict(discard_changes=True),True);client.ask('simulation.wait')
  client.ask('simulation.pause',{},True);client.ask('component.select',dict(id='R1'),True)
  baseline=client.ask('simulation.wait');event('baseline',baseline)
  (out/'ready.json').write_text(json.dumps(dict(pid=host.pid,endpoint=str(endpoint),selected=baseline['selected']),indent=2)+'\n')
  origin=next(p for p in baseline['components'] if p['id']=='R1')['position'];previous=baseline['command_revision'];deadline=time.monotonic()+180;observed=False
  while time.monotonic()<deadline:
   assert host.poll() is None,'host retired during observation'
   state=client.ask('project.inspect')
   if state['command_revision']!=previous:
    event('command-change',state);previous=state['command_revision']
   part=next((p for p in state['components'] if p['id']=='R1'),None)
   if not observed and part and part['position']==dict(x=origin['x']+10,y=origin['y']):
    assert state['wires']==baseline['wires'] and state['nets']==baseline['nets'] and state['result']==baseline['result'],'move changed connectivity/physics'
    assert state['undo_name']=='Move component','unexpected history'
    event('one-grid-move-observed',state);result['passed']=True;observed=True;deadline=min(deadline,time.monotonic()+30)
   time.sleep(.1)
  result['observation_lease_ended']=True
  if observed:
   path=out/'keyboard-moved.elab';client.ask('project.save',dict(path=str(path)),True);client.ask('simulation.wait');result['saved_project']=str(path)
  client.ask('app.close',dict(discard_changes=not observed),True);client.close();client=None
  result['host_exit_code']=host.wait(timeout=15);result['normal_closure']=result['host_exit_code']==0
 except BaseException as error:
  result['error']=type(error).__name__+': '+str(error)
  raise
 finally:
  if client:client.close()
  if host and host.poll() is None:
   host.terminate()
   try:host.wait(timeout=10)
   except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
  if host:result['final_host_exit_code']=host.returncode
  result['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
