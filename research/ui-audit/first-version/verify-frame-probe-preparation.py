"""Read-only preparation integrity; no capture or runtime qualification."""
import gzip
import hashlib
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'scripts'))
from wiring_replay import validate_release


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    folder = ROOT / 'docs/validation/frame-probe-preparation'
    checkpoint = json.loads((folder / 'checkpoint.json').read_text())
    assert checkpoint['status'] == 'offline_checks_passed'
    assert checkpoint['capture_executed'] is False
    assert checkpoint['screen_permission_requested'] is False
    assert checkpoint['alternate_capture_authorization_received'] is True
    assert checkpoint['eligible_latency_observations'] == 0
    assert checkpoint['latency_gate_passed'] is None
    assert set(checkpoint['source_sha256']) == {
        'tools/frame-probe/main.swift', 'tools/frame-probe/check.sh',
        'tools/frame-probe/README.md',
    }
    for name, expected in checkpoint['source_sha256'].items():
        assert digest(ROOT / name) == expected, name
    names = {p.relative_to(folder).as_posix() for p in folder.rglob('*') if p.is_file()}
    assert names == set(checkpoint['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in checkpoint['evidence_sha256'].items():
        assert digest(folder / name) == expected, name
    assert json.loads((folder / 'check-stdout.txt').read_text()) == {
        'self_tests_passed': True, 'capture_executed': False,
        'eligible_latency_observations': 0,
    }
    assert (folder / 'check-stderr.txt').read_bytes() == b''
    check = checkpoint['pure_check']
    assert check['exit_code'] == 0 and check['duration_s'] > 0
    assert check['swift_mode'] == 6 and check['warnings_as_errors'] is True
    assert checkpoint['compiler_output_artifact'] == 'compiler-version.txt.gz'
    compiler = gzip.decompress((folder / checkpoint['compiler_output_artifact']).read_bytes()).decode()
    assert 'Apple Swift version 6.4' in compiler
    assert 'Capture requires prior human authorization' in (folder / 'help.txt').read_text()
    assert checkpoint['initial_pure_location_failure']['exit_code'] is None
    assert checkpoint['sdk_header']['capture_runtime_semantics_validated'] is False
    assert checkpoint['cua_api_preflight']['native_input_or_frame_timestamps_exposed'] is False
    receipt = ROOT / 'research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json'
    identity = validate_release(ROOT, receipt)
    review = checkpoint['elab_review']
    assert identity['receipt_sha256'] == review['receipt_sha256']
    assert identity['release'] == review['version'] + ' /' + review['build'] == '0.148.0 /165'
    qualified = json.loads(receipt.read_text())
    assert qualified['native_sha256'] == review['native_sha256']
    assert qualified['source_commit'] == review['source_commit']
    assert review['canonical_promoted'] is False
    # The compiled tool/header digests are historical provenance, not current
    # installation requirements. A later compiler rebuild may differ in bytes.
    print(json.dumps(dict(preparation_integrity_verified=True, capture_executed=False,
                          eligible_latency_observations=0, latency_gate_passed=None)))


if __name__ == '__main__':
    main()
