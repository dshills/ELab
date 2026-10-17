import importlib.util,json
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');d=root/'build/first-version-ui-load-check'
s=importlib.util.spec_from_file_location('cap',root/'scripts/capacity-demo.py');m=importlib.util.module_from_spec(s);s.loader.exec_module(m)
c=m.helper.Client(str(d/'endpoint.json'),command_preconditions=True)
try:
 base,source,parts,_=m.build_chain(c,'pulse-source','potentiometer',50,True);c.ask('simulation.pause',{},True);c.ask('simulation.wait')
 ground=next(x['id'] for x in base['components'] if x['kind']=='ground'); edits=[{'command':'wire.connect','params':{'from':parts[48]+'.wiper','to':parts[48]+'.1'}}]
 def place(id,kind,position,value=None):
  p={'id':id,'kind':kind,'position':position}
  if value is not None:p['value']=value
  edits.append({'command':'component.place','params':p})
 def wire(a,b):edits.append({'command':'wire.connect','params':{'from':a,'to':b}})
 for i,index in enumerate([5,15,25,35],1):
  id='StressC'+str(i);place(id,'capacitor',{'x':-500+i*200,'y':800},'100nF');wire(id+'.1',parts[index]+'.wiper');wire(id+'.2',ground+'.reference')
 place('StressM','nmos',{'x':600,'y':400});wire('StressM.gate',source+'.positive');wire('StressM.source',ground+'.reference');wire('StressM.drain',parts[25]+'.wiper')
 place('StressLoad','resistor',{'x':600,'y':0},'1k');place('StressLED','led',{'x':600,'y':200});wire('StressLoad.1',source+'.positive');wire('StressLoad.2','StressLED.anode');wire('StressLED.cathode',parts[25]+'.wiper')
 for i in range(6):
  id='StressR'+str(i+1);place(id,'resistor',{'x':1000,'y':i*140},'10k');wire(id+'.1',source+'.positive');wire(id+'.2',ground+'.reference')
 c.ask('edit.batch',{'name':'Dense RC switching stress','edits':edits},True)
 c.ask('simulation.transient.set',{'step_s':.001,'high_s':.1,'low_s':.1,'initial_voltage_v':0,'seconds_per_wall_second':1},True);c.ask('simulation.restart',{},True);p=c.ask('simulation.wait')
 print('Physical parts',p['simulated_component_count'],'nodes',len(p['nets']),'status',p['status'],'analysis',p['analysis']);print('diagnostics',p['diagnostics'][:1]);assert p['simulated_component_count']==64 and len(p['nets'])==100 and not p['transient']['stale']
 c.ask('scope.probes.set',{'ids':['v:StressC1','v:StressC2','v:StressC3','v:StressC4']},True);c.ask('component.select',{'id':'StressLoad'},True);c.ask('view.selection.center',{'id':'StressLoad'},True);c.ask('drawing.tool.set',{'tool':'pan'},True)
 c.ask('project.save',{'path':str(d/'Dense capacity interaction.elab')},True);p=c.ask('simulation.wait');(d/'dense-ready.json').write_text(json.dumps(p,indent=2));print('Dense supported project saved; four traces, centered load, paused.')
finally:c.close()
