import hashlib,importlib.util,json,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');stage=Path(__file__).resolve().parent
runtime=Path('/private/tmp/elab-first-version-165-runtime');out=Path('/private/tmp/elab-first-version-165-timing-retry');out.mkdir(mode=0o700)
sys.path.insert(0,str(root/'scripts'))
from demo_session import require_fresh_demo_session
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
assert json.loads((runtime/'runtime-report.json').read_text())['passed']
agent=runtime/'elab-agent';launcher=runtime/'agent';host=None;client=None
report=dict(passed=False,scope='Copied /165 native socket and headless stdio timing APIs; private field settlement/refusal feedback retains separate source-mounted proof',agent_sha256=hashlib.sha256(agent.read_bytes()).hexdigest())
protected=['components','wires','nets','document','run_id','electrical_revision','transient','mixed','npn','result','scope','undo_entry','redo_entry']

def page_data(c,identity):
 rows=[];pages=[];offset=0
 while True:
  page=c.ask('operation.samples',dict(id=identity,offset=offset,limit=200));assert page['offset']==offset;rows+=page['rows'];pages.append(page);offset=page['next_offset']
  if offset==page['total']:return pages,rows

def run_case(c,label):
 folder=out/label;folder.mkdir();require_fresh_demo_session(c.ask('project.inspect'))
 c.ask('example.npn-pulse',{},True);c.ask('simulation.pause',{},True);c.ask('simulation.restart',{},True);c.ask('simulation.wait')
 c.ask('simulation.verify',dict(duration_s=.5,probe_ids=['v:VIN.positive']),True);identity=c.ask('project.inspect')['last_operation'];deadline=time.monotonic()+15
 while True:
  capture=c.ask('operation.inspect',dict(id=identity))
  if capture['operation']['status']=='completed':break
  assert capture['operation']['status'] not in ['failed','canceled'] and time.monotonic()<deadline;time.sleep(.01)
 pages,rows=page_data(c,identity)
 c.ask('recording.view.set',dict(mode='recorded'),True)
 cfg=dict(id=identity,probe_id='v:VIN.positive',level='2.5 V',direction='both',start_s=0,end_s=.5,limit=8,offset=0)
 c.ask('recording.timing.set',cfg,True);baseline=c.ask('project.inspect');states=dict(baseline=baseline)
 assert baseline['recording']['timing']['total_crossings']>8
 def remember(name):
  state=c.ask('project.inspect')
  for key in protected:assert state.get(key)==baseline.get(key),(label,name,key)
  states[name]=state;return state
 cfg=dict(baseline['recording']['timing']['settings']);cfg['level']='3 V';cfg['offset']=0
 c.ask('recording.timing.set',cfg,True);field=remember('accepted-field')
 assert field['recording']['timing']['settings']['level']=='3 V' and field['recording']['timing']['settings'].get('offset',0)==0
 next_cfg=dict(field['recording']['timing']['settings']);next_cfg['offset']=field['recording']['timing']['next_offset']
 c.command_revision=baseline['command_revision'];c.ask('recording.timing.set',next_cfg,True,expect='conflict');refused=remember('stale-page-refused');assert refused==field
 c.ask('recording.timing.set',next_cfg,True);next_state=remember('next-page')
 assert next_state['recording']['timing']['settings'].get('offset',0)==8 and next_state['command_revision']==baseline['command_revision']+2
 invalid=dict(next_state['recording']['timing']['settings']);invalid['level']='bad';invalid['offset']=0
 c.ask('recording.timing.set',invalid,True,expect='invalid-params');invalid_state=remember('invalid-field-refused');assert invalid_state==next_state
 corrected=dict(next_state['recording']['timing']['settings']);corrected['level']='2 V';corrected['offset']=0
 c.ask('recording.timing.set',corrected,True);corrected_state=remember('corrected-field');assert corrected_state['recording']['timing']['settings'].get('offset',0)==0
 corrected_next=dict(corrected_state['recording']['timing']['settings']);corrected_next['offset']=corrected_state['recording']['timing']['next_offset']
 c.ask('recording.timing.set',corrected_next,True);corrected_page=remember('corrected-next');assert corrected_page['recording']['timing']['settings'].get('offset',0)==8
 previous=dict(corrected_page['recording']['timing']['settings']);previous['offset']=max(0,previous['offset']-previous['limit'])
 c.ask('recording.timing.set',previous,True);previous_state=remember('previous-page');assert previous_state['recording']['timing']['settings'].get('offset',0)==0
 after_capture=c.ask('operation.inspect',dict(id=identity));after_pages,after_rows=page_data(c,identity)
 assert capture['metadata']==after_capture['metadata'] and pages==after_pages and rows==after_rows
 path=folder/'Timing capture.elab';c.ask('project.save',dict(path=str(path)),True);assert not c.ask('simulation.wait')['document']['dirty']
 document=json.loads(path.read_text());assert document['recordings']['entries'][0]['data']==dict(metadata=capture['metadata'],rows=rows)
 fresh=p.helper.Client(command_preconditions=True,agent_binary=launcher)
 try:
  require_fresh_demo_session(fresh.ask('project.inspect'));fresh.ask('project.load',dict(path=str(path)),True);reopened=fresh.ask('simulation.wait');assert reopened['paused'] and not reopened['document']['dirty']
  assert fresh.ask('operation.inspect',dict(id=reopened['last_operation']))['metadata']==capture['metadata']
  _,fresh_rows=page_data(fresh,reopened['last_operation']);assert fresh_rows==rows
 finally:fresh.close()
 data=dict(passed=True,states=states,capture=capture,pages=pages,reopened=reopened,original_capture_unchanged=True,saved_sha256=hashlib.sha256(path.read_bytes()).hexdigest(),scope='Public API sequence only; no private UI field claim')
 (folder/'report.json').write_text(json.dumps(data,indent=2)+'\n');return data

try:
 client=p.helper.Client(command_preconditions=True,agent_binary=launcher);left=run_case(client,'headless');client.close();client=None
 with (out/'native-host.txt').open('x') as log:
  endpoint=out/'endpoint.json';host=subprocess.Popen([str(runtime/'native'),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);report['native_pid']=host.pid;deadline=time.monotonic()+30
  while not endpoint.exists():assert host.poll() is None and time.monotonic()<deadline;time.sleep(.05)
  client=p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=launcher);right=run_case(client,'native')
  client.ask('app.close',{},True);client.close();client=None;report['native_exit_code']=host.wait(timeout=15);assert report['native_exit_code']==0
 for name,a in left['states'].items():
  b=right['states'][name]
  for key in ['components','wires','nets','paused','transient_settings']:assert a[key]==b[key],(name,key)
  for key in ['result','transient','mixed','npn','scope']:p.compare(a.get(key),b.get(key),name+'.'+key)
  p.compare(a['recording']['timing'],b['recording']['timing'],name+'.timing')
 assert [row for page in left['pages'] for row in page['rows']]==[row for page in right['pages'] for row in page['rows']]
 report.update(passed=True,normal_native_closure=True,stage_matches=len(left['states']),stale_page_revision_refused=True,invalid_threshold_refused=True,corrected_and_previous_pages_match=True,capture_metadata_rows_preserved=True,fresh_agent_reopens=2)
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
