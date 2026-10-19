import datetime,hashlib,importlib.util,json,math,statistics,subprocess,sys,time
from pathlib import Path
root=Path('/Users/dshills/Development/projects/ELab');base=root/'build/native-flasher-161';out=base/'observations';sys.path.insert(0,str(root/'scripts'));from wiring_replay import validate_release
proof=validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json');identity=json.loads((base/'identity.json').read_text());agent=Path(proof['agent']);binary=root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery';endpoint=out/'endpoint.json';assert not endpoint.exists();assert hashlib.sha256(binary.read_bytes()).hexdigest()==identity['native_sha256']
spec=importlib.util.spec_from_file_location('parity',root/'scripts/first-version-parity.py');p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)
r=dict(passed=False,scope='Native /161 Examples-loaded capacitor-timed flasher, live oscillation, C2 voltage probe backfill, Q2 trace focus, cursors and two native capacitance edits; exact agent reference and independent scaling/signal checks',release=proof,identity=identity,events=[],reference={},periods={},started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat());host=None;c=None;ref=None
def save():
 f=out/'report.pending';f.write_text(json.dumps(r,indent=2)+'\n');f.replace(out/'report.json')
def mark(name):(out/name).write_text('{}\n')
def event(name,s):r['events'].append(dict(name=name,utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),state=s));save()
def wait(name):
 deadline=time.monotonic()+900
 while not (out/name).exists():assert host.poll() is None,'host exited before '+name;assert time.monotonic()<deadline,'UI lease expired: '+name;time.sleep(.05)
 s=c.ask('simulation.wait');event(name,s);return s
def reference(name):r['reference'][name]=ref.ask('simulation.wait');save();return r['reference'][name]
def equivalent(a,b):
 for key in ['components','wires','nets','selected','drawing','paused','transient_settings','undo_name','redo_name']:assert a.get(key)==b.get(key),key
 for key in ['result','scope','transient','mixed','trace_focus']:p.compare(a.get(key),b.get(key),key)
def protected(a,b,scope=False,selection=False):
 keys=['components','wires','nets','transient','mixed','run_id','electrical_revision','document','undo_entry','redo_entry','paused']
 if selection:keys.append('selected')
 if scope:keys+=['scope','trace_focus']
 for key in keys:assert a.get(key)==b.get(key),key
def advance_to(target):
 at=ref.ask('simulation.wait')['transient']['sample']['time_s']
 while target-at>1e-10:
  ref.ask('simulation.step.start',dict(duration_s=min(1.,target-at)),True);at=ref.ask('simulation.wait')['transient']['sample']['time_s']
def clock_reach(target,label):
 deadline=time.monotonic()+45;observations=[];s=None
 while True:
  assert host.poll() is None;assert time.monotonic()<deadline,'live clock target timeout'
  s=c.ask('project.inspect');assert not s['paused'] and s['status']!='error' and not s['transient']['stale'] and not s['mixed']['stale'],s.get('diagnostics');at=s['transient']['sample']['time_s']
  if not observations or at-observations[-1]>=.1:observations.append(at)
  if at>=target:break
  time.sleep(.05)
 r[label+'-clock-times']=observations;event(label+'-clock-target',s);mark(label+'-verified.json')
def metrics(s,label):
 traces={t['probe']['id']:t['points'] for t in s['scope']['traces']};a,b=traces['i:D1'],traces['i:D2'];assert len(a)==len(b)==len(s['transient']['samples']);edges={};periods={};brackets={}
 for probe,points in [('i:D1',a),('i:D2',b)]:
  assert min(x['value'] for x in points)<1e-5 and max(x['value'] for x in points)>.002
  pairs=[(x,y) for x,y in zip(points,points[1:]) if x['value']<.001<=y['value'] and y['time_s']>.4];assert len(pairs)>=2,(label,probe,len(pairs));edges[probe]=[y['time_s'] for x,y in pairs];brackets[probe]=[[x['time_s'],y['time_s']] for x,y in pairs];periods[probe]=statistics.median([y-x for x,y in zip(edges[probe],edges[probe][1:])])
 opposite=sum((x['value']>.001)!=(y['value']>.001) for x,y in zip(a,b));assert opposite>len(a)/2
 mixed=s['mixed'];assert mixed['kcl_residual_a']<1e-9 and abs(mixed['power_residual_w'])<1e-8
 for q in mixed['transistors']:
  values=traces['v:'+q['transistor_id']+'.collector'];assert math.isclose(values[-1]['value'],q['vce_v'],rel_tol=1e-10,abs_tol=1e-12)
 result=dict(passed=True,step_s=s['transient_settings']['step_s'],period_s=periods,edges_s=edges,edge_brackets_s=brackets,opposite_led_rows=opposite,total_rows=len(a),threshold_a=.001,sampled_transition_times=True)
 r['periods'][label]=result;target=dict(a_s=edges['i:D1'][-2],b_s=edges['i:D1'][-1],a_text=f"{edges['i:D1'][-2]*1000:.0f} ms",b_text=f"{edges['i:D1'][-1]*1000:.0f} ms")
 (out/('cursor-'+label+'.json')).write_text(json.dumps(target,indent=2)+'\n');save();return result,target
def measure(label,base_state,target):
 private=wait('private-a-'+label);protected(base_state,private,True,True);mark('private-a-'+label+'-verified.json')
 ref.ask('scope.cursor.set',dict(cursor='a',time_s=target['a_text']),True);ref.ask('scope.cursor.select',dict(cursor='b'),True);reference('a-and-select-b-'+label)
 selected=wait('a-and-select-b-'+label);protected(base_state,selected,selection=True);equivalent(selected,r['reference']['a-and-select-b-'+label]);mark('a-and-select-b-'+label+'-verified.json')
 private=wait('private-b-'+label);protected(selected,private,True,True);mark('private-b-'+label+'-verified.json')
 ref.ask('scope.cursor.set',dict(cursor='b',time_s=target['b_text']),True);reference('timing-'+label);timing=wait('timing-'+label);protected(base_state,timing,selection=True);equivalent(timing,r['reference']['timing-'+label]);scope=timing['scope'];want=target['b_s']-target['a_s'];assert math.isclose(scope['delta_s'],want,abs_tol=1e-12) and math.isclose(scope['interval_hz'],1/want,rel_tol=1e-10)
 reading=next(x for x in scope['readings'] if x['probe_id']=='i:D1');assert reading['a']>=.001 and reading['b']>=.001;mark('timing-'+label+'-verified.json');return timing
def stale(before,after):
 assert after['paused'] and after['transient']['stale'] and after['mixed']['stale'] and after['scope']['stale'];assert before['transient']['sample']==after['transient']['sample'] and before['transient']['samples']==after['transient']['samples'];assert [t['points'] for t in before['scope']['traces']]==[t['points'] for t in after['scope']['traces']]
with (out/'native-host.txt').open('x') as log:
 try:
  ref=p.helper.Client(command_preconditions=True,agent_binary=agent);ref.ask('example.flasher',{},True);ref.ask('simulation.pause',{},True);ref.ask('simulation.restart',{},True);initial=reference('restart-10')
  host=subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2);r['pid']=host.pid;save();deadline=time.monotonic()+20
  while not endpoint.exists():assert host.poll() is None;assert time.monotonic()<deadline;time.sleep(.05)
  c=p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=agent);blank=c.ask('simulation.wait');assert not blank['components'] and not blank['wires'];event('blank',blank);mark('ready.json')
  loaded=wait('loaded-paused');assert loaded['paused'] and not loaded['transient']['stale']
  for key in ['components','wires','nets','transient_settings']:assert loaded[key]==initial[key],key
  mark('loaded-paused-verified.json');start=wait('restart-10');equivalent(start,initial);mark('restart-10-verified.json')
  running=wait('running-10');assert not running['paused'];clock_reach(1.6,'running-10');paused=wait('paused-10');assert paused['paused'];advance_to(paused['transient']['sample']['time_s']);reference('paused-10');equivalent(paused,r['reference']['paused-10']);first,targets=metrics(paused,'10');mark('paused-10-verified.json')
  ids=paused['scope']['selected'][:];assert ids[-1]=='v:V1.positive';ids[-1]='v:C2';ref.ask('scope.probes.set',dict(ids=ids),True);reference('probe-c2');probed=wait('probe-c2');protected(paused,probed);equivalent(probed,r['reference']['probe-c2']);trace=next(t for t in probed['scope']['traces'] if t['probe']['id']=='v:C2');cap=next(x for x in probed['mixed']['capacitors'] if x['id']=='C2');assert len(trace['points'])==len(probed['transient']['samples']) and math.isclose(trace['points'][-1]['value'],cap['voltage_v'],rel_tol=1e-10,abs_tol=1e-12);r['new_probe_backfills_history']=True;mark('probe-c2-verified.json')
  ref.ask('scope.trace.select',dict(id='v:Q2.collector'),True);reference('focus-q2');focused=wait('focus-q2');protected(probed,focused);equivalent(focused,r['reference']['focus-q2']);assert focused['trace_focus']['objects']==['Q2'];mark('focus-q2-verified.json');timing=measure('10',focused,targets)
  previous=timing
  for part in ['C1','C2']:
   ref.ask('component.select',dict(id=part),True);reference('selected-'+part);selected=wait('selected-'+part);equivalent(selected,r['reference']['selected-'+part]);mark('selected-'+part+'-verified.json');private=wait('private-'+part);protected(selected,private,True,True);mark('private-'+part+'-verified.json')
   ref.ask('editor.value.set',dict(value='20uF'),True);ref.ask('component.set',dict(id=part,value='20uF'),True);reference('edited-'+part);edited=wait('edited-'+part);equivalent(edited,r['reference']['edited-'+part]);stale(selected,edited);previous=edited;mark('edited-'+part+'-verified.json')
  ref.ask('simulation.restart',{},True);reference('restart-20');start=wait('restart-20');equivalent(start,r['reference']['restart-20']);assert start['transient']['sample']['time_s']==0 and 'a' not in start['scope'] and 'b' not in start['scope'];mark('restart-20-verified.json')
  wait('running-20');clock_reach(3.4,'running-20');paused=wait('paused-20');assert paused['paused'];advance_to(paused['transient']['sample']['time_s']);reference('paused-20');equivalent(paused,r['reference']['paused-20']);second,targets=metrics(paused,'20');ratio=second['period_s']['i:D1']/first['period_s']['i:D1'];assert abs(ratio-2)<.06;r['capacitance_period_ratio']=ratio;mark('paused-20-verified.json');measure('20',paused,targets)
  wait('allow-save-copy');c.ask('project.save',dict(path=str(out/'Native timed flasher.elab')),True);ref.ask('project.save',dict(path=str(out/'Agent timed flasher.elab')),True)
  a=json.loads((out/'Native timed flasher.elab').read_text());b=json.loads((out/'Agent timed flasher.elab').read_text())
  for key in a:
   if key!='view':assert a[key]==b[key],key
  r['saved_fields_except_view_match']=True;r['passed']=True;save();mark('all-verified.json');wait('finish');c.ask('app.close',{},True);c.close();c=None;r['host_exit_code']=host.wait(timeout=15);r['normal_closure']=r['host_exit_code']==0
 except BaseException as error:r['error']=type(error).__name__+': '+str(error);raise
 finally:
  if ref:ref.close()
  if c:c.close()
  if host and host.poll() is None:
   host.terminate()
   try:host.wait(timeout=10)
   except subprocess.TimeoutExpired:host.kill();host.wait(timeout=5)
  if host:r['final_host_exit_code']=host.returncode
  r['ended_utc']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
