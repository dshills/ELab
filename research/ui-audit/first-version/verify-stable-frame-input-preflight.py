"""Read-only verification of a failed input-session setup and normal cleanup."""
import ast
import gzip
import hashlib
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'scripts'))
from wiring_replay import validate_release


def read(path):
    data = path.read_bytes()
    return json.loads(gzip.decompress(data) if path.suffix == '.gz' else data)


def main():
    folder = ROOT / 'docs/validation/stable-frame-input-preflight'
    checkpoint = read(folder / 'checkpoint.json')
    assert checkpoint['status'] == 'stable_input_setup_not_qualified'
    assert not checkpoint['capture_executed'] and not checkpoint['test_input_operated']
    assert checkpoint['eligible_latency_observations'] == 0 and checkpoint['latency_gate_passed'] is None
    assert not checkpoint['root_cause_established'] and not checkpoint['ordinary_startup_qualified']
    assert not checkpoint['owner_acceptance'] and checkpoint['sealed_review_unchanged']
    build = checkpoint['diagnostic_build']
    assert build['exit_code'] == 0 and build['swift_mode'] == 6 and build['warnings_as_errors']
    assert not build['capture_executed']
    names = {p.relative_to(folder).as_posix() for p in folder.rglob('*') if p.is_file()}
    assert names == set(checkpoint['evidence_sha256']) | {'checkpoint.json'}
    assert not any('endpoint.json' == p.name for p in folder.rglob('*'))
    for name, expected in checkpoint['evidence_sha256'].items():
        assert hashlib.sha256((folder / name).read_bytes()).hexdigest() == expected, name
    identity = validate_release(ROOT, ROOT / 'research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json')
    assert identity['receipt_sha256'] == checkpoint['release_receipt_sha256']
    receipt = read(ROOT / 'research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json')
    assert receipt['native_sha256'] == checkpoint['native_sha256']
    profiles = {}
    for case in ['network-denied', 'network-default']:
        ast.parse((folder / ('session-' + case + '.py.txt')).read_text())
        base = folder / case
        report = read(base / 'session-report.json')
        assert report['release'] == identity and report['status'] == 'closed_normally'
        assert report['native_exit_code'] == report['final_native_exit_code'] == 0
        assert not report['forced_cleanup'] and report['owned_process_absent'] and report['owned_process_absent_after_cleanup']
        assert report['endpoint_removed'] and report['sealed_review_unchanged']
        assert report['shared_geometry_before'] == report['shared_geometry_after']
        assert all(row['denied'] and row['errno'] in [1, 13] for row in report['controls'].values())
        assert report['controls'] == read(base / 'permission-probe-stdout.txt.gz')
        assert report['controls'] == checkpoint['cases'][case]['controls']
        assert '--smoke' not in report['argv']
        for flag in ['--agent-endpoint', '--recovery-directory', '--appearance-file']:
            assert report['argv'][report['argv'].index(flag) + 1].startswith(report['stage'] + '/')
        baseline, final, expected = [read(base / (name + '.json.gz')) for name in ['baseline', 'final', 'agent-expected']]
        for key in ['components', 'wires', 'result']:
            assert baseline[key] == final[key], (case, key)
        assert next(part['value'] for part in final['components'] if part['id'] == 'R1') == 1000
        assert next(part['value'] for part in expected['components'] if part['id'] == 'R1') == 2000
        assert final['result']['current_a'] == .0029167965774405296 and not final['result']['stale']
        assert expected['result']['current_a'] == .001476723549138745 and not expected['result']['stale']
        assert read(base / 'baseline.elab')['graph'] == read(base / 'final.elab')['graph']
        profiles[case] = (base / 'session.sb').read_text()
    assert profiles['network-default'] == profiles['network-denied'].replace('(deny network-outbound (remote ip "*:*"))\n', '')
    first = read(folder / 'network-denied/registration-observation.json')['observation']
    second = read(folder / 'network-default/registration-stdout.txt.gz')
    assert [first['pid'], second['pid']] == [26624, 27806]
    for observed in [first, second]:
        assert observed['process_registered'] and not observed['terminated']
        assert observed['bundle_identifier'] is None and not observed['capture_executed']
    assert checkpoint['recorded_refusal']['exit_code'] == 1
    assert checkpoint['all_matching_native_processes_absent']
    print(json.dumps(dict(preflight_integrity_verified=True, input_setup_qualified=False,
                          normal_closes=2, capture_executed=False,
                          eligible_latency_observations=0, latency_gate_passed=None)))


if __name__ == '__main__':
    main()
