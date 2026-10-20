"""Read-only /180 release replay; no application or user settings are opened."""
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

def main():
 cp=read(HERE/'checkpoint.json')
 actual={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name!='checkpoint.json'};assert actual==set(cp['evidence_sha256'])
 for relative,digest in cp['evidence_sha256'].items():
  p=HERE/relative;assert not p.is_symlink() and p.resolve().is_relative_to(HERE) and sha(p)==digest,relative
 for relative,digest in cp['dependencies'].items():assert sha(ROOT/relative)==digest,relative
 receipt_path=ROOT/'research/ui-audit/first-version/integration-discovery-review-checkpoint.json'
 assert sha(receipt_path)==cp['receipt_sha256'] and receipt_path.read_bytes()==(HERE/'Acceptance.json').read_bytes()
 receipt=read(receipt_path);review=Path(receipt['review_path']);count=receipt['manifest_file_entries']
 identity=validate_release(ROOT,receipt_path);assert identity['release']=='0.163.0 /180' and identity['runtime_inputs_verified']==358 and identity['manifest_file_entries_verified']==count==473
 assert sha(review/'RELEASE_MANIFEST.json')==cp['manifest_sha256'] and (HERE/'Release-manifest.json').read_bytes()==(review/'RELEASE_MANIFEST.json').read_bytes()
 for target,entry in read(HERE/'selected-source-files.json').items():assert sha(HERE/target)==entry['sha256']==sha(review/entry['source'])
 assert len(receipt['quality']['source_sha256'])==655 and receipt['quality']['passed']
 assert all(sha(ROOT/n)==h for n,h in receipt['quality']['source_sha256'].items())
 source=ROOT/'docs/validation/integration-discovery'
 assert sha(source/'checkpoint.json')==cp['source_proof_checkpoint_sha256']==receipt['source_proof_checkpoint_sha256']
 module('numerical_discovery_source_replay',source/'verify.py').main()
 built=read(HERE/'Build-result.json');assert built['clean_build_vcs'] and built['source_commit']==receipt['source_commit']
 admission=module('numerical_discovery_signed_admission',HERE/'sources/signed_bundle.py.txt')
 assert built['source_bundle']==admission.bundle_tree(review/'ELab.app')
 admission.admit(review/'ELab.app',built,ROOT/'build/first-version-180-tools/build/elab')
 runtime=read(HERE/'Runtime.json');assert runtime['passed'] and runtime['shared_geometry_unchanged']
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
 previous.captured=lambda path:HERE/'pins'/(str(Path(path).relative_to('/private/tmp/elab-first-version-180-signed-acceptance-final'))+'.gz')
 previous.floating();previous.pins();previous.geometry();integration(previous.equal)
 pages=module('scoped_recorded_paging_replay',ROOT/'docs/validation/shutdown-layout-review-176/verify.py');pages.verify_paging(HERE/'paging',read(HERE/'Paging-plan.json'))
 assert len(receipt['preserved_bundles'])==38
 for label,entry in receipt['preserved_bundles'].items():
  folder=ROOT/'build'/('ELab-'+label+'-review');manifest=read(folder/'RELEASE_MANIFEST.json');actual=members(folder);actual.pop('RELEASE_MANIFEST.json')
  assert actual==manifest['files'] and len(actual)==entry['files_verified'] and sha(folder/'RELEASE_MANIFEST.json')==entry['manifest_sha256']
 for key,name in [('preserved_original_147_candidate','ELab-0.130.0-147-review'),('preserved_failed_164_candidate','ELab-0.147.0-164-review'),('preserved_unsigned_171_candidate','ELab-0.154.0-171-unsigned-candidate'),('preserved_failed_172_candidate','ELab-0.155.0-172-failed-candidate')]:
  folder=ROOT/'build'/name;actual=members(folder);actual.pop('CANDIDATE_MANIFEST.json');assert actual==read(folder/'CANDIDATE_MANIFEST.json')['files'] and len(actual)==receipt[key]['files_verified'] and sha(folder/'CANDIDATE_MANIFEST.json')==receipt[key]['manifest_sha256']
 assert sha(ROOT/'build/ELab.app/Contents/MacOS/ELab')==receipt['canonical']['native_sha256'] and sha(ROOT/receipt['sdk']['archive'])==receipt['sdk']['sha256']
 archive=ROOT/'build/ELab-0.163.0-180-review.zip';ar=read(HERE/'archive/report.json')
 assert ar['passed'] and ar['files_verified']==count+1==474 and sha(archive)==cp['archive_sha256']==ar['archive_sha256']
 expected=members(review);assert expected==ar['expected_files'];verify_archive(archive,review.name,expected);assert members(Path(ar['extracted_review']))==expected
 examples=read(HERE/'archive/example-report.json');assert examples['passed'] and len(examples['cases'])==3 and examples['agent_sha256']==receipt['agent_sha256']
 assert all(c['fixture_unchanged'] and c['saved_reopen_fresh'] for c in examples['cases'])
 assert not any(read(HERE/'report.json')[k] for k in ('canonical_promoted','new_physical_input','new_visible_latency','owner_acceptance','full_mvp_complete'))
 print('Verified /180: exact sources/signature/ZIP, two-field numerical guidance, six engine families/twelve paired states, inherited geometry/pin/recovery/floating/paging and preserved reviews.')

if __name__=='__main__':main()
