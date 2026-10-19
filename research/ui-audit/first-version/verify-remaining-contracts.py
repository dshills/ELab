"""Read-only proof of the remaining application discovery annotations."""

import gzip
import hashlib
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'scripts'))
from package_review import members


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def raw(folder, name):
    path = folder / name
    return path.read_bytes() if path.exists() else gzip.decompress((folder / (name + '.gz')).read_bytes())


def read(folder, name):
    return json.loads(raw(folder, name))


def main():
    folder = ROOT / 'docs/validation/remaining-contracts'
    cp = read(folder, 'checkpoint.json')
    assert cp['status'] == 'source_metadata_validated_clean_package_pending'
    assert cp['annotated_commands_before'] == 142 and cp['annotated_commands_after'] == 162
    for key in ['ordinary_catalog_changed', 'public_command_ids_changed', 'handlers_changed', 'models_changed', 'project_format_changed', 'sdk_changed', 'new_release_created', 'canonical_promoted', 'owner_acceptance']:
        assert cp[key] is False
    assert {p.relative_to(folder).as_posix() for p in folder.rglob('*') if p.is_file()} == set(cp['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in cp['evidence_sha256'].items():
        assert sha(folder / name) == expected, name
    for name, expected in cp['source_sha256'].items():
        assert sha(ROOT / name) == expected, name
    assert sha(ROOT / 'scripts/check.sh') == cp['quality_runner_sha256']
    assert sha(ROOT / 'third_party/vellum/sdk.lock.json') == cp['sdk_lock_sha256']
    assert sha(ROOT / 'research/13-native-command-inventory.json') == cp['native_ledger_sha256']
    before = read(folder, 'before.json')
    assert before['source_commit'] == 'cbbc97136978999e708433d1c6ab11a95ed606d4'
    assert before['exit_code'] == 1 and before['original_failure_reproduced'] and before['missing_contracts'] == 20
    for name, expected in before['source_sha256'].items():
        original = raw(folder, 'original-' + Path(name).name + '.txt')
        assert hashlib.sha256(original).hexdigest() == expected
        if not name.endswith('_test.go'):
            assert original == subprocess.check_output(['git', 'show', before['source_commit'] + ':' + name], cwd=ROOT, timeout=15)
    assert raw(folder, 'before-stdout.txt').count(b'missing application contract') == 20
    replay = read(folder, 'final-test-original.json')
    assert replay['exit_code'] == 1 and replay['original_failure_reproduced'] and replay['missing_contracts'] == 20
    assert raw(folder, 'final-test-original-stdout.txt').count(b'missing application contract') == 20
    focused = read(folder, 'focused.json')
    assert focused['passed'] and focused['exit_code'] == 0 and sha(ROOT / 'workbench/application_capabilities_test.go') == focused['test_sha256']
    names = [line.split('--- PASS: ', 1)[1].split(' (', 1)[0] for line in raw(folder, 'focused-stdout.txt').decode().splitlines() if '--- PASS: ' in line]
    assert sum(not any(other.startswith(name + '/') for other in names) for name in names) == 31
    quality = read(folder, 'quality.json')
    assert quality == cp['source_quality'] and quality['passed'] and quality['exit_code'] == 0 and quality['source_sha256'] == cp['source_sha256']
    assert b'Ran 54 tests' in raw(folder, 'quality-stderr.txt')
    baseline_info = raw(folder, 'baseline-agent-build-info.txt')
    assert b'vcs.modified=false' in baseline_info and before['source_commit'].encode() in baseline_info
    assert b'vcs.modified=true' in raw(folder, 'elab-agent-build-info.txt')
    external = read(folder, 'external.json')
    assert external['passed'] and external['socket_host_exit_code'] == 0 and external['socket_host_absent']
    assert external['contracts_before'] == 142 and external['contracts_after'] == external['public_commands'] == 162 and external['remaining_without_contract'] == []
    baseline, stdio, socket = [read(folder, name + '.json') for name in ['baseline', 'stdio', 'socket']]
    assert baseline['catalog'] == stdio['catalog'] == socket['catalog'] and stdio['details'] == socket['details']
    old, current = baseline['details'], stdio['details']
    assert set(old) == set(current) and len(current) == 162
    assert sum('x-command-contract' in c['input_schema'] for c in old.values()) == 142
    assert all('x-command-contract' in c['input_schema'] for c in current.values())
    changed = [name for name in current if current[name] != old[name]]
    assert changed == external['detailed_commands_changed'] and len(changed) == 20
    assert set(changed) == {name for name in old if 'x-command-contract' not in old[name]['input_schema']}
    for name in current:
        assert old[name]['output_schema'] == current[name]['output_schema'] and old[name]['mutation'] == current[name]['mutation']
    for label, data in [('stdio', stdio), ('socket', socket)]:
        assert not data['idle']['changed'] and not data['idle']['timed_out'] and data['idle']['coalesced'] and 'snapshot' not in data['idle']
        assert data['frame']['phase'] == 0 and data['event']['changed']
        assert data['detailed']['document'] == data['before']['document'] and data['detailed']['diagnostic_presentation']['details_visible']
        display = data['display']
        assert display['document']['dirty'] and display['undo_entry']['name'] == 'Change display preferences' and not display['display']['animate']
        assert display['display']['preferences']['direction'] == 'electron'
        for state in data['theme_stages']:
            assert state['appearance']['status'] == 'ready' and state['appearance']['saved_theme'] == state['theme']
            for key in ['components', 'wires', 'result', 'run_id', 'electrical_revision', 'document', 'undo_entry', 'redo_entry']:
                assert state.get(key) == display.get(key), (label, key)
        assert data['disabled']['appearance']['status'] == 'disabled'
        assert data['persistence_error']['appearance']['status'] == 'error' and data['persistence_error']['appearance']['error']
        assert [s['example']['info']['id'] for s in data['aliases']] == ['led', 'rc', 'npn', 'blink', 'flasher', 'switch-npn', 'rectifier']
        assert data['reset']['paused'] and data['reset']['components'] == data['original']['components'] and data['reset']['document']['generation'] > data['original']['document']['generation']
        assert len(data['disconnected']['wires']) == len(data['connected']['wires']) - 1
        for key in ['wires', 'run_id', 'undo_entry']:
            assert data['no_op'][key] == data['disconnected'][key]
        assert len(data['restored']['wires']) == len(data['connected']['wires'])
        private = folder / 'private-files' / label
        saved = read(private, 'manual-project.elab')
        assert saved['graph']['components'] == data['before']['components'] and saved['graph']['wires'] == data['before']['wires']
        assert read(private, 'appearance.json')['theme'] == data['theme_stages'][-1]['theme']
        assert raw(private, 'corrupt-appearance.json') == b'not JSON' and data['manual_save_unchanged']
    preservation = read(folder, 'preservation.json')
    assert preservation['passed'] and preservation['old_release_admission_refused'] and len(preservation['sealed_reviews']) == 29
    assert sha(ROOT / 'research/ui-audit/first-version/component-startup-review-checkpoint.json') == preservation['receipt_sha256']
    for name, report in preservation['sealed_reviews'].items():
        sealed = ROOT / 'build' / name
        manifest = json.loads((sealed / 'RELEASE_MANIFEST.json').read_text())
        actual = members(sealed)
        assert actual.pop('RELEASE_MANIFEST.json')['sha256'] == report['manifest_sha256']
        assert actual == manifest['files'] and len(actual) == report['entries']
    assert sha(ROOT / 'build/ELab-0.152.0-169-review.zip') == preservation['current_archive_sha256']
    assert sha(ROOT / 'build/ELab.app/Contents/MacOS/ELab') == preservation['canonical_native_sha256']
    print(json.dumps(dict(remaining_contracts_verified=True, original_missing_contracts=20, focused_passed_leaf_cases=31, annotated_commands=162, annotation_omissions=0, ordinary_catalog_unchanged=True, complete_behavior_acceptance=False, clean_package_qualified=False)))


if __name__ == '__main__':
    main()
