"""Read-only replay of sweep smoke fixture evidence and source admission."""
from pathlib import Path
import gzip
import hashlib
import json
import sys

folder = Path(__file__).resolve().parent
root = folder.parents[2]
sys.path.insert(0, str(root/'scripts'))
from package_review import members

sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
read = lambda p: json.loads(p.read_text())
checkpoint = read(folder/'checkpoint.json')
actual = {p.relative_to(folder).as_posix(): sha(p) for p in folder.rglob('*') if p.is_file() and p.name != 'checkpoint.json'}
assert actual == checkpoint['evidence_sha256']
quality = read(folder/'quality-default.json')
assert quality['passed'] and quality['environment'] == {} and quality['sources_unchanged']
assert len(quality['source_sha256']) == 650
assert all(sha(root/name) == digest for name,digest in quality['source_sha256'].items())
assert sha(root/'scripts/check.sh') == quality['quality_runner_sha256']
previous = read(root/'docs/validation/dc-sweep-handoff/quality-default.json')
assert set(previous['source_sha256']) == set(quality['source_sha256'])
assert [name for name,digest in quality['source_sha256'].items() if previous['source_sha256'][name] != digest] == ['desktop/dc_sweep_smoke.go']
assert sha(root/'research/13-native-command-inventory.json') == checkpoint['native_inventory_sha256']
old = root/'docs/validation/dc-sweep-handoff'
assert sha(old/'checkpoint.json') == checkpoint['prior_sweep_checkpoint_sha256']
assert all(sha(old/name) == digest for name,digest in read(old/'checkpoint.json')['evidence_sha256'].items())
failed = json.loads(gzip.decompress((folder/'copied-runtime-failed.json.gz').read_bytes()))
assert not failed['passed'] and len(failed['stages']) == 1 and failed['stages'][0]['exit_code'] == 1
assert 'nativeDCSweepSmoke: native DC sweep: context deadline exceeded' in gzip.decompress((folder/'copied-native-failed.txt.gz').read_bytes()).decode()
for name,needle in [('native-diagnostic.txt.gz','context deadline exceeded'),('native-diagnostic-second.txt.gz','"event":"elab.native.smoke","passed":true')]:
    assert needle in gzip.decompress((folder/name).read_bytes()).decode()
assert [r['observed_exec_exit_code'] for r in read(folder/'diagnostics.json')['diagnostics']] == [1,0]
for relative,expected in checkpoint['preserved_artifacts'].items():
    assert sha(root/relative) == expected
for relative,manifest_name in checkpoint['preserved_manifests'].items():
    p = root/relative
    files = members(p)
    files.pop(manifest_name)
    assert files == read(p/manifest_name)['files']
assert sha(root/'build/ELab.app/Contents/MacOS/ELab') == checkpoint['canonical_native_sha256']
print(dict(passed=True, default_quality=True, go_sources=650, source_difference='smoke fixture only', original_native_timeout=True, corrected_diagnostic_pass=True, replacement_qualified=False, capture_executed=False))
