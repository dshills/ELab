"""Read-only NPN ledger, source, image archive, numerical and cleanup replay."""
from pathlib import Path
import gzip
import hashlib
import json
import math
import statistics
import subprocess
import sys

from reference import independent

folder = Path(__file__).resolve().parent
root = folder.parents[2]


def contents(path):
    return path.read_bytes() if path.exists() else gzip.decompress(path.with_name(path.name + '.gz').read_bytes())


def read(path):
    return json.loads(contents(path))


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


checkpoint = read(folder / 'checkpoint.json')
assert {p.relative_to(folder).as_posix(): sha(p.read_bytes()) for p in folder.rglob('*')
        if p.is_file() and p.name != 'checkpoint.json'} == checkpoint['evidence_sha256']
assert all(sha(Path(name).read_bytes()) == digest for name, digest in checkpoint['preserved_artifacts'].items())
quality = read(root / 'docs/validation/sweep-smoke-admission/quality-default.json')
assert len(quality['source_sha256']) == 650
assert all(sha((root / name).read_bytes()) == digest for name, digest in quality['source_sha256'].items())
plan = read(root / 'research/37-visible-latency-cohort-plan.json')
assert plan['cohort_trials_run'] == 0 and len(plan['trials']) == 60  # immutable declaration
ledger = read(folder / 'cohort-outcomes.json')
assert [r['id'] for r in ledger['outcomes']] == [f'npn-{i:02}' for i in range(1, 21)]
assert ledger['scheduled'] == ledger['retained_outcomes'] == ledger['available'] == 20
assert ledger['unavailable'] == ledger['replacements'] == 0
assert ledger['overall_prd_performance_pass'] is None
session = folder / 'session'
for name, info in read(session / 'original-file-manifest.json').items():
    raw = contents(session / name)
    assert sha(raw) == info['sha256'] and len(raw) == info['bytes']
report = read(session / 'report.json')
assert all(report[k] for k in ['owned_process_absent', 'endpoint_removed', 'baseline_file_unchanged',
    'shared_geometry_unchanged', 'review_unchanged', 'copied_bundle_unchanged', 'canonical_unchanged'])
assert not report['forced_cleanup'] and report['open_exit_code'] == 0 and report['native_exit_status'] is None
close = read(session / 'close-response.json')
assert close['closed'] and close['shutdown']['exit_ready'] and close['shutdown']['status'] == 'completed'
assert read(session / 'cua-postflight.json')['running_elab'] == []
host = read(session / 'host-before.json')
assert all(host[key] == plan['expected_host'][key] for key in ['model', 'os_version', 'os_build'])
declaration = read(session / 'producer-declaration.json')
assert declaration['ids'] == [r['id'] for r in ledger['outcomes']]
assert declaration['replacement_attempts'] == 0 and not declaration['app_or_recorder_changed']
assert sha(gzip.decompress((session / 'producers/worker.py.gz').read_bytes())) == declaration['worker_sha256']
assert sha(gzip.decompress((session / 'producers/reference.py.gz').read_bytes())) == declaration['reference_sha256']
assert sha((root / 'internal/conformance/testdata/electrical-reference.json').read_bytes()) == declaration['frozen_reference_sha256']
assert sha((root / 'scripts/freeze-electrical-reference.py').read_bytes()) == declaration['reference_generator_sha256']
preflight = read(session / 'producers/preflight.json')
assert preflight['passed'] and not any(preflight[k] for k in ['native_started', 'capture_executed', 'ui_input'])
assert len(preflight['sessions']) == 2
for pair in preflight['sessions']:
    independent(pair['before'])
    independent(pair['after'])
subprocess.run([sys.executable, '-B', str(root / 'scripts/freeze-electrical-reference.py')], check=True)
frames = images = stale = 0
for outcome in ledger['outcomes']:
    trial = session / 'trials' / outcome['id']
    original = read(trial / 'outcome.json')
    analysis = read(trial / 'analysis.json')
    assert original['status'] == outcome['original_status'] == 'collected_pending_pixel_analysis'
    assert analysis == outcome['analysis'] and outcome['available']
    assert analysis['measurement_status'] == 'available' and len(analysis['rejected_mutations']) == 9
    assert sha(contents(trial / (outcome['id'] + '-baseline.elab'))) == original['baseline_sha256']
    assert contents(trial / (outcome['id'] + '-final.elab'))
    subprocess.run([sys.executable, '-B', str(folder / 'analyze.py'), str(trial), '--verify'],
                   check=True, stdout=subprocess.DEVNULL)
    frames += analysis['complete_frames']
    images += analysis['unique_images']
    stale += analysis['states'].get('committed-value-stale-result', 0)
assert frames == 3260 and images == 3130 and stale == 12
for key in ['marker_to_committed_ms', 'marker_to_fresh_ms']:
    values = [r['analysis'][key] for r in ledger['outcomes']]
    p95 = sorted(values)[math.ceil(.95 * len(values)) - 1]
    assert ledger['statistics_ms'][key] == dict(min=min(values), median=statistics.median(values), max=max(values), nearest_rank_p95=p95)
assert ledger['percentile_rank'] == 19
assert ledger['statistics_ms']['marker_to_fresh_ms']['nearest_rank_p95'] < plan['thresholds_ms']['first_valid_updated_sample']
assert ledger['statistics_ms']['marker_to_committed_ms']['nearest_rank_p95'] > plan['thresholds_ms']['visible_acknowledgment']
assert ledger['threshold_assessment']['acknowledgment'] == 'inconclusive_controller_overhead_included'
assert ledger['threshold_assessment']['fresh_result'] == 'supported_for_this_fixture_within_software_composition_method'
print(dict(passed=True, scheduled_outcomes=20, available_bounds=20, unavailable=0,
           complete_frames=frames, original_pngs=images, stale_frames=stale,
           committed_p95_ms=ledger['statistics_ms']['marker_to_committed_ms']['nearest_rank_p95'],
           fresh_p95_ms=ledger['statistics_ms']['marker_to_fresh_ms']['nearest_rank_p95'],
           overall_prd_performance_pass=None, go_sources_unchanged=650))
