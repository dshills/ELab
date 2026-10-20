"""Read-only replay: no app launch, command execution, test run or capture."""
from pathlib import Path
import copy,gzip,hashlib,json,math,plistlib,stat
P=Path(__file__).resolve().parent;R=P.parents[2]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
raw=lambda n:gzip.decompress((P/'raw'/n).read_bytes())
read=lambda n:json.loads(raw(n))
cp=json.loads((P/'checkpoint.json').read_text());actual={str(p.relative_to(P)) for p in P.rglob('*') if p.is_file() and p.name!='checkpoint.json' and '__pycache__' not in p.parts}
assert actual==set(cp['files']) and all(sha(P/n)==h for n,h in cp['files'].items())
for n,v in json.loads((P/'manifest.json').read_text())['files'].items():
 data=gzip.decompress((P/n).read_bytes());assert len(data)==v['bytes'] and hashlib.sha256(data).hexdigest()==v['sha256']
r=json.loads((P/'report.json').read_text());b=read('build/build-result-signed.json.gz');c=read('native/controller.json.gz');state=read('producers/candidate-state.json.gz')
assert r['controller']==c and r['candidate']==state and r['reference']==read('reference/report.json.gz')
assert r['build_and_strict_signature_passed'] and b['passed'] and b['strict_bundle_signature_valid'] and b['clean_build_vcs']
assert r['source_commit']==b['source_commit']=='b49fd7ad4ef67b9cbb5002a47dd281e5f5287e2e'
assert (b['version'],b['build'])==('0.172.0','189')
assert not any(r[k] for k in ('native_recheck_available','original_native_failure_closed','native_cause_confirmed','qualified_replacement')) and r['planned_transport_cases_executed']==0
for n in ['elab','elab-agent','elab-conformance']:
 info=raw('build/'+n+'-build-info.txt.gz').decode();assert 'vcs.revision='+b['source_commit'] in info and 'vcs.modified=false' in info
q=json.loads((R/'docs/validation/recorded-time-first-settlement/report.json').read_text())['quality']
assert q['exit_code']==0 and q['sources_unchanged'] and len(q['source_sha256'])==670 and all(sha(R/n)==h for n,h in q['source_sha256'].items())
for key,path in [('source_proof_checkpoint_sha256','docs/validation/recorded-time-first-settlement/checkpoint.json'),('prior_release_receipt_sha256','research/ui-audit/first-version/recorded-step-review-checkpoint.json'),('canonical_sha256','build/ELab.app/Contents/MacOS/ELab'),('sdk_lock_sha256','third_party/vellum/sdk.lock.json')]:assert sha(R/path)==r[key]
def tree(folder):
 result={}
 for p in [folder,*sorted(folder.rglob('*'))]:
  assert not p.is_symlink() and (p.is_file() or p.is_dir())
  result[str(p.relative_to(folder))]={'mode':stat.S_IMODE(p.stat().st_mode),'sha256':sha(p) if p.is_file() else None}
 return result
candidate=Path(state['preserved_candidate']);assert not Path(state['built_location']).exists()
assert tree(candidate/'ELab.app')==b['source_bundle']
files={str(p.relative_to(candidate)):{'sha256':sha(p),'mode':oct(stat.S_IMODE(p.stat().st_mode))} for p in candidate.rglob('*') if p.is_file()}
assert files==state['candidate_members'] and not state['qualified_release'] and not state['original_native_failure_closed']
app=candidate/'ELab.app';info=plistlib.loads((app/'Contents/Info.plist').read_bytes());assert info['CFBundleShortVersionString']=='0.172.0' and info['CFBundleVersion']=='189' and info['CFBundleExecutable']=='ELab'
assert sha(app/'Contents/MacOS/ELab')==b['native_sha256']==c['native_sha256'] and sha(candidate/'elab-agent')==b['agent_sha256']==c['agent_sha256']
manifest=json.loads((app/'Contents/Resources/APP_MANIFEST.json').read_text());assert manifest['source_binary_sha256']==b['source_native_sha256'] and manifest['private_build'] and manifest['signed_identity']=='-'
assert sha(R/'build/first-version-189-tools/build/elab')==b['source_native_sha256']!=b['native_sha256']
d=state['temporary_runtime_decommission'];assert d['bytes_unchanged'] and d['after_normal_close'] and sha(Path(d['path']))==d['sha256']==b['native_sha256']
assert d['before_mode']=='0o755' and d['after_mode']=='0o644' and stat.S_IMODE(Path(d['path']).stat().st_mode)==0o644
expected=copy.deepcopy(b['source_bundle']);expected['Contents/MacOS/ELab']['mode']=0o644;assert tree(Path(d['path']).parents[2])==expected
assert c['owned_process_absent'] and c['endpoint_removed'] and c['open_exit_code']==0 and c['native_exit_status'] is None and not c['forced_cleanup']
assert c['shutdown']['status']=='completed' and all(c['shutdown'][k] for k in ('workers_joined','recovery_finalized','exit_ready'))
assert all(c[k] for k in ('shared_geometry_unchanged','review_unchanged','copied_app_unchanged','canonical_unchanged','prior_review_unchanged'))
assert c['shared_geometry_before']==c['shared_geometry_after']
shared=Path.home()/'Library/Cogent Core';geo={str(p.relative_to(shared)):{'sha256':sha(p),'mode':oct(stat.S_IMODE(p.stat().st_mode))} for p in shared.rglob('*') if p.is_file()};assert geo==c['shared_geometry_after']
assert all(v['denied'] for v in read('native/geometry-control.json.gz').values())
assert read('native/post-stop-default-refusal.json.gz')['exit_code']==75 and read('native/post-stop-default-refusal.json.gz')['endpoint_absent']
assert read('producers/initial-preflight-failure.json.gz')['child_exit_code']==71 and not read('producers/initial-preflight-failure.json.gz')['native_app_started']
obs=read('native/case-observations.json.gz');assert obs['private_entry_cases_executed']==0 and not obs['rendered_original_case_passed'] and not obs['original_native_failure_closed']
a=read('native/fixture.json.gz');z=read('native/original-final.json.gz')
assert {k for k in a.keys()|z.keys() if a.get(k)!=z.get(k)}=={'command_revision','revision','project_work','viewport','document'}
assert a['recording']==z['recording'] and z['recording']['sample_index']==0 and z['paused'] and not z['recording']['playing']
assert raw('native/fixture.elab.gz')==raw('native/fixture-final.elab.gz') and hashlib.sha256(raw('native/fixture.elab.gz')).hexdigest()==r['reference']['source_sha256']
rows=read('native/capture-rows.json.gz');metadata=read('native/capture-metadata.json.gz');assert len(rows)==1001 and metadata['input_mode']=='charge' and metadata['settings']['initial_voltage_v']==0 and metadata['duration_s']==1
maxv=maxi=0
for i,row in enumerate(rows):
 t=row['time_s'];v,amps=row['values'];assert abs(t-i*.001)<1e-14;decay=math.exp(-t/.01);maxv=max(maxv,abs(v-5*(1-decay)));maxi=max(maxi,abs(amps-.0005*decay));assert abs(v-5*(1-decay))<1e-12 and abs(amps-.0005*decay)<1e-15
ref=r['reference'];assert ref['passed'] and ref['rows_restored_exact']==ref['independent_rc_rows']==1001 and not ref['native_transport_recheck_available'] and ref['native_matched_states']==[]
assert (maxv,maxi)==(ref['max_voltage_error_v'],ref['max_current_error_a'])
for stage in read('reference/states.json.gz'):
 want={'index-next':4,'time-previous-expected':2,'time-enter-control':3,'time-enter-previous-control':2}[stage['label']];s=stage['state'];assert s['recording']['sample_index']==want and s['recording']['time_s']==rows[want]['time_s'] and s['paused'] and not s['recording']['playing']
assert ref['shutdown']['status']=='completed' and ref['shutdown']['exit_ready'] and ref['shutdown']['workers_joined']
for n in ['fixture','original-final']:assert read('native/'+n+'-bridge-exit.json.gz')['exit_code']==0
assert read('reference/reference-bridge-exit.json.gz')['exit_code']==0
for n in ['fixture-ui','focus-render-check']:
 assert raw('native/'+n+'.jpg.gz').startswith(b'\xff\xd8');ax=raw('native/'+n+'-ax.txt.gz').decode();assert 'sample 0 / 1000' in ax and 'Recorded seek time' in ax
print(json.dumps(dict(evidence_verified=True,clean_build_and_strict_signature=True,candidate_preserved=True,native_recheck_available=False,original_native_failure_closed=False,qualified_replacement=False,independent_rc_rows=1001,shared_geometry_unchanged=True,normal_close=True,app_started=False)))
