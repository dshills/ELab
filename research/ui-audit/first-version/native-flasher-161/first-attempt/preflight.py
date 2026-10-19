import importlib.util,json,statistics,sys
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-flasher-161';sys.path.insert(0,str(root/'scripts'));spec=importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p);agent=root/'build/ELab-0.144.0-161-review/elab-agent';c=p.helper.Client(command_preconditions=True,agent_binary=agent)
def advance(duration):
 while duration>1e-12:
  step=min(1.,duration);c.ask('simulation.step',dict(duration_s=step),True);c.ask('simulation.wait');duration-=step

def edges(s):
 traces={t['probe']['id']:t['points'] for t in s['scope']['traces']};out={}
 for probe in ['i:D1','i:D2']:
  points=traces[probe];out[probe]=[b['time_s'] for a,b in zip(points,points[1:]) if a['value']<.001<=b['value'] and b['time_s']>.4]
 return out
try:
 c.ask('example.flasher',{},True);c.ask('simulation.pause',{},True);c.ask('simulation.restart',{},True);initial=c.ask('simulation.wait');advance(1.6);first=c.ask('simulation.wait');first_edges=edges(first)
 for part in ['C1','C2']:
  c.ask('component.select',dict(id=part),True);c.ask('editor.value.set',dict(value='20uF'),True);c.ask('component.set',dict(id=part,value='20uF'),True)
 c.ask('simulation.restart',{},True);advance(3.4);second=c.ask('simulation.wait');second_edges=edges(second)
 period=lambda e:statistics.median([b-a for a,b in zip(e['i:D1'],e['i:D1'][1:])]);a,b=period(first_edges),period(second_edges)
 r=dict(passed=True,initial_probe_ids=initial['scope']['selected'],step_s=initial['transient_settings']['step_s'],initial_selected=initial['selected'],period_10uf_s=a,period_20uf_s=b,ratio=b/a,first_edges=first_edges,second_edges=second_edges,initial_capacitors=[x for x in initial['components'] if x['kind']=='capacitor']);assert abs(b/a-2)<.06;(base/'preflight.json').write_text(json.dumps(r,indent=2)+'\n');print(json.dumps(r))
finally:c.close()
