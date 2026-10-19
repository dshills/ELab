import datetime,hashlib,importlib.util,json,math,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-edit-159';out=base/'observations';sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/numeric-placement-review-checkpoint.json')
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';endpoint=out/'endpoint.json';baseline_file=out/'Baseline.elab';baseline_sha=hashlib.sha256(baseline_file.read_bytes()).hexdigest()
assert not endpoint.exists();assert hashlib.sha256(binary.read_bytes()).hexdigest()=='063bd50f74507cab0156ea9ff98c1488c59c26e7bd71c5777c0d903f7c463537'
for line in subprocess.check_output(['/bin/ps','-axo','comm='],text=True).splitlines():assert str(binary) not in line,'old test host running'
result=dict(passed=False,scope='Native /159 resistor entry/Apply, Undo/Redo, wire Delete/reconnect and invalid-value recovery; public seeded prior circuit and independent agent parity',release=proof,native_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),baseline_sha256=baseline_sha,events=[],agent_reference={},started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
client=None;reference=None;host=None

def save():
 p=out/'report.pending';p.write_text(json.dumps(result,indent=2)+'\n');p.replace(out/'report.json')
def event(name,state):result['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=state));save()
def marker(name): (out/name).write_text(json.dumps(dict(pid=host.pid))+'\n')
def wait(name):
 deadline=time.monotonic()+900
 while not (out/name).exists():
  assert host.poll() is None,'original host exited before '+name
  assert time.monotonic()<deadline,'UI observation lease ended: '+name
  time.sleep(.1)
 s=client.ask('simulation.wait');event(name,s);assert hashlib.sha256(baseline_file.read_bytes()).hexdigest()==baseline_sha;return s
def equivalent(a,b):
 for k in ['components','wires','nets','selected','drawing','undo_name','redo_name','paused']:assert a.get(k)==b.get(k),(k,a.get(k),b.get(k))
 parity.compare(a.get('result'),b.get('result'));parity.compare(a['scope'],b['scope'])
def unchanged(a,b):
 for k in ['components','wires','nets','result','scope','run_id','electrical_revision','document','undo_entry','redo_entry','undo_name','redo_name','paused']:assert a.get(k)==b.get(k),k
def seed(c):
 c.ask('project.load',dict(path=str(baseline_file)),True);c.ask('simulation.wait');c.ask('component.select',dict(id='R1'),True);return c.ask('simulation.wait')
def ref(name):result['agent_reference'][name]=reference.ask('simulation.wait')
with (out/'host.txt').open('x') as log:
 try:
  reference=parity.helper.Client(command_preconditions=True,agent_binary=agent);ref_baseline=seed(reference);ref('baseline')
  reference.ask('editor.value.set',dict(value='2k'),True);reference.ask('component.set',dict(id='R1',value='2k'),True);ref('edited')
  reference.ask('edit.undo',{},True);ref('undo');reference.ask('edit.redo',{},True);ref('redo')
  reference.ask('component.select',dict(id='W4'),True);reference.ask('object.delete',dict(id='W4'),True);ref('deleted')
  reference.ask('drawing.tool.set',dict(tool='wire'),True);reference.ask('wire.terminals.set',dict(**{'from':'GND1.reference','to':'V1.negative'}),True);reference.ask('wire.connect',dict(**{'from':'GND1.reference','to':'V1.negative'}),True);ref('reconnected')
  reference.ask('component.select',dict(id='R1'),True);ref('selected-again');reference.ask('editor.value.set',dict(value='0'),True);reference.ask('component.set',dict(id='R1',value='0'),True,expect='invalid-params');ref('refused')
  reference.ask('editor.value.set',dict(value='2.2k'),True);reference.ask('component.set',dict(id='R1',value='2.2k'),True);ref('corrected');save()
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);result['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():assert host.poll() is None;assert time.monotonic()<deadline;time.sleep(.05)
  client=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);native_baseline=seed(client);event('baseline',native_baseline);equivalent(native_baseline,ref_baseline);marker('ready.json')
  private=wait('2k-private');unchanged(native_baseline,private);marker('2k-private-verified.json')
  s=wait('edited');equivalent(s,result['agent_reference']['edited']);assert s['result']['current_a']<native_baseline['result']['current_a'] and .00147<s['result']['current_a']<.00148;marker('edited-verified.json')
  for name in ['undo','redo','deleted','reconnected']:
   s=wait(name);equivalent(s,result['agent_reference'][name]);marker(name+'-verified.json')
  assert len(s['wires'])==4 and s['wires'][-1]['id']=='W5' and not s['result']['stale']
  selected=wait('selected-again');equivalent(selected,result['agent_reference']['selected-again']);marker('selected-again-verified.json')
  refused=wait('refused');equivalent(refused,result['agent_reference']['refused']);unchanged(selected,refused);marker('refused-verified.json')
  private=wait('corrected-private');unchanged(refused,private);marker('corrected-private-verified.json')
  corrected=wait('corrected');equivalent(corrected,result['agent_reference']['corrected']);assert math.isclose(corrected['result']['current_a'],.001344737452917332,abs_tol=5e-12);marker('corrected-verified.json')
  wait('allow-save-copy');client.ask('project.save',dict(path=str(out/'Native edited LED.elab')),True);saved=client.ask('simulation.wait');event('agent-saved-native-edit',saved)
  reference.ask('project.save',dict(path=str(out/'Agent edited LED.elab')),True);reference.ask('simulation.wait');a=json.loads((out/'Native edited LED.elab').read_text());b=json.loads((out/'Agent edited LED.elab').read_text())
  for key in a:
   if key!='view':assert a[key]==b[key],key
  result['saved_fields_except_view_match']=True;result['passed']=True;marker('all-verified.json');save();wait('finish-observation');client.ask('app.close',{},True);client.close();client=None;result['host_exit_code']=host.wait(timeout=15);result['normal_closure']=result['host_exit_code']==0
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
