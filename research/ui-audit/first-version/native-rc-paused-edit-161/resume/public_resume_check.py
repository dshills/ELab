import gzip,hashlib,importlib.util,json,math,sys
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-rc-resume-161';out=base/'observations';sys.path.insert(0,str(root/'scripts'));spec=importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
original=json.loads((out/'report.json').read_text());agent=Path(original['release']['agent']);fixture=root/'build/native-rc-edit-161/observations/Native edited RC.elab';assert hashlib.sha256(fixture.read_bytes()).hexdigest()==original['fixture_sha256'];c=p.helper.Client(command_preconditions=True,agent_binary=agent);report=dict(passed=False,scope='Fresh packaged-agent actual simulation.resume on a stale RC trace; complements native direct Resume and explicit-reset full trace comparator',states={})
try:
 assert c.ask('project.inspect')['components']==[];c.ask('project.load',dict(path=str(fixture)),True);c.ask('simulation.pause',{},True);c.ask('simulation.input.set',dict(mode='charge'),True);c.ask('simulation.restart',{},True);c.ask('simulation.step',dict(duration_s=.02),True);before=c.ask('simulation.wait');report['states']['baseline']=before
 c.ask('editor.value.set',dict(value='10k'),True);c.ask('component.set',dict(id='R1',value='10k'),True);edited=c.ask('simulation.wait');report['states']['stale']=edited;assert edited['transient']['stale'] and edited['transient']['sample']==before['transient']['sample']
 resumed=c.ask('simulation.resume',{},True);report['states']['resume_response']=resumed
 assert resumed['run_id']==edited['run_id']+1 and resumed['transient']['run_id']==resumed['run_id'] and not resumed['transient']['stale'];assert resumed['transient']['sample']['time_s']==0 and resumed['transient']['sample']['voltage_v']==0;assert resumed['transient']['time_constant_s']==.01
 c.ask('simulation.pause',{},True);paused=c.ask('simulation.wait');report['states']['paused']=paused
 traces={x['probe']['id']:x['points'] for x in paused['scope']['traces']}
 for i,row in enumerate(paused['transient']['samples']):
  voltage=5*(-math.expm1(-row['time_s']/.01));current=(5-voltage)/10000
  assert math.isclose(row['voltage_v'],voltage,rel_tol=1e-10,abs_tol=1e-12) and math.isclose(row['current_a'],current,rel_tol=1e-10,abs_tol=1e-12)
  for probe,want in [('v:C1',voltage),('i:R1',current)]:assert math.isclose(traces[probe][i]['value'],want,rel_tol=1e-10,abs_tol=1e-12)
 report.update(passed=True,new_run_starts_at_zero=True,time_constant_s=.01,agent_sha256=hashlib.sha256(agent.read_bytes()).hexdigest())
finally:
 c.close();report['agent_exit_code']=c.process.returncode
 with (out/'public-direct-resume.json.gz').open('wb') as f:
  with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:z.write((json.dumps(report,indent=2)+'\n').encode())
print(json.dumps({k:v for k,v in report.items() if k!='states'}))
