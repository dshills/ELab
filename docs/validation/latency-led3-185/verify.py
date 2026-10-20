"""Read-only whole LED3 replay; no app, recording, input or archive extraction."""
from pathlib import Path
from fractions import Fraction
import gzip,hashlib,io,json,math,subprocess,sys,tarfile
if sys.flags.optimize:raise SystemExit('Verification requires assertions; optimized Python is refused')
sys.dont_write_bytecode=True
from PIL import Image
HERE=Path(__file__).resolve().parent;ROOT=HERE.parents[2]
h=lambda raw:hashlib.sha256(raw).hexdigest()
def raw(name):return gzip.decompress((HERE/'session'/(name+'.gz')).read_bytes())
def read(name):return json.loads(raw(name))
def main():
 cp=json.loads((HERE/'checkpoint.json').read_text());actual={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p!=HERE/'checkpoint.json'};assert actual==set(cp['files'])
 for n,digest in cp['files'].items():
  p=HERE/n;assert not p.is_symlink() and p.resolve().is_relative_to(HERE) and h(p.read_bytes())==digest,n
 for n,v in json.loads((HERE/'manifest.json').read_text()).items():
  data=(HERE/n).read_bytes();decoded=gzip.decompress(data);assert h(data)==v['stored_sha256'] and h(decoded)==v['original_sha256'] and len(decoded)==v['original_bytes']
 artifacts=json.loads((HERE/'artifacts.json').read_text());plan=json.loads((HERE/'declaration.json').read_text());summary=json.loads((HERE/'summary.json').read_text());ledger=read('cohort-result.json')
 planpath=ROOT/'research/49-current-led-timing-plan.json';assert planpath.read_bytes()==(HERE/'declaration.json').read_bytes() and h(planpath.read_bytes())==artifacts['plan_sha256']
 assert plan['declared_before_collection'] and plan['trials_run']==0 and plan['release']=='0.168.0 /185' and plan['cohort_id']==summary['cohort_id']=='led3-current-185'
 assert [t['id'] for t in plan['trials']]==[f'led3-{i:02}' for i in range(1,21)]
 assert ledger['scheduled']==ledger['attempted']==len(ledger['ledger'])==20 and ledger['halted_at'] is None and ledger['replacements']==0
 assert [v['id'] for v in ledger['ledger']]==[t['id'] for t in plan['trials']] and all(v['status']=='available' for v in ledger['ledger'])
 receiptpath=ROOT/'research/ui-audit/first-version/inspector-width-review-checkpoint.json';assert h(receiptpath.read_bytes())==artifacts['current_receipt_sha256'];receipt=json.loads(receiptpath.read_text())
 assert receipt['native_sha256']==summary['native_sha256']==plan['native_sha256'] and receipt['agent_sha256']==summary['agent_sha256']==plan['agent_sha256']
 assert artifacts['current_go_sources']==plan['current_go_sources']==receipt['quality']['source_sha256'] and len(plan['current_go_sources'])==659
 for n,digest in plan['current_go_sources'].items():assert h((ROOT/n).read_bytes())==digest,n
 for n,digest in plan['pinned_files'].items():assert h(Path(n).read_bytes())==digest,n
 assert h((ROOT/'third_party/vellum/sdk.lock.json').read_bytes())==artifacts['sdk_lock_sha256']
 assert h((ROOT/'docs/validation/native-baseline-admission-173/checkpoint.json').read_bytes())==artifacts['prior_native_admission_checkpoint_sha256']
 for n,digest in artifacts['legacy_method_dependencies'].items():assert h((ROOT/n).read_bytes())==digest,n
 # Exact adaptation audit: no pixel, clock, geometry, state, physics or
 # corruption predicate changes from the previously qualified LED analyzer.
 old=gzip.decompress((HERE/'producers/analyze-original-173.py.txt.gz').read_bytes()).decode()
 expected=old.replace('proof=Path(__file__).resolve().parent',"proof=Path('/Users/dshills/Development/projects/ELab/docs/validation/latency-followup-173/led')").replace('sys.path.insert(0,str(proof.parent))','sys.path.insert(0,str(Path(__file__).resolve().parent))').replace('ddd0e0cfdfad56c25c9d2d4d0882a1bb79095197101c9e79a844626bfaaa7a1a',plan['native_sha256']).replace('folder=Path(sys.argv[1])',"if sys.flags.optimize:raise SystemExit('Verification requires assertions')\nfolder=Path(sys.argv[1])")
 assert (HERE/'analyze.py').read_text()==expected
 original_store=gzip.decompress((HERE/'producers/frame-store-original-173.py.txt.gz').read_bytes()).decode()
 expected_store=original_store.replace('ROOT = Path(__file__).resolve().parents[3]',"ROOT = Path('/Users/dshills/Development/projects/ELab')").replace("ARCHIVE_ROOT = ROOT / 'build/validation-evidence/latency-followup-173'","ARCHIVE_ROOT = ROOT / 'build/validation-evidence/latency-led3-185'")
 assert (HERE/'frame_store.py').read_text()==expected_store
 assert h((HERE/'reference.py.txt').read_bytes())==plan['pinned_files'][str(ROOT/'build/latency-led3-185-work/reference.py')]
 from importlib.machinery import SourceFileLoader
 import importlib.util
 spec=importlib.util.spec_from_loader('frozen_led_law',SourceFileLoader('frozen_led_law',str(HERE/'reference.py.txt')));ref=importlib.util.module_from_spec(spec);spec.loader.exec_module(ref)
 stores=json.loads((HERE/'image-stores.json').read_text());assert [v['id'] for v in stores['stores']]==[t['id'] for t in plan['trials']]
 profile=json.loads((HERE/'dependencies/led.json.txt').read_text());assert h((HERE/'dependencies/led.json.txt').read_bytes())==plan['pinned_files'][str(ROOT/'tools/baseline-admission/profiles/led.json')]
 fresh_bounds=[];ack_bounds=[];frames_total=images_total=bytes_total=0
 for entry,store,expected_case in zip(ledger['ledger'],stores['stores'],summary['trial_results']):
  ident=entry['id'];folder=HERE/'session/trials'/ident;prefix='trials/'+ident+'/';a=read(prefix+'analysis.json');action=read(prefix+'capture-action.json');admission=read(prefix+'admission.json');bindings=read(prefix+'independent-bindings.json');capture_report=read(prefix+'capture/report.json');rows=[json.loads(s) for s in raw(prefix+'capture/frames.jsonl').splitlines()];marker=read(prefix+'capture/marker-'+ident.replace('-','_')+'.json');source_store=read(prefix+'capture/frame-store.json');assert {k:v for k,v in store.items() if k!='id'}==source_store
  assert (folder/'capture/frame-store.json').read_bytes()==raw(prefix+'capture/frame-store.json')
  path=ROOT/store['archive_path'];assert path.is_file(),'Restore missing archive; never regenerate a trial: '+str(path)
  assert path.stat().st_size==store['archive_bytes'] and h(path.read_bytes())==store['archive_sha256'] and path.stat().st_mode&0o777==0o600 and path.parent.stat().st_mode&0o777==0o700
  assert admission['admitted'] and action['input_call_completed'] and action['marker_calls']==action['apply_calls']==1 and action['after_ax_has_expected_readout'] and not action.get('error')
  for k in ('trial_id','attempt_id','frame_index','png','png_sha256','tool_sha256','profile_sha256'):assert admission[k]==bindings[k]
  assert admission['attempt_id']==action['attempt_id']==read(prefix+'accepted-handshake.json')['attempt_id'] and admission['trial_id']==ident
  manifest=read(prefix+'capture/image-manifest.json');assert len(manifest)==capture_report['unique_images']==store['member_count']
  with tarfile.open(path,'r:xz') as tar:
   members=tar.getmembers();assert len(members)==len({m.name for m in members})==len(manifest)
   first_data=None
   for m in members:
    assert m.isfile() and m.mode==0o600 and Path(m.name).name==m.name and m.name in manifest and m.size<=64<<20
    data=tar.extractfile(m).read();assert len(data)==m.size==manifest[m.name]['bytes'] and h(data)==manifest[m.name]['sha256']
    if m.name==admission['png']:first_data=data
  assert first_data is not None and h(first_data)==admission['png_sha256']
  rgba=Image.open(io.BytesIO(first_data)).convert('RGBA');assert rgba.size==(profile['width'],profile['height']);regions=[]
  for region in profile['regions']:
   crop=rgba.crop(region['rect']);assert crop.getchannel('A').getextrema()==(255,255);digest=h(crop.convert('RGB').tobytes());regions.append(dict(name=region['name'],rgb_sha256=digest,matches=digest==region['rgb_sha256']))
  assert regions==admission['result']['regions'] and all(v['matches'] for v in regions)
  scale=Fraction(capture_report['timebase_numerator'],capture_report['timebase_denominator']);tick=int(marker['mach_ticks'])
  committed=next(r for r in rows if r.get('event')=='frame' and r['index']==a['first_committed_parameter_frame']);fresh=next(r for r in rows if r.get('event')=='frame' and r['index']==a['first_fresh_frame'])
  ack_bound=Fraction(int(committed['display_mach_ticks'])-tick)*scale/1000000;fresh_bound=Fraction(int(fresh['display_mach_ticks'])-tick)*scale/1000000
  assert float(ack_bound)==a['marker_to_committed_ms']==entry['marker_to_committed_ms']==expected_case['marker_to_committed_ms'] and float(fresh_bound)==a['marker_to_fresh_ms']==entry['marker_to_fresh_ms']==expected_case['marker_to_fresh_ms']
  assert [fresh_bound.numerator,fresh_bound.denominator]==expected_case['fresh_bound_fraction_ms'] and [ack_bound.numerator,ack_bound.denominator]==expected_case['committed_bound_fraction_ms']
  assert len(a['rejected_mutations'])==8 and a['measurement_status']=='available' and capture_report['eligible_latency_observations']==0
  before=read(prefix+'before-apply.json');after=read(prefix+'after-apply.json');paired=read(prefix+'agent-expected.json');laws=read(prefix+'independent.json');ref.independent(before);assert ref.independent(after)==laws['native']==expected_case['native_led_law'] and ref.independent(paired)==laws['agent']==expected_case['agent_led_law'];ref.compare_states(after,paired)
  # The frozen full-recording analyzer redecodes every unique image, validates
  # every frame and all paired/public/independent physics, and repeats its eight
  # corruption controls. --verify writes no new analysis or trial.
  replay=subprocess.run([sys.executable,'-B',str(HERE/'analyze.py'),str(folder),'--verify'],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,timeout=60)
  assert replay.returncode==0,(ident,replay.stdout)
  fresh_bounds.append(fresh_bound);ack_bounds.append(ack_bound);frames_total+=a['complete_frames'];images_total+=a['unique_images'];bytes_total+=capture_report['image_bytes']
  print(ident+' replayed',flush=True)
 rank=math.ceil(Fraction(95,100)*len(fresh_bounds));assert rank==19 and len(fresh_bounds)==20
 p95=sorted(fresh_bounds)[rank-1];ack95=sorted(ack_bounds)[rank-1]
 assert [p95.numerator,p95.denominator]==summary['p95_exact_fraction_ms'] and float(p95)==summary['marker_to_fresh_p95_ms'] and float(ack95)==summary['marker_to_committed_p95_ms']
 assert p95<=500 and ack95>100 and summary['fresh_sample_500ms_supported'] and summary['acknowledgment_100ms_status'].startswith('inconclusive')
 assert frames_total==summary['complete_frames']==3266 and images_total==summary['unique_images']==3127 and bytes_total==summary['image_bytes']==432248941
 assert summary['available']==summary['attempted']==summary['scheduled']==20 and summary['replacements']==summary['unavailable']==summary['not_attempted']==0 and not summary['pooled_with_earlier_cohorts']
 assert summary['raw_eligible_latency_observations']==0 and not any(summary[k] for k in ('overall_prd_performance_pass','owner_acceptance','full_mvp_complete','canonical_promoted'))
 observation=stores['cua_observation'];p=ROOT/observation['archive_path'];assert p.stat().st_size==observation['archive_bytes'] and h(p.read_bytes())==observation['archive_sha256'] and p.stat().st_mode&0o777==0o600
 with tarfile.open(p,'r:xz') as tar:
  members=tar.getmembers();assert len(members)==1 and members[0].isfile() and members[0].name==observation['member'] and members[0].mode==0o600;data=tar.extractfile(members[0]).read();assert len(data)==observation['bytes'] and h(data)==observation['sha256']
 assert sum(v['archive_bytes'] for v in stores['stores'])+observation['archive_bytes']==artifacts['archive_total_bytes']==64174632
 session=read('report.json');assert session['owned_process_absent'] and session['endpoint_removed'] and not session['forced_cleanup'] and session['open_exit_code']==0 and session['native_exit_status'] is None
 for key in ('shared_geometry_unchanged','review_unchanged','copied_bundle_unchanged','canonical_unchanged','baseline_file_unchanged'):assert session[key]
 terminal=json.loads((HERE/'terminal-results.json').read_text());assert terminal['session_wrapper_exit_code']==terminal['worker_wrapper_exit_code']==terminal['open_exit_code']==0 and terminal['native_exact_exit_status'] is None
 first_inspection=read('trials/led3-01/inspection-selection.json');assert first_inspection['original_first_and_fresh_images_visually_inspected'] and first_inspection['inspected_before_trial_02']
 assert h((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes())==receipt['canonical']['native_sha256']
 print(json.dumps(dict(passed=True,available=20,scheduled=20,complete_frames=frames_total,unique_images=images_total,go_pillow_region_checks=80,corruption_exercises=160,p95_ms=float(p95),p95_exact_fraction_ms=[p95.numerator,p95.denominator],fresh_sample_500ms_supported=True,acknowledgment_100ms='inconclusive',raw_flags_unchanged=True,overall_prd_performance_pass=False)))
if __name__=='__main__':main()
