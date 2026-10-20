"""Read-only source-mode and immutable-capture replay; starts no application."""
from pathlib import Path
from importlib.machinery import SourceFileLoader
import gzip,hashlib,importlib.util,json,math,sys
if sys.flags.optimize:raise SystemExit('Verification requires assertions; optimized Python is refused')
HERE=Path(__file__).resolve().parent;ROOT=HERE.parents[2]
def sha(data):return hashlib.sha256(data).hexdigest()
def raw(name):return gzip.decompress((HERE/'raw'/(name+'.gz')).read_bytes())
def read(name):return json.loads(raw(name))
def module(name,path):
 spec=importlib.util.spec_from_loader(name,SourceFileLoader(name,str(path)));m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m

def main():
 cp=json.loads((HERE/'checkpoint.json').read_text())['files']
 assert set(cp)=={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name!='checkpoint.json'}
 for name,digest in cp.items():
  p=HERE/name;assert not p.is_symlink() and sha(p.read_bytes())==digest,name
 for name,entry in json.loads((HERE/'manifest.json').read_text()).items():
  stored=(HERE/name).read_bytes();data=gzip.decompress(stored)
  assert sha(stored)==entry['stored_sha256'] and sha(data)==entry['original_sha256'] and len(data)==entry['original_bytes']
 report=json.loads((HERE/'report.json').read_text());receipt_path=ROOT/'research/ui-audit/first-version/recorded-dc-review-checkpoint.json';assert sha(receipt_path.read_bytes())==report['receipt_sha256'];receipt=json.loads(receipt_path.read_text())
 assert receipt['source_commit']==report['source_commit'] and receipt['agent_sha256']==report['agent_sha256']
 assert sha((ROOT/'docs/validation/junction-boundary-review-179/producers/collect.py.txt').read_bytes())==report['helper_sha256']
 release=ROOT/'docs/validation/recorded-dc-review-183';assert sha((release/'checkpoint.json').read_bytes())==report['release_checkpoint_sha256']
 for name,digest in json.loads((release/'checkpoint.json').read_text())['evidence_sha256'].items():assert sha((release/name).read_bytes())==digest,name
 assert len(receipt['quality']['source_sha256'])==658 and receipt['quality']['passed']
 for name,digest in receipt['quality']['source_sha256'].items():assert sha((ROOT/name).read_bytes())==digest,name
 sys.path.insert(0,str(ROOT/'scripts'));from package_review import members
 folder=Path(receipt['review_path']);actual=members(folder);actual.pop('RELEASE_MANIFEST.json')
 assert actual==json.loads((folder/'RELEASE_MANIFEST.json').read_text())['files'] and len(actual)==621
 assert sha((folder/'RELEASE_MANIFEST.json').read_bytes())==report['release_manifest_sha256']
 assert sha(Path('/private/tmp/elab-first-version-183-signed-runtime/elab-agent').read_bytes())==receipt['agent_sha256']
 assert sha((ROOT/receipt['sdk']['archive']).read_bytes())==receipt['sdk']['sha256']
 assert sha((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes())==receipt['canonical']['native_sha256']
 reference=module('mode_reference',HERE/'producers/reference.py.txt');sys.modules['reference']=reference
 analyzer=module('mode_analysis',HERE/'producers/analyze.py.txt')
 final=read('final/report.json');summary=read('final/analysis.json');plan=read('final/plan.json')
 assert final['passed'] and final['agent_unchanged'] and summary['passed'] and len(final['results'])==len(plan['cases'])==12
 assert plan['receipt_sha256']==report['receipt_sha256'] and plan['agent_sha256']==report['agent_sha256']
 assert plan['bounds']==dict(voltage_v=1e-9,current_a=1e-9,live_record_value_abs=1e-10,live_record_value_rel=1e-9,charge_continuity_v=1e-9)
 totals=dict(rows=0,matched=0,laws=0,continuity=0,finals=0,kcl_nodes=0)
 for result,case,analysis in zip(final['results'],plan['cases'],summary['cases']):
  assert result['passed'] and result['case']==case
  label=case['name']+'-'+case['mode'];prefix='final/'+label+'/'
  assert read(prefix+'result.json')==result and analysis['label']==label
  a,b=read(prefix+'first.json'),read(prefix+'second.json');assert a['rows']==b['rows'] and a['metadata']['statistics']==b['metadata']['statistics']
  assert reference.check_rows(case['name'],case['mode'],a)==result['laws']
  assert raw(prefix+'first.json')==raw(prefix+'first-after-restart.json')
  for data,filename in [(a,'first.json'),(b,'second.json')]:
   origin=data['metadata']['operation_id'];assert raw(prefix+'reopened-'+origin+'.json')==raw(prefix+filename)
  baseline,live=read(prefix+'baseline.json'),read(prefix+'live.json');before,pending=read(prefix+'pending-before.json'),read(prefix+'pending.json');changed,restarted,reopened=read(prefix+'changed.json'),read(prefix+'restarted.json'),read(prefix+'reopened.json')
  for number in ('first','second'):
   left,right=read(prefix+number+'-source-before.json'),read(prefix+number+'-source-after.json')
   for key in ('components','wires','drawing','transient','mixed','npn','result','scope','paused','run_id','electrical_revision','actuation_revision','undo_entry','redo_entry','transient_settings'):assert left.get(key)==right.get(key),(label,number,key)
  matched=0
  for i,trace in enumerate(live['scope']['traces']):
   assert trace['probe']==a['metadata']['probes'][i]
   assert [(point['time_s'],point['value']) for point in trace['points']]==[(row['time_s'],row['values'][i]) for row in a['rows']],(label,trace['probe']['id'])
   for point in trace['points']:
    assert any(row['time_s']==point['time_s'] and math.isclose(row['values'][i],point['value'],abs_tol=1e-10,rel_tol=1e-9) for row in a['rows']);matched+=1
  assert matched==result['live_record_values_matched']
  other=('discharge' if case['mode']!='discharge' else 'charge') if case['name']=='rc' else ('low' if case['mode']!='low' else 'high')
  assert pending['transient']['pending_input_mode']==other
  for key in ('sample','samples','input_mode'):assert pending['transient'][key]==before['transient'][key]
  assert math.isclose(changed['transient']['sample']['time_s'],.101,abs_tol=1e-12)
  assert restarted['run_id']==changed['run_id']+1 and restarted['transient']['sample']['time_s']==0
  for state in (changed,restarted):assert state['transient']['input_mode']==other and abs(state['transient']['sample']['input_v']-(0. if other in ('low','discharge') else 5.))<1e-9
  continuity=0
  for i,probe in enumerate(a['metadata']['probes']):
   if probe['id'].startswith('v:C'):
    points=[p for p in changed['scope']['traces'][i]['points'] if p['time_s']==.1];last=live['scope']['traces'][i]['points'][-1]['value']
    assert len(points)>=2 and all(abs(p['value']-last)<1e-9 for p in points);continuity+=1
  assert continuity==result['charge_continuity_channels']
  for key in ('components','wires','drawing','electrical_revision','undo_entry','redo_entry','transient_settings'):assert all(s.get(key)==baseline.get(key) for s in (before,pending,changed,restarted)),(label,key)
  kind='npn' if case['name']=='blink' else 'mixed'
  if kind in a['metadata']:
   strip=lambda value:{k:v for k,v in value.items() if k!='run_id'}
   assert strip(live[kind])==strip(a['metadata'][kind]);totals['finals']+=1
   if kind=='mixed':
    kcl=analyzer.graph_kcl(a['metadata']);assert kcl==analysis['independent_final_kcl'];totals['kcl_nodes']+=kcl['nodes']
  assert reopened['paused'] and not reopened['document']['dirty'] and reopened['document']['recording_count']==2 and reopened['transient']['input_mode']=='auto' and reopened['transient']['sample']['time_s']==0
  assert reopened['components']==baseline['components'] and reopened['wires']==baseline['wires'] and reopened['transient_settings']==baseline['transient_settings']
  for cap in (p for p in a['metadata']['graph']['components'] if p['kind']=='capacitor'):
   wanted=cap.get('initial_voltage_v',case['initial'])
   for state in (baseline,restarted,reopened):
    got=state['transient']['sample']['voltage_v'] if case['name']=='rc' else next(p for p in state['mixed']['capacitors'] if p['id']==cap['id'])['voltage_v']
    assert abs(got-wanted)<1e-9
  assert sha(raw(prefix+'project.elab'))==result['saved_sha256']
  totals['rows']+=len(a['rows']);totals['matched']+=matched;totals['laws']+=result['laws']['independent_law_checks'];totals['continuity']+=continuity
 assert totals==dict(rows=1224,matched=6423,laws=2456,continuity=12,finals=9,kcl_nodes=33)
 assert [summary[k] for k in ('recorded_rows','live_record_values_matched','independent_row_law_checks','charge_continuity_channels','complete_final_states_matched','independent_final_kcl_nodes')]==list(totals.values())
 processes=requests=0
 for label in ('discovery','initial','final'):
  result=read(label+'/report.json');exits=[result['process']] if label=='discovery' else result['process_exits']
  if label=='initial':assert not result['passed'] and len(result['results'])==3 and "'kcl_a': 0.006582940296305045" in result['failure']
  for outcome in exits:
   assert outcome['exit_code']==0
   transcript=[json.loads(line) for line in raw(label+'/'+outcome['name']+'-protocol.jsonl').splitlines()];req=[r['request'] for r in transcript if 'request' in r];res=[r['response'] for r in transcript if 'response' in r]
   assert len(req)==len(res)==outcome['requests']
   for i,response in enumerate(res):assert req[i]['id']==response['id']==f'junction-{i+1}' and 'error' not in response
   close_results=[res[i]['result'] for i,r in enumerate(req) if r['command']=='app.close'];assert close_results and close_results[-1]['closed']
   terminal=res[-1]['result'] if req[-1]['command']=='app.close.inspect' else res[-1]['result']['shutdown']
   assert terminal==outcome['shutdown'] and outcome['shutdown']['status']=='completed'
   assert all(outcome['shutdown'][k] for k in ('workers_joined','recovery_finalized','exit_ready'))
   assert not raw(label+'/'+outcome['name']+'-stderr.txt')
   processes+=1;requests+=len(req)
 assert processes==report['all_declared_normal_processes']==32 and requests==report['all_declared_requests']==678
 assert 'KeyError:' in raw('stage/analysis-initial-field-error.txt').decode()
 assert not any(report[k] for k in ('new_product_defect','source_changed','new_release','native_input','performance_qualified','owner_acceptance','full_mvp_complete'))
 print(json.dumps(dict(cases=12,recorded_rows=1224,live_values=6423,row_laws=2456,final_kcl_nodes=33,normal_processes=32,requests=678,new_product_defect=False)))

if __name__=='__main__':main()
