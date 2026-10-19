"""Read-only /166 package qualification; source/mounted and runtime scopes differ."""
import gzip
import hashlib
import importlib.util
import json
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


def main():
    folder = ROOT / 'docs/validation/preset-review-166'
    checkpoint = read(folder / 'checkpoint.json')
    names = {p.relative_to(folder).as_posix() for p in folder.rglob('*') if p.is_file()}
    assert names == set(checkpoint['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in checkpoint['evidence_sha256'].items():
        assert sha(folder / name) == expected, name
    receipt_path = ROOT / 'research/ui-audit/first-version/value-preset-review-checkpoint.json'
    identity = validate_release(ROOT, receipt_path)
    receipt = read(receipt_path)
    assert identity['release'] == '0.149.0 /166' and identity['source_commit'] == 'c28c564919e61fe9b76de148337c5ba9b332e11d'
    assert identity['receipt_sha256'] == checkpoint['receipt_sha256']
    review = Path(receipt['review_path'])
    assert (review / 'Evidence/Acceptance.json').read_bytes() == receipt_path.read_bytes()
    info = plistlib.loads((review / 'ELab.app/Contents/Info.plist').read_bytes())
    assert info['CFBundleShortVersionString'] == '0.149.0' and info['CFBundleVersion'] == '166'
    assert not receipt['canonical_promoted'] and receipt['owner_acceptance'].startswith('Pending:')
    assert receipt['p0_counts'] == dict(implemented=17, partial=10)
    for name, expected in receipt['quality']['source_sha256'].items():
        assert sha(ROOT / name) == expected, name
    assert receipt['quality']['passed'] and receipt['quality']['exit_code'] == 0
    runtime = read(folder / 'runtime.json')
    assert runtime == read(review / 'Evidence/Runtime.json') and runtime['passed']
    assert runtime['copied_bundle_unchanged'] and runtime['agent_unchanged'] and runtime['controller_unchanged']
    assert runtime['shared_geometry_unchanged'] and runtime['shared_geometry_before'] == runtime['shared_geometry_after']
    assert runtime['stages'] == receipt['copied_runtime']
    assert [s['name'] for s in runtime['stages']] == ['native-smoke', 'public-suites']
    assert all(s['status'] == 'completed' and s['exit_code'] == 0 for s in runtime['stages'])
    assert runtime['controls']['allowed_read']
    for category in ['files', 'network', 'shared_geometry']:
        assert all(row['denied'] and row['errno'] in [1, 13] for row in runtime['controls'][category].values())
    public = read(folder / 'public-report.json.gz')
    assert public == read(review / 'Evidence/Public-report.json.gz') and public['passed']
    assert [a['name'] for a in public['adapters']] == ['in-process', 'headless-stdio', 'headless-socket', 'native-socket']
    acceptance = read(folder / 'acceptance/report.json')
    assert acceptance == receipt['acceptance'] and acceptance['passed'] and acceptance['native_host_closed_normally']
    assert acceptance['shared_geometry_unchanged'] and acceptance['preset_value_commands']['values'] == 4
    assert acceptance['preset_value_commands']['native_headless_stage_matches'] == 8
    assert acceptance['live_grid']['final_time_s'] >= 3.41 and acceptance['live_grid']['paused_physics_stable']
    for name in ['value_correction', 'live_measurement_ownership', 'recorded_cursors', 'recording_targets']:
        assert acceptance[name]['passed']
    assert all(case['native_endpoint_matches'] for case in acceptance['reference_parity']['cases'])
    spec = importlib.util.spec_from_file_location('preset_parity', ROOT / 'scripts/first-version-parity.py')
    parity = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(parity)
    for control in ['r-1k', 'r-2k', 'c-1u', 'c-100u']:
        native = read(folder / ('acceptance/native-preset-values-' + control + '.json.gz'))
        headless = read(folder / ('acceptance/headless-preset-values-' + control + '.json.gz'))
        assert native['control'] == headless['control'] == control
        for phase in ['accepted', 'undone']:
            for key in ['components', 'wires', 'nets', 'drawing', 'paused']:
                assert native[phase][key] == headless[phase][key], (control, phase, key)
            for key in ['result', 'transient']:
                parity.compare(native[phase].get(key), headless[phase].get(key), control + '.' + phase + '.' + key)
    for path in (folder / 'acceptance').rglob('*'):
        if path.is_file():
            assert path.read_bytes() == (review / 'Evidence/Runtime-acceptance' / path.relative_to(folder / 'acceptance')).read_bytes()
    preserved = read(folder / 'preservation.json')
    assert len(preserved['preserved_bundles']) == len(receipt['preserved_bundles']) == 25
    assert preserved['preserved_bundles'] == receipt['preserved_bundles']
    for label, expected in preserved['preserved_bundles'].items():
        assert sha(ROOT / 'build' / ('ELab-' + label + '-review') / 'RELEASE_MANIFEST.json') == expected['manifest_sha256']
    assert sha(ROOT / 'build/ELab.app/Contents/MacOS/ELab') == receipt['canonical']['native_sha256']
    archive = read(folder / 'archive/package-report.json.gz')
    assert archive['passed'] and archive['source_review_unchanged'] and archive['files_verified'] == 901
    assert archive['archive_bytes'] == 48570118
    archive_path = ROOT / 'build/ELab-0.149.0-166-review.zip'
    assert sha(archive_path) == archive['archive_sha256'] == '6f7f9d05e97ce1f61b6944dcfa9b47c7b46e39e79b823834a708868c34c70465'
    assert members(review) == archive['expected_files']
    verify_archive(archive_path, review.name, archive['expected_files'])
    extracted = read(folder / 'archive/example-report.json')
    assert extracted['passed'] and extracted['agent_sha256'] == identity['agent_sha256']
    assert len(extracted['cases']) == 3 and all(case['fixture_unchanged'] and case['saved_reopen_fresh'] for case in extracted['cases'])
    readme = (ROOT / 'README.md').read_text()
    assert re.findall(r'\*\*Private review ready: ([0-9.]+) /([0-9]+)\.\*\*', readme) == [('0.149.0', '166')]
    assert '](build/ELab-0.149.0-166-review/ELab.app)' in readme
    assert '](research/ui-audit/first-version/value-preset-review-checkpoint.json)' in readme
    assert 'python3 -B scripts/prepare-vellum.py' in readme and '"$ELAB_SDK_CACHE/scripts/package-app.py"' in readme
    assert './build/elab --smoke --smoke-timeout 2m' in readme
    print(json.dumps(dict(passed=True, release=identity['release'], source_commit=identity['source_commit'],
                          manifest_entries=900, previous_reviews=25, preset_values=4,
                          copied_native_headless_stages=8, archive_files=901,
                          owner_acceptance=False, canonical_promoted=False)))


if __name__ == '__main__':
    main()
