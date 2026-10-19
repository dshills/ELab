"""Read-only artifact binding and all-image replay; creates no UI or recordings."""
from pathlib import Path
import gzip
import hashlib
import json
import subprocess
import sys

folder = Path(__file__).resolve().parent
root = folder.parents[2]
read = lambda p: json.loads(p.read_bytes() if p.exists() else
    gzip.decompress(p.with_name(p.name + '.gz').read_bytes()))
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
checkpoint = read(folder / 'checkpoint.json')
actual = {p.relative_to(folder).as_posix(): sha(p) for p in folder.rglob('*')
          if p.is_file() and p.name != 'checkpoint.json'}
assert actual == checkpoint['evidence_sha256']
assert all(sha(root / name) == digest for name, digest in checkpoint['source_sha256'].items())
assert all(sha(Path(name)) == digest for name, digest in checkpoint['preserved_artifacts'].items())
quality = read(root / 'docs/validation/sweep-smoke-admission/quality-default.json')
assert len(quality['source_sha256']) == 650
assert all(sha(root / name) == digest for name, digest in quality['source_sha256'].items())
checks = json.loads(gzip.decompress((folder / 'display-calibration-fixture-check-diagnostic.txt.gz').read_bytes()))
assert checks['offline_checks_passed'] and checks['layout_cases'] == 7
assert not checks['ui_created'] and not checks['capture_executed']
assert gzip.decompress((folder / 'display-calibration-fixture-format-diagnostic.txt.gz').read_bytes()) == b''
for name in ['initial-guard-failure', 'diagnostic-repeat']:
    attempt = folder / name
    session = read(attempt / 'report.json')
    assert all(session[k] for k in ['owned_process_absent', 'endpoint_removed',
        'shared_geometry_unchanged', 'review_unchanged', 'copied_bundle_unchanged',
        'canonical_unchanged', 'baseline_file_unchanged'])
    assert not session['forced_cleanup'] and session['open_exit_code'] == 0
    assert session['native_exit_status'] is None
    close = read(attempt / 'close-response.json')
    assert close['closed'] and close['shutdown']['exit_ready']
    assert close['shutdown']['status'] == 'completed'
    states = [read(attempt / f) for f in ['initial.json', 'final.json', 'agent-expected.json']]
    for key in ['components', 'wires', 'nets', 'model', 'circuit', 'result']:
        assert states[0][key] == states[1][key] == states[2][key]
    result = read(attempt / 'experiment-report.json')
    analysis = read(attempt / 'pixel-analysis.json')
    if name == 'initial-guard-failure':
        assert result['status'] == 'failed' and result['process_exit_codes'] == dict(fixture=1, display=0, window=0)
        assert not analysis['calibration_attempt_accepted']
        assert analysis['streams']['window-capture']['outside_fixture_lifetime'] == 5
    else:
        assert result['process_exit_codes'] == dict(fixture=0, display=0, window=0)
        assert analysis['calibration_attempt_accepted'] and analysis['helper_error'] is None
        assert all(x['outside_fixture_lifetime'] == 0 for x in analysis['streams'].values())
    assert analysis['eligible_latency_observations'] == 0 and analysis['latency_gate_passed'] is None
    subprocess.run([sys.executable, '-B', str(folder / 'analyze.py'), str(attempt)], check=True)
assert checkpoint['eligible_latency_observations'] == 0
assert checkpoint['latency_gate_passed'] is None
print(dict(passed=True, attempts_retained=2, static_repeat_passed=True,
           initial_guard_failure_resolved=False, go_sources_unchanged=650,
           eligible_latency_observations=0))
