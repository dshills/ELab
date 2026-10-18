from pathlib import Path
import importlib.util,json,math,sys
root=Path('/Users/dshills/Development/projects/ELab');sys.path.insert(0,str(root/'scripts'))
s=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');m=importlib.util.module_from_spec(s);s.loader.exec_module(m);m.helper.ROOT=root/'build/first-version-143-tools'
p=root/'build/pot-gesture-ui';load=lambda n:json.loads((p/(n+'.json')).read_text())
before,drag,undo=load('before-drag'),load('after-drag'),load('after-undo')
wiper=lambda state:next(x['wiper_position'] for x in state['components'] if x['id']=='Timing')
assert wiper(before)==.5 and wiper(undo)==.5
assert undo['components']==before['components'] and undo['wires']==before['wires']
assert undo['undo_entry']==before['undo_entry'] and undo['redo_entry']==drag['undo_entry']
assert undo['paused'] and undo['mixed']['stale'] and undo['transient']['stale'] and not undo['document']['dirty']
assert abs(before['mixed']['capacitors'][0]['voltage_v']-2.5*(-math.expm1(-2)))<.002
agent=m.helper.Client(command_preconditions=True)
try:
 agent.ask('project.load',{'path':str(p/'Pot-before.elab')},True);agent.ask('simulation.wait');agent.ask('simulation.step',{'duration_s':.005},True);a_before=agent.ask('simulation.wait')
 agent.ask('potentiometer.wiper.set',{'id':'Timing','position':wiper(drag)},True);a_drag=agent.ask('simulation.wait')
 tip=a_drag['undo_entry']['command_revision'];agent.ask('edit.undo',{'expected_entry_revision':tip},True);a_undo=agent.ask('project.inspect')
 for name,native,peer in [('before',before,a_before),('drag',drag,a_drag),('undo',undo,a_undo)]:
  assert native['components']==peer['components'] and native['wires']==peer['wires'],name
  m.compare(native['mixed'],peer['mixed'],name+'.mixed');m.compare(native['transient'],peer['transient'],name+'.transient')
 (p/'agent-states.json').write_text(json.dumps({'before':a_before,'drag':a_drag,'undo':a_undo},indent=2))
finally:agent.close()
report={'passed':True,'scope':'one mounted physical mouse drag and one native Cmd+Z on the exact review-143 executable; separate agent replay','wiper_before':.5,'wiper_after_drag':wiper(drag),'wiper_after_one_undo':wiper(undo),'command_revision_delta_drag':drag['command_revision']-before['command_revision'],'new_wiper_event_count':len([e for e in drag['transient']['events'] if e['mode'].startswith('wiper:Timing:')])-len([e for e in before['transient']['events'] if e['mode'].startswith('wiper:Timing:')]),'undo_entry':drag['undo_entry'],'before_undo_tip_restored':True,'physical_time_s':drag['transient']['sample']['time_s'],'capacitor_voltage_v':drag['mixed']['capacitors'][0]['voltage_v'],'independent_thevenin_reference_v':2.5*(-math.expm1(-2)),'independent_reference_tolerance_v':.002,'run_electrical_and_charge_preserved_during_drag':True,'undo_marks_results_stale_and_restores_clean_source':True,'agent_comparison':'Every reported mixed/transient field including traces/events except independent run/electrical identities; rel 1e-9/abs 1e-12; exact graph equality at all three phases','limits':['One paused capacitor-loaded pot at one window/configuration','No agent interleaving during held drag, loss of focus mid-drag, running drag, VoiceOver/IME or latency percentile claim','Slider and exact percent field remain absent from external AX snapshot; physical drag located using actual screenshot']}
(p/'report.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(report,indent=2))
c=m.helper.Client(str(p/'endpoint.json'),command_preconditions=True)
try:
 c.ask('project.save',{'path':str(p/'Pot-after-undo.elab')},True);c.ask('simulation.wait');c.ask('app.close',{},True)
finally:c.close()
