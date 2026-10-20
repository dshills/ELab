"""Read-only native MOSFET evidence replay; no app, settings or UI access."""
from pathlib import Path
from importlib.machinery import SourceFileLoader
import gzip,hashlib,importlib.util,json,math,struct,sys
if sys.flags.optimize:raise SystemExit('Verification requires assertions; optimized Python is refused')
HERE=Path(__file__).resolve().parent;ROOT=HERE.parents[2]
def sha(data):return hashlib.sha256(data).hexdigest()
def raw(name):return gzip.decompress((HERE/'raw'/(name+'.gz')).read_bytes())
def read(name):return json.loads(raw(name))
def module(name,path):
 spec=importlib.util.spec_from_loader(name,SourceFileLoader(name,str(path)));m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m

def equal(a,b,path=''):
 if isinstance(a,(int,float)) and not isinstance(a,bool) and isinstance(b,(int,float)) and not isinstance(b,bool):assert math.isclose(a,b,rel_tol=1e-9,abs_tol=1e-12),(path,a,b)
 elif isinstance(a,dict):
  assert set(a)==set(b),(path,set(a)^set(b))
  for key in a:
   if key not in ('run_id','electrical_revision'):equal(a[key],b[key],path+'.'+key)
 elif isinstance(a,list):
  assert len(a)==len(b),(path,len(a),len(b))
  for i,(x,y) in enumerate(zip(a,b)):equal(x,y,path+f'[{i}]')
 else:assert a==b,(path,a,b)

def main():
 cp=json.loads((HERE/'checkpoint.json').read_text())['files'];assert set(cp)=={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name!='checkpoint.json'}
 for name,digest in cp.items():
  p=HERE/name;assert not p.is_symlink() and sha(p.read_bytes())==digest,name
 for name,entry in json.loads((HERE/'manifest.json').read_text()).items():
  stored=(HERE/name).read_bytes();original=gzip.decompress(stored) if entry['compressed'] else stored
  assert sha(stored)==entry['stored_sha256'] and sha(original)==entry['original_sha256'] and len(original)==entry['original_bytes']
 report=json.loads((HERE/'report.json').read_text());receipt_path=ROOT/'research/ui-audit/first-version/recorded-dc-review-checkpoint.json';assert sha(receipt_path.read_bytes())==report['receipt_sha256'];receipt=json.loads(receipt_path.read_text())
 for name,digest in report['dependencies'].items():assert sha((ROOT/name).read_bytes())==digest,name
 assert len(receipt['quality']['source_sha256'])==658 and receipt['quality']['passed']
 for name,digest in receipt['quality']['source_sha256'].items():assert sha((ROOT/name).read_bytes())==digest,name
 release=ROOT/'docs/validation/recorded-dc-review-183';assert sha((release/'checkpoint.json').read_bytes())==report['release_checkpoint_sha256']
 for name,digest in json.loads((release/'checkpoint.json').read_text())['evidence_sha256'].items():assert sha((release/name).read_bytes())==digest
 sys.path.insert(0,str(ROOT/'scripts'));from package_review import members
 folder=Path(receipt['review_path']);actual=members(folder);actual.pop('RELEASE_MANIFEST.json');assert actual==json.loads((folder/'RELEASE_MANIFEST.json').read_text())['files'] and len(actual)==621
 assert sha((folder/'RELEASE_MANIFEST.json').read_bytes())==report['release_manifest_sha256']
 assert sha(Path('/private/tmp/elab-first-version-183-signed-runtime/elab-agent').read_bytes())==receipt['agent_sha256']
 assert sha((ROOT/receipt['sdk']['archive']).read_bytes())==receipt['sdk']['sha256']
 assert sha((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes())==receipt['canonical']['native_sha256']
 reference=module('mosfet_reference',HERE/'producers/reference.py.txt')
 final=read('reference/report.json');assert final['passed'] and len(final['results'])==10 and len(final['fresh_reopens'])==3
 for result in final['results']:
  label=result['label'];native=read('native/'+label+'.json');saved=read('reference/'+label+'.json');expected=saved['expected'];assert saved['result']==result
  assert native['paused'] and native['last_actor']=='user' and not native['mixed']['stale'] and not native['transient']['stale']
  for key in ('components','wires','nets','transient_settings','selected','paused'):assert native[key]==expected[key],(label,key)
  for key in ('mixed','transient','scope'):equal(native[key],expected[key],label+'.'+key)
  assert reference.laws(native)==result['laws'] and sha(raw('native/'+label+'.json'))==result['native_state_sha256']
  if label.endswith(('-low','-high')):assert native['step_work']['actor']=='user' and native['step_work']['status']=='completed'
  if label=='pmos-low':
   m=native['mixed']['mosfets'][0];assert m['vgs_v']<0 and m['vds_v']<0 and m['drain_current_a']<0 and m['power_w']>0 and native['mixed']['leds'][0]['current_a']>0
  if label=='cmos-midpoint':assert all(abs(m['drain_current_a'])>.05 and m['power_w']>0 for m in native['mixed']['mosfets'])
 for name in ('nmos','pmos','cmos'):
  reopened=read('reference/'+name+'-fresh.json');assert reopened['source_unchanged']
  filename=name+'-midpoint.elab' if name=='cmos' else name+'-high.elab';document=read('native/'+filename);state=reopened['state'];assert sha(raw('native/'+filename))==reopened['source_sha256']
  assert state['paused'] and not state['document']['dirty'] and state['transient']['sample']['time_s']==0 and state['components']==document['graph']['components'] and state['wires']==document['graph']['wires']
  assert reference.laws(state)==reopened['laws']
  saved=read('native/'+filename.removesuffix('.elab')+'-saved.json');assert not saved['document']['dirty']
 before,private,applied,midpoint=(read('native/'+name+'.json') for name in ('cmos-edit-before','cmos-edit-private','cmos-edit-applied','cmos-midpoint'))
 assert before==private and before['paused'] and applied['paused'] and applied['mixed']['stale'] and applied['transient']['stale']
 assert next(p for p in before['components'] if p['id']=='VIN')['value']==5 and next(p for p in applied['components'] if p['id']=='VIN')['value']==2.5
 assert before['transient']['sample']==applied['transient']['sample']
 assert {k:v for k,v in before['mixed'].items() if k!='stale'}=={k:v for k,v in applied['mixed'].items() if k!='stale'}
 assert applied['undo_entry']['actor']=='user' and applied['electrical_revision']==before['electrical_revision']+1
 assert midpoint['transient']['sample']['time_s']==0 and not midpoint['mixed']['stale']
 unexpected=read('native/cmos-selector-unexpected-running.json');assert not unexpected['paused'] and unexpected['selected']=='VIN' and unexpected['transient']['sample']['time_s']>0
 old=read('native/cmos-high.json');assert old['paused'] and old['transient']['sample']['time_s']==.002 and old['components']==unexpected['components'] and old['wires']==unexpected['wires']
 for suffix in ('open','highlight'):
  ax=raw('native/cmos-selector-'+suffix+'.ax.txt').decode();assert 'Ⅱ Paused' in ax and 'sample 2 ms' in ax
 direct_before,direct_after=read('native/menu-focus-before.json'),read('native/menu-focus-after.json')
 assert direct_before['paused'] and direct_after['paused'] and direct_before['selected']=='VIN' and direct_after['selected']=='MN'
 for key in ('components','wires','mixed','transient','scope','run_id','electrical_revision','undo_entry','redo_entry'):assert direct_before.get(key)==direct_after.get(key)
 assert '(selected) MN' in raw('native/menu-focus-keyboard-choice.ax.txt').decode()
 controller=read('native/controller.json');assert controller['same_pid_after_exec'] and controller['owned_pid']==42824 and controller['owned_process_absent'] and controller['endpoint_removed'] and not controller['forced_cleanup'] and controller['open_exit_code']==0 and controller['native_exit_status'] is None
 assert all(controller[k] for k in ('shared_geometry_unchanged','review_unchanged','copied_app_unchanged','canonical_unchanged')) and controller['shared_geometry_before']==controller['shared_geometry_after']
 assert controller['shutdown']['status']=='completed' and all(controller['shutdown'][k] for k in ('workers_joined','recovery_finalized','exit_ready')) and controller['eligible_latency_observations']==0
 assert read('native/close-response.json')['closed']
 for name,size in report['images'].items():
  data=(HERE/'raw/native'/name).read_bytes();assert data[:2]==b'\xff\xd8';offset=2;observed=None
  while offset<len(data):
   assert data[offset]==255
   while data[offset]==255:offset+=1
   marker=data[offset];offset+=1
   if marker in (1,216,217) or 208<=marker<=215:continue
   length=struct.unpack('>H',data[offset:offset+2])[0];assert length>=2
   if marker in (192,193,194,195,197,198,199,201,202,203,205,206,207):
    h,w=struct.unpack('>HH',data[offset+3:offset+7]);observed=dict(width=w,height=h,media_type='image/jpeg');break
   offset+=length
  assert observed==size
 assert (report['images']['nmos-initial.jpg']['width'],report['images']['nmos-initial.jpg']['height'])==(1053,802)
 assert (report['images']['zoomed.jpg']['width'],report['images']['zoomed.jpg']['height'])==(5054,1410)
 requests=0;processes=0
 for label in ('reference-initial','reference'):
  result=read(label+'/report.json')
  if label=='reference-initial':assert not result['passed'] and "KeyError: 'led_model'" in result['failure']
  for outcome in result['process_exits']:
   rows=[json.loads(line) for line in raw(label+'/'+outcome['name']+'-protocol.jsonl').splitlines()];req=[r['request'] for r in rows if 'request' in r];res=[r['response'] for r in rows if 'response' in r]
   assert len(req)==len(res)==outcome['requests'] and outcome['exit_code']==0
   for i,r in enumerate(res):assert r['id']==req[i]['id']==f'junction-{i+1}' and 'error' not in r
   work=res[-1]['result'] if req[-1]['command']=='app.close.inspect' else res[-1]['result']['shutdown'];assert work==outcome['shutdown'] and work['status']=='completed' and all(work[k] for k in ('workers_joined','recovery_finalized','exit_ready'))
   assert not raw(label+'/'+outcome['name']+'-stderr.txt');processes+=1;requests+=len(req)
 assert processes==7 and requests==63
 observers=observer_requests=0
 for name in json.loads((HERE/'manifest.json').read_text()):
  if name.startswith('raw/native/') and name.endswith('-bridge-exit.json.gz'):
   relative=name.removeprefix('raw/').removesuffix('.gz');exit=read(relative);assert exit['exit_code']==0 and exit['closed_connection_only'];stem=relative.removesuffix('-bridge-exit.json')
   rows=[json.loads(line) for line in raw(stem+'-protocol.jsonl').splitlines()];req=[r['request'] for r in rows if 'request' in r];res=[r['response'] for r in rows if 'response' in r]
   assert len(req)==len(res)==exit['requests']
   assert all(r['command'] in ('project.inspect','project.save') for r in req)
   for i,r in enumerate(res):assert r['id']==req[i]['id'] and 'error' not in r
   observers+=1;observer_requests+=len(req)
 assert observers==16 and observer_requests==22
 assert sha((ROOT/'docs/pmos.md').read_bytes())==report['pmos_documentation_sha256'] and 'does not currently display that power value' in (ROOT/'docs/pmos.md').read_text()
 assert 'existing probes show voltage, current, power' in raw('stage/pmos-before.md').decode()
 assert not any(report[k] for k in ('initial_named_pause_established','hybrid_item_return_preserved_pause','new_product_defect_established','source_changed','new_release','owner_acceptance','full_mvp_complete')) and report['direct_pointer_keyboard_menu_preserved_pause']
 print(json.dumps(dict(accepted_native_states=10,fresh_reopens=3,reference_processes=7,observer_connections=16,direct_menu_pause_preserved=True,hybrid_menu_outcome='unqualified',native_exact_exit_status=None,new_source=False)))

if __name__=='__main__':main()
