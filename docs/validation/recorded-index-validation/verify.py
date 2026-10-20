"""Read-only source/metadata replay, not a native runtime or new test run."""
from pathlib import Path
import gzip
import hashlib
import json
import sys

if sys.flags.optimize:
    raise SystemExit('Assertions required')
P = Path(__file__).resolve().parent
R = P.parents[2]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def raw(name):
    return gzip.decompress((P / 'raw/stage' / (name + '.gz')).read_bytes())


def read(name):
    return json.loads(raw(name))


checkpoint = json.loads((P / 'checkpoint.json').read_text())['files']
assert set(checkpoint) == {str(f.relative_to(P)) for f in P.rglob('*') if f.is_file() and f != P / 'checkpoint.json'}
for name, digest in checkpoint.items():
    path = P / name
    assert not path.is_symlink() and path.resolve().is_relative_to(P)
    assert sha(path) == digest, name
for name, info in json.loads((P / 'manifest.json').read_text()).items():
    path = P / name
    value = gzip.decompress(path.read_bytes())
    assert sha(path) == info['stored_sha256']
    assert len(value) == info['original_bytes'] and hashlib.sha256(value).hexdigest() == info['original_sha256']
report = json.loads((P / 'report.json').read_text())
quality = read('full-quality.json')
outcome = read('quality-result.json')
before = read('quality-before.json')
assert report['issue'] == 'RECORD-INDEX-01' and report['source_qualified']
assert quality['exit_code'] == outcome['exit_code'] == 0 and quality['sources_unchanged']
assert quality['source_sha256'] == before['go_sources'] and len(quality['source_sha256']) == 664
assert sum(not n.startswith('tools/') for n in quality['source_sha256']) == 661
for name, digest in quality['source_sha256'].items():
    assert sha(R / name) == digest, name
for key in ('all_go_sources_unchanged', 'sdk_lock_unchanged', 'canonical_unchanged', 'prior_release_receipt_unchanged'):
    assert outcome[key]
assert outcome['elapsed_seconds'] == quality['elapsed_seconds'] == report['quality']['elapsed_seconds']
assert sha(R / 'third_party/vellum/sdk.lock.json') == report['sdk_lock_sha256']
assert sha(R / 'build/ELab.app/Contents/MacOS/ELab') == report['canonical_sha256']
receipt_path = R / 'research/ui-audit/first-version/inspector-width-review-checkpoint.json'
assert sha(receipt_path) == report['prior_release_receipt_sha256']
receipt = json.loads(receipt_path.read_text())
assert receipt['quality']['source_sha256']['desktop/recording.go'] == report['original_application_source_sha256']
assert hashlib.sha256(raw('recording-original.go.txt')).hexdigest() == report['original_application_source_sha256']
assert sha(R / 'desktop/recording.go') == report['fixed_application_source_sha256']
for name in ('recording.go', 'recording_index_test.go', 'recording_index_offscreen_test.go'):
    assert (P / 'sources' / (name + '.txt')).read_bytes() == (R / 'desktop' / name).read_bytes()
assert (P / 'sources/13-native-command-inventory.json.txt').read_bytes() == (R / 'research/13-native-command-inventory.json').read_bytes()

old = read('ledger-before.json')
new = json.loads((R / 'research/13-native-command-inventory.json').read_text())
review = read('ledger-review.json')
assert review['source_changes'] == ['desktop/recording.go'] and len(review['non_line_site_changes']) == 2
for key in old:
    if key not in ('source_sha256', 'sites'):
        assert old[key] == new[key], key
assert len(new['source_sha256']) == 72 and len(new['sites']) == 400
assert len(new['public_commands']) == 164 and len(new['commands_with_source_routes']) == 132
assert [n for n in old['source_sha256'] if old['source_sha256'][n] != new['source_sha256'][n]] == ['desktop/recording.go']
changes = [{'before': a, 'after': b} for a, b in zip(old['sites'], new['sites'])
           if {k: v for k, v in a.items() if k != 'line'} != {k: v for k, v in b.items() if k != 'line'}]
assert changes == review['non_line_site_changes']
terminal = read('terminal-results.json')
assert terminal['original_compile']['exit_code'] == terminal['original_diagnostic']['exit_code'] == 1
assert terminal['original_regression_final']['exit_code'] == 1 and terminal['original_regression_final']['leaf_failures'] == 4
assert terminal['focused_final']['exit_code'] == terminal['full_quality']['exit_code'] == 0
original = raw('original-regression-final-output.txt').decode()
final = raw('focused-final-output.txt').decode()
for name in ('TestMountedNumericOwnerRecordedIndexRefusesFractionAndCorrects',
             'TestNativeRecordedIndexSharesStrictIntegerValidation/0.5',
             'TestNativeRecordedIndexSharesStrictIntegerValidation/1.5',
             'TestNativeRecordedIndexSharesStrictIntegerValidation/19.75'):
    assert '--- FAIL: ' + name in original
    assert '--- PASS: ' + name in final
assert b'sample 0\xe2\x86\x921; command revision 5\xe2\x86\x926; error=""' in raw('reproduced-output.txt')
assert report['new_clean_replacement_required'] and not report['owner_acceptance'] and not report['full_mvp_complete']
print(json.dumps(dict(passed=True, source_qualified=True, original_regression_leaf_failures=4,
                      application_go_sources=661, all_go_sources=664,
                      quality_seconds=quality['elapsed_seconds'], commands_unchanged=164,
                      source_routes_unchanged=132, clean_replacement_required=True,
                      physical_input_qualified=False, app_opened=False)))
