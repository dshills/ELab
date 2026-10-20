"""Read-only Inspector source regression replay; opens no application."""
from pathlib import Path
import copy,gzip,hashlib,json,sys
if sys.flags.optimize:raise SystemExit('Verification requires assertions; optimized Python is refused')
HERE=Path(__file__).resolve().parent;ROOT=HERE.parents[2]
def sha(data):return hashlib.sha256(data).hexdigest()
def raw(name):return gzip.decompress((HERE/'raw/stage'/(name+'.gz')).read_bytes())
def read(name):return json.loads(raw(name))
def main():
 cp=json.loads((HERE/'checkpoint.json').read_text())['files']
 assert set(cp)=={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name!='checkpoint.json'}
 for name,digest in cp.items():
  p=HERE/name;assert not p.is_symlink() and p.resolve().is_relative_to(HERE) and sha(p.read_bytes())==digest,name
 for name,entry in json.loads((HERE/'manifest.json').read_text()).items():
  stored=(HERE/name).read_bytes();data=gzip.decompress(stored)
  assert sha(stored)==entry['stored_sha256'] and sha(data)==entry['original_sha256'] and len(data)==entry['original_bytes']
 report=json.loads((HERE/'report.json').read_text());q=read('full-quality.json')
 assert q['exit_code']==0 and q['sources_unchanged'] and q['source_files']==662 and q['source_sha256']==read('source-before-quality.json')
 for p,digest in q['source_sha256'].items():assert sha((ROOT/p).read_bytes())==digest,p
 receipt_path=ROOT/'research/ui-audit/first-version/recorded-dc-review-checkpoint.json'
 assert sha(receipt_path.read_bytes())==report['prior_receipt_sha256'];previous=json.loads(receipt_path.read_text())
 current={p:h for p,h in q['source_sha256'].items() if not p.startswith('tools/')};old=previous['quality']['source_sha256']
 assert len(current)==659 and set(current)-set(old)=={'desktop/inspector_width_offscreen_test.go'} and not set(old)-set(current)
 assert {p for p in old if old[p]!=current[p]}=={'desktop/view.go','desktop/layout.go','desktop/rail_labels_smoke.go'}
 for p in ('view','layout','rail_labels_smoke'):
  assert sha(raw(p+'-original.go.txt'))==old['desktop/'+p+'.go']
  assert sha(raw(p+'-final.go.txt'))==current['desktop/'+p+'.go']
 assert sha(raw('inspector_width_offscreen_test-final.go.txt'))==current['desktop/inspector_width_offscreen_test.go']
 bad=raw('original-final-regression.log').decode();fixed=raw('fixed-focused-final-fixture.log').decode()
 assert bad.count('--- FAIL: TestMountedInspectorTextGroupsUseAvailableRailWidth/')==4 and bad.count('--- FAIL: TestMountedInspectorWidthRespectsConstrainedRails/')==2
 assert 'FAIL' not in fixed and fixed.count('--- PASS: TestMountedInspectorTextGroupsUseAvailableRailWidth/')==4 and fixed.count('--- PASS: TestMountedInspectorWidthRespectsConstrainedRails/')==2
 assert 'Ran 65 tests' in raw('full-quality.log').decode() and '|Inspector|' in raw('check-final.sh.txt').decode()
 assert (ROOT/'scripts/check.sh').read_bytes()==raw('check-final.sh.txt')
 ledger=json.loads((ROOT/'research/13-native-command-inventory.json').read_text());a=read('native-inventory-original.json');b=read('native-inventory-candidate.json');assert ledger==b
 assert {p for p in a['source_sha256'] if a['source_sha256'][p]!=b['source_sha256'][p]}=={'desktop/layout.go','desktop/view.go'}
 x,y=copy.deepcopy(a),copy.deepcopy(b)
 for v in (x,y):
  v.pop('source_sha256')
  for site in v['sites']:site.pop('line')
 assert x==y and len(b['sites'])==400 and len(b['public_commands'])==164 and len(b['commands_with_source_routes'])==132
 assert read('native-inventory-comparison.json')['passed']
 sys.path.insert(0,str(ROOT/'scripts'));from package_review import members
 reviews=dict(previous['preserved_bundles']);reviews['0.166.0-183']=dict(manifest_sha256=sha((Path(previous['review_path'])/'RELEASE_MANIFEST.json').read_bytes()),files_verified=previous['manifest_file_entries'])
 assert len(reviews)==41
 for label,entry in reviews.items():
  folder=ROOT/'build'/('ELab-'+label+'-review');actual=members(folder);actual.pop('RELEASE_MANIFEST.json')
  assert actual==json.loads((folder/'RELEASE_MANIFEST.json').read_text())['files'] and len(actual)==entry['files_verified'] and sha((folder/'RELEASE_MANIFEST.json').read_bytes())==entry['manifest_sha256']
 assert sha((ROOT/previous['sdk']['archive']).read_bytes())==previous['sdk']['sha256']
 assert sha((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes())==previous['canonical']['native_sha256']
 assert report['source_qualified'] and report['replacement_packaging_pending'] and report['original_subcase_failures']==report['fixed_subcases']==6
 assert not any(report[k] for k in ('new_release','native_fix_qualified','solver_changed','commands_changed','sdk_changed','owner_acceptance','full_mvp_complete'))
 print(json.dumps(dict(original_failures=6,fixed_mounted_subcases=6,application_sources=659,tool_sources=3,full_quality_exit=0,preserved_reviews=41,native_replacement_pending=True)))
if __name__=='__main__':main()
