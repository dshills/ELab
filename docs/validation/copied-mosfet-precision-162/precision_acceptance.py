import hashlib,importlib.util,json,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');stage=Path(__file__).resolve().parent
runtime=Path('/private/tmp/elab-first-version-162-runtime');out=Path('/private/tmp/elab-first-version-162-precision');out.mkdir(mode=0o700)
sys.path.insert(0,str(root/'scripts'))
from runtime_isolation import pinned_child_runner
_,run_child=pinned_child_runner()
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');parity=importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
assert json.loads((runtime/'runtime-report.json').read_text())['passed']
review=root/'build/ELab-0.145.0-162-review';agent=runtime/'elab-agent'
assert hashlib.sha256(agent.read_bytes()).digest()==hashlib.sha256((review/'elab-agent').read_bytes()).digest()
r=dict(passed=False,scope='Exact clean copied /162 native socket/headless stdio with development-file and outbound-IP denial; strict MOSFET cutoff voltage reference, live/captured six-probe traces and fresh public reopens',agent_sha256=hashlib.sha256(agent.read_bytes()).hexdigest())
host=None

def reproduce(label,endpoint=None):
 cmd=[sys.executable,'-B',str(stage/'precision_reproduce.py'),'--agent-binary',str(agent),'--artifacts',str(out/label)]
 if endpoint:cmd+=['--connect',str(endpoint)]
 with (out/(label+'-output.txt')).open('x') as log:run_child(cmd,cwd=root,stdout=log,stderr=-2,timeout=45,check=True)
 report=json.loads((out/label/'report.json').read_text());assert report['passed'] and report['agent_sha256']==r['agent_sha256']
 return report

try:
 left=reproduce('headless');r['headless']={kind:dict(live=case['live_check'],captured=case['captured_check']) for kind,case in left['cases'].items()}
 with (out/'native-host.txt').open('x') as log:
  endpoint=out/'endpoint.json';host=subprocess.Popen([str(runtime/'native'),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);r['native_pid']=host.pid
  deadline=time.monotonic()+30
  while not endpoint.exists():assert host.poll() is None and time.monotonic()<deadline;time.sleep(.05)
  right=reproduce('native',endpoint)
  r['native']={kind:dict(live=case['live_check'],captured=case['captured_check']) for kind,case in right['cases'].items()}
  r['native_exit_code']=host.wait(timeout=15);assert r['native_exit_code']==0
 exclusions={'operation_id','created_at','input_revision','electrical_revision','document_generation'}
 for kind in ['nmos','pmos']:
  a,b=left['cases'][kind],right['cases'][kind]
  for key in ['components','wires','nets','paused','transient_settings']:assert a['live'][key]==b['live'][key],(kind,key)
  for key in ['mixed','transient','scope','npn','result']:parity.compare(a['live'].get(key),b['live'].get(key),kind+'.'+key)
  for side in ['live_check','captured_check']:assert a[side]==b[side]
  assert [row for page in a['pages'] for row in page['rows']]==[row for page in b['pages'] for row in page['rows']]
  parity.compare({k:v for k,v in a['capture']['metadata'].items() if k not in exclusions},{k:v for k,v in b['capture']['metadata'].items() if k not in exclusions},kind+'.metadata')
 r.update(passed=True,normal_native_closure=True,full_graph_physics_scope_and_capture_rows_match=True,capture_metadata_provenance_exclusions=sorted(exclusions),fresh_agent_reopens=4)
except BaseException as error:
 r['error']=type(error).__name__+': '+str(error);raise
finally:
 if host and host.poll() is None:
  host.terminate()
  try:host.wait(timeout=10)
  except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
 if host:r['final_native_exit_code']=host.returncode
 (out/'report.json').write_text(json.dumps(r,indent=2)+'\n')
print(json.dumps(r))
