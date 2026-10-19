"""Read-only Verify source/evidence integrity; no copied-release or OS-input pass."""
import gzip
import hashlib
import json
from pathlib import Path
import stat
import subprocess

ROOT = Path(__file__).resolve().parents[3]


def read(path):
    data = path.read_bytes()
    return json.loads(gzip.decompress(data) if path.suffix == '.gz' else data)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def raw(folder, name):
    return gzip.decompress((folder / (name + '.gz')).read_bytes())


def main():
    folder = ROOT / 'docs/validation/verification-handoff'
    checkpoint = read(folder / 'checkpoint.json')
    assert checkpoint['status'] == 'source_repair_validated_replacement_pending'
    assert not any(checkpoint[key] for key in ['new_app_release_created', 'canonical_promoted', 'owner_acceptance', 'public_commands_changed', 'models_changed', 'project_format_changed', 'sdk_changed'])
    assert checkpoint['owner_walkthrough'] == 'Not tried yet'
    assert {p.relative_to(folder).as_posix() for p in folder.rglob('*') if p.is_file()} == set(checkpoint['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in checkpoint['evidence_sha256'].items():
        assert sha(folder / name) == expected, name
    for name, expected in checkpoint['source_sha256'].items():
        assert sha(ROOT / name) == expected, name
    for name in ['main.go', 'verification.go', 'viewport.go', 'smoke.go', 'recording_compare_smoke.go', 'project_export_smoke.go', 'verification_contracts_smoke.go', 'verification_help_smoke.go', 'scope_context_smoke.go']:
        original = subprocess.check_output(['git', 'show', checkpoint['original_source_commit'] + ':desktop/' + name], cwd=ROOT, timeout=15)
        assert original == (folder / ('original-' + name + '.txt')).read_bytes()
    before = read(folder / 'before.json')
    assert before['exit_code'] == 1 and before['original_failure_reproduced']
    assert b'operations=0' in raw(folder, 'before-stdout.txt') and b'expected command revision' in raw(folder, 'before-stdout.txt')
    for name, expected in before['source_sha256'].items():
        assert sha(folder / ('original-' + Path(name).name + '.txt')) == expected
    replay = read(folder / 'final-test-original.json')
    assert replay['exit_code'] == 1 and replay['failed_test_nodes'] == 13 and replay['failed_leaf_cases'] == 11
    failures = [line.split('--- FAIL: ', 1)[1].split(' (', 1)[0] for line in raw(folder, 'final-test-original-stdout.txt').decode().splitlines() if '--- FAIL: ' in line]
    assert sum(not any(x.startswith(n + '/') for x in failures) for n in failures) == 11
    assert b'invalid quantity started capture' in raw(folder, 'final-test-original-stdout.txt')
    additional = read(folder / 'invalid-continuation-before.json')
    assert additional['reproduced'] and additional['exit_code'] == 1
    assert b'editing sample step forgot the invalid duration' in raw(folder, 'invalid-continuation-before-stdout.txt')
    for name, expected in additional['source_sha256'].items():
        assert sha(folder / ('initial-repair-' + Path(name).name + '.txt')) == expected
    assert sha(folder / 'invalid-continuation.go.txt') == additional['overlay_test_sha256']
    focused = read(folder / 'focused.json')
    assert focused['passed'] and focused['exit_code'] == 0
    for name, expected in focused['test_sha256'].items():
        assert sha(ROOT / name) == expected
    assert b'TestMountedVerificationFieldsMatchAgentCapture/verify-cap-C1' in raw(folder, 'focused-stdout.txt')
    assert b'\nPASS\n' in raw(folder, 'focused-stdout.txt')
    quality = read(folder / 'quality.json')
    assert quality == checkpoint['source_quality'] and quality['passed'] and quality['exit_code'] == 0
    assert quality['command'] == './scripts/check.sh' and quality['source_sha256'] == checkpoint['source_sha256']
    assert b'Ran 54 tests' in raw(folder, 'quality-stderr.txt')
    assert 'TestMounted(Verification|' in (ROOT / 'scripts/check.sh').read_text()
    for prefix in ['first-', 'second-', 'third-']:
        earlier = read(folder / (prefix + 'native-smoke.json'))
        assert not earlier['passed'] and earlier['native_exit_code'] == 1 and earlier['owned_process_absent']
        assert not earlier.get('forced_cleanup', False)
    assert b'choose two different retained recordings' in raw(folder, 'first-native-smoke-stderr.txt')
    assert b'expected command revision 1760; current command revision is 1761' in raw(folder, 'third-native-smoke-stderr.txt')
    native = read(folder / 'native-smoke.json')
    assert native == checkpoint['general_native_smoke'] and native['passed']
    assert native['build_exit_code'] == native['native_exit_code'] == 0 and native['owned_process_absent']
    events = [json.loads(line) for line in raw(folder, 'native-smoke-stdout.txt').splitlines() if line.startswith(b'{"event":"elab.native.smoke"')]
    assert events == [dict(event='elab.native.smoke', passed=True)]
    assert b'vcs.modified=true' in raw(folder, 'native-build-info.txt')
    inventory = read(ROOT / 'research/13-native-command-inventory.json')
    assert len(inventory['source_sha256']) == 70 and len(inventory['sites']) == 393
    assert len(inventory['public_commands']) == 162 and len(inventory['commands_with_source_routes']) == 132
    assert sum(site['status'] == 'requires-binding-review' for site in inventory['sites']) == 170 and not inventory['issues']
    for name, expected in inventory['source_sha256'].items():
        assert sha(ROOT / name) == expected
    review = read(folder / 'inventory-review.json')
    assert review['public_commands_unchanged'] and review['aliases_and_routes_unchanged'] and not review['issues']
    preservation = read(folder / 'preservation.json')
    receipt = ROOT / 'research/ui-audit/first-version/value-preset-review-checkpoint.json'
    assert sha(receipt) == preservation['receipt_sha256']
    sealed = Path(preservation['review_path'])
    assert sha(sealed / 'RELEASE_MANIFEST.json') == preservation['manifest_sha256']
    manifest = read(sealed / 'RELEASE_MANIFEST.json')['files']
    assert len(manifest) == preservation['manifest_entries'] == 900
    assert {p.relative_to(sealed).as_posix() for p in sealed.rglob('*') if p.is_file()} == set(manifest) | {'RELEASE_MANIFEST.json'}
    for name, expected in manifest.items():
        path = sealed / name
        assert not path.is_symlink() and sha(path) == expected['sha256']
        assert oct(stat.S_IMODE(path.stat().st_mode)) == expected['mode']
    assert sha(ROOT / 'build/ELab.app/Contents/MacOS/ELab') == preservation['canonical_native_sha256']
    assert sha(ROOT / 'build/ELab-0.149.0-166-review.zip') == preservation['archive_sha256']
    print(json.dumps(dict(source_repair_integrity_verified=True, original_failed_leaf_cases=11, source_quality_passed=True, development_native_smoke_passed=True, clean_replacement_qualified=False, owner_acceptance=False)))


if __name__ == '__main__':
    main()
