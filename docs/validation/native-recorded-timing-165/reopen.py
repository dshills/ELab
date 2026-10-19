import hashlib,importlib.util,json,shutil,sys
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=Path(__file__).resolve().parent;out=base/'observations';runtime=Path('/private/tmp/elab-first-version-165-runtime');temp=Path('/private/tmp/elab-native-timing-165-reopens');temp.mkdir(mode=0o700)
sys.path.insert(0,str(root/'scripts'))
from demo_session import require_fresh_demo_session
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
assert json.loads((runtime/'runtime-report.json').read_text())['passed']
r=dict(passed=False,scope='Two fresh copied headless agents reopen the exact native-menu and matching headless saved files. Metadata and all 521 waveform rows must match each original file exactly; no private UI claim.',files={})
try:
 for label in ['Native','Agent']:
  src=out/(label+' timing.elab');dest=temp/src.name;shutil.copyfile(src,dest);document=json.loads(src.read_text());saved=document['recordings']['entries'][0]['data'];c=p.helper.Client(command_preconditions=True,agent_binary=runtime/'agent')
  try:
   require_fresh_demo_session(c.ask('project.inspect'));c.ask('project.load',dict(path=str(dest)),True);state=c.ask('simulation.wait');assert state['paused'] and not state['document']['dirty'] and state['document']['recording_count']==1
   operation=c.ask('operation.inspect',dict(id=state['last_operation']));assert operation['metadata']==saved['metadata'];rows=[];pages=[];offset=0
   while True:
    page=c.ask('operation.samples',dict(id=state['last_operation'],offset=offset,limit=200));pages.append(page);rows+=page['rows'];offset=page['next_offset']
    if offset==page['total']:break
   assert rows==saved['rows'] and len(rows)==521
   r['files'][label]=dict(passed=True,sha256=hashlib.sha256(src.read_bytes()).hexdigest(),state=state,capture=operation,pages=pages,metadata_rows_exact=True)
  finally:c.close()
 r.update(passed=True,fresh_agent_reopens=2)
finally:(out/'fresh-reopens.json').write_text(json.dumps(r,indent=2)+'\n')
print(json.dumps(dict(passed=r['passed'],fresh_agent_reopens=r.get('fresh_agent_reopens',0))))
