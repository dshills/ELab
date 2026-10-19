import datetime,hashlib,importlib.util,json,math,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-rc-resume-161';out=base/'observations';sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json');identity=json.loads((base/'identity.json').read_text());agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';fixture=root/'build/native-rc-edit-161/observations/Native edited RC.elab';fixture_sha=hashlib.sha256(fixture.read_bytes()).hexdigest();endpoint=out/'endpoint.json';assert not endpoint.exists();assert hashlib.sha256(binary.read_bytes()).hexdigest()==identity['native_sha256']
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
r=dict(passed=False,scope='Native /161 direct Resume after paused resistance edit; prior native-edited 20k fixture publicly loaded, then actual 10k Apply/Resume/Pause; fresh reference and independent RC law',release=proof,identity=identity,fixture_sha256=fixture_sha,events=[],reference={},started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat());host=None;c=None;ref=None;baseline_sha=None
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def save():
 f=out/'report.pending';f.write_text(json.dumps(r,indent=2)+'\n');f.replace(out/'report.json')
def event(name,s):r['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=s));save()
def mark(name):(out/name).write_text('{}\n')
def wait(name):
 deadline=time.monotonic()+900
 while not (out/name).exists():assert host.poll() is None,'host exited before '+name;assert time.monotonic()<deadline,'UI lease expired: '+name;time.sleep(.05)
 s=c.ask('simulation.wait');event(name,s);assert sha(fixture)==fixture_sha
 if baseline_sha:assert sha(out/'Baseline RC.elab')==baseline_sha
 return s
def equivalent(a,b):
 for key in ['components','wires','nets','selected','drawing','paused','transient_settings','undo_name','redo_name']:assert a.get(key)==b.get(key),key
 for key in ['result','scope','transient']:p.compare(a.get(key),b.get(key),key)
def protected(a,b):
 for key in ['components','wires','nets','selected','paused','scope','transient','run_id','electrical_revision','document','undo_entry','redo_entry']:assert a.get(key)==b.get(key),key
def reference(name):r['reference'][name]=ref.ask('simulation.wait');return r['reference'][name]
def seed(target,label):
 target.ask('project.load',dict(path=str(fixture)),True);target.ask('simulation.pause',{},True);target.ask('component.select',dict(id='R1'),True);target.ask('simulation.input.set',dict(mode='charge'),True);target.ask('simulation.restart',{},True);target.ask('simulation.step',dict(duration_s=.02),True);target.ask('project.save',dict(path=str(out/label)),True);return target.ask('simulation.wait')
def stale(before,s):
 assert s['paused'] and s['transient']['stale'] and s['scope']['stale'];assert s['transient']['sample']==before['transient']['sample'];assert s['transient']['samples']==before['transient']['samples'];assert s['transient']['time_constant_s']==before['transient']['time_constant_s'];assert s['transient']['run_id']==before['transient']['run_id']
def numerical(s,resistance):
 t=s['transient'];assert not t['stale'] and not s['scope']['stale'] and math.isclose(t['time_constant_s'],resistance*1e-6,abs_tol=1e-12)
 traces={x['probe']['id']:x['points'] for x in s['scope']['traces']}
 for i,sample in enumerate(t['samples']):
  at=sample['time_s'];v=5*(-math.expm1(-at/(resistance*1e-6)));current=(5-v)/resistance
  for key,want in [('input_v',5),('voltage_v',v),('current_a',current),('energy_j',.5e-6*v*v)]:assert math.isclose(sample[key],want,rel_tol=1e-10,abs_tol=1e-12),(key,at)
  for probe,want in [('v:C1',v),('i:R1',current)]:assert len(traces[probe])==len(t['samples']) and math.isclose(traces[probe][i]['value'],want,rel_tol=1e-10,abs_tol=1e-12),(probe,at)
with (out/'native-host.txt').open('x') as log:
 try:
  ref=p.helper.Client(command_preconditions=True,agent_binary=agent);seed(ref,'Agent baseline RC.elab');reference('baseline');ref.ask('editor.value.set',dict(value='10k'),True);ref.ask('component.set',dict(id='R1',value='10k'),True);reference('edited-stale');ref.ask('simulation.restart',{},True);reference('reference-fresh-start');save()
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);r['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():assert host.poll() is None;assert time.monotonic()<deadline;time.sleep(.05)
  c=p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);baseline=seed(c,'Baseline RC.elab');baseline_sha=sha(out/'Baseline RC.elab');r['baseline_sha256']=baseline_sha;event('baseline',baseline);equivalent(baseline,r['reference']['baseline']);numerical(baseline,20000);mark('ready.json')
  private=wait('private-10k');protected(baseline,private);mark('private-10k-verified.json')
  edited=wait('edited-stale');equivalent(edited,r['reference']['edited-stale']);stale(baseline,edited);assert next(x for x in edited['components'] if x['id']=='R1')['value']==10000;mark('edited-stale-verified.json')
  running=wait('live-running');assert not running['paused'] and running['transient']['sample']['time_s']>0;assert running['transient']['samples'][0]['time_s']==0 and running['transient']['samples'][0]['voltage_v']==0 and not running['transient']['stale'];mark('live-running-verified.json')
  paused=wait('live-paused');assert paused['paused'];numerical(paused,10000);at=paused['transient']['sample']['time_s'];ref.ask('simulation.step',dict(duration_s=at),True);reference('live-paused');equivalent(paused,r['reference']['live-paused'])
  for key in ['components','wires','nets','document','electrical_revision','undo_entry','redo_entry']:assert paused.get(key)==edited.get(key),key
  assert paused['run_id']==edited['run_id']+1 and paused['transient']['run_id']==paused['run_id'];assert paused['transient']['samples'][0]['time_s']==0 and paused['transient']['samples'][0]['voltage_v']==0
  time.sleep(.15);stable=c.ask('simulation.wait');protected(paused,stable);r['live_paused_time_s']=at;r['live_paused_stable']=True;mark('live-paused-verified.json')
  wait('allow-save-copy');c.ask('project.save',dict(path=str(out/'Native resume RC.elab')),True);ref.ask('project.save',dict(path=str(out/'Agent resume RC.elab')),True)
  a=json.loads((out/'Native resume RC.elab').read_text());b=json.loads((out/'Agent resume RC.elab').read_text())
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
