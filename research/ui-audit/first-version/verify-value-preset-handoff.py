"""Read-only source-repair/evidence integrity; no replacement or OS-input pass."""
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


def main():
    folder = ROOT / 'docs/validation/value-preset-handoff'
    checkpoint = read(folder / 'checkpoint.json')
    assert checkpoint['status'] == 'source_repair_validated_replacement_pending'
    assert not checkpoint['new_app_release_created'] and not checkpoint['canonical_promoted']
    assert not checkpoint['owner_acceptance'] and checkpoint['owner_walkthrough'] == 'Not tried yet'
    assert not any(checkpoint[key] for key in ['public_commands_changed', 'models_changed', 'project_format_changed', 'sdk_changed'])
    names = {p.relative_to(folder).as_posix() for p in folder.rglob('*') if p.is_file()}
    assert names == set(checkpoint['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in checkpoint['evidence_sha256'].items():
        assert sha(folder / name) == expected, name
    for name, expected in checkpoint['source_sha256'].items():
        assert sha(ROOT / name) == expected, name
    assert sha(ROOT / 'scripts/check.sh') == checkpoint['quality_runner_sha256']
    for name in ['value_apply.go', 'component_inspector.go']:
        original = subprocess.check_output(['git', 'show', checkpoint['original_source_commit'] + ':desktop/' + name], cwd=ROOT, timeout=15)
        assert original == (folder / ('original-' + name + '.txt')).read_bytes()
    before = read(folder / 'before.json')
    replay = read(folder / 'final-test-original.json')
    assert before['exit_code'] == replay['exit_code'] == 1
    assert replay['failed_cases'] == checkpoint['original_final_test_failures'] == 4
    assert gzip.decompress((folder / 'before-stdout.txt.gz').read_bytes()).count(b'one preset click failed:') == 4
    assert gzip.decompress((folder / 'final-test-original-stdout.txt.gz').read_bytes()).count(b'one preset click differs from the public agent:') == 4
    quality = read(folder / 'quality.json')
    assert quality == checkpoint['source_quality'] and quality['passed'] and quality['exit_code'] == 0
    assert quality['command'] == './scripts/check.sh' and quality['source_sha256'] == checkpoint['source_sha256']
    assert b'Ran 54 tests' in gzip.decompress((folder / 'quality-stderr.txt.gz').read_bytes())
    native = read(folder / 'native-smoke.json')
    assert native == checkpoint['general_native_smoke'] and native['passed']
    assert native['build_exit_code'] == native['native_exit_code'] == 0 and native['owned_process_absent']
    events = [json.loads(line) for line in gzip.decompress((folder / 'native-smoke-stdout.txt.gz').read_bytes()).splitlines()
              if line.startswith(b'{"event":"elab.native.smoke"')]
    assert events == [dict(event='elab.native.smoke', passed=True)]
    assert b'vcs.modified=true' in gzip.decompress((folder / 'native-build-info.txt.gz').read_bytes())
    inventory = read(ROOT / 'research/13-native-command-inventory.json')
    assert len(inventory['source_sha256']) == 69 and len(inventory['sites']) == 393
    assert len(inventory['public_commands']) == 162 and len(inventory['commands_with_source_routes']) == 132
    assert sum(site['status'] == 'requires-binding-review' for site in inventory['sites']) == 167
    assert inventory['issues'] == []
    review = read(folder / 'inventory-review.json')
    assert review['public_commands_unchanged'] and review['aliases_and_routes_unchanged'] and review['issues'] == []
    preservation = read(folder / 'preservation.json')
    receipt_path = ROOT / 'research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json'
    assert sha(receipt_path) == preservation['receipt_sha256']
    sealed = Path(read(receipt_path)['review_path'])
    manifest = read(sealed / 'RELEASE_MANIFEST.json')['files']
    assert len(manifest) == preservation['manifest_entries'] == 788
    assert {p.relative_to(sealed).as_posix() for p in sealed.rglob('*') if p.is_file()} == set(manifest) | {'RELEASE_MANIFEST.json'}
    for name, expected in manifest.items():
        path = sealed / name
        assert not path.is_symlink() and sha(path) == expected['sha256']
        assert oct(stat.S_IMODE(path.stat().st_mode)) == expected['mode']
    assert sha(ROOT / 'build/ELab.app/Contents/MacOS/ELab') == preservation['canonical_native_sha256']
    assert sha(ROOT / 'build/ELab-0.148.0-165-review.zip') == preservation['private_archive_sha256']
    print(json.dumps(dict(source_repair_integrity_verified=True, original_failed_presets=4,
                          source_quality_passed=True, development_native_smoke_passed=True,
                          clean_replacement_qualified=False, owner_acceptance=False)))


if __name__ == '__main__':
    main()
