import datetime,hashlib,importlib.util,json,math,subprocess,sys,threading,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-close-158';out=base/'observations';sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/native-text-review-checkpoint.json')
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';endpoint=out/'endpoint.json';working=out/'Working.elab';agent_file=out/'Agent expected.elab';assert not endpoint.exists() and not working.exists()
assert hashlib.sha256(binary.read_bytes()).hexdigest()=='570b805b42a9f3ee01acb6fecd3bdea5420c57a0b38dd247d1c52689127d82b5'
for line in subprocess.check_output(['/bin/ps','-axo','comm='],text=True).splitlines():assert str(binary) not in line,'old test host running'
result=dict(passed=False,scope='Actual /158 native value typing/Apply, Cmd+W/Cancel, Cmd+W/Save and continue, original normal close and fresh agent reopen',release=proof,native_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),events=[])
client=None;reference=None;host=None;watcher=None;watch_thread=None;watch_events=[];watch_errors=[]
def save():
 p=out/'report.pending';p.write_text(json.dumps(result,indent=2)+'\n');p.replace(out/'report.json')
def event(name,state):result['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=state));save()
def marker(name): (out/name).write_text(json.dumps(dict(pid=host.pid))+'\n')
def wait_marker(name):
 deadline=time.monotonic()+900
 while time.monotonic()<deadline:
  assert host.poll() is None,'original native host exited before '+name
  if (out/name).exists():return client.ask('project.inspect')
  time.sleep(.1)
 raise TimeoutError('UI observation lease ended: '+name)
def seed(c):
 c.ask('example.reset',dict(discard_changes=True),True);c.ask('simulation.wait');c.ask('component.select',dict(id='R1'),True);return c.ask('simulation.wait')
def unchanged(a,b,all_measurements=True):
 keys=['components','wires','nets','result','run_id','electrical_revision','document','undo_entry','redo_entry','undo_name','redo_name','paused','status','selected','transient_settings','drawing','closed']
 if all_measurements:keys.append('scope')
 for k in keys:assert a.get(k)==b.get(k),(k,a.get(k),b.get(k))
def equivalent(a,b):
 for k in ['components','wires','nets','selected','paused','transient_settings','undo_name','redo_name']:assert a.get(k)==b.get(k),k
 parity.compare(a['result'],b['result']);parity.compare(a['scope'],b['scope'])
def watch():
 try:
  s=watcher.ask('project.inspect');revision=s['revision']
  while True:
   p=watcher.ask('events.poll',dict(after_revision=revision,timeout_ms=1000));revision=p['revision'];watch_events.append(p)
   if p['closed']:return
 except BaseException as error:watch_errors.append(type(error).__name__+': '+str(error))
with (out/'host.txt').open('x') as log:
 try:
  reference=parity.helper.Client(command_preconditions=True,agent_binary=agent);seed(reference);reference.ask('project.save',dict(path=str(out/'Agent baseline.elab')),True);reference.ask('simulation.wait')
  reference.ask('editor.value.set',dict(value='2.2k'),True);reference.ask('component.set',dict(id='R1',value='2.2k'),True);ref_dirty=reference.ask('simulation.wait');reference.ask('app.close',{},True,expect='unsaved-changes');ref_after_refusal=reference.ask('project.inspect');unchanged(ref_dirty,ref_after_refusal)
  reference.ask('project.save',dict(path=str(agent_file)),True);reference.ask('simulation.wait');ref_closed=reference.ask('app.close',{},True);assert ref_closed['closed'];event('agent-close-reference',ref_closed)
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);result['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():
   assert host.poll() is None;assert time.monotonic()<deadline;time.sleep(.05)
  client=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);seed(client);client.ask('project.save',dict(path=str(working)),True);baseline=client.ask('simulation.wait');original=working.read_bytes();(out/'Original baseline.elab').write_bytes(original);event('baseline',baseline);marker('ready.json')
  private=wait_marker('private-value-observed');unchanged(baseline,private);event('private-value',private);marker('private-verified.json')
  edited=wait_marker('apply-ui-observed');assert edited['circuit']['resistance_ohm']==2200 and not edited['result']['stale'] and edited['document']['dirty'];assert math.isclose(edited['result']['current_a'],.001344737452917332,rel_tol=1e-10);assert working.read_bytes()==original;equivalent(edited,ref_dirty);event('edited',edited);marker('edit-verified.json')
  first=wait_marker('first-close-prompt-observed');unchanged(edited,first);assert working.read_bytes()==original;event('first-close-prompt',first);marker('first-prompt-verified.json')
  canceled=wait_marker('cancel-ui-observed');unchanged(edited,canceled);assert working.read_bytes()==original;event('close-canceled',canceled);marker('cancel-verified.json')
  second=wait_marker('second-close-prompt-observed');unchanged(edited,second);assert working.read_bytes()==original;event('second-close-prompt',second)
  watcher=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);watch_thread=threading.Thread(target=watch,daemon=True);watch_thread.start();marker('save-close-ready.json')
  result['host_exit_code']=host.wait(timeout=300);assert result['host_exit_code']==0;result['normal_closure']=True
  if watch_thread:watch_thread.join(timeout=3)
  result['close_observer_publications']=watch_events;result['close_observer_errors']=watch_errors
  snapshots=[p['snapshot'] for p in watch_events if 'snapshot' in p];closed=next((s for s in reversed(snapshots) if s['closed']),None);result['closed_snapshot_available']=closed is not None
  if closed:
   event('native-closed',closed);assert not closed['document']['dirty'];equivalent(closed,ref_closed);assert closed['project_work']['actor']=='user' and closed['project_work']['command']=='project.save'
  saved=json.loads(working.read_text());expected=json.loads(agent_file.read_text());assert working.read_bytes()!=original
  for k in saved:
   if k!='view':assert saved[k]==expected[k],k
  result['saved_fields_except_view_match_agent']=True;result['saved_sha256']=hashlib.sha256(working.read_bytes()).hexdigest();result['saved_bytes']=working.stat().st_size
  reopen=parity.helper.Client(command_preconditions=True,agent_binary=agent)
  try:
   reopen.ask('project.load',dict(path=str(working)),True);fresh=reopen.ask('simulation.wait');assert fresh['circuit']['resistance_ohm']==2200 and not fresh['result']['stale'];assert fresh['components']==edited['components'] and fresh['wires']==edited['wires'] and fresh['nets']==edited['nets'];parity.compare(fresh['result'],edited['result']);event('fresh-agent-reopened',fresh)
  finally:reopen.close()
  result['passed']=True;marker('all-verified.json')
 except BaseException as error:result['error']=type(error).__name__+': '+str(error);raise
 finally:
  if host and host.poll() is None:
   host.terminate()
   try:host.wait(timeout=10)
   except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
  if client:client.close()
  if watcher:watcher.close()
  if watch_thread:watch_thread.join(timeout=2)
  if reference:reference.close()
  if host:result['final_host_exit_code']=host.returncode
  result['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
