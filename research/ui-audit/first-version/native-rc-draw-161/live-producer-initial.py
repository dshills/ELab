import gzip,importlib.util,json,math,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');out=root/'build/native-rc-draw-161-retry/observations';sys.path.insert(0,str(root/'scripts'));spec=importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
construction=json.loads((out/'report.json').read_text());assert (out/'timing-verified.json').exists();baseline=next(e['state'] for e in construction['events'] if e['name']=='timing');agent=Path(construction['release']['agent']);ref=p.helper.Client(command_preconditions=True,agent_binary=agent);c=p.helper.Client(str(out/'endpoint.json'),command_preconditions=True,agent_binary=agent);report=dict(passed=False,scope='Actual native Resume/Pause on the natively built RC circuit; paused full transient/scope against fresh agent replay to observed time, independent discharge law; no wall-time equivalence')
def save():(out/'live-report.json').write_text(json.dumps(report,indent=2)+'\n')
def wait(name):
 deadline=time.monotonic()+300
 while not (out/name).exists():assert time.monotonic()<deadline,'live UI lease expired';time.sleep(.05)
 s=c.ask('simulation.wait');report[name]=s;save();return s
def mark(name):(out/name).write_text('{}\n')
try:
 document=dict(graph=dict(components=baseline['components'],wires=baseline['wires']),transient_settings=baseline['transient_settings'],scope_probes=baseline['scope']['selected']);p.construct(ref,document)
 ref.ask('simulation.input.set',dict(mode='charge'),True)
 for _ in range(10):ref.ask('simulation.step',dict(duration_s=.001),True)
 ref.ask('simulation.input.set',dict(mode='discharge'),True)
 for _ in range(10):ref.ask('simulation.step',dict(duration_s=.001),True)
 for cursor,at in [('a',.01),('b',.02)]:ref.ask('scope.cursor.set',dict(cursor=cursor,time_s=at),True)
 reference=ref.ask('simulation.wait')
 for key in ['scope','transient']:p.compare(baseline[key],reference[key])
 mark('live-ready.json');running=wait('live-running');assert not running['paused'] and running['transient']['sample']['time_s']>.02;mark('live-running-verified.json')
 paused=wait('live-paused');assert paused['paused'];duration=paused['transient']['sample']['time_s']-.02;assert duration>0
 for key in ['components','wires','nets','selected','document','undo_entry','redo_entry','run_id','electrical_revision']:assert paused[key]==baseline[key],key
 ref.ask('simulation.step',dict(duration_s=duration),True);reference=ref.ask('simulation.wait')
 for key in ['scope','transient']:p.compare(paused[key],reference[key])
 peak=5*(-math.expm1(-1));sample=paused['transient']['sample'];want=peak*math.exp(-(sample['time_s']-.01)/.01);assert math.isclose(sample['voltage_v'],want,rel_tol=1e-10,abs_tol=1e-12);assert math.isclose(sample['current_a'],-want/10000,rel_tol=1e-10,abs_tol=1e-12)
 time.sleep(.15);stable=c.ask('simulation.wait')
 for key in ['transient','scope','run_id','electrical_revision','document','undo_entry','redo_entry']:assert paused[key]==stable[key],key
 report.update(passed=True,paused_time_s=sample['time_s'],paused_voltage_v=sample['voltage_v'],independent_discharge_law=True,full_reference_transient_scope_parity=True,paused_physics_stable=True);save();mark('live-paused-verified.json')
finally:ref.close();c.close();save()
