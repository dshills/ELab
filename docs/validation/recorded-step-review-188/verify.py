"""Read-only /188 release replay; no application or user settings are opened."""
from pathlib import Path
from importlib.machinery import SourceFileLoader
import copy,gzip,hashlib,importlib.util,json,math,sys
if sys.flags.optimize:raise SystemExit('Verification requires assertions; optimized Python is refused')
HERE=Path(__file__).resolve().parent;ROOT=HERE.parents[2]
sys.path.insert(0,str(ROOT/'scripts'))
from package_review import members,verify_archive
from wiring_replay import validate_release

def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def read(p):
 data=p.read_bytes();return json.loads(gzip.decompress(data) if p.suffix=='.gz' else data)
def module(name,path):
 spec=importlib.util.spec_from_loader(name,SourceFileLoader(name,str(path)));result=importlib.util.module_from_spec(spec)
 previous=sys.dont_write_bytecode
 try:sys.dont_write_bytecode=True;spec.loader.exec_module(result)
 finally:sys.dont_write_bytecode=previous
 return result

def integration(equal):
 old=ROOT/'docs/validation/integration-discovery/raw/original-final'
 old_catalog=read(old/'ordinary.json.gz');old_details=read(old/'details.json.gz')
 families=[('flasher','elab.mixed-mna-sdirk2.v1'),('rc-high-pass','elab.network-mna-sdirk2.v1'),('rc','elab.ideal-rc-exact.v1'),('blink','elab.npn-switch-quasistatic.v1'),('rlc','elab.rlc-mna-radau5.v1'),('waveform','elab.network-waveforms-mna-sdirk2.v1')]
 results={}
 for label in ('native','headless'):
  discovery=read(HERE/'integration'/(label+'-integration-discovery.json.gz'))
  assert discovery['catalog']==old_catalog
  excluded={'view','viewport','revision','command_revision','last_actor'}
  assert {k:v for k,v in discovery['before'].items() if k not in excluded}=={k:v for k,v in discovery['after'].items() if k not in excluded}
  for command in ('component.place','component.set'):
   a,b=copy.deepcopy(old_details[command]),copy.deepcopy(discovery['details'][command])
   old_text=a['capabilities'][0]['input_schema']['x-network-numerics'].pop('time_integration')
   new_text=b['capabilities'][0]['input_schema']['x-network-numerics'].pop('time_integration')
   assert 'retain backward Euler' in old_text and 'retain backward Euler' not in new_text
   assert 'SDIRK2' in new_text and 'historical' in new_text and all(engine in new_text for _,engine in families)
   assert a==b,command
  values=[]
  for name,engine in families:
   value=read(HERE/'integration'/(label+'-integration-'+name+'.json.gz'));assert value['name']==name and value['engine']==engine
   assert value['baseline']['transient']['engine']==value['live']['transient']['engine']==engine
   assert value['baseline']['transient']['sample']['time_s']==0 and value['live']['transient']['sample']['time_s']==.001
   assert not value['live']['transient']['stale'];values.append(value)
  results[label]=(discovery,values)
 assert results['native'][0]['details']==results['headless'][0]['details']
 for a,b in zip(results['native'][1],results['headless'][1]):
  for phase in ('baseline','live'):
   for key in ('components','wires','transient_settings','source_sampling','source_warnings'):assert a[phase].get(key)==b[phase].get(key)
   for key in ('result','mixed','transient','scope'):equal(a[phase].get(key),b[phase].get(key))
 fixture=gzip.decompress((HERE/'integration/integration-waveform.elab.gz').read_bytes())
 accepted=read(HERE/'Reference-acceptance.json.gz')['integration_discovery_commands']
 assert accepted['passed'] and accepted['paired_physics_states']==12 and accepted['cases_per_adapter']==6 and accepted['corrected_detailed_text_fields']==2
 assert hashlib.sha256(fixture).hexdigest()==accepted['fixture_sha256'] and accepted['fixture_unchanged']
 assert accepted==read(HERE/'Acceptance.json')['packaged_integration_discovery']

def terminals(equal):
 source=ROOT/'docs/validation/terminal-shape-validation/raw'
 folder=HERE/'terminal';values={}
 original_inputs={}
 for case in ('scope-empty','pins-short'):
  for name in ('original.elab','malformed.elab'):
   p=folder/'terminal-input'/case/(name+'.gz')
   reference=source/'original-final'/case/(name+'.gz')
   assert gzip.decompress(p.read_bytes())==gzip.decompress(reference.read_bytes())
   original_inputs[(case,name)]=read(p)
 bad_record=folder/'terminal-input/malformed-record.json.gz'
 assert gzip.decompress(bad_record.read_bytes())==gzip.decompress((source/'import-controls/malformed-record.json.gz').read_bytes())
 for label in ('native','headless'):
  outcomes=[]
  for case in ('scope-empty','pins-short'):
   directory=folder/(label+'-terminal-'+case);result=read(directory/'result.json.gz')
   assert result['case']==case
   before,refused=result['before'],result['refused']
   excluded={'project_work','revision','command_revision','last_actor','view','viewport'}
   assert {k:v for k,v in before.items() if k not in excluded}=={k:v for k,v in refused.items() if k not in excluded}
   assert before['document']['dirty'] and refused['project_work']['status']=='failed' and refused['project_work']['error']['code']=='invalid-document'
   scan_before,scanned=result['scan_before'],result['scanned']
   assert not scanned['recovery']['candidates'] and len(scanned['recovery']['problems'])==1 and 'terminal metadata' in scanned['recovery']['problems'][0]['message']
   for field in ('components','wires','transient','mixed','run_id','electrical_revision','document','undo_entry','redo_entry'):assert scan_before.get(field)==scanned.get(field)
   capsule_file=directory/'recovery'/('rec-'+('a'*24)+'-0000000001.elab-recovery.gz')
   capsule=read(capsule_file);assert hashlib.sha256(gzip.decompress(capsule_file.read_bytes())).hexdigest()==result['capsule_sha256']
   assert hashlib.sha256(json.dumps(capsule['document'],separators=(',',':')).encode()).hexdigest()==capsule['document_sha256']
   assert capsule['document']==original_inputs[(case,'malformed.elab')]
   saved=directory/'surviving-edit.elab.gz';document=read(saved);reopened=result['reopened']
   assert hashlib.sha256(gzip.decompress(saved.read_bytes())).hexdigest()==result['saved_sha256']
   assert document['graph']['components']==reopened['components']==before['components'] and document['graph']['wires']==reopened['wires']==before['wires']
   assert not reopened['document']['dirty'] and reopened['transient']['sample']['time_s']==0 and abs(reopened['transient']['sample']['current_a']-5/20000)<1e-12
   valid=result['valid'];assert not valid['document']['dirty'] and valid['components']==original_inputs[(case,'original.elab')]['graph']['components']
   outcomes.append(result)
  control=read(folder/(label+'-terminal-import.json.gz'))
  assert {k:v for k,v in control['before'].items() if k not in excluded}=={k:v for k,v in control['after'].items() if k not in excluded}
  assert control['after']['project_work']['status']=='failed' and control['after']['project_work']['error']['code']=='invalid-recording'
  values[label]=outcomes
 paired=0
 for a,b in zip(values['native'],values['headless']):
  assert a['case']==b['case']
  for phase in ('before','refused','scanned','reopened','valid'):
   for field in ('components','wires','transient_settings'):assert a[phase].get(field)==b[phase].get(field)
   for field in ('result','mixed','transient','scope'):equal(a[phase].get(field),b[phase].get(field))
   paired+=1
 summary=read(HERE/'Reference-acceptance.json.gz')['terminal_validation_commands']
 assert summary['passed'] and summary['paired_physics_stages']==paired==10
 assert summary['invalid_document_refusals']==summary['recovery_problem_cases']==summary['surviving_saved_fresh_reopens']==4 and summary['standalone_import_refusals']==2
 assert summary==read(HERE/'Acceptance.json')['packaged_terminal_validation']

def recording_final(equal):
 source=ROOT/'docs/validation/recording-final-consistency/raw/original'
 folder=HERE/'recording-final';paired=0;values={}
 fields={name:(['current_a'] if name=='led' else ['base_current_a','collector_current_a','emitter_current_a','base_resistor_id','load_resistor_id','led_id']) for name in ('led','npn','blink')}
 excluded={'project_work','revision','command_revision','last_actor','view','viewport'}
 def equivalent(a,b):assert {k:v for k,v in a.items() if k not in excluded}=={k:v for k,v in b.items() if k not in excluded}
 for name,mutations in fields.items():
  for filename in ['good.json','bad-embedded.elab']+[f+'.json' for f in mutations]:
   assert gzip.decompress((folder/'recording-final-input'/name/(filename+'.gz')).read_bytes())==gzip.decompress((source/name/(filename+'.gz')).read_bytes())
 for label in ('native','headless'):
  results=[]
  for name,mutations in fields.items():
   result=read(folder/(label+'-recording-final-'+name+'-result.json.gz'))
   assert result['name']==name and result['byte_identical_export_before_and_after'] and len(result['cases'])==len(mutations)
   good=gzip.decompress((source/name/'good.json.gz').read_bytes());assert hashlib.sha256(good).hexdigest()==result['good_sha256']
   for suffix in ('good','after-refusal'):
    assert gzip.decompress((folder/(label+'-recording-final-'+name+'-'+suffix+'.json.gz')).read_bytes())==good
   for case,field in zip(result['cases'],mutations):
    assert case['field']==field;equivalent(case['before'],case['after'])
    assert case['before']['document']['dirty'] and case['after']['project_work']['error']['code']=='invalid-recording'
   equivalent(result['embedded_before'],result['embedded_after'])
   assert result['embedded_after']['project_work']['error']['code']=='invalid-document'
   results.append(result)
  values[label]=results
 for a,b in zip(values['native'],values['headless']):
  for left,right in zip(a['cases'],b['cases']):
   assert left['field']==right['field']
   for field in ('components','wires','drawing','paused'):assert left['after'].get(field)==right['after'].get(field)
   for field in ('result','npn','mixed','transient','scope'):equal(left['after'].get(field),right['after'].get(field))
   paired+=1
 result=read(HERE/'Reference-acceptance.json.gz')['recording_final_commands']
 assert result['passed'] and result['invalid_recording_refusals']==26 and result['invalid_document_refusals']==6
 assert result['intact_byte_identical_exports']==result['post_refusal_byte_identical_exports']==6
 assert result['paired_preserved_stages']==paired==13 and result['original_inputs_unchanged']
 assert result==read(HERE/'Acceptance.json')['packaged_recording_final']

def recorded_index(equal):
 expected_reports=[]
 ignored={'view','viewport','revision','command_revision','last_actor'}
 stable=lambda value:{k:v for k,v in value.items() if k not in ignored}
 for label in ('native','headless'):
  value=read(HERE/'recorded-index'/(label+'-recorded-index.json.gz'))
  rows=value['rows'];assert len(rows)==21 and [p['id'] for p in value['metadata']['probes']]==['v:C1','i:R1']
  assert value['metadata']['settings']['initial_voltage_v']==0 and value['metadata']['input_mode']=='charge'
  for row in rows:
   decay=math.exp(-row['time_s']/.01);assert abs(row['values'][0]-5*(1-decay))<1e-12 and abs(row['values'][1]-.0005*decay)<1e-15
  assert [entry['index'] for entry in value['refusals']]==[.5,1.5,19.75]
  for entry in value['refusals']:assert stable(entry['before'])==stable(entry['refused'])
  assert [entry['recording']['sample_index'] for entry in value['integer_stages']]==[4,20,0]
  for entry,index in zip(value['integer_stages'],(4,20,0)):
   assert entry['recording']['time_s']==rows[index]['time_s']
   assert {k:v for k,v in stable(entry).items() if k!='recording'}=={k:v for k,v in stable(value['before']).items() if k!='recording'}
  a=gzip.decompress((HERE/'recorded-index'/(label+'-recorded-index-baseline.elab.gz')).read_bytes());b=gzip.decompress((HERE/'recorded-index'/(label+'-recorded-index-final.elab.gz')).read_bytes())
  assert a==b and hashlib.sha256(a).hexdigest()==value['source_sha256'] and value['saved_bytes_unchanged']
  expected_reports.append(value)
 assert expected_reports[0]['rows']==expected_reports[1]['rows']
 for a,b in zip(expected_reports[0]['integer_stages'],expected_reports[1]['integer_stages']):
  for key in ('components','wires','nets','paused'):assert a[key]==b[key]
  for key in ('scope','readings','sample_index','time_s','playing'):equal(a['recording'][key],b['recording'][key])
 summary=read(HERE/'Reference-acceptance.json.gz')['recorded_index_commands'];assert summary==read(HERE/'Acceptance.json')['packaged_recorded_index']
 assert summary['passed'] and summary['invalid_index_refusals']==6 and summary['paired_integer_stages']==3 and summary['independent_rc_rows']==42 and summary['stored_rows_and_metadata_unchanged'] and summary['original_and_final_save_bytes_equal']

def recorded_steps(equal):
 expected=[('index-next',{'index':3},.00390625,'next',4),('index-previous',{'index':3},.00390625,'previous',2),('time-next',{'time_s':'3 ms'},.00390625,'next',4),('time-previous',{'time_s':'3 ms'},.00390625,'previous',2),('speed-next',{'index':8},.0078125,'next',9),('speed-previous',{'index':8},.00390625,'previous',7)]
 ignored={'view','viewport','revision','command_revision','last_actor'}
 stable=lambda state:{k:v for k,v in state.items() if k not in ignored}
 results=[]
 for label in ('native','headless'):
  value=read(HERE/'recorded-index'/(label+'-recorded-index.json.gz'));results.append(value)
  stages=value['step_stages'];assert len(stages)==6
  for stage,(name,seek,speed,direction,index) in zip(stages,expected):
   assert (stage['name'],stage['seek'],stage['speed'],stage['direction'],stage['expected_index'])==(name,seek,speed,direction,index)
   before,after=stage['before'],stage['accepted'];viewer=after['recording']
   assert not viewer['playing'] and viewer['sample_index']==index and viewer['time_s']==value['rows'][index]['time_s'] and viewer['seconds_per_wall_second']==speed
   assert {k:v for k,v in stable(before).items() if k!='recording'}=={k:v for k,v in stable(after).items() if k!='recording'}
  boundaries=value['step_boundaries'];assert [(v['index'],v['direction']) for v in boundaries]==[(0,'previous'),(20,'next')]
  for boundary in boundaries:assert stable(boundary['before'])==stable(boundary['refused'])
 for a,b in zip(results[0]['step_stages'],results[1]['step_stages']):
  for key in ('components','wires','nets','paused'):assert a['accepted'][key]==b['accepted'][key]
  for key in ('scope','readings','sample_index','time_s','playing','seconds_per_wall_second'):equal(a['accepted']['recording'][key],b['accepted']['recording'][key])
 summary=read(HERE/'Reference-acceptance.json.gz')['recorded_index_commands'];assert summary==read(HERE/'Acceptance.json')['packaged_recorded_index']
 assert summary['passed'] and summary['paired_step_stages']==6 and summary['step_boundary_refusals']==4

def main():
 cp=read(HERE/'checkpoint.json')
 actual={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name!='checkpoint.json'};assert actual==set(cp['evidence_sha256'])
 for relative,digest in cp['evidence_sha256'].items():
  p=HERE/relative;assert not p.is_symlink() and p.resolve().is_relative_to(HERE) and sha(p)==digest,relative
 for relative,digest in cp['dependencies'].items():assert sha(ROOT/relative)==digest,relative
 receipt_path=ROOT/'research/ui-audit/first-version/recorded-step-review-checkpoint.json'
 assert sha(receipt_path)==cp['receipt_sha256'] and receipt_path.read_bytes()==(HERE/'Acceptance.json').read_bytes()
 receipt=read(receipt_path);review=Path(receipt['review_path']);count=receipt['manifest_file_entries']
 identity=validate_release(ROOT,receipt_path);assert identity['release']=='0.171.0 /188' and identity['runtime_inputs_verified']==359 and identity['manifest_file_entries_verified']==count
 assert sha(review/'RELEASE_MANIFEST.json')==cp['manifest_sha256'] and (HERE/'Release-manifest.json').read_bytes()==(review/'RELEASE_MANIFEST.json').read_bytes()
 for target,entry in read(HERE/'selected-source-files.json').items():assert sha(HERE/target)==entry['sha256']==sha(review/entry['source'])
 assert len(receipt['quality']['source_sha256'])==666 and receipt['quality']['passed']
 assert all(sha(ROOT/n)==h for n,h in receipt['quality']['source_sha256'].items())
 source=ROOT/'docs/validation/recorded-step-settlement'
 assert sha(source/'checkpoint.json')==cp['source_proof_checkpoint_sha256']==receipt['source_proof_checkpoint_sha256']
 module('recorded_play_source_replay',source/'verify.py')
 built=read(HERE/'Build-result.json');assert built['clean_build_vcs'] and built['source_commit']==receipt['source_commit']
 admission=module('numerical_discovery_signed_admission',HERE/'sources/signed_bundle.py.txt')
 assert built['source_bundle']==admission.bundle_tree(review/'ELab.app')
 admission.admit(review/'ELab.app',built,ROOT/'build/first-version-188-tools/build/elab')
 runtime=read(HERE/'Runtime.json');assert runtime['passed'] and runtime['shared_geometry_unchanged']
 native_text=gzip.decompress((HERE/'native-smoke.log.txt.gz').read_bytes()).decode()
 inspector_events=[json.loads(line[line.index('{'):]) for line in native_text.splitlines() if '"event":"elab.inspector-width.smoke"' in line]
 assert len(inspector_events)==1 and inspector_events[0]==receipt['packaged_inspector_text']
 inspector=inspector_events[0];assert inspector['passed'] and len(inspector['labels'])==11
 assert {v['example'] for v in inspector['labels']}=={'rc','nmos','pmos','cmos'}
 for v in inspector['labels']:assert v['group_width_px']>0 and v['frame_width_px']>=v['group_width_px']*.9 and v['frame_width_px']<=v['group_width_px']+1 and v['actual_width_px']<=v['frame_width_px']+1 and v['complete_text'] and not v['truncated'] and v['shaped_right_px']<=v['actual_width_px']+1
 assert runtime['source_bundle']==built['source_bundle'] and runtime['source_agent_sha256']==built['agent_sha256']
 assert all(stage['status']=='completed' and stage['exit_code']==0 for stage in runtime['stages'])
 public=read(HERE/'Public-report.json.gz');assert public['passed'] and [a['name'] for a in public['adapters']]==['in-process','headless-stdio','headless-socket','native-socket']
 exits,shutdowns=read(HERE/'Process-exits.json.gz'),read(HERE/'Shutdown-responses.json.gz')
 assert len(exits)==102 and len(shutdowns)==112
 for outcome in exits.values():assert outcome['exit_code']==0 and not outcome['graceful_wait_timed_out'] and not any(outcome.get(k) for k in ('wait_error','close_error','context_error_before_cancel'))
 for outcome in shutdowns.values():
  assert not outcome.get('error') and not outcome['response'].get('error');state=outcome['response']['result'];work=state['shutdown']
  assert state['closed'] and work['status']=='completed' and work['workers_joined'] and work['exit_ready']
 accepted=read(HERE/'Reference-acceptance.json.gz');assert accepted['passed'] and accepted['native_host_closed_normally'] and accepted['shared_geometry_unchanged']
 previous=module('scoped_geometry_pin_floating_replay',ROOT/'docs/validation/geometry-action-review-179/verify.py')
 previous.HERE=HERE
 previous.captured=lambda path:HERE/'pins'/(str(Path(path).relative_to('/private/tmp/elab-first-version-188-signed-acceptance-final'))+'.gz')
 previous.floating();previous.pins();previous.geometry();integration(previous.equal);terminals(previous.equal);recording_final(previous.equal);recorded_index(previous.equal);recorded_steps(previous.equal)
 pages=module('scoped_recorded_paging_replay',ROOT/'docs/validation/shutdown-layout-review-176/verify.py');pages.verify_paging(HERE/'paging',read(HERE/'Paging-plan.json'))
 assert len(receipt['preserved_bundles'])==44
 for label,entry in receipt['preserved_bundles'].items():
  folder=ROOT/'build'/('ELab-'+label+'-review');manifest=read(folder/'RELEASE_MANIFEST.json');actual=members(folder);actual.pop('RELEASE_MANIFEST.json')
  assert actual==manifest['files'] and len(actual)==entry['files_verified'] and sha(folder/'RELEASE_MANIFEST.json')==entry['manifest_sha256']
 for key,name in [('preserved_original_147_candidate','ELab-0.130.0-147-review'),('preserved_failed_164_candidate','ELab-0.147.0-164-review'),('preserved_unsigned_171_candidate','ELab-0.154.0-171-unsigned-candidate'),('preserved_failed_172_candidate','ELab-0.155.0-172-failed-candidate'),('preserved_failed_182_candidate','ELab-0.165.0-182-review'),('preserved_failed_184_candidate','ELab-0.167.0-184-review')]:
  folder=ROOT/'build'/name;actual=members(folder);actual.pop('CANDIDATE_MANIFEST.json');assert actual==read(folder/'CANDIDATE_MANIFEST.json')['files'] and len(actual)==receipt[key]['files_verified'] and sha(folder/'CANDIDATE_MANIFEST.json')==receipt[key]['manifest_sha256']
 assert sha(ROOT/'build/ELab.app/Contents/MacOS/ELab')==receipt['canonical']['native_sha256'] and sha(ROOT/receipt['sdk']['archive'])==receipt['sdk']['sha256']
 archive=ROOT/'build/ELab-0.171.0-188-review.zip';ar=read(HERE/'archive/report.json')
 assert ar['passed'] and ar['files_verified']==count+1 and sha(archive)==cp['archive_sha256']==ar['archive_sha256']
 expected=members(review);assert expected==ar['expected_files'];verify_archive(archive,review.name,expected);assert members(Path(ar['extracted_review']))==expected
 examples=read(HERE/'archive/example-report.json');assert examples['passed'] and len(examples['cases'])==3 and examples['agent_sha256']==receipt['agent_sha256']
 assert all(c['fixture_unchanged'] and c['saved_reopen_fresh'] for c in examples['cases'])
 assert 'ZIP-extracted /188' in examples['scope']
 assert not any(read(HERE/'report.json')[k] for k in ('canonical_promoted','new_physical_input','new_visible_latency','owner_acceptance','full_mvp_complete'))
 print('Verified /188: clean source/signature/ZIP, recorded Previous/Next field-settlement and inherited native/regression checks and 44 preserved reviews.')

if __name__=='__main__':main()
