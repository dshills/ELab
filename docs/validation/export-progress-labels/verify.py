"""Read-only source delta, reviewed ledger and recorded quality admission."""
from pathlib import Path
import gzip
import hashlib
import json

folder = Path(__file__).resolve().parent
root = folder.parents[2]
read = lambda p: json.loads(p.read_text())
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
checkpoint = read(folder / 'checkpoint.json')
assert {p.relative_to(folder).as_posix(): sha(p) for p in folder.rglob('*')
        if p.is_file() and p.name != 'checkpoint.json'} == checkpoint['evidence_sha256']
source = read(folder / 'source-review.json')
assert all(sha(Path(p)) == digest for p, digest in source['preserved_artifacts'].items())
assert sha(root / 'docs/validation/sweep-smoke-admission/quality-default.json') == source['previous_quality_sha256']
old = read(root / 'docs/validation/sweep-smoke-admission/quality-default.json')
quality = read(folder / 'quality/report.json')
assert quality['passed'] and quality['exit_code'] == 0 and quality['sources_unchanged_during_checks']
assert quality['command'] == './scripts/check.sh' and quality['changed_go_sources'] == ['desktop/project_work.go']
assert len(quality['source_sha256']) == source['source_count'] == 650
assert all(sha(root / path) == digest for path, digest in quality['source_sha256'].items())
assert [p for p in old['source_sha256'] if old['source_sha256'][p] != quality['source_sha256'][p]] == ['desktop/project_work.go']
assert sha(folder / 'original-project_work.go.txt') == source['original_sha256'] == old['source_sha256']['desktop/project_work.go']
assert sha(folder / 'corrected-project_work.go.txt') == source['corrected_sha256'] == quality['source_sha256']['desktop/project_work.go']
assert sha(root / 'research/13-native-command-inventory.json') == source['native_ledger_sha256']
previous_ledger = json.loads(gzip.decompress((folder / 'original-native-inventory.json.gz').read_bytes()))
current_ledger = read(root / 'research/13-native-command-inventory.json')
review = read(folder / 'inventory-review.json')
assert review['passed'] and len(review['changed_sites']) == 1
assert review['source_count'] == 72 and review['site_count'] == 397 and review['route_count'] == 132 and review['public_command_count'] == 164
for key in current_ledger:
    if key not in ['sites', 'source_sha256']:
        assert current_ledger[key] == previous_ledger[key]
for before, after in zip(previous_ledger['sites'], current_ledger['sites']):
    assert {k: v for k, v in before.items() if k != 'line'} == {k: v for k, v in after.items() if k != 'line'}
failed = read(folder / 'initial-ledger-refusal/report.json')
assert not failed['passed'] and failed['exit_code'] == 1 and failed['sources_unchanged_during_checks']
assert b'native sources/routes changed' in gzip.decompress((folder / 'initial-ledger-refusal/stdout.txt.gz').read_bytes())
assert b'Ran 54 tests' in gzip.decompress((folder / 'quality/stderr.txt.gz').read_bytes())
assert source['package_qualification'] == 'pending' and not source['shared_command_semantics_changed']
print(dict(passed=True, source_go_files=650, changed_go_sources=['desktop/project_work.go'],
           full_quality_seconds=quality['elapsed_s'], initial_ledger_refusal_retained=True,
           public_commands=164, replacement_package_qualified=False))
