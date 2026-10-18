import datetime,hashlib,importlib.util,json,math,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-draw-159';out=base/'observations';sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/numeric-placement-review-checkpoint.json')
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';endpoint=out/'endpoint.json';assert not endpoint.exists();assert hashlib.sha256(binary.read_bytes()).hexdigest()=='063bd50f74507cab0156ea9ff98c1488c59c26e7bd71c5777c0d903f7c463537'
for line in subprocess.check_output(['/bin/ps','-axo','comm='],text=True).splitlines():assert str(binary) not in line,'old test host running'
parts=[('dc-source',-200,0),('resistor',0,-110),('led',200,0),('ground',0,110)]
wires=[('V1.positive','R1.1'),('R1.2','D1.anode'),('D1.cathode','GND1.reference'),('GND1.reference','V1.negative')]
result=dict(passed=False,scope='Actual native /159 LED construction from blank through palette/exact placement/named wire menus, Undo/Redo; external agent parity',release=proof,native_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),events=[],agent_reference={})
client=None;reference=None;host=None

def save():
 p=out/'report.pending';p.write_text(json.dumps(result,indent=2)+'\n');p.replace(out/'report.json')
def event(name,state):result['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=state));save()
def marker(name): (out/name).write_text(json.dumps(dict(pid=host.pid))+'\n')
def wait_marker(name):
 deadline=time.monotonic()+900
 while time.monotonic()<deadline:
  assert host.poll() is None,'original host exited before '+name
  if (out/name).exists():return client.ask('simulation.wait')
  time.sleep(.1)
 raise TimeoutError('UI observation lease ended: '+name)
def equivalent(a,b):
 for k in ['components','wires','nets','selected','drawing','undo_name','redo_name','paused']:assert a.get(k)==b.get(k),(k,a.get(k),b.get(k))
 parity.compare(a.get('result'),b.get('result'));parity.compare(a['scope'],b['scope'])
with (out/'host.txt').open('x') as log:
 try:
  reference=parity.helper.Client(command_preconditions=True,agent_binary=agent)
  for index,(kind,x,y) in enumerate(parts,1):
   reference.ask('drawing.tool.set',dict(tool='place',kind=kind),True);reference.ask('drawing.position.set',dict(position=dict(x=x,y=y)),True);reference.ask('component.place',dict(kind=kind,position=dict(x=x,y=y)),True);result['agent_reference']['part-'+str(index)]=reference.ask('simulation.wait')
  reference.ask('drawing.tool.set',dict(tool='wire'),True)
  for index,(a,b) in enumerate(wires,1):
   reference.ask('wire.terminals.set',dict(**{'from':a,'to':b}),True);reference.ask('wire.connect',dict(**{'from':a,'to':b}),True);result['agent_reference']['wire-'+str(index)]=reference.ask('simulation.wait')
  reference.ask('edit.undo',{},True);result['agent_reference']['undo']=reference.ask('simulation.wait');reference.ask('edit.redo',{},True);result['agent_reference']['redo']=reference.ask('simulation.wait');save()
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);result['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():
   assert host.poll() is None;assert time.monotonic()<deadline;time.sleep(.05)
  client=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);baseline=client.ask('project.inspect');assert not baseline['components'] and not baseline['wires'];event('blank',baseline);marker('ready.json')
  for index in range(1,5):
   s=wait_marker('part-'+str(index)+'-ui-observed');event('part-'+str(index),s);equivalent(s,result['agent_reference']['part-'+str(index)]);assert len(s['components'])==index and not s['wires'];marker('part-'+str(index)+'-verified.json')
  for index in range(1,5):
   s=wait_marker('wire-'+str(index)+'-ui-observed');event('wire-'+str(index),s);equivalent(s,result['agent_reference']['wire-'+str(index)]);assert len(s['wires'])==index
   if index==4:assert not s['result']['stale'] and math.isclose(s['result']['current_a'],.0029167965774405296,abs_tol=5e-12)
   marker('wire-'+str(index)+'-verified.json')
  undone=wait_marker('undo-ui-observed');event('undo',undone);equivalent(undone,result['agent_reference']['undo']);assert len(undone['wires'])==3 and undone['result']['stale'];marker('undo-verified.json')
  redone=wait_marker('redo-ui-observed');event('redo',redone);equivalent(redone,result['agent_reference']['redo']);assert len(redone['wires'])==4 and not redone['result']['stale'];marker('redo-verified.json')
  wait_marker('allow-save-copy');client.ask('project.save',dict(path=str(out/'Native built LED.elab')),True);saved=client.ask('simulation.wait');event('agent-saved-native-build',saved)
  reference.ask('project.save',dict(path=str(out/'Agent built LED.elab')),True);reference.ask('simulation.wait');a=json.loads((out/'Native built LED.elab').read_text());b=json.loads((out/'Agent built LED.elab').read_text())
  for key in a:
   if key!='view':assert a[key]==b[key],key
  result['saved_fields_except_view_match']=True;result['passed']=True;marker('all-verified.json');save();wait_marker('finish-observation');client.ask('app.close',{},True);client.close();client=None;result['host_exit_code']=host.wait(timeout=15);result['normal_closure']=result['host_exit_code']==0
 except BaseException as error:result['error']=type(error).__name__+': '+str(error);raise
 finally:
  if reference:reference.close()
  if client:client.close()
  if host and host.poll() is None:
   host.terminate()
   try:host.wait(timeout=10)
   except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
  if host:result['final_host_exit_code']=host.returncode
  result['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
