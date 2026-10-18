import importlib.util,json,math,sys
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');sys.path.insert(0,str(root/'scripts'))
s=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');m=importlib.util.module_from_spec(s);s.loader.exec_module(m);m.helper.ROOT=root/'build/first-version-142-tools'
out=root/'build/live-controls-142-retry'
load=lambda n:json.loads((out/(n+'.json')).read_text())
baseline=load('before-charge');native={n:load(n) for n in ['queued-charge','charge-step','queued-discharge','discharge-step','restart']}
a=m.helper.Client(command_preconditions=True)
agent={}
try:
 a.ask('example.rc',{'discard_changes':True},True);a.ask('simulation.pause',{},True);a.ask('simulation.restart',{},True);a.ask('simulation.wait')
 for mode in ['charge','discharge']:
  a.ask('simulation.input.set',{'mode':mode},True);agent['queued-'+mode]=a.ask('project.inspect')
  a.ask('simulation.step.start',{'duration_s':.001},True);agent[mode+'-step']=a.ask('simulation.wait')
 a.ask('simulation.restart',{},True);agent['restart']=a.ask('simulation.wait')
 for name,actual in native.items():
  expected=agent[name]
  for k in ['components','wires','transient_settings']:assert actual[k]==expected[k]==baseline[k],(name,k)
  m.compare(actual['transient'],expected['transient'],name+'.transient')
  assert actual['last_actor']=='user' and actual['paused'] and not actual['transient']['stale']
  assert actual['electrical_revision']==baseline['electrical_revision']
  assert actual['run_id']==baseline['run_id']+(1 if name=='restart' else 0)
  for k in ['document','undo_entry','redo_entry']:
   assert actual.get(k)==baseline.get(k),(name,k)
 for mode,prior in [('charge','before-charge'),('discharge','charge-step')]:
  assert native['queued-'+mode]['transient']['sample']==load(prior)['transient']['sample']
  assert native['queued-'+mode]['transient']['pending_input_mode']==mode
  work=native[mode+'-step']['step_work'];assert work['actor']=='user' and work['status']=='completed' and work['completed_samples']==work['total_samples']==1
 v1=5*(-math.expm1(-.001/.01));v2=v1*math.exp(-.001/.01)
 for name,t,want in [('charge-step',.001,v1),('discharge-step',.002,v2)]:
  x=native[name]['transient']['sample'];assert x['time_s']==t and abs(x['voltage_v']-want)<1e-12
 assert native['restart']['transient']['sample']['time_s']==0 and native['restart']['transient']['input_mode']=='discharge'
 report={'passed':True,'phases':list(native),'comparison':'Every transient field (including retained samples/events) except independent run/electrical IDs; numeric rel 1e-9/abs 1e-12; exact graph/settings and native source/history invariants','independent_reference':{'law':'V1=5*(1-exp(-1ms/10ms)); V2=V1*exp(-1ms/10ms)','charge_voltage_v':v1,'discharge_voltage_v':v2,'absolute_tolerance_v':1e-12},'native_observed':{n:d['transient']['sample'] for n,d in native.items()},'limits':['RC-mode binding only; other graph mode inventories and numeric field entry not physically exercised','Not complete mounted 146-site binding coverage, VoiceOver/IME, or visible p95 evidence'],'headless_agent_actor':agent['charge-step']['last_actor']}
 (out/'agent-states.json').write_text(json.dumps(agent,indent=2));(out/'report.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(report,indent=2))
finally:a.close()
c=m.helper.Client(str(out/'endpoint.json'),command_preconditions=True)
try:
 latest=c.ask('project.inspect');assert latest['components']==baseline['components'] and latest['paused']
 c.ask('project.save',{'path':str(out/'RC-control-check.elab')},True);c.ask('simulation.wait');c.ask('app.close',{},True)
finally:c.close()
