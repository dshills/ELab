"""Read-only replay of the halted RC block; never launches an app or recorder."""
from pathlib import Path
from fractions import Fraction
import gzip
import hashlib
import io
import json
import math
import tarfile
import types

from PIL import Image

folder = Path(__file__).resolve().parent
root = folder.parents[2]


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def data(path):
    return path.read_bytes() if path.exists() else gzip.decompress(path.with_name(path.name + '.gz').read_bytes())


def read(path):
    return json.loads(data(path))


checkpoint = read(folder / 'checkpoint.json')
assert {p.relative_to(folder).as_posix(): sha(p.read_bytes()) for p in folder.rglob('*')
        if p.is_file() and p.name != 'checkpoint.json'} == checkpoint['evidence_sha256']
assert all(sha(Path(name).read_bytes()) == digest for name, digest in checkpoint['preserved_artifacts'].items())
quality = read(root / 'docs/validation/sweep-smoke-admission/quality-default.json')
assert len(quality['source_sha256']) == 650
assert all(sha((root / name).read_bytes()) == digest for name, digest in quality['source_sha256'].items())
for phase in ['collection', 'diagnostic']:
    base = folder / phase
    for name, info in read(base / 'original-file-manifest.json').items():
        raw = data(base / name)
        assert sha(raw) == info['sha256'] and len(raw) == info['bytes']
    report = read(base / 'report.json')
    assert all(report[k] for k in ['owned_process_absent', 'endpoint_removed', 'baseline_file_unchanged',
                                  'review_unchanged', 'copied_bundle_unchanged', 'canonical_unchanged'])
    assert not report['forced_cleanup'] and report['open_exit_code'] == 0
    close = read(base / 'close-response.json')
    assert close['closed'] and close['shutdown']['exit_ready'] and close['shutdown']['status'] == 'completed'
assert read(folder / 'collection/report.json')['shared_geometry_unchanged']
assert not read(folder / 'diagnostic/report.json')['shared_geometry_unchanged']
assert 'AssertionError' in data(folder / 'diagnostic/session-output.txt').decode()
status = read(folder / 'status.json')
assert status['retained_outcomes'] == status['unavailable'] == 2 and status['available'] == 0
assert status['not_attempted'] == [f'rc-{i:02}' for i in range(3, 21)]
assert status['replacement_attempts'] == 0 and status['full_cohort_p95'] is status['performance_gate_passed'] is None
trial = folder / 'collection/trials/rc-01'
before, after = [read(trial / (name + '-apply.json')) for name in ['before', 'after']]
cap = lambda state: next(p['value'] for p in state['components'] if p['id'] == 'Storage')
assert cap(before) == cap(after) == 1e-6
for key in ['components', 'wires', 'electrical_revision', 'run_id']:
    assert before[key] == after[key]
assert after['command_revision'] == before['command_revision'] + 1  # producer Pause only
assert after['paused'] and not before['paused']
assert before['transient']['time_constant_s'] == after['transient']['time_constant_s'] == .01
assert read(trial / 'outcome.json')['status'] == 'unavailable'
action = read(trial / 'capture-action.json')
assert action['input_call_invoked'] and action['input_call_completed'] and not action['after_ax_has_expected_readout']
assert 'CUA aborted' in read(folder / 'collection/trials/rc-02/outcome.json')['reason']
assert not (folder / 'collection/trials/rc-02/capture').exists()
reference = types.ModuleType('recorded_rc_reference')
exec(compile(data(folder / 'collection/producers/reference.py'), 'recorded_reference.py', 'exec'), reference.__dict__)
reference.independent(before)
reference.independent(after)
assert read(folder / 'collection/producers/preflight.json')['passed']

capture = trial / 'capture'
report, config = read(capture / 'report.json'), read(capture / 'configuration.json')
rows = [json.loads(line) for line in data(capture / 'frames.jsonl').splitlines()]
frames = [r for r in rows if r['event'] == 'frame']
marker = next(r for r in rows if r['event'] == 'marker')
assert marker == action['marker_acknowledgment'] == read(capture / 'marker-rc_01.json')
assert marker['frame_count_before'] == sum(r['event'] == 'frame' for r in rows[:rows.index(marker)])
assert len(frames) == report['frames_received_within_deadline'] == 166
assert [r['index'] for r in frames] == list(range(1, 167))
assert report['complete_frames'] == 163 and report['unique_images'] == 161
assert report['collector_error'] is report['command_error'] is report['journal_close_error'] is None
assert report['geometry_rechecked_after_stop'] and report['eligible_latency_observations'] == 0
assert report['latency_gate_passed'] is None
bounds = dict(x=2034, y=303, width=1053, height=802)
assert config['capture_plan']['window_global_points'] == bounds and config['seconds'] == 3
assert config['target_executable_sha256'] == checkpoint['native_sha256']
assert config['tool_executable_sha256'] == checkpoint['recorder_sha256']
assert not any(config[k] for k in ['audio', 'microphone', 'cursor', 'permission_requested'])
manifest = read(capture / 'image-manifest.json')
images = {}
with tarfile.open(capture / 'frames.tar.xz', 'r:xz') as archive:
    for member in archive:
        assert member.isfile() and member.mode == 0o600 and member.name in manifest and member.name not in images
        raw = archive.extractfile(member).read()
        assert sha(raw) == manifest[member.name]['sha256'] and len(raw) == manifest[member.name]['bytes']
        images[member.name] = raw
assert set(images) == set(manifest) and sum(map(len, images.values())) == report['image_bytes']
# Full first image is manually inspected. These static regions contain the
# accepted schematic value, tau and run; private draft/cursor/live waveform vary.
regions = [(618, 304, 704, 323), (330, 501, 390, 519), (260, 520, 310, 538)]
baseline = Image.open(folder / 'rc-01-baseline.png').convert('RGB')
expected = [baseline.crop(box).tobytes() for box in regions]
for raw in images.values():
    image = Image.open(io.BytesIO(raw)).convert('RGB')
    assert image.size == (1053, 802)
    assert [image.crop(box).tobytes() for box in regions] == expected
scale = Fraction(report['timebase_numerator'], report['timebase_denominator'])
start, deadline = int(config['start_mach_ticks']), int(config['deadline_mach_ticks'])
assert deadline - start == int(Fraction(3_000_000_000, 1) / scale)
complete = 0
for frame in frames:
    assert not frame['eligible_for_latency'] and frame['target_geometry_matches_at_receipt']
    assert start <= int(frame['received_mach_ticks']) <= deadline
    if frame['status'] != 0:
        assert not frame.get('png')
        continue
    displayed = int(frame['display_mach_ticks'])
    assert int(frame['presentation_value']) == math.floor(displayed * scale)
    assert frame['presentation_matches_converted_display_time'] and frame['screen_rect_points'] == bounds
    if displayed > deadline:
        assert frame['outside_presentation_window'] and not frame.get('png')
        continue
    assert frame['frame_geometry_validated'] and sha(images[frame['png']]) == frame['png_sha256']
    if frame['index'] > marker['frame_count_before'] and displayed > int(marker['mach_ticks']):
        assert abs(frame['since_acknowledged_marker_ms'] - float((displayed - int(marker['mach_ticks'])) * scale / 1_000_000)) < 1e-9
    else:
        assert 'since_acknowledged_marker_ms' not in frame
    complete += 1
assert complete == 163
diagnostic = folder / 'diagnostic'
for name in ['live-index', 'paused-index']:
    original, accepted = [read(diagnostic / f'{name}-{suffix}.json') for suffix in ['prepare', 'after']]
    assert cap(original) == 1e-6 and cap(accepted) == 2e-6
    assert accepted['run_id'] == original['run_id'] + 1
assert not read(diagnostic / 'live-index-after.json')['transient']['stale']
assert read(diagnostic / 'live-index-after.json')['transient']['time_constant_s'] == .02
assert read(diagnostic / 'paused-index-after.json')['paused'] and read(diagnostic / 'paused-index-after.json')['transient']['stale']
assert cap(read(diagnostic / 'live-coordinate-after.json')) == 1e-6
assert status['coordinate_call_completion'] is None and status['diagnostic_owner_resize_confirmed']
assert not read(diagnostic / 'report.json')['capture_executed']
print(dict(passed=True, retained_outcomes=2, available_bounds=0, complete_frames=163,
           original_pngs=161, diagnostic_coordinate_completion='unknown', shared_geometry_unchanged=False,
           full_cohort_p95=None, go_sources_unchanged=650))
