import datetime,hashlib,importlib.util,json,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/placement-number-pointer-replay';out=base/'observations';sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
proof=json.loads((root/'build/native-placement-158/observations/report.json').read_text())['release']
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';endpoint=out/'endpoint.json';assert not endpoint.exists()
assert hashlib.sha256(binary.read_bytes()).hexdigest()=='d2f4a49ba0a3a0b4d3165a0f6631a9e11d9e1608d5ca0a7b824d793154891972'
report=dict(scope='Development repair replay against packaged /158 agent; source and resistor tool seeded publicly, actual native X/Y and pointer Place',base_release=proof,development_identity=json.loads((root/'build/placement-number-repair/development-final-identity.json').read_text()),events=[])
client=None;host=None

def save(): (out/'report.json').write_text(json.dumps(report,indent=2)+'\n')
def record(name):
 s=client.ask('simulation.wait');report['events'].append(dict(name=name,state=s,utc=datetime.datetime.now(datetime.timezone.utc).isoformat()));save();return s
def marker(name): (out/name).write_text('{}\n')
def wait(name):
 deadline=time.monotonic()+900
 while not (out/name).exists():
  assert host.poll() is None;assert time.monotonic()<deadline,name;time.sleep(.1)
 record(name);marker(name+'-recorded')
with (out/'host.txt').open('x') as log:
 try:
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);report['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():assert host.poll() is None;assert time.monotonic()<deadline;time.sleep(.05)
  client=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent)
  client.ask('component.place',dict(kind='dc-source',position=dict(x=-200,y=0)),True)
  client.ask('drawing.tool.set',dict(tool='place',kind='resistor'),True)
  client.ask('drawing.position.set',dict(position=dict(x=-200,y=0)),True)
  record('prepared');marker('ready')
  for name in ['x-private','y-private','placed','finish']:wait(name)
  client.ask('project.save',dict(path=str(out/'Observed placement.elab')),True);record('saved');client.ask('app.close',{},True);client.close();client=None;report['normal_exit']=host.wait(timeout=15)==0
 except BaseException as error:report['error']=type(error).__name__+': '+str(error);raise
 finally:
  if client:client.close()
  if host and host.poll() is None:host.terminate();host.wait(timeout=10)
  if host:report['exit_code']=host.returncode
  save()
