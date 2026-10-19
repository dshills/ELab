import hashlib,importlib.util,json,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');out=Path('/private/tmp/elab-165-archive-rehearsal');sys.path.insert(0,str(root/'scripts'))
import runtime_isolation as isolation
from demo_session import require_fresh_demo_session
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
packaging=json.loads((out/'report.json').read_text());assert packaging['passed'];review=Path(packaging['extracted_review']);before=isolation.bundle_tree(review/'ELab.app');all_before=json.loads((review/'RELEASE_MANIFEST.json').read_text());runtime=Path('/private/tmp/elab-first-version-165-runtime');rr=json.loads((runtime/'runtime-report.json').read_text());assert rr['passed']
profile=out/'runtime.sb';profile.write_text(isolation.sandbox_profile([Path(v['root']) for v in rr['blocked_development'].values()]));assert profile.read_bytes()==(runtime/'runtime.sb').read_bytes()
work=out/'Empty working directory µ';work.mkdir();isolation.launcher(out/'native',profile,review/'ELab.app/Contents/MacOS/ELab',work);isolation.launcher(out/'agent',profile,review/'elab-agent',work)
endpoint=out/'owned-endpoint.json';host=client=reference=None
report=dict(passed=False,scope='Exact archived /165 app/agent extracted by macOS ditto into a Unicode path with spaces. Direct executable startup and public example load/save/reopen under the existing qualified denial policy. No Finder/LaunchServices/quarantine, default-home, clean-account, installation or physical UI claim.',archive_sha256=packaging['archive_sha256'],native_sha256=isolation.digest(review/'ELab.app/Contents/MacOS/ELab'),agent_sha256=isolation.digest(review/'elab-agent'),qualified_runtime_report_sha256=isolation.digest(runtime/'runtime-report.json'),same_denial_policy_sha256=isolation.digest(profile),cases={})
try:
 with (out/'native-host.txt').open('x') as log:host=subprocess.Popen([str(out/'native'),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2)
 report['native_pid']=host.pid;deadline=time.monotonic()+30
 while not endpoint.exists():assert host.poll() is None and time.monotonic()<deadline;time.sleep(.05)
 client=p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=out/'agent');reference=p.helper.Client(command_preconditions=True,agent_binary=out/'agent');require_fresh_demo_session(client.ask('project.inspect'));require_fresh_demo_session(reference.ask('project.inspect'))
 for example in sorted((review/'Examples').glob('*.elab')):
  states={}
  for label,c in [('native',client),('headless',reference)]:
   c.ask('project.load',dict(path=str(example),discard_changes=True),True);state=c.ask('simulation.wait');assert state['paused'] and not state['document']['dirty'];states[label]=state
   destination=out/(label+' '+example.name);c.ask('project.save',dict(path=str(destination)),True)
   fresh=p.helper.Client(command_preconditions=True,agent_binary=out/'agent')
   try:
    require_fresh_demo_session(fresh.ask('project.inspect'));fresh.ask('project.load',dict(path=str(destination)),True);reopened=fresh.ask('simulation.wait');assert reopened['paused'] and not reopened['document']['dirty']
    for key in ['components','wires','nets','paused']:assert state[key]==reopened[key],(example.name,label,key)
    for key in ['result','transient','mixed','npn','scope']:p.compare(state.get(key),reopened.get(key),key)
    report['cases'].setdefault(example.name,{})[label]=dict(state=state,reopened=reopened,saved_sha256=isolation.digest(destination),source_example_sha256=isolation.digest(example))
   finally:fresh.close()
  for key in ['components','wires','nets','paused']:assert states['native'][key]==states['headless'][key],(example.name,key)
  for key in ['result','transient','mixed','npn','scope']:p.compare(states['native'].get(key),states['headless'].get(key),key)
 assert len(report['cases'])==3
 client.ask('app.close',{},True);client.close();client=None;report['native_exit_code']=host.wait(timeout=15);assert report['native_exit_code']==0 and not endpoint.exists()
 assert isolation.bundle_tree(review/'ELab.app')==before
 for name,entry in all_before['files'].items():assert isolation.digest(review/name)==entry['sha256'] and oct((review/name).stat().st_mode&0o777)==entry['mode'],name
 report.update(passed=True,normal_native_closure=True,endpoint_removed=True,extracted_review_unchanged=True,example_count=3,fresh_agent_reopens=6)
except BaseException as error:report['error']=type(error).__name__+': '+str(error);raise
finally:
 if client:client.close()
 if reference:reference.close()
 if host and host.poll() is None:
  host.terminate()
  try:host.wait(timeout=10)
  except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
 if host:report['final_native_exit_code']=host.returncode
 (out/'runtime-report.json').write_text(json.dumps(report,indent=2)+'\n')
print(json.dumps({key:report.get(key) for key in ['passed','example_count','fresh_agent_reopens','native_exit_code','final_native_exit_code']}))
