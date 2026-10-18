import datetime,hashlib,importlib.util,json,subprocess,time,sys
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/keyboard-native-153-ui';out=base/'value';out.mkdir()
sys.path.insert(0,str(root/'scripts'))
spec=importlib.util.spec_from_file_location('value_parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
agent=root/'build/ELab-0.136.0-153-review/elab-agent';binary=base/'ELab-keyboard-153.app/Contents/MacOS/ELab-keyboard-153';endpoint=out/'endpoint.json'
assert hashlib.sha256(binary.read_bytes()).hexdigest()=='71dc5a02d876d7fd8937d2b99d18619b0dd79a5c736580e5d321b59b17d37d29'
for line in subprocess.check_output(['/bin/ps','-axo','command='],text=True).splitlines():assert not line.strip().startswith(str(binary)),'another harness instance is running'
def seed(client):
 client.ask('example.reset',dict(discard_changes=True),True);client.ask('simulation.wait');client.ask('component.select',dict(id='R1'),True);state=client.ask('simulation.wait');assert not state['paused'] and not state['result']['stale'];return state
reference=parity.helper.Client(command_preconditions=True,agent_binary=agent)
try:
 baseline_reference=seed(reference)
 reference.ask('editor.value.set',dict(value='2.2k'),True)
 reference.ask('component.set',dict(id='R1',value='2.2k'),True);applied=reference.ask('simulation.wait')
 reference.ask('edit.undo',{},True);undone=reference.ask('simulation.wait')
finally:reference.close()
expected=[dict(name='applied',state=applied),dict(name='undo',state=undone)]
(out/'agent-reference.json').write_text(json.dumps(dict(passed=True,agent_sha256=hashlib.sha256(agent.read_bytes()).hexdigest(),baseline=baseline_reference,stages=expected),indent=2)+'\n')
def compare(state,wanted):
 for key in ['components','wires','nets','selected','drawing','undo_name','redo_name','status','paused']:assert state[key]==wanted[key],key
 parity.compare(state.get('result'),wanted.get('result'))
result=dict(passed=False,scope='Native unit text entry / one Apply click / one Cmd-Z; live LED circuit; exact packaged-agent reference',binary_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),events=[],stages=[])
def save():
 temp=out/'report.pending';temp.write_text(json.dumps(result,indent=2)+'\n');temp.replace(out/'report.json')
def event(name,state):result['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=state));save()
client=None;host=None
with (out/'host.txt').open('x') as log:
 try:
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);result['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():
   assert host.poll() is None,'host exited before endpoint';assert time.monotonic()<deadline,'endpoint admission timeout';time.sleep(.05)
  client=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent)
  baseline=seed(client);event('baseline',baseline)
  (out/'ready.json').write_text(json.dumps(dict(pid=host.pid,endpoint=str(endpoint)),indent=2)+'\n')
  previous=baseline['command_revision'];deadline=time.monotonic()+300;index=0
  while time.monotonic()<deadline:
   assert host.poll() is None,'host retired during UI observation'
   state=client.ask('project.inspect')
   if state['command_revision']!=previous:event('command-change',state);previous=state['command_revision']
   if index<len(expected):
    try:compare(state,expected[index]['state'])
    except AssertionError:pass
    else:
     name=expected[index]['name'];event('matched-'+name,state);result['stages'].append(dict(name=name,matched_public_agent=True,command_revision=state['command_revision']));index+=1
     if index==len(expected):result['passed']=True;deadline=min(deadline,time.monotonic()+30)
     save()
   time.sleep(.1)
  result['observation_lease_ended']=True
  if result['passed']:
   path=out/'value-restored.elab';client.ask('project.save',dict(path=str(path)),True);client.ask('simulation.wait');result['saved_project']=str(path)
  client.ask('app.close',dict(discard_changes=not result['passed']),True);client.close();client=None
  result['host_exit_code']=host.wait(timeout=15);result['normal_closure']=result['host_exit_code']==0
 except BaseException as error:result['error']=type(error).__name__+': '+str(error);raise
 finally:
  if client:client.close()
  if host and host.poll() is None:
   host.terminate()
   try:host.wait(timeout=10)
   except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
  if host:result['final_host_exit_code']=host.returncode
  result['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
