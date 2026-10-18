from pathlib import Path
import json,sys,importlib.util
root=Path('/Users/dshills/Development/projects/ELab');out=root/'build/native-typing-157/observations';sys.path.insert(0,str(root/'scripts'))
spec=importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
agent=root/'build/ELab-0.140.0-157-review/elab-agent';c=m.helper.Client(str(out/'endpoint.json'),command_preconditions=True,agent_binary=agent);ref=m.helper.Client(command_preconditions=True,agent_binary=agent)
try:
 s=c.ask('project.inspect');report=json.loads((out/'report.json').read_text());before=report['events'][-1]['state']
 for k in ['components','wires','nets','document','transient','run_id','electrical_revision','paused','undo_entry','redo_entry','status','transient_settings','result']:assert s.get(k)==before.get(k),k
 for command,params,mutate in [('example.rc',dict(discard_changes=True),True),('simulation.wait',{},False),('simulation.pause',{},True),('simulation.input.set',dict(mode='charge'),True),('simulation.restart',{},True),('simulation.wait',{},False),('scope.probes.set',dict(ids=['v:C1','i:R1']),True),('simulation.step',dict(duration_s=.02),True),('simulation.wait',{},False),('scope.cursor.set',dict(cursor='a',time_s='14 ms'),True)]:ref.ask(command,params,mutate)
 r=ref.ask('project.inspect');m.compare(s['scope'],r['scope']);m.compare(s['transient'],r['transient']);assert s['scope']['a']['requested_s']==.014
 (out/'separate-focus-control.json').write_text(json.dumps(dict(passed=True,scope='One additional CUA click/focus observation, then Cmd+A/type and text observation, then Return',native=s,agent=r,domain_preserved=True),indent=2)+'\n');print('separate-focus control matches packaged agent at 14 ms')
finally:c.close();ref.close()
