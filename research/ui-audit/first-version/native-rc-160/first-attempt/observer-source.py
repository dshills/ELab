import datetime,hashlib,importlib.util,json,math,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-rc-160';out=base/'observations';sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/inspector-recovery-review-checkpoint.json');identity=json.loads((base/'identity.json').read_text());agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';assert hashlib.sha256(binary.read_bytes()).hexdigest()==identity['native_sha256'];endpoint=out/'endpoint.json';assert not endpoint.exists()
spec=importlib.util.spec_from_file_location('rc_parity',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
r=dict(passed=False,scope='Actual /160 native RC example, pause/restart, probe selection, charge/discharge steps, live resume/pause and cursor measurements; no native graph or trace seed',release=proof,identity=identity,events=[],reference={},started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat());c=None;ref=None;host=None;manual_sha=None

def save():
 f=out/'report.pending';f.write_text(json.dumps(r,indent=2)+'\n');f.replace(out/'report.json')
def event(name,s):r['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=s));save()
def mark(name):(out/name).write_text('{}\n')
def wait(name):
 deadline=time.monotonic()+900
 while not (out/name).exists():
  assert host.poll() is None,'original host exited before '+name;assert time.monotonic()<deadline,'UI observation lease ended for '+name;time.sleep(.1)
 s=c.ask('simulation.wait');event(name,s)
 if manual_sha:assert hashlib.sha256((out/'RC control reference.elab').read_bytes()).hexdigest()==manual_sha
 return s
def equivalent(a,b):
 for key in ['components','wires','nets','selected','paused','transient_settings','undo_name','redo_name']:assert a.get(key)==b.get(key),key
 p.compare(a['transient'],b['transient']);p.compare(a['scope'],b['scope'])
def protected(a,b,scope=False):
 for key in ['components','wires','nets','selected','document','transient_settings','undo_entry','redo_entry','electrical_revision']:assert a.get(key)==b.get(key),key
 if scope:
  for key in ['transient','scope','run_id','paused','status']:assert a.get(key)==b.get(key),key

def step(client,duration):client.ask('simulation.step.start',dict(duration_s=duration),True);return client.ask('simulation.wait')
def ref_state(name):r['reference'][name]=ref.ask('simulation.wait');return r['reference'][name]
def numeric(s,mode,origin=0.,initial=0.):
 for row in s['transient']['samples']:
  elapsed=row['time_s']-origin
  want=5+(initial-5)*math.exp(-elapsed/.01) if mode=='charge' else initial*math.exp(-elapsed/.01)
  current=(row['input_v']-want)/10000
  if elapsed>1e-12:assert row['input_v']==(5 if mode=='charge' else 0)
  for key,value in [('voltage_v',want),('current_a',current),('energy_j',.5e-6*want*want)]:assert math.isclose(row[key],value,rel_tol=1e-10,abs_tol=1e-12),(mode,key,row[key],value,row['time_s'])
def timing(s):
 scope=s['scope'];assert math.isclose(scope['delta_s'],.012,abs_tol=1e-12) and math.isclose(scope['interval_hz'],1/.012,rel_tol=1e-12)
 values={x['probe_id']:x for x in scope['readings']}
 for probe in ['v:C1','i:R1']:
  a,b=[5*(-math.expm1(-t/.01)) if probe=='v:C1' else .0005*math.exp(-t/.01) for t in [.004,.016]]
  for key,want in [('a',a),('b',b),('delta',b-a)]:assert math.isclose(values[probe][key],want,rel_tol=1e-10,abs_tol=1e-12),(probe,key)
with (out/'native-host.txt').open('x') as log:
 try:
  ref=p.helper.Client(command_preconditions=True,agent_binary=agent);ref.ask('example.rc',dict(discard_changes=True),True);ref.ask('simulation.pause',{},True);ref.ask('simulation.wait');ref.ask('simulation.restart',{},True);baseline_ref=ref_state('initial-restart')
  assert baseline_ref['transient_settings']['step_s']==.001 and baseline_ref['transient']['sample']['time_s']==0
  ref.ask('scope.probes.set',dict(ids=['v:C1','v:V1.positive']),True);ref_state('probe-1');ref.ask('scope.probes.set',dict(ids=['v:C1','i:R1']),True);ref_state('probe-2');ref.ask('project.save',dict(path=str(out/'Agent RC control reference.elab')),True);ref_state('control-baseline')
  ref.ask('simulation.input.set',dict(mode='charge'),True);ref_state('queued-charge')
  for i in range(1,11):step(ref,.001);ref_state('charge-'+str(i))
  peak=ref.ask('project.inspect')['transient']['sample']['voltage_v'];assert math.isclose(peak,5*(-math.expm1(-1)),rel_tol=1e-12)
  ref.ask('simulation.input.set',dict(mode='discharge'),True);ref_state('queued-discharge')
  for i in range(1,11):step(ref,.001);ref_state('discharge-'+str(i))
  ref.ask('simulation.restart',{},True);ref_state('restart-discharge');ref.ask('simulation.input.set',dict(mode='charge'),True);ref_state('queued-live-charge');save()
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);r['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():assert host.poll() is None;assert time.monotonic()<deadline;time.sleep(.05)
  c=p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);blank=c.ask('simulation.wait');assert not blank['components'] and not blank['wires'];event('blank',blank);mark('ready.json')
  loaded=wait('paused-loaded');assert loaded['paused'] and loaded['transient'] and not loaded['transient']['stale']
  for key in ['components','wires','nets','transient_settings']:assert loaded[key]==baseline_ref[key],key
  mark('paused-loaded-verified.json');s=wait('initial-restart');equivalent(s,baseline_ref);assert s['transient']['sample']['time_s']==0;mark('initial-restart-verified.json')
  for name in ['probe-1','probe-2']:
   s=wait(name);equivalent(s,r['reference'][name]);mark(name+'-verified.json')
  wait('allow-baseline-save');c.ask('project.save',dict(path=str(out/'RC control reference.elab')),True);baseline=c.ask('simulation.wait');event('control-baseline',baseline);manual_sha=hashlib.sha256((out/'RC control reference.elab').read_bytes()).hexdigest();r['manual_baseline_sha256']=manual_sha;equivalent(baseline,r['reference']['control-baseline']);mark('control-baseline-verified.json')
  charge=wait('queued-charge');equivalent(charge,r['reference']['queued-charge']);protected(baseline,charge);assert charge['transient']['sample']==baseline['transient']['sample'];mark('queued-charge-verified.json')
  s=wait('charge-10');equivalent(s,r['reference']['charge-10']);protected(baseline,s);numeric(s,'charge');assert math.isclose(s['transient']['sample']['time_s'],.01,abs_tol=1e-12);mark('charge-10-verified.json')
  queued=wait('queued-discharge');equivalent(queued,r['reference']['queued-discharge']);protected(baseline,queued);assert queued['transient']['sample']==s['transient']['sample'];mark('queued-discharge-verified.json')
  s=wait('discharge-10');equivalent(s,r['reference']['discharge-10']);protected(baseline,s);assert math.isclose(s['transient']['sample']['time_s'],.02,abs_tol=1e-12);assert math.isclose(s['transient']['sample']['voltage_v'],peak*math.exp(-1),rel_tol=1e-12)
  edge=[]
  for row in s['transient']['samples']:
   t=row['time_s'];v=5*(-math.expm1(-t/.01)) if t<=.01 else peak*math.exp(-(t-.01)/.01)
   for key,want in [('voltage_v',v),('current_a',(row['input_v']-v)/10000),('energy_j',.5e-6*v*v)]:assert math.isclose(row[key],want,rel_tol=1e-10,abs_tol=1e-12),(key,t)
   if math.isclose(t,.01,abs_tol=1e-12):edge.append(row)
  assert len(edge)==2 and edge[0]['input_v']==5 and edge[1]['input_v']==0 and edge[0]['voltage_v']==edge[1]['voltage_v'];r['capacitor_continuous_at_input_edge']=True;mark('discharge-10-verified.json')
  for name in ['restart-discharge','queued-live-charge']:
   s=wait(name);equivalent(s,r['reference'][name]);protected(baseline,s);mark(name+'-verified.json')
  live=wait('live-paused');assert live['paused'] and not live['transient']['stale'];live_t=live['transient']['sample']['time_s'];assert live_t>0;protected(baseline,live);numeric(live,'charge')
  n=round(live_t/.001);assert math.isclose(live_t,n*.001,abs_tol=1e-10)
  while n:
   count=min(n,1000);step(ref,count*.001);n-=count
  ref_state('live-paused');equivalent(live,r['reference']['live-paused']);time.sleep(.15);stable=c.ask('simulation.wait');protected(live,stable,True);r['live_pause_stable']=True;r['live_time_s']=live_t;mark('live-paused-verified.json')
  ref.ask('simulation.restart',{},True);ref_state('restart-charge')
  for _ in range(20):step(ref,.001)
  ref_state('charge-20');ref.ask('scope.cursor.set',dict(cursor='a',time_s='4 ms'),True);ref.ask('scope.cursor.select',dict(cursor='b'),True);ref_state('a-and-select-b');ref.ask('scope.cursor.set',dict(cursor='b',time_s='16 ms'),True);ref_state('timing');timing(r['reference']['timing']);ref.ask('scope.cursor.clear',dict(cursor='a'),True);ref_state('clear-a');save()
  s=wait('restart-charge');equivalent(s,r['reference']['restart-charge']);protected(baseline,s);mark('restart-charge-verified.json');s=wait('charge-20');equivalent(s,r['reference']['charge-20']);protected(baseline,s);numeric(s,'charge');assert len(s['transient']['samples'])==21;measurement_base=s;mark('charge-20-verified.json')
  s=wait('private-a');protected(measurement_base,s,True);mark('private-a-verified.json');s=wait('a-and-select-b');protected(measurement_base,s);equivalent(s,r['reference']['a-and-select-b']);selected=s;mark('a-and-select-b-verified.json');s=wait('private-b');protected(selected,s,True);mark('private-b-verified.json');s=wait('timing');protected(measurement_base,s);equivalent(s,r['reference']['timing']);timing(s);mark('timing-verified.json');s=wait('clear-a');protected(measurement_base,s);equivalent(s,r['reference']['clear-a']);assert 'a' not in s['scope'] and 'delta_s' not in s['scope'];mark('clear-a-verified.json')
  wait('allow-save-copy');c.ask('project.save',dict(path=str(out/'Native RC checked.elab')),True);ref.ask('project.save',dict(path=str(out/'Agent RC checked.elab')),True);a=json.loads((out/'Native RC checked.elab').read_text());b=json.loads((out/'Agent RC checked.elab').read_text())
  for key in a:
   if key!='view':assert a[key]==b[key],key
  r['saved_fields_except_view_match']=True;r['passed']=True;save();mark('all-verified.json');wait('finish');c.ask('app.close',{},True);c.close();c=None;r['host_exit_code']=host.wait(timeout=15);r['normal_closure']=r['host_exit_code']==0
 except BaseException as error:r['error']=type(error).__name__+': '+str(error);raise
 finally:
  if ref:ref.close()
  if c:c.close()
  if host and host.poll() is None:
   host.terminate()
   try:host.wait(timeout=10)
   except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
  if host:r['final_host_exit_code']=host.returncode
  r['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
