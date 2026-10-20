"""Read-only native admission replay. Requires Pillow and retained local archives."""
from pathlib import Path
from importlib.machinery import SourceFileLoader
import gzip,hashlib,importlib.util,io,json,sys,tarfile
if sys.flags.optimize:raise SystemExit('Verification requires assertions; optimized Python is refused')
sys.dont_write_bytecode=True
from PIL import Image
HERE=Path(__file__).resolve().parent;ROOT=HERE.parents[2]
sha=lambda data:hashlib.sha256(data).hexdigest()
def raw(name):return gzip.decompress((HERE/'session'/(name+'.gz')).read_bytes())
def read(name):return json.loads(raw(name))
def main():
 cp=json.loads((HERE/'checkpoint.json').read_text());assert set(cp['files'])=={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p!=HERE/'checkpoint.json'}
 for n,h in cp['files'].items():
  p=HERE/n;assert not p.is_symlink() and p.resolve().is_relative_to(HERE) and sha(p.read_bytes())==h,n
 for n,v in json.loads((HERE/'manifest.json').read_text()).items():
  data=(HERE/n).read_bytes();decoded=gzip.decompress(data);assert sha(data)==v['stored_sha256'] and sha(decoded)==v['original_sha256'] and len(decoded)==v['original_bytes']
 report=json.loads((HERE/'report.json').read_text());plan=json.loads((HERE/'declaration.json').read_text());analysis=json.loads((HERE/'analysis.json').read_text());assert report['passed'] and analysis['passed']
 assert report['cohort_trials']==report['eligible_latency_observations']==analysis['eligible_latency_observations']==0 and report['timing_percentile'] is None
 assert not report['whole_recording_visibility_qualified'] and not report['performance_gate_passed'] and not report['owner_acceptance'] and not report['full_mvp_complete']
 assert report['tested_version']=='0.156.0' and report['tested_build']=='173' and report['current_release']=='0.168.0 /185'
 current=ROOT/'research/ui-audit/first-version/inspector-width-review-checkpoint.json';assert sha(current.read_bytes())==report['current_receipt_sha256'];receipt=json.loads(current.read_text())
 assert report['current_go_sources']==receipt['quality']['source_sha256'] and len(report['current_go_sources'])==659
 for n,h in report['current_go_sources'].items():assert sha((ROOT/n).read_bytes())==h,n
 original=json.loads((ROOT/'research/ui-audit/first-version/sweep-handoff-review-checkpoint.json').read_text());assert original['native_sha256']==report['tested_native_sha256']==plan['native_sha256'] and original['agent_sha256']==report['tested_agent_sha256']==plan['agent_sha256']
 for binary in ('/private/tmp/elab-first-version-173-signed-runtime/ELab.app/Contents/MacOS/ELab','/private/tmp/elab-first-version-173-signed-runtime/elab-agent'):assert sha(Path(binary).read_bytes())==plan['pinned_files'][binary]
 assert sha((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes())==receipt['canonical']['native_sha256']
 for n in ['prepare.py','controller.mjs','profiles/rc.json']:
  canonical='tools/baseline-admission/'+n;saved=HERE/'dependencies'/(Path(n).name+'.txt');assert sha(saved.read_bytes())==plan['pinned_files'][str(ROOT/canonical)]
 for n in ('session.py','worker.py','controller.js','reference.py','launcher.swift'):
  assert sha(gzip.decompress((HERE/'producers'/(n+'.gz')).read_bytes()))==plan['pinned_files'][str(ROOT/'build/native-baseline-admission-173-work'/n)]
 assert sha((HERE/'reference.py.txt').read_bytes())==plan['pinned_files'][str(ROOT/'build/native-baseline-admission-173-work/reference.py')]
 spec=importlib.util.spec_from_loader('frozen_independent_rc',SourceFileLoader('frozen_independent_rc',str(HERE/'reference.py.txt')));ref=importlib.util.module_from_spec(spec);spec.loader.exec_module(ref)
 stores=json.loads((HERE/'image-archives.json').read_text());images={}
 for store in stores:
  p=ROOT/store['path'];assert p.exists(),'Restore missing local archive; do not recreate trials: '+str(p)
  assert p.stat().st_size==store['bytes'] and sha(p.read_bytes())==store['sha256'] and p.stat().st_mode&0o777==0o600 and p.parent.stat().st_mode&0o777==0o700
  decoded={}
  with tarfile.open(p,'r:xz') as tar:
   members=tar.getmembers();assert len(members)==len(store['files']) and len(members)==len({m.name for m in members})
   for m in members:
    assert m.isfile() and '/' not in m.name and m.name in store['files'] and m.size<=64<<20 and m.mode==0o600
    data=tar.extractfile(m).read();v=store['files'][m.name];assert len(data)==m.size==v['bytes'] and sha(data)==v['sha256'];decoded[m.name]=data
  images[store['case']]=decoded
 profile=json.loads((HERE/'dependencies/rc.json.txt').read_text());results=[]
 for case in plan['cases']:
  prefix='trials/'+case['id']+'/';read_case=lambda n:read(prefix+n);before=read_case('before-apply.json');after=read_case('after-apply.json');admission=read_case('admission.json');action=read_case('capture-action.json');bindings=read_case('independent-bindings.json');outcome=read_case('outcome.json');capture_report=read_case('capture/report.json')
  assert outcome['status']=='native_guard_passed' and outcome['recorder_exit_code']==0 and action['case_passed'] and admission['admitted']==case['expected_admitted']
  assert action['marker_calls']==action['apply_calls']==int(case['expected_admitted']) and action['input_call_completed']==case['expected_admitted']
  for key in ('trial_id','attempt_id','frame_index','png','png_sha256','tool_sha256','profile_sha256'):assert admission[key]==bindings[key]
  assert action['attempt_id']==read_case('accepted-handshake.json')['attempt_id']==admission['attempt_id']
  rows=[json.loads(line) for line in raw(prefix+'capture/frames.jsonl').splitlines()];frames=[r for r in rows if r.get('png')];markers=[r for r in rows if r['event']=='marker'];first=frames[0]
  assert len(frames)==capture_report['complete_frames'] and len(markers)==action['marker_calls']==capture_report['marker_count']
  assert first['index']==admission['frame_index'] and first['png']==admission['png'] and first['png_sha256']==admission['png_sha256']
  for key in ('frame_geometry_validated','target_geometry_matches_at_receipt','presentation_matches_converted_display_time'):assert first[key] is True
  pngs=images[case['id']];assert len(pngs)==capture_report['unique_images'] and sum(map(len,pngs.values()))==capture_report['image_bytes']<=64<<20
  for frame in frames:assert frame['png']=='frame-'+frame['png_sha256']+'.png' and sha(pngs[frame['png']])==frame['png_sha256']
  rgba=Image.open(io.BytesIO(pngs[first['png']])).convert('RGBA');assert rgba.size==(profile['width'],profile['height']);regions=[]
  for region in profile['regions']:
   crop=rgba.crop(region['rect']);assert crop.getchannel('A').getextrema()==(255,255);digest=sha(crop.convert('RGB').tobytes());regions.append(dict(name=region['name'],rgb_sha256=digest,matches=digest==region['rgb_sha256']))
  assert regions==admission['result']['regions'] and all(v['matches'] for v in regions)==case['expected_admitted']
  if case['expected_admitted']:
   assert markers[0]==action['marker_acknowledgment'] and markers[0]['frame_count_before']>=first['index'] and int(markers[0]['mach_ticks'])>=int(first['received_mach_ticks'])
   assert [e['name'] for e in action['events']]==['capture_start_requested','baseline_image_received','marker_requested','complete_acknowledgment_validated','input_call_will_invoke','input_call_completed','after_input_ax_observed']
  else:
   assert [v['name'] for v in regions if not v['matches']]==['accepted-capacitor'] and action['refusal']=='Error: Baseline was not admitted' and action['stale_receipt_refusal']=='Error: Admission binding differs: trial_id'
   assert before['components']==after['components'] and before['undo_entry']==after['undo_entry']
  delta=int(case['expected_admitted']);assert after['electrical_revision']==before['electrical_revision']+delta and after['run_id']==before['run_id']+delta and before['wires']==after['wires']
  parts=lambda s:{p['id']:p for p in s['components']};a,b=parts(before),parts(after);assert a['Storage']['value']==case['public_before_f'] and b['Storage']['value']==2e-6
  for name in a:
   x,y=dict(a[name]),dict(b[name])
   if name=='Storage':x.pop('value');y.pop('value')
   assert x==y
  paired=read_case('agent-matched-time.json');saved=read_case('independent.json');assert ref.independent(after)==saved['native'] and ref.independent(paired)==saved['agent']
  for key in ('time_s','input_v','voltage_v','current_a','energy_j'):assert abs(after['transient']['sample'][key]-paired['transient']['sample'][key])<1e-10
  assert not after['transient']['stale'] and capture_report['eligible_latency_observations']==0 and capture_report['collector_error'] is None and capture_report['geometry_rechecked_after_stop']
  baseline=raw(prefix+'baseline.elab');assert json.loads(baseline)['graph']['components']==before['components']
  results.append(dict(case=case['id'],frames=len(frames),images=len(pngs),regions=regions))
 assert sum(v['frames'] for v in results)==328 and sum(v['images'] for v in results)==326
 closed=read('report.json');assert closed['owned_process_absent'] and closed['endpoint_removed'] and not closed['forced_cleanup'] and closed['open_exit_code']==0 and closed['native_exit_status'] is None
 for k in ('shared_geometry_unchanged','review_unchanged','copied_bundle_unchanged','canonical_unchanged','baseline_file_unchanged'):assert closed[k]
 terminal=json.loads((HERE/'terminal-results.json').read_text());assert terminal['session_exit_code']==terminal['worker_exit_code']==0 and terminal['native_exit_status'] is None
 shutdown=read('close-response.json')['shutdown'];assert shutdown['status']=='completed' and shutdown['workers_joined'] and shutdown['exit_ready']
 assert 'tests 28' in gzip.decompress((HERE/'producers/controller-tests.txt.gz').read_bytes()).decode() and 'Ran 6 tests' in gzip.decompress((HERE/'producers/prepare-tests.txt.gz').read_bytes()).decode()
 print(json.dumps(dict(passed=True,native_guard_cases=2,marker_calls=1,apply_calls=1,refused_marker_and_apply_calls=0,complete_frames=328,unique_recorder_images=326,go_pillow_region_checks=8,application_go_sources_unchanged=659,eligible_latency_observations=0,timing_percentile=None)))
if __name__=='__main__':main()
