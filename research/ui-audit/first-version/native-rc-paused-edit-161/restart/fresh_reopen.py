import gzip,hashlib,importlib.util,json,math,sys
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-rc-edit-161';out=base/'observations';sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json');spec=importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p);agent=Path(proof['agent']);r=json.loads((out/'report.json').read_text());expected=next(e['state'] for e in r['events'] if e['name']=='live-paused');states={}
for label,name in [('native','Native edited RC.elab'),('reference','Agent edited RC.elab')]:
 c=p.helper.Client(command_preconditions=True,agent_binary=agent)
 try:
  blank=c.ask('simulation.wait');assert not blank['components'] and not blank['wires'];c.ask('project.load',dict(path=str(out/name)),True);s=c.ask('simulation.wait')
  for key in ['components','wires','nets','transient_settings']:assert s[key]==expected[key],key
  assert s['scope']['selected']==['v:C1','i:R1'] and not s['document']['dirty'] and s['paused'] and not s['transient']['stale'];sample=s['transient']['sample'];assert sample['time_s']==0 and sample['voltage_v']==0 and math.isclose(sample['current_a'],.00025,abs_tol=1e-12) and math.isclose(s['transient']['time_constant_s'],.02,abs_tol=1e-12);states[label]=s
 finally:c.close()
p.compare(states['native']['transient'],states['reference']['transient']);p.compare(states['native']['scope'],states['reference']['scope'])
with (out/'fresh-agent-reopen.json.gz').open('wb') as f:
 with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:z.write((json.dumps(dict(passed=True,blank_start=True,scope='Two fresh exact packaged agents reopen native/reference saved circuits at saved paused startup; does not persist live waveform/cursors or qualify native panels',states=states),indent=2)+'\n').encode())
print('Fresh RC reopening preserves graph/settings/probes and paused 0 V startup')
