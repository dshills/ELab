"""Read-only component startup source/evidence integrity; no promotion."""

import gzip
import hashlib
import json
from pathlib import Path
import stat
import subprocess

ROOT = Path(__file__).resolve().parents[3]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


def raw(folder, name):
    path = folder / name
    if path.exists():
        return path.read_bytes()
    return gzip.decompress((folder / (name + '.gz')).read_bytes())


def main():
    folder = ROOT / 'docs/validation/component-startup-handoff'
    cp = read(folder / 'checkpoint.json')
    assert cp['status'] == 'source_repair_validated_replacement_pending'
    assert not any(cp[key] for key in [
        'public_commands_changed', 'models_changed', 'project_format_changed',
        'sdk_changed', 'new_app_release_created', 'owner_acceptance', 'canonical_promoted',
    ])
    assert cp['owner_walkthrough'] == 'Not tried yet'
    assert {p.relative_to(folder).as_posix() for p in folder.rglob('*') if p.is_file()} == set(cp['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in cp['evidence_sha256'].items():
        assert sha(folder / name) == expected, name
    for name, expected in cp['source_sha256'].items():
        assert sha(ROOT / name) == expected, name
    assert sha(ROOT / 'scripts/check.sh') == cp['quality_runner_sha256']
    assert sha(ROOT / 'third_party/vellum/sdk.lock.json') == cp['sdk_lock_sha256']

    before = read(folder / 'before.json')
    assert before['original_failure_reproduced'] and before['exit_code'] == 1
    for name, expected in before['source_sha256'].items():
        original = raw(folder, 'original-' + Path(name).name + '.txt')
        assert hashlib.sha256(original).hexdigest() == expected
        if not name.endswith('_test.go'):
            assert original == subprocess.check_output(
                ['git', 'show', cp['original_source_commit'] + ':' + name], cwd=ROOT, timeout=15)
    assert raw(folder, 'before-stdout.txt').count(b'expected command revision') == 4
    replay = read(folder / 'final-test-original.json')
    assert replay['original_failure_reproduced'] and replay['exit_code'] == 1 and replay['failed_leaf_cases'] == 4
    assert raw(folder, 'final-test-original-stdout.txt').count(b'expected command revision') == 4
    focused = read(folder / 'focused.json')
    assert focused['passed'] and focused['exit_code'] == 0
    for name, expected in focused['test_sha256'].items():
        assert sha(ROOT / name) == expected
    names = [line.split('--- PASS: ', 1)[1].split(' (', 1)[0] for line in raw(folder, 'focused-stdout.txt').decode().splitlines() if '--- PASS: ' in line]
    assert sum(not any(other.startswith(name + '/') for other in names) for name in names) == 34
    quality = read(folder / 'quality.json')
    assert quality == cp['source_quality'] and quality['passed'] and quality['exit_code'] == 0
    assert quality['source_sha256'] == cp['source_sha256']
    assert b'Ran 54 tests' in raw(folder, 'quality-stderr.txt')
    native = read(folder / 'native-smoke.json')
    assert native == cp['general_native_smoke'] and native['passed'] and native['native_exit_code'] == 0 and native['owned_process_absent']
    events = [json.loads(line) for line in raw(folder, 'native-smoke-stdout.txt').splitlines() if line.startswith(b'{"event":"elab.native.smoke"')]
    assert events == [dict(event='elab.native.smoke', passed=True)]
    assert b'vcs.modified=true' in raw(folder, 'native-build-info.txt')
    for prefix in ['first', 'second', 'third', 'fourth']:
        failed = read(folder / (prefix + '-native-smoke.json'))
        assert not failed['passed'] and failed['native_exit_code'] == 1 and failed['owned_process_absent']

    ledger = read(ROOT / 'research/13-native-command-inventory.json')
    assert len(ledger['source_sha256']) == 71 and len(ledger['sites']) == 397
    assert len(ledger['public_commands']) == 162 and len(ledger['commands_with_source_routes']) == 132
    assert sum(site['status'] == 'requires-binding-review' for site in ledger['sites']) == 173 and not ledger['issues']
    old = read(ROOT / 'build/ELab-0.151.0-168-review/Evidence/Native-command-inventory.json')
    for key in ledger:
        if key not in ['source_sha256', 'sites']:
            assert ledger[key] == old[key], key
    review = read(folder / 'inventory-review.json')
    assert review['public_commands_aliases_and_routes_unchanged'] and len(review['changed_source']) == 6 and not review['issues']
    preservation = read(folder / 'preservation.json')
    assert sha(ROOT / 'research/ui-audit/first-version/startup-reset-review-checkpoint.json') == preservation['receipt_sha256']
    assert len(preservation['sealed_reviews']) == 28
    for name, report in preservation['sealed_reviews'].items():
        sealed = ROOT / 'build' / name
        assert sha(sealed / 'RELEASE_MANIFEST.json') == report['manifest_sha256']
        manifest = read(sealed / 'RELEASE_MANIFEST.json')['files']
        assert len(manifest) == report['manifest_entries']
        assert {p.relative_to(sealed).as_posix() for p in sealed.rglob('*') if p.is_file()} == set(manifest) | {'RELEASE_MANIFEST.json'}
        for name, expected in manifest.items():
            path = sealed / name
            assert not path.is_symlink() and sha(path) == expected['sha256'] and oct(stat.S_IMODE(path.stat().st_mode)) == expected['mode']
    assert sha(ROOT / 'build/ELab-0.151.0-168-review.zip') == preservation['archive_sha256']
    assert sha(ROOT / 'build/ELab.app/Contents/MacOS/ELab') == preservation['canonical_native_sha256']
    print(json.dumps(dict(source_repair_integrity_verified=True, original_failed_leaf_cases=4, focused_passed_leaf_cases=34, source_quality_passed=True, development_native_smoke_passed=True, clean_replacement_qualified=False, owner_acceptance=False)))


if __name__ == '__main__':
    main()
