import gzip,hashlib,importlib.util,json,sys
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');out=root/'build/first-version-160-tools/native-observations';sys.path.insert(0,str(root/'scripts'));spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
agent=root/'build/ELab-0.143.0-160-review/elab-agent';assert hashlib.sha256(agent.read_bytes()).hexdigest()=='bf177e5f3d136196745f09bb4f090bc87ce7b45c82218fd5991fc890acd86cc4';r=json.loads((out/'report.json').read_text());expected=next(e['state'] for e in r['events'] if e['name']=='corrected');c=p.helper.Client(command_preconditions=True,agent_binary=agent)
try:
 blank=c.ask('simulation.wait');assert not blank['components'] and not blank['wires'];c.ask('project.load',dict(path=str(out/'Native corrected LED.elab')),True);s=c.ask('simulation.wait')
 for k in ['components','wires','nets']:assert s[k]==expected[k],k
 p.compare(s['result'],expected['result']);p.compare(s['scope'],expected['scope']);assert not s['document']['dirty'] and not s['result']['stale']
 with (out/'fresh-agent-reopen.json.gz').open('wb') as f:
  with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:z.write((json.dumps(dict(passed=True,blank_start=True,scope='Exact packaged /160 agent fresh public load of actual native-corrected file; not native file-panel acceptance',state=s),indent=2)+'\n').encode())
finally:c.close()
print('Fresh packaged-agent reopen passes')
