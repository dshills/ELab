import csv,datetime,hashlib,importlib.util,io,json,math,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-rc-csv-156';out=base/'observations'
sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/recording-owner-review-checkpoint.json')
spec=importlib.util.spec_from_file_location('csv_parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';endpoint=out/'endpoint.json'
assert hashlib.sha256(binary.read_bytes()).hexdigest()=='24e98f76934fbc2956474ef0db5bd10ed7d71ef6adf1758039f6222110d56cdf'
for line in subprocess.check_output(['/bin/ps','-axo','command='],text=True).splitlines():
 assert 'ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery' not in line,'existing harness is running'
project=out/'RC reference.elab';export=out/'RC interval.csv';assert not project.exists() and not export.exists() and not endpoint.exists()
result=dict(passed=False,scope='Actual native Export scope CSV Cancel/Save; selected RC window and probe data; isolated /156',release=proof,native_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),events=[])
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
 raise TimeoutError('UI observation lease ended without required CSV stage')
def seed(c):
 c.ask('example.rc',dict(discard_changes=True),True);c.ask('simulation.wait');c.ask('simulation.pause',{},True)
 c.ask('simulation.input.set',dict(mode='charge'),True);c.ask('simulation.restart',{},True);c.ask('simulation.wait')
 c.ask('scope.probes.set',dict(ids=['v:C1','i:R1']),True);c.ask('simulation.step',dict(duration_s=.02),True);c.ask('simulation.wait')
 c.ask('scope.window.set',dict(mode='fixed',start_s=.004,end_s=.016),True);s=c.ask('simulation.wait')
 assert s['paused'] and not s['transient']['stale'] and len(s['transient']['samples'])==21
 return s

def unchanged(before,after):
 for key in ['components','wires','nets','selected','document','transient','scope','run_id','electrical_revision','paused','undo_entry','redo_entry','status','transient_settings','result']:
  assert (key in before)==(key in after) and before.get(key)==after.get(key),key

def parse(data,state):
 table=list(csv.reader(io.StringIO(data.decode())));assert table[0]==['run_id','electrical_revision','engine','window_start_s','window_end_s','sample_step_s','time_s','v:C1 [V]','i:R1 [A]'];assert len(table)==14
 for i,row in enumerate(table[1:]):
  assert len(row)==9 and int(row[0])==state['transient']['run_id'] and int(row[1])==state['transient']['electrical_revision'] and row[2]==state['transient']['engine']
  for j,v in [(3,.004),(4,.016),(5,.001),(6,.004+i*.001)]:assert math.isclose(float(row[j]),v,rel_tol=0,abs_tol=1e-12),(i,j)
  t=float(row[6]);assert math.isclose(float(row[7]),5*(1-math.exp(-t/.01)),rel_tol=1e-10,abs_tol=1e-12)
  assert math.isclose(float(row[8]),5*math.exp(-t/.01)/10000,rel_tol=1e-10,abs_tol=1e-12)
 return table
with (out/'host.txt').open('x') as log:
 try:
  reference=parity.helper.Client(command_preconditions=True,agent_binary=agent);ref_base=seed(reference)
  ref_path=out/'Agent interval.csv';reference.ask('export.scope.csv',dict(path=str(ref_path)),True);ref_exported=reference.ask('simulation.wait');result['agent_reference']=ref_exported;ref_data=ref_path.read_bytes();ref_table=parse(ref_data,ref_base);save()
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);result['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():
   assert host.poll() is None;assert time.monotonic()<deadline,'endpoint admission timeout';time.sleep(.05)
  client=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);seed(client);client.ask('project.save',dict(path=str(project)),True);baseline=client.ask('simulation.wait');project_bytes=project.read_bytes()
  for key in ['components','wires','nets','selected','paused','transient_settings']:assert baseline[key]==ref_base[key],key
  parity.compare(baseline['scope'],ref_base['scope']);parity.compare(baseline['transient'],ref_base['transient']);event('baseline',baseline);marker('ready.json',dict(pid=host.pid,export_path=str(export),project=str(project)))
  canceled=wait_for(lambda s:(out/'cancel-ui-observed').exists());event('picker-canceled',canceled);unchanged(baseline,canceled)
  for key in ['last_export','project_work']:assert (key in baseline)==(key in canceled) and baseline.get(key)==canceled.get(key),key
  assert not export.exists() and project.read_bytes()==project_bytes;marker('cancel-verified.json')
  wait_for(lambda s:(out/'export-ui-observed').exists())
  exported=wait_for(lambda s:s.get('last_export',{}).get('path')==str(export) and s.get('project_work',{}).get('status')=='completed');event('exported',exported);unchanged(baseline,exported)
  assert exported['project_work']['command']=='export.scope.csv' and exported['project_work']['actor']=='user'
  data=export.read_bytes();table=parse(data,baseline);assert [row[2:] for row in table]==[row[2:] for row in ref_table]
  artifact=exported['last_export'];assert artifact['format']=='csv' and artifact['bytes']==len(data) and artifact['sha256']==hashlib.sha256(data).hexdigest()
  assert artifact['document_generation']==baseline['document']['generation'] and artifact['run_id']==baseline['transient']['run_id'] and artifact['electrical_revision']==baseline['transient']['electrical_revision']
  assert project.read_bytes()==project_bytes and not list(out.glob('.elab-export-*'))
  result['csv_verification']=dict(rows=13,columns=table[0],native_sha256=hashlib.sha256(data).hexdigest(),agent_sha256=hashlib.sha256(ref_data).hexdigest(),bytes_equal=data==ref_data,all_fields_equal_except_independent_run_electrical_ids=True,independent_RC_law_passed=True,temporary_export_files_remaining=0)
  result['passed']=True;save();marker('export-verified.json')
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
