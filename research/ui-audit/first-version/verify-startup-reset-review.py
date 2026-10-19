"""Read-only admission for clean /168, its evidence, RC data and archive."""
import gzip
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import plistlib
import re
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'scripts'))
from package_review import members, verify_archive
from wiring_replay import validate_release


def read(path):
    data = path.read_bytes()
    return json.loads(gzip.decompress(data) if path.suffix == '.gz' else data)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def preserved(folder, manifest_name, expected):
    manifest = read(folder / manifest_name)
    actual = members(folder)
    assert actual.pop(manifest_name)['sha256'] == expected['manifest_sha256']
    assert len(manifest['files']) == expected['files_verified']
    assert actual == manifest['files'], str(folder)


def main():
    folder = ROOT / 'docs/validation/startup-reset-review-168'
    checkpoint = read(folder / 'checkpoint.json')
    assert {p.relative_to(folder).as_posix() for p in folder.rglob('*') if p.is_file()} == set(checkpoint['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in checkpoint['evidence_sha256'].items():
        assert sha(folder / name) == expected, name
    signature = read(folder / 'signature-preflight.json')
    assert signature == checkpoint['signature_preflight'] and signature['exit_code'] == 0
    assert signature['signature'] == 'adhoc' and signature['team_identifier'] is None
    display = gzip.decompress((folder / 'signature-preflight.txt.gz').read_bytes())
    assert b'Signature=adhoc' in display and b'TeamIdentifier=not set' in display
    receipt_path = ROOT / 'research/ui-audit/first-version/startup-reset-review-checkpoint.json'
    identity = validate_release(ROOT, receipt_path)
    receipt = read(receipt_path)
    assert identity['release'] == '0.151.0 /168' and identity['source_commit'] == '0f9c9842e7297c2f2d9c11a5a368a9f2430a7118'
    assert identity['receipt_sha256'] == checkpoint['receipt_sha256']
    assert identity['runtime_inputs_verified'] == 347 and identity['production_source_files_verified'] == 70
    assert identity['manifest_file_entries_verified'] == 1207 and identity['manifest_sha256'] == checkpoint['manifest_sha256']
    review = Path(receipt['review_path'])
    assert (review / 'Evidence/Acceptance.json').read_bytes() == receipt_path.read_bytes()
    info = plistlib.loads((review / 'ELab.app/Contents/Info.plist').read_bytes())
    assert info['CFBundleShortVersionString'] == '0.151.0' and info['CFBundleVersion'] == '168'
    assert not receipt['canonical_promoted'] and receipt['owner_acceptance'].startswith('Pending:')
    assert receipt['p0_counts'] == dict(implemented=17, partial=10)
    proof = ROOT / 'docs/validation/verification-startup-reset'
    source = read(proof / 'checkpoint.json')
    assert sha(proof / 'checkpoint.json') == checkpoint['source_proof_checkpoint_sha256']
    assert {k:v for k,v in receipt['quality'].items() if k != 'scope'} == source['source_quality']
    for name, expected in receipt['quality']['source_sha256'].items():
        assert sha(ROOT / name) == expected, name
    for path in proof.rglob('*'):
        if path.is_file():
            assert path.read_bytes() == (review / 'Evidence/Startup-reset-source-repair' / path.relative_to(proof)).read_bytes()
    assert source['original_failed_leaf_cases'] == receipt['startup_reset']['original_failed_leaf_cases'] == 4
    runtime = read(folder / 'runtime.json')
    assert runtime == read(review / 'Evidence/Runtime.json') and runtime['passed']
    assert runtime['copied_bundle_unchanged'] and runtime['agent_unchanged'] and runtime['controller_unchanged']
    assert runtime['shared_geometry_unchanged'] and runtime['shared_geometry_before'] == runtime['shared_geometry_after']
    assert runtime['stages'] == receipt['copied_runtime']
    assert [s['name'] for s in runtime['stages']] == ['native-smoke', 'public-suites']
    assert all(s['status'] == 'completed' and s['exit_code'] == 0 for s in runtime['stages'])
    assert runtime['controls']['allowed_read']
    for category in ['files', 'network', 'shared_geometry']:
        assert all(row['denied'] and row['errno'] in [1,13] for row in runtime['controls'][category].values())
    public = read(folder / 'public-report.json.gz')
    assert public == read(review / 'Evidence/Public-report.json.gz') and public['passed']
    assert [a['name'] for a in public['adapters']] == ['in-process', 'headless-stdio', 'headless-socket', 'native-socket']
    acceptance = read(folder / 'acceptance/report.json')
    assert acceptance == receipt['acceptance'] and acceptance['passed'] and acceptance['native_host_closed_normally'] and acceptance['shared_geometry_unchanged']
    vr = acceptance['verification_handoff_commands']
    assert vr == receipt['verification_regression_commands'] and vr['passed'] and vr['native_headless_rows_equal'] and vr['sample_rows_per_adapter'] == 424
    assert [c['name'] for c in vr['cases']] == ['duration', 'sample-step', 'default-voltage', 'capacitor-voltage']
    spec = importlib.util.spec_from_file_location('verify_parity', ROOT / 'scripts/first-version-parity.py')
    parity = importlib.util.module_from_spec(spec);spec.loader.exec_module(parity)
    rows_total = 0
    for case in vr['cases']:
        name = case['name']
        native = read(folder / ('acceptance/native-verify-' + name + '.json.gz'))
        headless = read(folder / ('acceptance/headless-verify-' + name + '.json.gz'))
        assert native['case'] == headless['case'] == name and native['settings'] == headless['settings']
        for key in ['graph','settings','input_mode','analysis','engine','probes','duration_s','sample_count']:
            assert native['metadata'][key] == headless['metadata'][key], (name,key)
        parity.compare(native['metadata']['statistics'], headless['metadata']['statistics'])
        assert native['probes'] == headless['probes'] and native['rows'] == headless['rows']
        assert len(native['rows']) == case['rows'] == native['metadata']['sample_count']
        rows_total += len(native['rows'])
        initial = {'default-voltage':2., 'capacitor-voltage':3.}.get(name,0.)
        voltage = next(i for i,p in enumerate(native['probes']) if p['id'] in ['v:C1','v:C1.1'])
        current = next(i for i,p in enumerate(native['probes']) if p['id'] == 'i:R1')
        errors = []
        for row in native['rows']:
            value = 5 + (initial-5)*math.exp(-row['time_s']/.01)
            error = abs(row['values'][voltage]-value);errors.append(error)
            assert error <= 2e-12 and abs(row['values'][current]-(5-value)/10000) <= 2e-15
        assert max(errors) == case['max_voltage_error_v'] == native['max_voltage_error_v']
        for result in [native, headless]:
            for key in ['verification_settings','components','wires','transient','run_id','electrical_revision','document','undo_entry','redo_entry']:
                assert result['refused'].get(key) == result['accepted'].get(key), (name,key)
            assert result['stale']['verification_settings']['input_mode'] == 'discharge'
            for key in ['components','wires','transient','run_id','electrical_revision','undo_entry','redo_entry']:
                assert result['before'].get(key) == result['after'].get(key), (name,key)
            assert result['reset']['verification_settings'] == dict(duration_s=result['settings']['duration_s'], sample_step_s=0, input_mode='auto', capacitor_initials=[])
    assert rows_total == 424
    coil = acceptance['coil_startup_reset_commands']
    assert coil == receipt['startup_reset']['copied_commands'] and coil['passed'] and coil['native_headless_rows_equal']
    assert coil['four_fresh_saved_project_reopens'] and coil['live_and_saved_startup_and_history_preserved']
    assert [c['example'] for c in coil['cases']] == ['rl','rlc']
    for case in coil['cases']:
        name = case['example']
        native = read(folder / ('acceptance/native-coil-' + name + '.json.gz'))
        headless = read(folder / ('acceptance/headless-coil-' + name + '.json.gz'))
        for phase,current in [('before',.03),('after',-.02)]:
            a,b = native[phase],headless[phase]
            for key in ['graph','settings','engine','input_mode','probes','duration_s','sample_count']:
                assert a['metadata'][key] == b['metadata'][key], (name,phase,key)
            parity.compare(a['metadata']['statistics'],b['metadata']['statistics'])
            assert a['probes'] == b['probes'] and a['rows'] == b['rows']
            assert len(a['rows']) == case[phase+'_rows']
            assert abs(a['rows'][0]['values'][0]-current) < 1e-12
        for outcome in [native,headless]:
            assert outcome['fresh_recordings_equal']
            for key in ['components','wires','transient','mixed','run_id','electrical_revision','undo_entry','redo_entry']:
                assert outcome['state'].get(key) == outcome['live'].get(key) == outcome['reset'].get(key), (name,key)
            assert outcome['reset']['verification_settings'] == dict(duration_s=.005,sample_step_s=0,input_mode='auto',capacitor_initials=[])
            reopened = outcome['reopened']
            assert reopened['paused'] and reopened['transient']['sample']['time_s'] == 0 and not reopened['document']['dirty'] and reopened['document']['recording_count'] == 2
            assert next(p['initial_current_a'] for p in reopened['components'] if p['id']=='L1') == -.02
            if name == 'rl':
                errors=[]
                for row in outcome['after']['rows']:
                    expected=.05+(-.02-.05)*math.exp(-row['time_s']/.01)
                    errors.append(abs(row['values'][0]-expected))
                    assert errors[-1] <= 1e-5 and abs(row['values'][1]-(5-100*expected)) <= .001
                assert max(errors) == outcome['rl_max_current_error_a'] == case['rl_max_current_error_a']
    assert acceptance['preset_value_commands']['values'] == 4 and acceptance['live_grid']['final_time_s'] >= 3.41 and acceptance['live_grid']['paused_physics_stable']
    for name in ['value_correction','live_measurement_ownership','recorded_cursors','recording_targets']:
        assert acceptance[name]['passed']
    assert all(c['native_endpoint_matches'] for c in acceptance['reference_parity']['cases'])
    for path in (folder / 'acceptance').rglob('*'):
        if path.is_file():
            assert path.read_bytes() == (review / 'Evidence/Runtime-acceptance' / path.relative_to(folder / 'acceptance')).read_bytes()
    preservation = read(folder / 'preservation.json')
    assert len(preservation['preserved_bundles']) == len(receipt['preserved_bundles']) == 27
    assert preservation['preserved_bundles'] == receipt['preserved_bundles']
    for label, expected in receipt['preserved_bundles'].items():
        preserved(ROOT / 'build' / ('ELab-' + label + '-review'), 'RELEASE_MANIFEST.json', expected)
    preserved(ROOT / 'build/ELab-0.130.0-147-review', 'CANDIDATE_MANIFEST.json', receipt['preserved_original_147_candidate'])
    preserved(ROOT / 'build/ELab-0.147.0-164-review', 'CANDIDATE_MANIFEST.json', receipt['preserved_failed_164_candidate'])
    assert sha(ROOT / 'build/ELab.app/Contents/MacOS/ELab') == receipt['canonical']['native_sha256']
    archive = read(folder / 'archive/package-report.json.gz')
    assert archive['passed'] and archive['source_review_unchanged'] and archive['files_verified'] == 1208 and archive['archive_bytes'] == 54326611
    archive_path = ROOT / 'build/ELab-0.151.0-168-review.zip'
    assert sha(archive_path) == archive['archive_sha256'] == '7c4c914ff58ce100e912aef663026432bc5df8b9c8bd66347f2f5d70f6ac005f'
    assert members(review) == archive['expected_files']
    verify_archive(archive_path, review.name, archive['expected_files'])
    assert (ROOT / 'build/ELab-0.151.0-168-review.zip.sha256').read_text() == archive['archive_sha256']+'  ELab-0.151.0-168-review.zip\n'
    extracted = read(folder / 'archive/example-report.json')
    assert extracted['passed'] and extracted['agent_sha256'] == identity['agent_sha256']
    assert len(extracted['cases']) == 3 and all(c['fixture_unchanged'] and c['saved_reopen_fresh'] for c in extracted['cases'])
    readme = (ROOT / 'README.md').read_text()
    assert re.findall(r'\*\*Private review ready: ([0-9.]+) /([0-9]+)\.\*\*', readme) == [('0.151.0','168')]
    assert '](build/ELab-0.151.0-168-review/ELab.app)' in readme
    assert '](research/ui-audit/first-version/startup-reset-review-checkpoint.json)' in readme
    print(json.dumps(dict(passed=True,release=identity['release'],source_commit=identity['source_commit'],manifest_entries=1207,runtime_source_inputs=347,previous_reviews=27,verify_rows_per_adapter=424,archive_files=1208,owner_acceptance=False,canonical_promoted=False)))


if __name__ == '__main__':
    main()
