"""Read-only follow-up ledger, local archives, refusal, physics and cleanup replay."""
from pathlib import Path
import gzip
import hashlib
import json
import math
import statistics
import subprocess
import sys
import types

from PIL import Image
import io

from frame_store import contents, load_images, read

folder = Path(__file__).resolve().parent
root = folder.parents[2]
sha = lambda raw: hashlib.sha256(raw).hexdigest()
checkpoint = read(folder / 'checkpoint.json')
assert {p.relative_to(folder).as_posix(): sha(p.read_bytes()) for p in folder.rglob('*')
        if p.is_file() and p.name != 'checkpoint.json'} == checkpoint['evidence_sha256']
assert all(sha(Path(name).read_bytes()) == digest for name, digest in checkpoint['preserved_artifacts'].items())
assert not list(folder.rglob('frames.tar.xz'))
quality = read(root / 'docs/validation/sweep-smoke-admission/quality-default.json')
assert len(quality['source_sha256']) == 650
assert all(sha((root / name).read_bytes()) == digest for name, digest in quality['source_sha256'].items())
plan_path = root / 'research/39-led-rc-followup-plan.json'
plan = read(plan_path)
assert plan['cohort_trials_run'] == 0 and len(plan['trials']) == 40  # immutable declaration
ledger = read(folder / 'cohort-outcomes.json')
assert [r['id'] for r in ledger['outcomes']] == [r['id'] for r in plan['trials']]
assert ledger['scheduled'] == 40 and ledger['attempted'] == 24 and ledger['available'] == 21
assert ledger['unavailable'] == 3 and ledger['not_attempted'] == 16 and ledger['replacements'] == 0
assert ledger['overall_prd_performance_pass'] is None
for kind in ['led', 'rc']:
    session = folder / kind / 'session'
    for name, info in read(session / 'original-file-manifest.json').items():
        raw = contents(session / name)
        assert sha(raw) == info['sha256'] and len(raw) == info['bytes']
    report = read(session / 'report.json')
    assert all(report[k] for k in ['owned_process_absent', 'endpoint_removed', 'baseline_file_unchanged',
        'review_unchanged', 'copied_bundle_unchanged', 'canonical_unchanged'])
    assert report['shared_geometry_unchanged'] == (kind == 'led')
    assert not report['forced_cleanup'] and report['open_exit_code'] == 0 and report['native_exit_status'] is None
    close = read(session / 'close-response.json')
    assert close['closed'] and close['shutdown']['exit_ready'] and close['shutdown']['status'] == 'completed'
    assert read(session / 'cua-postflight.json')['running_elab'] == []
    declaration = read(session / 'producer-declaration.json')
    assert declaration['plan_sha256'] == sha(plan_path.read_bytes())
    assert declaration['worker_sha256'] == sha(gzip.decompress((session / 'producers/worker.py.gz').read_bytes()))
    assert not declaration['app_or_recorder_changed'] and declaration['replacement_attempts'] == 0
    host = read(session / 'host-before.json')
    assert all(host[key] == plan['expected_host'][key] for key in ['model', 'os_version', 'os_build'])
assert contents(folder / 'led/session/input-controller.js') == contents(folder / 'rc/session/input-controller.js')
assert 'AssertionError' in contents(folder / 'rc/session/session-output.txt').decode()
assert 'stopped after unavailable rc2-04' in contents(folder / 'rc/session/worker-output.txt').decode()
frames = images = 0
frozen = read(root / 'internal/conformance/testdata/electrical-reference.json')
for n in range(1, 21):
    trial = folder / f'led/session/trials/led2-{n:02}'
    analysis = read(trial / 'analysis.json')
    outcome = ledger['outcomes'][n - 1]
    assert outcome['status'] == analysis['measurement_status'] == 'available' and outcome['analysis'] == analysis
    original = read(trial / 'outcome.json')
    assert original['status'] == outcome['original_status'] == 'collected_pending_pixel_analysis'
    assert sha(contents(trial / f'led2-{n:02}-baseline.elab')) == original['baseline_sha256']
    assert contents(trial / f'led2-{n:02}-final.elab')
    for name, resistance in [('before', 1000), ('after', 2000)]:
        state = read(trial / (name + '-apply.json'))
        expected = next(r for r in frozen['led'] if float(r['resistance_ohm']) == resistance)
        assert abs(state['result']['current_a'] - float(expected['current_a'])) < 1e-12
    subprocess.run([sys.executable, '-B', str(folder / 'led/analyze.py'), str(trial), '--verify'],
                   check=True, stdout=subprocess.DEVNULL)
    frames += analysis['complete_frames']
    images += analysis['unique_images']
assert frames == 3252 and images == 3098
for key in ['marker_to_committed_ms', 'marker_to_fresh_ms']:
    values = [r['analysis'][key] for r in ledger['outcomes'] if r['reference'] == 'led']
    p95 = sorted(values)[math.ceil(.95 * len(values)) - 1]
    assert ledger['cohorts']['led']['statistics_ms'][key] == dict(min=min(values), median=statistics.median(values), max=max(values), nearest_rank_p95=p95)
assert ledger['cohorts']['led']['percentile_rank'] == 19
assert ledger['cohorts']['led']['statistics_ms']['marker_to_fresh_ms']['nearest_rank_p95'] < 500
assert ledger['cohorts']['led']['statistics_ms']['marker_to_committed_ms']['nearest_rank_p95'] > 100
rc = folder / 'rc/session/trials'
subprocess.run([sys.executable, '-B', str(folder / 'rc/analyze.py'), str(rc / 'rc2-01'), '--verify'],
               check=True, stdout=subprocess.DEVNULL)
reference = types.ModuleType('recorded_rc_reference')
exec(compile(gzip.decompress((folder / 'rc/session/producers/reference.py.gz').read_bytes()),
             'recorded_reference.py', 'exec'), reference.__dict__)
observed = read(folder / 'rc/session/overlap-observations.json')['observations']
annotation = read(folder / 'rc/image-annotations.json')
for n in range(1, 5):
    trial = rc / f'rc2-{n:02}'
    raw_images = load_images(trial / 'capture')
    rows = [json.loads(line) for line in contents(trial / 'capture/frames.jsonl').splitlines()]
    complete = [r for r in rows if r.get('png')]
    report = read(trial / 'capture/report.json')
    assert len(complete) == report['complete_frames']
    for row in complete:
        assert sha(raw_images[row['png']]) == row['png_sha256'] and not row['eligible_for_latency']
    before, after = [read(trial / (name + '-apply.json')) for name in ['before', 'after']]
    reference.independent(before)
    reference.independent(after)
    cap = lambda state: next(p['value'] for p in state['components'] if p['id'] == 'Storage')
    if n < 4:
        assert cap(after) == 2e-6 and after['electrical_revision'] == before['electrical_revision'] + 1
    else:
        assert cap(after) == cap(before) == 1e-6 and after['run_id'] == before['run_id']
        assert after['command_revision'] == before['command_revision'] + 1
    if n > 1:
        observation = observed[n - 2]
        cache = {}
        unknown = []
        for row in complete:
            if row['png'] not in cache:
                image = Image.open(io.BytesIO(raw_images[row['png']])).convert('RGB')
                cache[row['png']] = any(sha(image.crop(r['box']).tobytes()) != r['rgb_sha256'] for r in annotation['static_regions'])
            if cache[row['png']]:
                unknown.append(row)
        assert len(unknown) == observation['unknown_complete_frames']
        assert sum(cache.values()) == observation['unknown_unique_images']
        assert unknown[0]['index'] == observation['first_unknown_frame'] and unknown[0]['png'] == observation['first_unknown_png']
        if n < 4:
            refused = subprocess.run([sys.executable, '-B', str(folder / 'rc/analyze.py'), str(trial)], capture_output=True, text=True)
            assert refused.returncode != 0 and 'Occluded or unknown UI region' in refused.stderr
for n in range(5, 21):
    outcome = ledger['outcomes'][20 + n - 1]
    assert outcome['id'] == f'rc2-{n:02}' and outcome['status'] == 'not_attempted'
    assert not (rc / f'rc2-{n:02}').exists()
assert ledger['cohorts']['rc']['full_cohort_p95'] is None
subprocess.run([sys.executable, '-B', str(folder / 'check_frame_store.py')], check=True)
print(dict(passed=True, led_available=20, led_complete_frames=frames, led_original_pngs=images,
           led_fresh_p95_ms=ledger['cohorts']['led']['statistics_ms']['marker_to_fresh_ms']['nearest_rank_p95'],
           rc_attempted=4, rc_available=1, rc_unavailable=3, rc_not_attempted=16,
           rc_settings_equality=False, overall_prd_performance_pass=None, go_sources_unchanged=650))
