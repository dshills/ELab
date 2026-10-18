import datetime,hashlib,importlib.util,json,math,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-rc-cursors-156';out=base/'observations'
sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/recording-owner-review-checkpoint.json')
spec=importlib.util.spec_from_file_location('cursor_parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';endpoint=out/'endpoint.json'
assert hashlib.sha256(binary.read_bytes()).hexdigest()=='24e98f76934fbc2956474ef0db5bd10ed7d71ef6adf1758039f6222110d56cdf'
for line in subprocess.check_output(['/bin/ps','-axo','command='],text=True).splitlines():
 assert 'ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery' not in line,'existing harness is running'
project=out/'RC reference.elab';assert not project.exists() and not endpoint.exists()
result=dict(passed=False,scope='Actual native RC cursor field entry, focus-loss selection, timing and Clear; isolated /156',release=proof,native_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),events=[],agent_reference={})
client=None;reference=None;host=None

def save():
 pending=out/'report.pending';pending.write_text(json.dumps(result,indent=2)+'\n');pending.replace(out/'report.json')
def event(name,state):
 result['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=state));save()
def marker(name,data=None):
 (out/name).write_text(json.dumps(data or dict(pid=host.pid),indent=2)+'\n')
def wait_for(predicate,seconds=300):
 deadline=time.monotonic()+seconds;last_revision=None
 while time.monotonic()<deadline:
  assert host.poll() is None,'original native host exited'
  state=client.ask('project.inspect')
  if state['command_revision']!=last_revision:
   result['last_observed']=state;last_revision=state['command_revision'];save()
  if predicate(state):return state
  time.sleep(.1)
 raise TimeoutError('UI observation lease ended without required cursor stage')
def seed(c):
 c.ask('example.rc',dict(discard_changes=True),True);c.ask('simulation.wait');c.ask('simulation.pause',{},True)
 c.ask('simulation.input.set',dict(mode='charge'),True);c.ask('simulation.restart',{},True);c.ask('simulation.wait')
 c.ask('scope.probes.set',dict(ids=['v:C1','i:R1']),True)
 c.ask('simulation.step',dict(duration_s=.02),True);s=c.ask('simulation.wait')
 assert s['paused'] and not s['transient']['stale'] and math.isclose(s['transient']['sample']['time_s'],.02,abs_tol=1e-12)
 assert len(s['transient']['samples'])==21
 return s

def unchanged(before,after,scope=False):
 keys=['components','wires','nets','document','transient','run_id','electrical_revision','paused','undo_entry','redo_entry','status','transient_settings','result']
 if scope:keys.append('scope')
 for key in keys:assert (key in before)==(key in after) and before.get(key)==after.get(key),key

def equivalent(native,ref):
 for key in ['components','wires','nets','selected','paused','transient_settings','undo_name','redo_name']:assert (key in native)==(key in ref) and native.get(key)==ref.get(key),key
 parity.compare(native['scope'],ref['scope']);parity.compare(native['transient'],ref['transient'])

def timing(state):
 scope=state['scope'];assert math.isclose(scope['delta_s'],.012,abs_tol=1e-12) and math.isclose(scope['interval_hz'],1/.012,rel_tol=1e-12)
 values={v['probe_id']:v for v in scope['readings']}
 for probe in ['v:C1','i:R1']:
  a,b=[5*(1-math.exp(-t/.01)) if probe=='v:C1' else 5*math.exp(-t/.01)/10000 for t in [.004,.016]]
  for field,want in [('a',a),('b',b),('delta',b-a)]:assert math.isclose(values[probe][field],want,rel_tol=1e-10,abs_tol=1e-12),(probe,field,values[probe][field],want)
with (out/'host.txt').open('x') as log:
 try:
  reference=parity.helper.Client(command_preconditions=True,agent_binary=agent);ref_base=seed(reference)
  reference.ask('scope.cursor.set',dict(cursor='a',time_s='4 ms'),True);reference.ask('scope.cursor.select',dict(cursor='b'),True);result['agent_reference']['a-and-select-b']=reference.ask('project.inspect')
  reference.ask('scope.cursor.set',dict(cursor='b',time_s='16 ms'),True);result['agent_reference']['timing']=reference.ask('project.inspect');timing(result['agent_reference']['timing'])
  reference.ask('scope.cursor.clear',dict(cursor='a'),True);result['agent_reference']['clear-a']=reference.ask('project.inspect');save()
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);result['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():
   assert host.poll() is None;assert time.monotonic()<deadline,'endpoint admission timeout';time.sleep(.05)
  client=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);seed(client);client.ask('project.save',dict(path=str(project)),True);baseline=client.ask('simulation.wait');equivalent(baseline,ref_base);project_bytes=project.read_bytes();event('baseline',baseline);marker('ready.json',dict(pid=host.pid,project=str(project)))
  private_a=wait_for(lambda s:(out/'private-a-observed').exists());event('private-a',private_a);unchanged(baseline,private_a,True);marker('private-a-verified.json')
  selected=wait_for(lambda s:(out/'select-ui-observed').exists());event('a-and-select-b',selected);unchanged(baseline,selected);equivalent(selected,result['agent_reference']['a-and-select-b']);marker('select-verified.json')
  private_b=wait_for(lambda s:(out/'private-b-observed').exists());event('private-b',private_b);unchanged(selected,private_b,True);marker('private-b-verified.json')
  measured=wait_for(lambda s:(out/'timing-ui-observed').exists());event('timing',measured);unchanged(baseline,measured);equivalent(measured,result['agent_reference']['timing']);timing(measured);marker('timing-verified.json')
  cleared=wait_for(lambda s:(out/'clear-ui-observed').exists());event('clear-a',cleared);unchanged(baseline,cleared);equivalent(cleared,result['agent_reference']['clear-a']);assert 'a' not in cleared['scope'] and 'delta_s' not in cleared['scope'] and 'interval_hz' not in cleared['scope'];assert project.read_bytes()==project_bytes
  result['passed']=True;save();marker('clear-verified.json')
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
