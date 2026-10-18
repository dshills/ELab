import datetime,hashlib,importlib.util,json,math,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-live-owner-157';out=base/'observations'
sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/live-measurement-review-checkpoint.json')
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';endpoint=out/'endpoint.json'
assert hashlib.sha256(binary.read_bytes()).hexdigest()=='01cc111a62ab758e4e48e24191784d4ee584aed66cc1740c2044db119ba89998'
for line in subprocess.check_output(['/bin/ps','-axo','comm='],text=True).splitlines():assert str(binary) not in line,'existing harness running'
project=out/'RC reference.elab';assert not project.exists() and not endpoint.exists()
result=dict(passed=False,scope='Actual native private live cursor A and interval start input across an agent restart; isolated byte-identical /157',release=proof,native_sha256=hashlib.sha256(binary.read_bytes()).hexdigest(),started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),events=[],agent_reference={})
client=None;reference=None;host=None

def save():
 p=out/'report.pending';p.write_text(json.dumps(result,indent=2)+'\n');p.replace(out/'report.json')
def event(name,state):
 result['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=state));save()
def marker(name): (out/name).write_text(json.dumps(dict(pid=host.pid))+'\n')
def wait_marker(name):
 deadline=time.monotonic()+900
 while time.monotonic()<deadline:
  assert host.poll() is None,'original native host exited'
  if (out/name).exists():return client.ask('project.inspect')
  time.sleep(.1)
 raise TimeoutError('UI stage lease expired: '+name)
def seed(c):
 c.ask('example.rc',dict(discard_changes=True),True);c.ask('simulation.wait');c.ask('simulation.pause',{},True)
 c.ask('simulation.input.set',dict(mode='charge'),True);c.ask('simulation.restart',{},True);c.ask('simulation.wait');c.ask('scope.probes.set',dict(ids=['v:C1','i:R1']),True)
 c.ask('simulation.step',dict(duration_s=.02),True);s=c.ask('simulation.wait');assert len(s['transient']['samples'])==21 and s['paused'];return s
def restart(c):
 c.ask('simulation.restart',{},True);c.ask('simulation.wait');c.ask('simulation.step',dict(duration_s=.02),True);return c.ask('simulation.wait')
def unchanged(before,after,scope=False):
 keys=['components','wires','nets','document','transient','mixed_transient','run_id','electrical_revision','paused','undo_entry','redo_entry','status','transient_settings','result']
 if scope:keys.append('scope')
 for key in keys:assert (key in before)==(key in after) and before.get(key)==after.get(key),(key,before.get(key),after.get(key))
def equivalent(native,ref):
 for key in ['components','wires','nets','selected','paused','transient_settings','undo_name','redo_name']:assert native.get(key)==ref.get(key),key
 parity.compare(native['scope'],ref['scope']);parity.compare(native['transient'],ref['transient'])
with (out/'host.txt').open('x') as log:
 try:
  reference=parity.helper.Client(command_preconditions=True,agent_binary=agent);refbase=seed(reference)
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);result['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():
   assert host.poll() is None;assert time.monotonic()<deadline;time.sleep(.05)
  client=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);seed(client);client.ask('project.save',dict(path=str(project)),True);baseline=client.ask('simulation.wait');equivalent(baseline,refbase);project_bytes=project.read_bytes();event('baseline',baseline);marker('ready.json')
  for label,command,params in [('cursor-a','scope.cursor.set',dict(cursor='a',time_s='6 ms')),('interval-start','scope.window.set',dict(mode='fixed',start_s='6 ms',end_s='20 ms'))]:
   before=client.ask('project.inspect');event(label+'-before',before);marker(label+'-ready.json')
   private=wait_marker(label+'-private-observed');event(label+'-private',private);unchanged(before,private,True);marker(label+'-private-verified.json')
   wait_marker(label+'-allow-restart');reset=restart(client);refreset=restart(reference);equivalent(reset,refreset);assert reset['run_id']==before['run_id']+1 and reset['document']==before['document'];assert 'a' not in reset['scope'] and reset['scope']['window']['mode']=='live';event(label+'-restarted',reset);marker(label+'-restarted.json')
   entered=wait_marker(label+'-reset-enter-observed');event(label+'-reset-enter',entered);unchanged(reset,entered,True);assert not entered.get('error');marker(label+'-reset-verified.json')
   fresh=wait_marker(label+'-fresh-observed');event(label+'-fresh',fresh);unchanged(reset,fresh);reference.ask(command,params,True);ref=reference.ask('project.inspect');equivalent(fresh,ref);result['agent_reference'][label+'-fresh']=ref
   assert project.read_bytes()==project_bytes;marker(label+'-fresh-verified.json')
   if label=='cursor-a':
    client.ask('scope.cursor.clear',dict(cursor='a'),True);reference.ask('scope.cursor.clear',dict(cursor='a'),True)
    client.ask('scope.window.set',dict(mode='fixed',start_s=.004,end_s=.018),True);reference.ask('scope.window.set',dict(mode='fixed',start_s=.004,end_s=.018),True)
  result['passed']=True;save();marker('all-verified.json');wait_marker('finish-observation');client.ask('app.close',{},True);client.close();client=None;result['host_exit_code']=host.wait(timeout=15);result['normal_closure']=result['host_exit_code']==0
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
