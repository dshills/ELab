import gzip,hashlib,importlib.util,json,subprocess,sys,time
from pathlib import Path

root=Path('/Users/dshills/Development/projects/ELab');stage=root/'build/first-version-162-tools';runtime=Path('/private/tmp/elab-first-version-162-runtime');review=root/'build/ELab-0.145.0-162-review';out=Path('/private/tmp/elab-first-version-162-edge');out.mkdir(mode=0o700)
sys.path.insert(0,str(root/'scripts'))
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
rr=json.loads((runtime/'runtime-report.json').read_text());assert rr['passed']
assert hashlib.sha256((runtime/'elab-agent').read_bytes()).hexdigest()==hashlib.sha256((review/'elab-agent').read_bytes()).hexdigest()
report=dict(passed=False,scope='Exact clean copied /162 native socket and headless stdio, independent RC equations, all scope/CSV rows, cursors and extrema; no physical input claim')
host=None;client=None
def reproduce(label,endpoint=None):
 command=[sys.executable,'-B',str(stage/'edge_reproduce.py'),'--agent',str(review/'elab-agent'),'--output',str(out/label)]
 if endpoint:command+=['--connect',str(endpoint)]
 with (out/(label+'-output.txt')).open('x') as log:r=subprocess.run(command,stdout=log,stderr=-2,timeout=30)
 assert r.returncode==0,label
 result=json.loads((out/label/'report.json').read_text());assert result['passed'] and result['agent_exit_code']==0
 return result
with (out/'native-host.txt').open('x') as log:
 try:
  report['headless']=reproduce('headless')
  endpoint=out/'endpoint.json';host=subprocess.Popen([str(runtime/'native'),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);report['native_pid']=host.pid
  deadline=time.monotonic()+30
  while not endpoint.exists():
   assert host.poll() is None,'native exited before endpoint';assert time.monotonic()<deadline,'endpoint timeout';time.sleep(.05)
  report['native']=reproduce('native',endpoint)
  left=json.loads(gzip.decompress((out/'native/state.json.gz').read_bytes()));right=json.loads(gzip.decompress((out/'headless/state.json.gz').read_bytes()))
  for stage_name in ['state','measured']:
   for key in ['components','wires','nets','paused','transient_settings']:assert left[stage_name][key]==right[stage_name][key],key
   for key in ['scope','transient']:parity.compare(left[stage_name][key],right[stage_name][key])
  report['complete_transient_scope_parity']=True
  client=parity.helper.Client(str(endpoint),command_preconditions=True,agent_binary=runtime/'agent')
  client.ask('project.save',dict(path=str(out/'RC checked.elab')),True);saved=client.ask('simulation.wait');assert not saved['document']['dirty']
  client.ask('app.close',{},True);client.close();client=None
  report['native_exit_code']=host.wait(timeout=15);assert report['native_exit_code']==0
  report['normal_native_closure']=True;report['passed']=True
 except BaseException as error:
  report['error']=type(error).__name__+': '+str(error);raise
 finally:
  if client:client.close()
  if host and host.poll() is None:
   host.terminate()
   try:host.wait(timeout=10)
   except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
  if host:report['final_native_exit_code']=host.returncode
  (out/'report.json').write_text(json.dumps(report,indent=2)+'\n')
print(json.dumps(report))
