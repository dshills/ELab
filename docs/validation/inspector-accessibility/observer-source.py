import datetime,hashlib,importlib.util,json,math,stat,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/ax-recovery-diagnostic';out=base/'observations';sys.path.insert(0,str(root/'scripts'))
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
receipt=json.loads((root/'research/ui-audit/first-version/numeric-placement-review-checkpoint.json').read_text());review=Path(receipt['review_path']);manifest=json.loads((review/'RELEASE_MANIFEST.json').read_text())
for name,entry in manifest['files'].items():
 f=review/name;assert f.is_file() and not f.is_symlink();assert hashlib.sha256(f.read_bytes()).hexdigest()==entry['sha256'];assert oct(stat.S_IMODE(f.stat().st_mode))==entry['mode']
agent=review/'elab-agent';assert hashlib.sha256(agent.read_bytes()).hexdigest()==receipt['agent_sha256'];binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';identity=json.loads((base/'development-identity.json').read_text());assert hashlib.sha256(binary.read_bytes()).hexdigest()==identity['native_sha256'];endpoint=out/'endpoint.json';assert not endpoint.exists()
fixture=p.helper.Client(command_preconditions=True,agent_binary=agent)
try:
 fixture.ask('project.load',dict(path=str(out/'Baseline.elab')),True);fixture.ask('component.set',dict(id='R1',value='2k'),True);fixture.ask('project.save',dict(path=str(out/'Baseline.elab')),True)
finally:fixture.close()
baseline_sha=hashlib.sha256((out/'Baseline.elab').read_bytes()).hexdigest();r=dict(passed=False,scope='Development grouped-inspector actual native invalid-value recovery; public seeded file/selection and independent packaged /159 agent reference',identity=identity,agent_sha256=receipt['agent_sha256'],sealed_159_entries_verified=len(manifest['files']),baseline_sha256=baseline_sha,events=[],reference={});host=None;c=None;ref=None

def save():
 f=out/'report.pending';f.write_text(json.dumps(r,indent=2)+'\n');f.replace(out/'report.json')
def event(name,s):r['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=s));save()
def mark(name):(out/name).write_text('{}\n')
def wait(name):
 deadline=time.monotonic()+900
 while not (out/name).exists():
  assert host.poll() is None,'owned host exited before '+name;assert time.monotonic()<deadline,'lease ended for '+name;time.sleep(.1)
 s=c.ask('simulation.wait');event(name,s);assert hashlib.sha256((out/'Baseline.elab').read_bytes()).hexdigest()==baseline_sha;return s
def seed(client):
 client.ask('project.load',dict(path=str(out/'Baseline.elab')),True);client.ask('component.select',dict(id='R1'),True);return client.ask('simulation.wait')
def equivalent(a,b):
 for k in ['components','wires','nets','drawing','selected','undo_name','redo_name','paused']:assert a.get(k)==b.get(k),k
 p.compare(a['result'],b['result']);p.compare(a['scope'],b['scope'])
def unchanged(a,b):
 for k in ['components','wires','nets','result','scope','run_id','electrical_revision','document','undo_entry','redo_entry','undo_name','redo_name','paused']:assert a.get(k)==b.get(k),k
with (out/'native-host.txt').open('x') as log:
 try:
  ref=p.helper.Client(command_preconditions=True,agent_binary=agent);r['reference']['baseline']=seed(ref);ref.ask('editor.value.set',dict(value='0'),True);ref.ask('component.set',dict(id='R1',value='0'),True,expect='invalid-params');r['reference']['refused']=ref.ask('simulation.wait');ref.ask('editor.value.set',dict(value='2.2k'),True);ref.ask('component.set',dict(id='R1',value='2.2k'),True);r['reference']['corrected']=ref.ask('simulation.wait');save()
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);r['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():assert host.poll() is None;assert time.monotonic()<deadline;time.sleep(.05)
  c=p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);s=seed(c);event('baseline',s);equivalent(s,r['reference']['baseline']);mark('ready.json')
  private=wait('invalid-private');unchanged(s,private);mark('invalid-private-verified.json');refused=wait('refused');equivalent(refused,r['reference']['refused']);unchanged(s,refused);mark('refused-verified.json')
  private=wait('corrected-private');unchanged(refused,private);mark('corrected-private-verified.json');corrected=wait('corrected');equivalent(corrected,r['reference']['corrected']);assert math.isclose(corrected['result']['current_a'],.001344737452917332,abs_tol=5e-12);mark('corrected-verified.json')
  wait('allow-save');c.ask('project.save',dict(path=str(out/'Native corrected LED.elab')),True);ref.ask('project.save',dict(path=str(out/'Agent corrected LED.elab')),True);a=json.loads((out/'Native corrected LED.elab').read_text());b=json.loads((out/'Agent corrected LED.elab').read_text())
  for k in a:
   if k!='view':assert a[k]==b[k],k
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
