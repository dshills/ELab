import datetime,gzip,hashlib,importlib.util,json,math,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-rc-draw-161-retry';out=base/'observations';sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json');identity=json.loads((base/'identity.json').read_text());agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';assert hashlib.sha256(binary.read_bytes()).hexdigest()==identity['native_sha256'];endpoint=out/'endpoint.json';assert not endpoint.exists()
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
r=dict(passed=False,scope='Native /161 RC from blank through palette, coordinates, value Apply and named wire menus, followed by restart, probes, manual charge/discharge and cursors; no native graph/trace seed',release=proof,identity=identity,events=[],reference={},started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat());host=None;c=None;ref=None
def save():
 f=out/'report.pending';f.write_text(json.dumps(r,indent=2)+'\n');f.replace(out/'report.json')
def mark(name):(out/name).write_text('{}\n')
def event(name,s):r['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=s));save()
def wait(name):
 deadline=time.monotonic()+900
 while not (out/name).exists():
  assert host.poll() is None,'host exited before '+name;assert time.monotonic()<deadline,'UI observation lease expired: '+name;time.sleep(.1)
 s=c.ask('simulation.wait');event(name,s);return s
def equivalent(a,b):
 for key in ['components','wires','nets','selected','drawing','paused','transient_settings','undo_name','redo_name']:assert a.get(key)==b.get(key),(key,a.get(key),b.get(key))
 for key in ['transient','scope','result']:p.compare(a.get(key),b.get(key),key)
def protected(a,b):
 for key in ['components','wires','nets','selected','document','transient','scope','run_id','electrical_revision','undo_entry','redo_entry','paused']:assert a.get(key)==b.get(key),key
def record_ref(name):r['reference'][name]=ref.ask('simulation.wait');return r['reference'][name]
def step(target,duration):target.ask('simulation.step.start',dict(duration_s=duration),True);return target.ask('simulation.wait')
def electrical(s):
 samples=s['transient']['samples'];traces={t['probe']['id']:t['points'] for t in s['scope']['traces']};peak=5*(-math.expm1(-1));edge=[]
 for i,row in enumerate(samples):
  at=row['time_s'];want=5*(-math.expm1(-at/.01)) if at<=.01 else peak*math.exp(-(at-.01)/.01)
  for key,value in [('voltage_v',want),('current_a',(row['input_v']-want)/10000),('energy_j',.5e-6*want*want)]:assert math.isclose(row[key],value,rel_tol=1e-10,abs_tol=1e-12),(key,at)
  for probe,value in [('v:C1',want),('i:R1',(row['input_v']-want)/10000)]:
   points=traces[probe];assert len(points)==len(samples);assert points[i]['time_s']==at and math.isclose(points[i]['value'],value,rel_tol=1e-10,abs_tol=1e-12),(probe,at,points[i]['value'],value)
  if math.isclose(at,.01,abs_tol=1e-12):edge.append(row)
 assert len(edge)==2 and edge[0]['input_v']==5 and edge[1]['input_v']==0 and edge[0]['voltage_v']==edge[1]['voltage_v']
 assert min(x['value'] for x in traces['i:R1'])==edge[1]['current_a']
with (out/'native-host.txt').open('x') as log:
 try:
  ref=p.helper.Client(command_preconditions=True,agent_binary=agent);ref.ask('simulation.pause',{},True);record_ref('paused-blank')
  for i,(kind,x,y) in enumerate([('pulse-source',-200,0),('resistor',0,-110),('capacitor',200,0),('ground',0,110)],1):
   ref.ask('drawing.tool.set',dict(tool='place',kind=kind),True);ref.ask('drawing.position.set',dict(position=dict(x=x,y=y)),True);ref.ask('component.place',dict(kind=kind,position=dict(x=x,y=y)),True);record_ref('part-'+str(i))
   if i==2:
    ref.ask('editor.value.set',dict(value='10k'),True);ref.ask('component.set',dict(id='R1',value='10k'),True);record_ref('resistor-10k')
  ref.ask('drawing.tool.set',dict(tool='wire'),True)
  for i,(a,b) in enumerate([('V1.positive','R1.1'),('R1.2','C1.1'),('C1.2','GND1.reference'),('GND1.reference','V1.negative')],1):
   ref.ask('wire.terminals.set',dict(**{'from':a,'to':b}),True);ref.ask('wire.connect',dict(**{'from':a,'to':b}),True);record_ref('wire-'+str(i))
  ref.ask('simulation.restart',{},True);record_ref('restart');ref.ask('scope.probes.set',dict(ids=['v:C1','i:R1']),True);record_ref('probes');ref.ask('simulation.input.set',dict(mode='charge'),True);record_ref('queued-charge')
  for _ in range(10):step(ref,.001)
  record_ref('charge-10');ref.ask('simulation.input.set',dict(mode='discharge'),True);record_ref('queued-discharge');step(ref,.001);record_ref('discharge-1')
  for _ in range(9):step(ref,.001)
  record_ref('discharge-10');ref.ask('scope.cursor.set',dict(cursor='a',time_s='10 ms'),True);ref.ask('scope.cursor.select',dict(cursor='b'),True);record_ref('a-and-select-b');ref.ask('scope.cursor.set',dict(cursor='b',time_s='20 ms'),True);record_ref('timing');save()
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);r['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():assert host.poll() is None;assert time.monotonic()<deadline;time.sleep(.05)
  c=p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);blank=c.ask('simulation.wait');assert not blank['components'] and not blank['wires'];event('blank',blank);mark('ready.json')
  names=['paused-blank','part-1','part-2','resistor-10k','part-3','part-4','wire-1','wire-2','wire-3','wire-4','restart','probes','queued-charge','charge-10','queued-discharge','discharge-1','discharge-10','a-and-select-b','timing']
  before=None
  for name in names:
   if name=='resistor-10k':
    private=wait('resistor-private');protected(before,private);mark('resistor-private-verified.json')
   if name=='a-and-select-b':
    private=wait('private-a');protected(before,private);mark('private-a-verified.json')
   if name=='timing':
    private=wait('private-b');protected(before,private);mark('private-b-verified.json')
   s=wait(name);equivalent(s,r['reference'][name])
   if name in ['discharge-1','discharge-10','timing']:electrical(s)
   if name=='timing':
    scope=s['scope'];assert math.isclose(scope['delta_s'],.01,abs_tol=1e-12);values={x['probe_id']:x for x in scope['readings']};peak=5*(-math.expm1(-1));end=peak*math.exp(-1)
    for probe,a,b in [('v:C1',peak,end),('i:R1',-peak/10000,-end/10000)]:
     for key,want in [('a',a),('b',b),('delta',b-a)]:assert math.isclose(values[probe][key],want,rel_tol=1e-10,abs_tol=1e-12),(probe,key)
   before=s;mark(name+'-verified.json')
  wait('allow-save-copy');c.ask('project.save',dict(path=str(out/'Native built RC.elab')),True);ref.ask('project.save',dict(path=str(out/'Agent built RC.elab')),True)
  a=json.loads((out/'Native built RC.elab').read_text());b=json.loads((out/'Agent built RC.elab').read_text())
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
