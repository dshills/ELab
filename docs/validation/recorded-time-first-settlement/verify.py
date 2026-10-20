"""Read-only source evidence replay. Does not launch apps, tests or capture tools."""
from pathlib import Path
import gzip,hashlib,json,re
P=Path(__file__).resolve().parent;R=P.parents[2]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
read=lambda n:gzip.decompress((P/'raw'/n).read_bytes())
cp=json.loads((P/'checkpoint.json').read_text())
actual={str(p.relative_to(P)) for p in P.rglob('*') if p.is_file() and p.name!='checkpoint.json' and '__pycache__' not in p.parts}
assert actual==set(cp['files'])
assert all(sha(P/n)==h for n,h in cp['files'].items())
m=json.loads((P/'manifest.json').read_text())
for n,item in m['files'].items():
 data=gzip.decompress((P/n).read_bytes());assert len(data)==item['bytes'] and hashlib.sha256(data).hexdigest()==item['sha256']
r=json.loads((P/'report.json').read_text());q=r['quality']
assert q['exit_code']==0 and q['sources_unchanged'] and len(q['source_sha256'])==670
assert all(sha(R/n)==h for n,h in q['source_sha256'].items())
assert json.loads(read('full-quality-final.json.gz'))==q
oldq=json.loads(read('full-quality.json.gz'));assert oldq['exit_code']==1 and oldq['sources_unchanged'] and oldq['source_sha256']==q['source_sha256']
assert b'TestNativeInventoryMatchesReviewedLedger' in read('quality-output.txt.gz')
assert b'does not contain package' in read('order-regression-original.txt.gz')
a=json.loads(read('native-inventory-before.json.gz'));b=json.loads(read('native-inventory.json.gz'))
n='desktop/recorded_playback_actions.go';a['source_sha256'][n]=b['source_sha256'][n]
for site in a['sites']:
 if site['file']==n:site['line']+=3
assert a==b==json.loads((R/'research/13-native-command-inventory.json').read_text())
assert len(b['source_sha256'])==73 and len(b['sites'])==402 and len(b['public_commands'])==164 and len(b['commands_with_source_routes'])==132
old=read('recorded_playback_actions-original.go.txt.gz').decode()
expected=old.replace('var recordedPlaybackFields = []struct {','// Managed time can publish focus-loss Change while the first native barrier is\n// pending. Settle it before legacy numeric controls so its own callback belongs\n// to that barrier. Later-field, stale-origin and unrelated-work guards stay exact.\nvar recordedPlaybackFields = []struct {').replace('}{{"recorded-index", true}, {"recorded-time", false}, {"recorded-speed", true}}','}{{"recorded-time", false}, {"recorded-index", true}, {"recorded-speed", true}}')
assert expected.encode()==read('recorded_playback_actions.go.txt.gz')==(R/n).read_bytes()
original=read('original-overlay-result.txt.gz').decode()
assert len(re.findall(r'^    --- FAIL: TestRecordedTransportManagedTimeBeforeFirstSettlement',original,re.M))==3
assert len(re.findall(r'^    --- PASS: TestRecordedTransportManagedTimeBeforeFirstSettlement',original,re.M))==12
assert original.count('order=[recorded-index] index=4 error="Recorded playback entries changed. Review them and try again."')==3
final=read('order-regression-final.txt.gz').decode();mounted=read('order-mounted-final.txt.gz').decode()
assert len(re.findall(r'^    --- PASS: TestRecordedTransportManagedTimeBeforeFirstSettlement',final,re.M))==15
assert len(re.findall(r'^--- PASS:',final,re.M))==12 and len(re.findall(r'^    --- PASS:',final,re.M))==52
assert len(re.findall(r'^--- PASS:',mounted,re.M))==8 and len(re.findall(r'^    --- PASS:',mounted,re.M))==41
assert '--- FAIL:' not in final+mounted and '\nPASS\n' in final and '\nPASS\n' in mounted
for key,path in [('prior_receipt_sha256','research/ui-audit/first-version/recorded-step-review-checkpoint.json'),('canonical_sha256','build/ELab.app/Contents/MacOS/ELab'),('sdk_lock_sha256','third_party/vellum/sdk.lock.json')]:assert sha(R/path)==r['preserved'][key]
prior=json.loads((R/'research/ui-audit/first-version/recorded-step-review-checkpoint.json').read_text())['quality']['source_sha256']
current={n:h for n,h in q['source_sha256'].items() if not n.startswith('tools/')}
assert len(prior)==666 and len(current)==667
assert set(current)-set(prior)=={'desktop/recorded_playback_order_test.go'} and not set(prior)-set(current)
assert {n for n in prior if prior[n]!=current[n]}=={'desktop/recorded_playback_actions.go','desktop/recorded_playback_actions_test.go','desktop/recorded_step_actions_test.go'}
assert r['source_scheduling_repaired'] and not any(r[k] for k in ('native_failure_closed','native_cause_confirmed','qualified_replacement','new_version'))
print(json.dumps(dict(evidence_verified=True,source_scheduling_repaired=True,original_scheduling_failures=3,final_scheduling_cases=15,full_quality_passed=True,native_failure_closed=False,native_cause_confirmed=False,new_version=False,app_started=False)))
