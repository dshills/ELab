"""Read-only RC ledger, preserved-image, numerical and cleanup verification."""
from pathlib import Path
import gzip
import hashlib
import json
import statistics
import subprocess
import sys
import types

folder = Path(__file__).resolve().parent
root = folder.parents[2]


def data(path):
    return path.read_bytes() if path.exists() else gzip.decompress(path.with_name(path.name + '.gz').read_bytes())


def read(path):
    return json.loads(data(path))


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
assert len(plan['trials']) == 60 and plan['cohort_trials_run'] == 0  # preserved declaration, not progress
ledger = read(folder / 'cohort-outcomes.json')
assert [r['id'] for r in ledger['outcomes']] == [f'rc-{i:02}' for i in range(1, 21)]
assert ledger['retained_outcomes'] == ledger['scheduled'] == 20
assert ledger['available'] == 12 and ledger['unavailable'] == 8 and ledger['replacements'] == 0
assert ledger['full_cohort_p95'] is ledger['performance_gate_passed'] is None
assert [r['id'] for r in ledger['outcomes'] if r['available']] == [f'rc-{i:02}' for i in range(9, 21)]
initial = root / 'docs/validation/latency-rc-halt-173'
assert read(initial / 'collection/trials/rc-01/outcome.json')['status'] == 'unavailable'
assert 'CUA aborted' in read(initial / 'collection/trials/rc-02/outcome.json')['reason']
for phase in ['remaining', 'tail']:
    base = folder / phase
    for name, info in read(base / 'original-file-manifest.json').items():
        raw = data(base / name)
        assert sha(raw) == info['sha256'] and len(raw) == info['bytes']
    report = read(base / 'report.json')
    assert all(report[k] for k in ['owned_process_absent', 'endpoint_removed', 'baseline_file_unchanged',
        'shared_geometry_unchanged', 'review_unchanged', 'copied_bundle_unchanged', 'canonical_unchanged'])
    assert not report['forced_cleanup'] and report['open_exit_code'] == 0 and report['native_exit_status'] is None
    close = read(base / 'close-response.json')
    assert close['closed'] and close['shutdown']['exit_ready'] and close['shutdown']['status'] == 'completed'
    host = read(base / 'host-before.json')
    assert all(host[key] == plan['expected_host'][key] for key in ['model', 'os_version', 'os_build'])
assert read(folder / 'tail/cua-postflight.json')['running_elab'] == []
assert 'Remaining RC block stopped after unavailable rc-08' in data(folder / 'remaining/worker-output.txt').decode()
assert 'rc_outcomes' in data(folder / 'tail/worker-output.txt').decode()
reference = types.ModuleType('recorded_rc_reference')
exec(compile(gzip.decompress((folder / 'tail/producers/reference.py.gz').read_bytes()),
             'recorded_reference.py', 'exec'), reference.__dict__)
occluded_frames = 0
private_images = 0
for n in range(3, 9):
    trial = folder / 'remaining/trials' / f'rc-{n:02}'
    report = read(trial / 'capture/report.json')
    rows = [json.loads(line) for line in data(trial / 'capture/frames.jsonl').splitlines()]
    storage = read(trial / 'capture/private-image-storage.json')['images']
    assert len(storage) == report['unique_images'] == 1
    for name, info in storage.items():
        raw = Path(info['local_path']).read_bytes()
        assert sha(raw) == info['sha256'] == '47ef9a1cd0223e2df5f10ec280935ed9f5af8e04d65c04a01505f5c2e6c00758'
        assert len(raw) == info['bytes'] and name == 'frame-' + info['sha256'] + '.png'
        assert not (trial / 'capture' / name).exists()
    complete = [r for r in rows if r.get('png')]
    assert len(complete) == report['complete_frames'] and all(r['png'] in storage for r in complete)
    occluded_frames += len(complete)
    private_images += len(storage)
    before, after = [read(trial / (name + '-apply.json')) for name in ['before', 'after']]
    reference.independent(before)
    reference.independent(after)
    cap = lambda state: next(p['value'] for p in state['components'] if p['id'] == 'Storage')
    if n < 8:
        assert cap(after) == 2e-6 and after['run_id'] == before['run_id'] + 1
        matched = read(trial / 'agent-matched-time.json')
        reference.independent(matched)
        assert after['transient']['sample'] == matched['transient']['sample']
    else:
        assert cap(after) == cap(before) == 1e-6 and after['run_id'] == before['run_id']
assert occluded_frames == 16 and private_images == 6
frames = images = 0
available = []
for n in range(9, 21):
    trial = folder / 'tail/trials' / f'rc-{n:02}'
    analysis = read(trial / 'analysis.json')
    assert read(trial / 'outcome.json')['status'] == 'collected_pending_pixel_analysis'
    assert analysis == ledger['outcomes'][n - 1]['analysis']
    subprocess.run([sys.executable, '-B', str(folder / 'analyze.py'), str(trial), '--verify'],
                   check=True, stdout=subprocess.DEVNULL)
    original = read(trial / 'outcome.json')
    assert sha(data(trial / f'rc-{n:02}-baseline.elab')) == original['baseline_sha256']
    assert data(trial / f'rc-{n:02}-final.elab')
    frames += analysis['complete_frames']
    images += analysis['unique_images']
    available.append(analysis['marker_to_fresh_ms'])
assert frames == 1991 and images == 1968
assert ledger['available_only_descriptive_ms'] == dict(min=min(available), median=statistics.median(available), max=max(available))
print(dict(passed=True, scheduled_outcomes=20, available_bounds=12, unavailable=8,
           complete_available_frames=frames, available_original_pngs=images,
           private_occluded_frames=occluded_frames, full_cohort_p95=None, go_sources_unchanged=650))
