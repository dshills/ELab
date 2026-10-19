import collections
import gzip
import hashlib
import json
from pathlib import Path
import shutil
import stat
import subprocess

root=Path('/Users/dshills/Development/projects/ELab')
base=Path(__file__).resolve().parent
out=root/'docs/validation/recorded-timing-direction'
assert json.loads((base/'quality.json').read_text())['passed']
assert json.loads((base/'native-smoke.json').read_text())['passed']
assert not out.exists();out.mkdir(parents=True)
def copy(source,target):
    destination=out/target;destination.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(source,destination)
def compressed(source,target):
    destination=out/target;destination.parent.mkdir(parents=True,exist_ok=True)
    with destination.open('wb') as f:
        with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:z.write(source.read_bytes())
for name in ['original-regression.txt','fixture-first-attempt.txt','focused-first-attempt.txt','focused-final.txt','quality.json','quality.txt','before-smoke-protocol-quality.json','before-smoke-protocol-quality.txt','native-smoke.json','native-smoke.txt','inventory.json','previous-inventory.json','inventory-review.json']:
    copy(base/name,name)
compressed(base/'native-build-info.txt','native-build-info.txt.gz')
for name in ['owner-only-fixture-failure-quality.json','owner-only-fixture-failure-quality.txt','owner-only-pulse.txt']:copy(base/name,name)
for p in (base/'native-smoke-before-owner-fixture').iterdir():copy(p,'native-smoke-before-owner-fixture/'+p.name)
for p in (base/'native-smoke-first-attempt').iterdir():copy(p,'native-smoke-first-attempt/'+p.name)
for name in ['recording_compare_smoke.go.before.txt','recording_pulses_smoke.go.before.txt']:copy(base/name,'smoke-original/'+name)
native=root/'build/native-timing-menu-163-retry'
for name in ['setup.py','observe.py','restore_checked.py','producer-corrections.json','identity.json']:copy(native/name,'native-original/'+name)
for p in (native/'observations').iterdir():
    if p.suffix=='.jpg' or p.name in ['cleanup.json','ui-conclusion.json','pointer-attempt.json']:copy(p,'native-original/'+p.name)
    elif p.suffix=='.txt':compressed(p,'native-original/'+p.name+'.gz')
compressed(native/'observations/report.json','native-original/report.json.gz')
for p in (native/'observations/requests').glob('*.json'):copy(p,'native-original/requests/'+p.name)
for label in ['Native','Agent']:compressed(native/'observations'/(label+' timing.elab'),'native-original/'+label+' timing.elab.gz')
interrupted=root/'build/native-timing-menu-163'
for name in ['observe.py','setup.py','restore_checked.py']:copy(interrupted/name,'approval-interrupted/'+name)
for name in ['cleanup.json','interruption.json','baseline-save-analysis.json']:copy(interrupted/'observations'/name,'approval-interrupted/'+name)
compressed(interrupted/'observations/report.json','approval-interrupted/report.json.gz')
files=['desktop/recorded_timing_actions.go','desktop/recording_timing.go','desktop/recorded_timing_offscreen_test.go','desktop/recorded_timing_direction_darwin_test.go','desktop/recorded_timing_direction_offscreen_darwin_test.go','desktop/recorded_timing_menu_darwin_test.go','desktop/recording_compare_smoke.go','desktop/recording_pulses_smoke.go','desktop/recording_timing_smoke.go']
identity=dict(source_base=subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip(),production_and_test_sha256={name:hashlib.sha256((root/name).read_bytes()).hexdigest() for name in files},native_inventory_sha256=hashlib.sha256((root/'research/13-native-command-inventory.json').read_bytes()).hexdigest(),scope='Source repair with mounted native-menu callback regressions and general development native smoke. No fixed physical typing/menu replay, clean replacement or owner acceptance claim.')
for name in files:copy(root/name,'source/'+Path(name).name+'.txt')
for name in ['recorded_timing_actions.go','recording_timing.go']:
    original=subprocess.check_output(['git','show','HEAD:desktop/'+name],cwd=root)
    destination=out/'source-original'/(name+'.txt');destination.parent.mkdir(exist_ok=True);destination.write_bytes(original)
(out/'source-identity.json').write_text(json.dumps(identity,indent=2)+'\n')
previous=json.loads((root/'docs/validation/recorded-timing-handoff/preservation.json').read_text())
records=previous['review_manifests'];records['ELab-0.146.0-163-review']=dict(entries=501,manifest_sha256='997128f748eda91ae53c9e573e59b6650e0be8a8b3c9ad367c5d43e66d2e61a2')
receipt=json.loads((root/'research/ui-audit/first-version/recorded-timing-review-checkpoint.json').read_text())
original=receipt['preserved_original_147_candidate'];records['ELab-0.130.0-147-review']=dict(entries=original['files_verified'],manifest_sha256=original['manifest_sha256'])
for name,record in records.items():
    folder=root/'build'/name;p=folder/'RELEASE_MANIFEST.json';assert hashlib.sha256(p.read_bytes()).hexdigest()==record['manifest_sha256'];entries=json.loads(p.read_text())['files'];assert len(entries)==record['entries']
    assert {str(p.relative_to(folder)) for p in folder.rglob('*') if p.is_file()}==set(entries)|{'RELEASE_MANIFEST.json'}
    for relative,expected in entries.items():
        p=folder/relative;assert not p.is_symlink() and hashlib.sha256(p.read_bytes()).hexdigest()==expected['sha256'];assert oct(stat.S_IMODE(p.stat().st_mode))==expected['mode']
assert hashlib.sha256((root/'build/ELab.app/Contents/MacOS/ELab').read_bytes()).hexdigest()==previous['canonical_native_sha256']
assert hashlib.sha256((root/'third_party/vellum/sdk.lock.json').read_bytes()).hexdigest()==previous['sdk_lock_sha256']
preservation=dict(passed=True,review_manifests=records,qualified_reviews=24,original_147_candidate_preserved=True,canonical_native_sha256=previous['canonical_native_sha256'],sdk_lock_sha256=previous['sdk_lock_sha256'],scope='Sealed historical app/agent contents preserved; current repaired source/ledger intentionally differs from /163.')
(out/'preservation.json').write_text(json.dumps(preservation,indent=2)+'\n')
patch=subprocess.check_output(['git','diff','--binary'],cwd=root)
with (out/'source-repair.patch.gz').open('wb') as f:
    with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:z.write(patch)
print('Copied source and native-failure evidence; README/verifier/checkpoint remain to seal.')
