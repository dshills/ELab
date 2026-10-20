"""Read-only DC scope failure and source-repair replay; starts no application."""
from pathlib import Path
import gzip,hashlib,json,sys
if sys.flags.optimize:raise SystemExit('Verification requires assertions; optimized Python is refused')
HERE=Path(__file__).resolve().parent;ROOT=HERE.parents[2]
def sha(data):return hashlib.sha256(data).hexdigest()
def raw(name):return gzip.decompress((HERE/'raw'/(name+'.gz')).read_bytes())
def read(name):return json.loads(raw(name))
def main():
 cp=json.loads((HERE/'checkpoint.json').read_text())['files']
 assert set(cp)=={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name!='checkpoint.json'}
 for name,digest in cp.items():
  p=HERE/name;assert not p.is_symlink() and sha(p.read_bytes())==digest,name
 for name,entry in json.loads((HERE/'manifest.json').read_text()).items():
  stored=(HERE/name).read_bytes();data=gzip.decompress(stored)
  assert sha(stored)==entry['stored_sha256'] and sha(data)==entry['original_sha256'] and len(data)==entry['original_bytes']
 report=json.loads((HERE/'report.json').read_text());quality=read('stage/full-quality.json')
 assert quality['exit_code']==0 and quality['sources_unchanged'] and quality['source_files']==661
 assert quality['source_sha256']==read('stage/source-before-quality.json')
 for name,digest in quality['source_sha256'].items():assert sha((ROOT/name).read_bytes())==digest,name
 previous=gzip.decompress((ROOT/'docs/validation/recording-final-consistency/raw/stage/full-quality.json.gz').read_bytes());old=json.loads(previous)['source_sha256'];current=quality['source_sha256']
 assert set(current)-set(old)=={'desktop/recording_dc_test.go'} and not set(old)-set(current)
 assert {p for p in old if old[p]!=current[p]}=={'desktop/recording.go'}
 assert sha(raw('stage/recording-original.go.txt'))==old['desktop/recording.go']
 assert sha(raw('stage/recording-final.go.txt'))==current['desktop/recording.go']
 assert sha(raw('stage/recording_dc-final.go.txt'))==current['desktop/recording_dc_test.go']
 old_log=raw('stage/original-focused.log').decode();assert old_log.count('recorded scope cannot render ui: node "recorded-index"')==2
 fixed_log=raw('stage/fixed-focused.log').decode();assert 'FAIL' not in fixed_log and fixed_log.count('--- PASS: TestNativeSingleSampleRecordingsRenderAndRetainTransientSeeking/example.')==2
 assert 'Ran 65 tests' in raw('stage/full-quality.log').decode()
 assert 'ui: node "recorded-index": ui: numeric control requires finite ordered bounds' in raw('failed-native/native-host.txt').decode()
 failure=raw('failed-release/acceptance-final-output.txt').decode();assert 'agent terminated during project.load' in failure and 'recording_final_cases.py' in failure
 assert 'CalledProcessError' in raw('failed-release/acceptance-run-output.txt').decode()
 assert read('failed-release/runtime-report.json')['passed']
 built=read('failed-release/build-result-signed.json');assert built['clean_build_vcs'] and built['source_commit']==report['source_parent_commit']
 candidate=ROOT/'build/ELab-0.165.0-182-review';assert not (candidate/'RELEASE_MANIFEST.json').exists()
 assert (candidate/'CANDIDATE_MANIFEST.json').read_bytes()==raw('failed-release/CANDIDATE_MANIFEST.json')
 assert sha((candidate/'CANDIDATE_MANIFEST.json').read_bytes())==report['candidate_manifest_sha256']
 sys.path.insert(0,str(ROOT/'scripts'));from package_review import members
 actual=members(candidate);actual.pop('CANDIDATE_MANIFEST.json');assert actual==read('failed-release/CANDIDATE_MANIFEST.json')['files']
 ledger=json.loads((ROOT/'research/13-native-command-inventory.json').read_text());old_ledger=read('stage/native-inventory-original.json')
 assert ledger==read('stage/native-inventory-candidate.json')
 assert {p for p in ledger['source_sha256'] if ledger['source_sha256'][p]!=old_ledger['source_sha256'][p]}=={'desktop/recording.go'}
 for key in ledger:
  if key=='source_sha256':continue
  if key=='sites':
   strip=lambda values:[{k:v for k,v in row.items() if k!='line'} for row in values]
   assert strip(ledger[key])==strip(old_ledger[key])
  else:assert ledger[key]==old_ledger[key]
 source=ROOT/'docs/validation/recording-final-consistency';assert sha((source/'checkpoint.json').read_bytes())==report['prior_recording_final_checkpoint_sha256']
 for name,digest in json.loads((source/'checkpoint.json').read_text())['files'].items():assert sha((source/name).read_bytes())==digest
 receipt_path=ROOT/'research/ui-audit/first-version/terminal-validation-review-checkpoint.json';assert sha(receipt_path.read_bytes())==report['prior_receipt_sha256'];receipt=json.loads(receipt_path.read_text())
 for label,entry in receipt['preserved_bundles'].items():
  folder=ROOT/'build'/('ELab-'+label+'-review');actual=members(folder);actual.pop('RELEASE_MANIFEST.json');assert actual==json.loads((folder/'RELEASE_MANIFEST.json').read_text())['files'] and len(actual)==entry['files_verified']
 folder=Path(receipt['review_path']);actual=members(folder);actual.pop('RELEASE_MANIFEST.json');assert actual==json.loads((folder/'RELEASE_MANIFEST.json').read_text())['files']
 assert sha((ROOT/receipt['sdk']['archive']).read_bytes())==receipt['sdk']['sha256']
 assert sha((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes())==receipt['canonical']['native_sha256']
 assert report['source_qualified'] and report['replacement_packaging_pending'] and report['native_host_exit_code'] is None
 assert not any(report[k] for k in ('solver_changed','new_release','native_fix_qualified','owner_acceptance','full_mvp_complete'))
 print(json.dumps(dict(original_source_failures=2,fixed_dc_families=2,validated_selections=8,failed_candidate_preserved=True,application_sources=658,full_quality_exit=0,native_replacement_pending=True)))
if __name__=='__main__':main()
