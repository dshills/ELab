"""Read-only final Inspector source/failed-candidate replay; opens no app."""
from pathlib import Path
import gzip,hashlib,json,sys
if sys.flags.optimize:raise SystemExit('Verification requires assertions; optimized Python is refused')
HERE=Path(__file__).resolve().parent;ROOT=HERE.parents[2]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
def raw(name):return gzip.decompress((HERE/'raw/stage'/(name+'.gz')).read_bytes())
def read(name):return json.loads(raw(name))
def main():
 cp=json.loads((HERE/'checkpoint.json').read_text())['files'];assert set(cp)=={str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p!=HERE/'checkpoint.json'}
 for name,digest in cp.items():assert not (HERE/name).is_symlink() and sha(HERE/name)==digest,name
 for name,v in json.loads((HERE/'manifest.json').read_text()).items():
  data=gzip.decompress((HERE/name).read_bytes());assert sha(HERE/name)==v['stored_sha256'] and hashlib.sha256(data).hexdigest()==v['original_sha256'] and len(data)==v['original_bytes']
 r=json.loads((HERE/'report.json').read_text());q=read('full-quality.json')
 assert q['exit_code']==0 and q['sources_unchanged'] and q['source_files']==662 and q['source_sha256']==read('source-before-quality.json')
 assert all(sha(ROOT/p)==h for p,h in q['source_sha256'].items())
 prior=HERE/'prior-width';pc=json.loads((prior/'checkpoint.json').read_text())['files']
 assert sha(prior/'checkpoint.json')==r['prior_source_checkpoint_sha256']
 assert all(sha(prior/p)==h for p,h in pc.items()) and set(pc)=={str(p.relative_to(prior)) for p in prior.rglob('*') if p.is_file() and p!=prior/'checkpoint.json'}
 old=json.loads(gzip.decompress((prior/'raw/stage/full-quality.json.gz').read_bytes()))['source_sha256'];current=q['source_sha256']
 assert set(old)==set(current) and {p for p in old if old[p]!=current[p]}=={'desktop/rail_labels_smoke.go'}
 assert hashlib.sha256(raw('rail_labels_smoke-original.go.txt')).hexdigest()==old['desktop/rail_labels_smoke.go']
 assert hashlib.sha256(raw('rail_labels_smoke-final.go.txt')).hexdigest()==current['desktop/rail_labels_smoke.go']
 assert (ROOT/'research/13-native-command-inventory.json').read_bytes()==gzip.decompress((prior/'raw/stage/native-inventory-candidate.json.gz').read_bytes())
 before=gzip.decompress((prior/'raw/stage/original-final-regression.log.gz').read_bytes()).decode();fixed=gzip.decompress((prior/'raw/stage/fixed-focused-final-fixture.log.gz').read_bytes()).decode()
 assert before.count('--- FAIL: TestMountedInspectorTextGroupsUseAvailableRailWidth/')==4 and before.count('--- FAIL: TestMountedInspectorWidthRespectsConstrainedRails/')==2
 assert 'FAIL' not in fixed and 'Ran 65 tests' in raw('full-quality.log').decode()
 assert fixed.count('--- PASS: TestMountedInspectorTextGroupsUseAvailableRailWidth/')==4 and fixed.count('--- PASS: TestMountedInspectorWidthRespectsConstrainedRails/')==2
 assert (ROOT/'scripts/check.sh').read_bytes()==gzip.decompress((prior/'raw/stage/check-final.sh.txt.gz').read_bytes())
 log=raw('failed-native-smoke.log.txt').decode();events=[json.loads(line[line.index('{'):]) for line in log.splitlines() if '"event":"elab.inspector-width.smoke"' in line]
 assert len(events)==1 and not events[0]['passed'] and len(events[0]['labels'])==11
 for v in events[0]['labels']:assert v['complete_text'] and not v['truncated'] and v['shaped_right_px']<=v['actual_width_px']+1 and 'frame_width_px' not in v
 assert any(v['allocated_width_px']<v['group_width_px']*.9 for v in events[0]['labels'])
 failed=read('failed-runtime-report.json');assert not failed['passed'] and failed['stages'][0]['exit_code']==1
 assert read('failed-build-result.json')['clean_build_vcs'] and read('failed-build-result.json')['source_commit']==r['source_parent_commit']
 sys.path.insert(0,str(ROOT/'scripts'));from package_review import members
 folder=ROOT/'build/ELab-0.167.0-184-review';assert not (folder/'RELEASE_MANIFEST.json').exists() and sha(folder/'CANDIDATE_MANIFEST.json')==r['candidate_manifest_sha256']
 assert (folder/'CANDIDATE_MANIFEST.json').read_bytes()==raw('CANDIDATE_MANIFEST.json');actual=members(folder);actual.pop('CANDIDATE_MANIFEST.json');assert actual==read('CANDIDATE_MANIFEST.json')['files']
 previous=json.loads((ROOT/'research/ui-audit/first-version/recorded-dc-review-checkpoint.json').read_text())
 assert sha(ROOT/'build/ELab.app/Contents/MacOS/ELab')==previous['canonical']['native_sha256'] and sha(ROOT/previous['sdk']['archive'])==previous['sdk']['sha256']
 assert r['source_qualified'] and r['replacement_packaging_pending'] and not any(r[k] for k in ('new_release','native_fix_qualified','solver_changed','commands_changed','sdk_changed','owner_acceptance','full_mvp_complete'))
 print(json.dumps(dict(passed=True,original_width_failures=6,failed_184_measurement_preserved=True,application_sources=659,tool_sources=3,full_quality_exit=0,native_replacement_pending=True)))
if __name__=='__main__':main()
