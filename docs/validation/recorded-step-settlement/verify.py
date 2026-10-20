"""Read-only archived evidence/current-source replay; starts no app or tests."""
from pathlib import Path
import collections,gzip,hashlib,json,sys
if sys.flags.optimize:raise SystemExit('Assertions required')
P=Path(__file__).resolve().parent;R=P.parents[2]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
def raw(n):return gzip.decompress((P/'raw/stage'/(n+'.gz')).read_bytes())
def read(n):return json.loads(raw(n))
checkpoint=json.loads((P/'checkpoint.json').read_text())['files']
assert set(checkpoint)=={str(p.relative_to(P)) for p in P.rglob('*') if p.is_file() and p!=P/'checkpoint.json'}
for n,h in checkpoint.items():
 p=P/n;assert not p.is_symlink() and p.resolve().is_relative_to(P) and sha(p)==h,n
for n,v in json.loads((P/'manifest.json').read_text()).items():
 p=P/n;data=gzip.decompress(p.read_bytes());assert sha(p)==v['stored_sha256'] and len(data)==v['original_bytes'] and hashlib.sha256(data).hexdigest()==v['original_sha256'],n
report=json.loads((P/'report.json').read_text());q=read('full-quality.json');before=read('quality-before.json');outcome=read('quality-result.json')
assert report['issue']=='RECORD-STEP-01' and report['source_qualified'] and report['clean_replacement_required']
assert q['exit_code']==outcome['exit_code']==0 and q['sources_unchanged'] and q['source_sha256']==before['go_sources']
assert len(q['source_sha256'])==669 and sum(not n.startswith('tools/') for n in q['source_sha256'])==666
assert q['elapsed_seconds']==outcome['elapsed_seconds']==report['quality']['elapsed_seconds']
assert all(sha(R/n)==h for n,h in q['source_sha256'].items())
assert all(outcome[k] for k in ('all_go_sources_unchanged','sdk_lock_unchanged','canonical_unchanged','prior_release_receipt_unchanged'))
for p in (P/'sources').rglob('*.txt'):
 n=str(p.relative_to(P/'sources'))[:-4];assert p.read_bytes()==(R/n).read_bytes(),n
for path,key in [('build/ELab.app/Contents/MacOS/ELab','canonical_sha256'),('third_party/vellum/sdk.lock.json','sdk_lock_sha256'),('research/ui-audit/first-version/recorded-playback-review-checkpoint.json','prior_release_receipt_sha256')]:assert sha(R/path)==report[key]
old=read('ledger-before.json');new=read('ledger-candidate.json');review=read('ledger-review.json')
assert new==json.loads((R/'research/13-native-command-inventory.json').read_text())
assert [len(new[k]) for k in ('source_sha256','sites','public_commands','commands_with_source_routes')]==[73,402,164,132]
assert all(old[k]==new[k] for k in old if k not in ('source_sha256','sites'))
assert review['source_changes']==[n for n,h in new['source_sha256'].items() if old['source_sha256'].get(n)!=h]==['desktop/main.go','desktop/recorded_playback_actions.go','desktop/recording.go','desktop/view.go','desktop/viewport.go']
def sites(r):return collections.Counter(json.dumps({k:v for k,v in s.items() if k!='line'},sort_keys=True) for s in r['sites'])
a,b=sites(old),sites(new)
assert review['removed']==[json.loads(s) for s,n in (a-b).items() for _ in range(n)] and len(review['removed'])==9
assert review['added']==[json.loads(s) for s,n in (b-a).items() for _ in range(n)] and len(review['added'])==10
original=raw('original-regression-final-output.txt').decode();final=raw('second-focused-output.txt').decode()
leaves=[v for v in original.splitlines() if '--- FAIL: ' in v and '/' in v.split('--- FAIL: ',1)[1]]
assert len(leaves)==report['original_final_leaf_failures']==24 and '--- FAIL:' not in final and '\nPASS\n' in final
prior=json.loads((R/'research/ui-audit/first-version/recorded-playback-review-checkpoint.json').read_text())
for name in ('recording.go','recorded_playback_actions.go'):
 assert hashlib.sha256(raw(name+'-original.txt')).hexdigest()==prior['quality']['source_sha256']['desktop/'+name]==report['original_production_source_sha256'][name]
def failures(n):return [v for v in raw(n).decode().splitlines() if '--- FAIL: ' in v and '/' in v.split('--- FAIL: ',1)[1]]
assert len(failures('original-diagnostic-output.txt'))==10 and len(failures('original-boundary-output.txt'))==4
assert report['original_diagnostic_leaf_failures']==14
for name in ('TestRecordedStepWaitsForFieldsAndIgnoresOtherTransportAdmissions','TestRecordedStepRefusesInterveningWork','TestRecordedStepRetainsInvalidPositionUntilCorrection','TestRecordedStepAfterOwnSpeedAndNaturalPlaybackEnd','TestMountedNumericOwnerRecordedStepAfterPrivateEntry','TestMountedNumericOwnerRecordedStepBoundaryDraft','TestMountedNumericOwnerRecordedStepWithoutFocusLoss','TestMountedNumericOwnerRecordedStepBoundaryRefusesWithoutDraft','TestMountedNumericOwnerRecordedStepPrivateEntryRefusesAgent','TestNativeSingleSampleRecordingsRenderAndRetainTransientSeeking','TestNativeRecordingPickerAndControlsMatchAgentCommands','TestMountedNumericOwnerRecordedPlayAfterPrivateEntry','TestRecordedPlayRefusesInterveningWork'):
 assert '--- PASS: '+name+' ' in final,name
assert '"app_started"' not in raw('quality-result.json').decode()
print(json.dumps(dict(passed=True,application_go_sources=666,all_go_sources=669,original_leaf_failures=24,source_qualified=True,clean_replacement_required=True,app_started=False)))
