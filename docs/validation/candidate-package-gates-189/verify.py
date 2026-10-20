"""Read-only data/ZIP/source replay; no extraction, app, test or subprocess."""
from pathlib import Path,PurePosixPath
import gzip,hashlib,json,math,stat,tarfile,zipfile
P=Path(__file__).resolve().parent;R=P.parents[2]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
raw=lambda n:gzip.decompress((P/'raw'/n).read_bytes())
read=lambda n:json.loads(raw(n))
cp=json.loads((P/'checkpoint.json').read_text());files={str(p.relative_to(P)) for p in P.rglob('*') if p.is_file() and p.name!='checkpoint.json' and '__pycache__' not in p.parts};assert files==set(cp['files']) and all(sha(P/n)==h for n,h in cp['files'].items())
m=json.loads((P/'manifest.json').read_text())
for n,v in m['files'].items():
 data=gzip.decompress((P/n).read_bytes());assert len(data)==v['bytes'] and hashlib.sha256(data).hexdigest()==v['sha256']
meta=m['data_archive'];assert sha(P/meta['path'])==meta['sha256'] and (P/meta['path']).stat().st_size==meta['bytes'];entries={}
with tarfile.open(P/meta['path'],mode='r:gz') as tar:
 for member in tar:
  path=PurePosixPath(member.name);assert member.isfile() and not path.is_absolute() and '..' not in path.parts and str(path)==member.name and member.name not in entries
  e=meta['entries'][member.name];assert member.size==e['bytes'];data=tar.extractfile(member).read();assert len(data)==e['bytes'] and hashlib.sha256(data).hexdigest()==e['sha256'];entries[member.name]=data
assert set(entries)==set(meta['entries']) and len(entries)==299 and sum(len(v) for v in entries.values())==40662416
get=lambda n:json.loads(entries[n]);r=json.loads((P/'report.json').read_text());a=get('acceptance/report.json');q=get('paging/report.json');z=read('archive/report.json.gz');assert r['acceptance']==a and r['paging']==q and r['archive']==z
assert a['passed'] and q['passed'] and z['passed'] and a['native_host_closed_normally'] and a['shared_geometry_unchanged']
assert not any(r[k] for k in ('release_qualified','native_time_previous_closed','promoted','new_version')) and not a['qualified_replacement'] and not a['native_time_previous_closed'] and not z['release_qualified'] and not z['native_time_previous_closed'] and not z['promoted']
index=a['recorded_index_commands'];assert index['passed'] and index['invalid_index_refusals']==6 and index['paired_integer_stages']==3 and index['paired_step_stages']==6 and index['step_boundary_refusals']==4 and index['independent_rc_rows']==42
ignored={'view','viewport','revision','command_revision','last_actor'}
preserved=lambda v:{k:x for k,x in v.items() if k not in ignored}
rows_checked=0
for label in ['native','headless']:
 v=get('acceptance/'+label+'-recorded-index.json');rows=v['rows'];assert len(rows)==21 and [p['id'] for p in v['metadata']['probes']]==['v:C1','i:R1']
 for row in rows:
  decay=math.exp(-row['time_s']/.01);assert abs(row['values'][0]-5*(1-decay))<1e-12 and abs(row['values'][1]-.0005*decay)<1e-15;rows_checked+=1
 for item in v['refusals']+v['step_boundaries']:assert preserved(item['before'])==preserved(item['refused'])
 assert len(v['refusals'])==3 and len(v['step_boundaries'])==2 and len(v['integer_stages'])==3 and len(v['step_stages'])==6
 for step in v['step_stages']:
  assert step['accepted']['recording']['sample_index']==step['expected_index'] and not step['accepted']['recording']['playing'];assert {k:x for k,x in preserved(step['before']).items() if k!='recording'}=={k:x for k,x in preserved(step['accepted']).items() if k!='recording'}
 first=entries['acceptance/'+label+'-recorded-index-baseline.elab'];last=entries['acceptance/'+label+'-recorded-index-final.elab'];assert first==last and hashlib.sha256(first).hexdigest()==v['source_sha256'] and v['saved_bytes_unchanged']
assert rows_checked==42
for section in ['recording_final_commands','terminal_validation_commands','integration_discovery_commands','geometry_action_commands','pinned_file_commands','floating_source_commands','sweep_capture_commands','value_correction','preset_value_commands','verification_handoff_commands','coil_startup_reset_commands','component_startup_commands','discovery_commands','live_measurement_ownership','recorded_cursors','recording_targets','reference_parity']:assert a[section]['passed']
assert a['recording_final_commands']['invalid_recording_refusals']==26 and a['recording_final_commands']['invalid_document_refusals']==6
assert a['terminal_validation_commands']['paired_physics_stages']==10 and a['integration_discovery_commands']['paired_physics_states']==12 and a['geometry_action_commands']['paired_stages']==24 and a['preset_value_commands']['values']==4
assert a['discovery_commands']['commands']==164 and a['live_grid']['paused_physics_stable'] and a['live_grid']['save_reopen_fresh'] and a['live_grid']['first_time_s']>=2.2 and a['live_grid']['second_time_s']>=3.41
assert q['behavior_passed'] and q['source_file_unchanged'] and q['process_exit_code']==0 and q['requests']==95 and len(q['cases'])==30 and not q['findings']
plan=read('stage/paging-plan.json.gz');fixture=entries['paging/recorded-rc.elab'];assert hashlib.sha256(fixture).hexdigest()==plan['fixture_sha256'];recorded=json.loads(fixture)['recordings']['entries'][0]['data'];assert len(recorded['rows'])==209
transcript=[json.loads(line) for line in entries['paging/transcript.jsonl'].splitlines()];assert len(transcript)==95
for i,case in enumerate(q['cases']):
 before,event,after=transcript[3+i*3:6+i*3];assert json.loads(before['request_json'])['command']==json.loads(after['request_json'])['command']=='project.inspect';assert before['response']['result']==after['response']['result'] and case['snapshot_unchanged']
 request=json.loads(event['request_json']);response=event['response'];assert request['command']==case['command']
 if case['error'] is not None:assert response['error']['code']==case['error']
 else:
  assert 'error' not in response
  if request['command']=='operation.samples':
   params=request['params'];offset=params.get('offset') or 0;limit=params.get('limit') or 100;rows=recorded['rows'][offset:offset+limit];assert response['result']=={'id':params['id'],'offset':offset,'next_offset':offset+len(rows),'total':209,'probes':recorded['metadata']['probes'],'rows':rows}
assert transcript[2]['response']['result']==transcript[-2]['response']['result'];closed=get('paging/close-response.json');assert closed['closed'] and all(closed['shutdown'][k] for k in ('workers_joined','recovery_finalized','exit_ready'))
zip_path=Path(z['archive']);assert sha(zip_path)==z['archive_sha256'] and zip_path.stat().st_size==z['archive_bytes'] and z['files_verified']==114 and z['ditto_exit_code']==0 and z['strict_extracted_signature_passed']
expected=z['members'];prefix=Path(z['source_package']).name
with zipfile.ZipFile(zip_path) as source:
 names=source.namelist();assert len(names)==len(set(names))==len(expected) and set(names)=={prefix+'/'+n for n in expected}
 for item in source.infolist():
  path=PurePosixPath(item.filename);mode=item.external_attr>>16;assert str(path)==item.filename and '..' not in path.parts and not path.is_absolute() and stat.S_ISREG(mode) and item.create_system==3
  v=expected[str(path.relative_to(prefix))];assert hashlib.sha256(source.read(item)).hexdigest()==v['sha256'] and oct(stat.S_IMODE(mode))==v['mode']
def members(folder):
 out={}
 for p in sorted(folder.rglob('*')):
  assert not p.is_symlink() and (p.is_dir() or p.is_file())
  if p.is_file():out[str(p.relative_to(folder))]={'sha256':sha(p),'mode':oct(stat.S_IMODE(p.stat().st_mode))}
 return out
assert members(Path(z['source_package']))==members(Path(z['extracted_package']))==expected
manifest=read('archive/package/CANDIDATE_MANIFEST.json.gz');payload=dict(expected);payload.pop('CANDIDATE_MANIFEST.json');assert manifest['files']==payload and not manifest['release_qualified']
status=read('archive/package/CANDIDATE_STATUS.json.gz');assert not status['release_qualified'] and not status['native_time_previous_closed'] and not status['promoted'] and status['owner_walkthrough']=='Not tried yet'
def compare(left,right):
 if isinstance(left,dict):
  left={k:v for k,v in left.items() if k not in ('run_id','electrical_revision')};right={k:v for k,v in right.items() if k not in ('run_id','electrical_revision')};assert left.keys()==right.keys()
  for k in left:compare(left[k],right[k])
 elif isinstance(left,list):
  assert isinstance(right,list) and len(left)==len(right)
  for x,y in zip(left,right):compare(x,y)
 elif isinstance(left,(int,float)) and not isinstance(left,bool):assert math.isclose(left,right,rel_tol=1e-9,abs_tol=1e-12)
 else:assert left==right
assert len(z['examples'])==3 and len(z['process_exits'])==6
for example in z['examples']:
 name=example['example'];original=raw('archive/package/Examples/'+name+'.gz');saved=raw('archive/saved-examples/'+name+'.gz');assert hashlib.sha256(original).hexdigest()==example['source_fixture_sha256'] and hashlib.sha256(saved).hexdigest()==example['saved_sha256'] and example['fresh_saved_reopen'] and example['source_unchanged']
 left=example['loaded'];right=example['reopened']
 for k in ['components','wires','nets','transient_settings','scope_probes']:assert left.get(k)==right.get(k)
 for k in ['result','transient','mixed','npn']:compare(left.get(k),right.get(k))
for e in z['process_exits']:assert e['exit_code']==0 and e['shutdown']['status']=='completed' and all(e['shutdown'][k] for k in ('workers_joined','recovery_finalized','exit_ready'))
display=r['display_state'];assert display['output']==read('display/observation.json.gz') and hashlib.sha256(raw('display/main.swift.txt.gz')).hexdigest()==display['source_sha256']
assert display['output']['valid_main_display'] and display['output']['main_display_active'] and not display['output']['main_display_asleep'] and display['output']['session_login_complete'] and display['output']['session_on_console']
assert not any(display['output'][k] for k in ('capture_performed','ui_events_posted','settings_changed'))
qsource=json.loads((R/'docs/validation/recorded-time-first-settlement/report.json').read_text())['quality'];assert qsource['exit_code']==0 and qsource['sources_unchanged'] and len(qsource['source_sha256'])==670 and all(sha(R/n)==h for n,h in qsource['source_sha256'].items())
for key,path in [('source_checkpoint_sha256','docs/validation/recorded-time-first-settlement/checkpoint.json'),('native_checkpoint_sha256','docs/validation/native-recorded-transport-189/checkpoint.json'),('frame_public_checkpoint_sha256','docs/validation/frame-and-public-gates-189/checkpoint.json'),('canonical_sha256','build/ELab.app/Contents/MacOS/ELab'),('prior_release_receipt_sha256','research/ui-audit/first-version/recorded-step-review-checkpoint.json')]:assert sha(R/path)==r[key]
b=read('stage/build-result-signed.json.gz');assert r['source_commit']==z['source_commit']==b['source_commit'] and status['native_sha256']==b['native_sha256'] and z['agent_sha256']==status['agent_sha256']==b['agent_sha256']==plan['agent_sha256']
print(json.dumps(dict(evidence_verified=True,source_unchanged=True,packaged_acceptance=True,paging_cases=30,paging_requests=95,independent_rc_rows=42,extracted_files=114,fresh_example_processes=6,release_qualified=False,native_time_previous_closed=False,promoted=False,app_started=False)))
