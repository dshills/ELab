"""Read-only navigation discovery source/protocol proof; not release admission."""

import gzip
import hashlib
import json
from pathlib import Path
import stat
import subprocess

ROOT = Path(__file__).resolve().parents[3]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def raw(folder, name):
    path = folder / name
    return path.read_bytes() if path.exists() else gzip.decompress((folder / (name + '.gz')).read_bytes())


def read(folder, name):
    return json.loads(raw(folder, name))


def main():
    folder = ROOT / 'docs/validation/navigation-contracts'
    cp = read(folder, 'checkpoint.json')
    assert cp['status'] == 'source_metadata_validated_clean_package_pending'
    assert not any(cp[key] for key in ['ordinary_catalog_changed', 'public_command_ids_changed', 'handlers_changed', 'models_changed', 'project_format_changed', 'sdk_changed', 'new_release_created', 'canonical_promoted', 'owner_acceptance'])
    assert {p.relative_to(folder).as_posix() for p in folder.rglob('*') if p.is_file()} == set(cp['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in cp['evidence_sha256'].items():
        assert sha(folder / name) == expected, name
    for name, expected in cp['source_sha256'].items():
        assert sha(ROOT / name) == expected, name
    assert sha(ROOT / 'scripts/check.sh') == cp['quality_runner_sha256']
    assert sha(ROOT / 'third_party/vellum/sdk.lock.json') == cp['sdk_lock_sha256']
    assert sha(ROOT / 'research/13-native-command-inventory.json') == cp['native_ledger_sha256']
    before = read(folder, 'before.json')
    assert before['exit_code'] == 1 and before['original_failure_reproduced']
    for name, expected in before['source_sha256'].items():
        original = raw(folder, 'original-' + Path(name).name + '.txt')
        assert hashlib.sha256(original).hexdigest() == expected
        if not name.endswith('_test.go'):
            assert original == subprocess.check_output(['git', 'show', before['source_commit'] + ':' + name], cwd=ROOT, timeout=15)
    assert raw(folder, 'before-stdout.txt').count(b'missing navigation contract') == 4
    replay = read(folder, 'final-test-original.json')
    assert replay['exit_code'] == 1 and replay['original_failure_reproduced'] and replay['missing_contract_cases'] == 4
    assert raw(folder, 'final-test-original-stdout.txt').count(b'missing navigation contract') == 4
    focused = read(folder, 'focused.json')
    assert focused['passed'] and focused['exit_code'] == 0 and sha(ROOT / 'workbench/navigation_capabilities_test.go') == focused['test_sha256']
    names = [line.split('--- PASS: ', 1)[1].split(' (', 1)[0] for line in raw(folder, 'focused-stdout.txt').decode().splitlines() if '--- PASS: ' in line]
    assert sum(not any(other.startswith(name + '/') for other in names) for name in names) == 26
    quality = read(folder, 'quality.json')
    assert quality == cp['source_quality'] and quality['passed'] and quality['exit_code'] == 0 and quality['source_sha256'] == cp['source_sha256']
    assert b'Ran 54 tests' in raw(folder, 'quality-stderr.txt')
    assert b'vcs.modified=true' in raw(folder, 'development-agent-build-info.txt')
    external = read(folder, 'external.json')
    assert external['passed'] and external['socket_host_exit_code'] == 0 and external['socket_host_absent']
    assert external['contracts_before'] == 138 and external['contracts_after'] == 142 and external['public_commands'] == 162
    assert len(external['remaining_without_contract']) == 20
    stdio, socket = read(folder, 'stdio.json'), read(folder, 'socket.json')
    assert stdio['catalog'] == socket['catalog'] and stdio['details'] == socket['details']
    baseline = read(folder, 'baseline-detailed.json')
    current = stdio['details']
    changed = [name for name in baseline if baseline[name] != current[name]]
    assert changed == external['detailed_commands_changed'] == ['selection.navigate', 'view.selection.center', 'view.pan', 'view.zoom']
    assert set(baseline) == set(current) and len(current) == 162
    assert sum('x-command-contract' in c['input_schema'] for c in baseline.values()) == 138
    assert sum('x-command-contract' in c['input_schema'] for c in current.values()) == 142
    for name in current:
        assert baseline[name]['output_schema'] == current[name]['output_schema'] and baseline[name]['mutation'] == current[name]['mutation']
    for observation in [stdio, socket]:
        before = observation['before']
        for phase in ['selected', 'centered', 'panned', 'zoomed']:
            state = observation[phase]
            for key in ['components', 'wires', 'transient', 'mixed', 'run_id', 'electrical_revision', 'document', 'undo_entry', 'redo_entry']:
                assert state.get(key) == before.get(key), (phase, key)
        assert observation['selected']['selected'] == before['wires'][0]['id']
        assert observation['selected']['view'] == before['view'] and not observation['centered']['viewport']['following']
        assert observation['zoomed']['view']['center'] == observation['panned']['view']['center']
        assert observation['zoomed']['view']['zoom'] == min(4, observation['panned']['view']['zoom'] * 2)
    preservation = read(folder, 'preservation.json')
    assert preservation['passed'] and preservation['old_release_admission_refused']
    assert len(preservation['sealed_reviews']) == 29
    assert sha(ROOT / 'research/ui-audit/first-version/component-startup-review-checkpoint.json') == preservation['receipt_sha256']
    for name, report in preservation['sealed_reviews'].items():
        sealed = ROOT / 'build' / name
        manifest_path = sealed / 'RELEASE_MANIFEST.json'
        assert sha(manifest_path) == report['manifest_sha256']
        files = json.loads(manifest_path.read_text())['files']
        assert len(files) == report['entries']
        assert {p.relative_to(sealed).as_posix() for p in sealed.rglob('*') if p.is_file()} == set(files) | {'RELEASE_MANIFEST.json'}
        for name, expected in files.items():
            path = sealed / name
            assert not path.is_symlink() and sha(path) == expected['sha256'] and oct(stat.S_IMODE(path.stat().st_mode)) == expected['mode']
    assert sha(ROOT / 'build/ELab-0.152.0-169-review.zip') == preservation['current_archive_sha256']
    assert sha(ROOT / 'build/ELab.app/Contents/MacOS/ELab') == preservation['canonical_native_sha256']
    print(json.dumps(dict(navigation_discovery_verified=True, original_missing_contracts=4, focused_passed_leaf_cases=26, annotated_commands=142, remaining_without_contract=20, ordinary_catalog_unchanged=True, clean_package_qualified=False, owner_acceptance=False)))


if __name__ == '__main__':
    main()
