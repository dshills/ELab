"""Read-only saved-coil reset source/evidence integrity; no release promotion."""
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
    folder = ROOT / 'docs/validation/verification-startup-reset'
    cp = read(folder / 'checkpoint.json')
    assert cp['status'] == 'source_repair_validated_replacement_pending'
    assert not any(cp[key] for key in ['public_commands_changed','models_changed','project_format_changed','sdk_changed','new_app_release_created','owner_acceptance','canonical_promoted'])
    assert cp['owner_walkthrough'] == 'Not tried yet'
    assert {p.relative_to(folder).as_posix() for p in folder.rglob('*') if p.is_file()} == set(cp['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in cp['evidence_sha256'].items():
        assert sha(folder / name) == expected, name
    for name, expected in cp['source_sha256'].items():
        assert sha(ROOT / name) == expected, name
    assert sha(ROOT / 'scripts/check.sh') == cp['quality_runner_sha256']
    assert sha(ROOT / 'third_party/vellum/sdk.lock.json') == cp['sdk_lock_sha256']
    original = subprocess.check_output(['git','show',cp['original_source_commit']+':desktop/verification_actions.go'],cwd=ROOT,timeout=15)
    assert original == (folder / 'original-verification_actions.go.txt').read_bytes()
    before = read(folder / 'before.json')
    assert before['original_failure_reproduced'] and before['exit_code'] == 1
    for name, expected in before['source_sha256'].items():
        assert sha(folder / ('original-' + Path(name).name + '.txt')) == expected
    assert raw(folder,'before-stdout.txt').count(b'captured_current=0.03 A; saved project requires -0.02 A') == 2
    replay = read(folder / 'final-test-original.json')
    assert replay['exit_code'] == 1 and replay['failed_leaf_cases'] == 4
    assert raw(folder,'final-test-original-stdout.txt').count(b'Use project startup retained scenario current:') == 2
    focused = read(folder / 'focused.json')
    assert focused['passed'] and focused['exit_code'] == 0
    for name, expected in focused['test_sha256'].items():
        assert sha(ROOT / name) == expected
    assert b'\nPASS\n' in raw(folder,'focused-stdout.txt')
    quality = read(folder / 'quality.json')
    assert quality == cp['source_quality'] and quality['passed'] and quality['exit_code'] == 0
    assert quality['source_sha256'] == cp['source_sha256']
    assert b'Ran 54 tests' in raw(folder,'quality-stderr.txt')
    native = read(folder / 'native-smoke.json')
    assert native == cp['general_native_smoke'] and native['passed'] and native['native_exit_code'] == 0 and native['owned_process_absent']
    events = [json.loads(line) for line in raw(folder,'native-smoke-stdout.txt').splitlines() if line.startswith(b'{"event":"elab.native.smoke"')]
    assert events == [dict(event='elab.native.smoke',passed=True)]
    assert b'vcs.modified=true' in raw(folder,'native-build-info.txt')
    ledger = read(ROOT / 'research/13-native-command-inventory.json')
    assert len(ledger['source_sha256']) == 70 and len(ledger['sites']) == 393 and len(ledger['public_commands']) == 162 and len(ledger['commands_with_source_routes']) == 132
    assert sum(s['status'] == 'requires-binding-review' for s in ledger['sites']) == 170 and not ledger['issues']
    review = read(folder / 'inventory-review.json')
    assert review['public_commands_aliases_and_routes_unchanged'] and review['changed_source'] == ['desktop/verification_actions.go'] and not review['issues']
    preservation = read(folder / 'preservation.json')
    assert sha(ROOT / 'research/ui-audit/first-version/verification-review-checkpoint.json') == preservation['receipt_sha256']
    sealed = Path(preservation['review_path'])
    assert sha(sealed / 'RELEASE_MANIFEST.json') == preservation['manifest_sha256']
    manifest = read(sealed / 'RELEASE_MANIFEST.json')['files']
    assert len(manifest) == preservation['manifest_entries'] == 1073
    assert {p.relative_to(sealed).as_posix() for p in sealed.rglob('*') if p.is_file()} == set(manifest) | {'RELEASE_MANIFEST.json'}
    for name, expected in manifest.items():
        path = sealed / name
        assert not path.is_symlink() and sha(path) == expected['sha256'] and oct(stat.S_IMODE(path.stat().st_mode)) == expected['mode']
    assert sha(ROOT / 'build/ELab-0.150.0-167-review.zip') == preservation['archive_sha256']
    assert sha(ROOT / 'build/ELab.app/Contents/MacOS/ELab') == preservation['canonical_native_sha256']
    print(json.dumps(dict(source_repair_integrity_verified=True,original_failed_leaf_cases=4,source_quality_passed=True,development_native_smoke_passed=True,clean_replacement_qualified=False,owner_acceptance=False)))


if __name__ == '__main__':
    main()
