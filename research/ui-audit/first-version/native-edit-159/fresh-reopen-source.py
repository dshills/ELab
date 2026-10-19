import gzip,hashlib,importlib.util,json,sys
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');out=root/'build/native-edit-159/observations';sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
proof=validate_release(root,root/'research/ui-audit/first-version/numeric-placement-review-checkpoint.json');r=json.loads((out/'report.json').read_text());expected=next(e['state'] for e in r['events'] if e['name']=='corrected')
c=p.helper.Client(command_preconditions=True,agent_binary=Path(proof['agent']))
try:
 blank=c.ask('simulation.wait');assert not blank['components'] and not blank['wires'];c.ask('project.load',dict(path=str(out/'Native edited LED.elab')),True);s=c.ask('simulation.wait')
 for key in ['components','wires','nets']: assert s[key]==expected[key],key
 p.compare(s['result'],expected['result']);p.compare(s['scope'],expected['scope']);assert not s['document']['dirty'] and not s['result']['stale'];assert s['circuit']['resistance_ohm']==2200
 data=dict(passed=True,blank_start=True,scope='Fresh exact packaged agent public load of actual native-edited circuit; not a native file panel',agent_sha256=proof['agent_sha256'],state=s)
 with (out/'fresh-agent-reopen.json.gz').open('wb') as f:
  with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:z.write((json.dumps(data,indent=2)+'\n').encode())
finally:c.close()
(out/'source-release-identity.json').write_text(json.dumps(proof,indent=2)+'\n');print('Fresh packaged-agent reopen and immutable release verification passed')
