"""Read-only source/evidence verification; no native/UI/privacy/capture operation."""
from pathlib import Path
import gzip
import hashlib
import json
import sys

folder=Path(__file__).resolve().parent;root=folder.parents[2]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
read=lambda p:json.loads(p.read_text())
cp=read(folder/'checkpoint.json')
actual={p.relative_to(folder).as_posix():sha(p) for p in folder.rglob('*') if p.is_file() and p.name!='checkpoint.json'}
assert actual==cp['evidence_sha256']
assert len(cp['source_sha256'])==7 and all(sha(root/n)==d for n,d in cp['source_sha256'].items())
assert all(sha(root/n)==d for n,d in cp['frozen_window_source_sha256'].items())
assert sha(root/'build/display-frame-probe-tools/display-frame-probe')==cp['tool_sha256']
assert sha(root/'build/frame-probe-tools/frame-probe')==cp['frozen_window_recorder_sha256']
r=read(folder/'report.json');assert r['self_tests_passed'] and len(r['cases'])==46 and len(r['journal_cases'])==7
assert len(set(r['cases']))==46 and len(set(r['journal_cases']))==7
assert r['capture_executed'] is False and r['ui_or_privacy_api_called'] is False and r['eligible_latency_observations']==0
assert r==json.loads(gzip.decompress((folder/'display-frame-probe-check-final.txt.gz').read_bytes()))
assert gzip.decompress((folder/'display-frame-probe-format.txt.gz').read_bytes())==b''
assert all(sha(root/n)==d for n,d in cp['preserved_artifacts'].items())
receipt=root/'research/ui-audit/first-version/sweep-handoff-review-checkpoint.json'
assert sha(receipt)==cp['review_receipt_sha256']
quality=read(root/'docs/validation/sweep-smoke-admission/quality-default.json')
assert len(quality['source_sha256'])==cp['go_sources_checked_unchanged']==650
assert all(sha(root/n)==d for n,d in quality['source_sha256'].items())
sys.path.insert(0,str(root/'scripts'));from package_review import members
review=Path(read(receipt)['review_path']);files=members(review);files.pop('RELEASE_MANIFEST.json')
assert files==read(review/'RELEASE_MANIFEST.json')['files'] and len(files)==400
assert cp['capture_executed'] is False and cp['eligible_latency_observations']==0
print(dict(passed=True,offline_checks=53,go_sources_unchanged=650,review_173_entries_preserved=400,capture_executed=False,runtime_calibration_qualified=False))
