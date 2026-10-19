"""Verify retained window-capture calibration, never collect or grade latency."""
import gzip
import hashlib
import json
import math
from pathlib import Path
import struct
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'scripts'))
from wiring_replay import validate_release


def read(path):
    data = path.read_bytes()
    return json.loads(gzip.decompress(data) if path.suffix == '.gz' else data)


def main():
    folder = ROOT / 'docs/validation/frame-probe-calibration'
    checkpoint = read(folder / 'checkpoint.json')
    assert checkpoint['status'] == 'nominal_window_capture_calibrated'
    assert checkpoint['capture_executed'] and checkpoint['owner_authorized_method']
    assert not checkpoint['screen_permission_requested'] and not checkpoint['input_operated']
    assert checkpoint['eligible_latency_observations'] == 0 and checkpoint['latency_gate_passed'] is None
    assert not checkpoint['display_visibility_validated'] and not checkpoint['physical_clock_calibration']
    names = {p.relative_to(folder).as_posix() for p in folder.rglob('*') if p.is_file()}
    assert names == set(checkpoint['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in checkpoint['evidence_sha256'].items():
        assert hashlib.sha256((folder / name).read_bytes()).hexdigest() == expected, name
    for name, expected in checkpoint['source_sha256'].items():
        assert hashlib.sha256((ROOT / name).read_bytes()).hexdigest() == expected, name
    identity = validate_release(ROOT, ROOT / 'research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json')
    report = read(folder / 'passing/report.json')
    assert report['release'] == identity and identity['receipt_sha256'] == checkpoint['release_receipt_sha256']
    assert report['status'] == 'window_capture_calibration_only'
    assert report['probe_sha256'] == checkpoint['probe_sha256']
    assert report['native_exit_code'] == report['recorder_exit_code'] == 0
    assert report['endpoint_removed'] and report['owned_process_absent'] and report['sealed_review_unchanged']
    assert not report['forced_cleanup'] and not report['input_operated']
    assert report['preflight'] == dict(screen_access_previously_granted=True, permission_requested=False, capture_executed=False)
    capture = folder / 'passing/capture'
    captured = read(capture / 'report.json')
    assert report['capture_report'] == captured and captured['collector_error'] is captured['command_error'] is None
    assert captured['pid'] == report['owned_pid'] == 22961 and captured['window_id'] == 151880
    assert captured['eligible_latency_observations'] == 0 and captured['latency_gate_passed'] is None
    assert captured['filter'] == 'desktopIndependentWindow'
    assert not captured['unobscured_display_visibility_validated']
    rows = [json.loads(line) for line in gzip.decompress((capture / 'frames.jsonl.gz').read_bytes()).splitlines()]
    frames = [row for row in rows if row['event'] == 'frame']
    markers = [row for row in rows if row['event'] == 'marker']
    assert len(markers) == 1 and markers[0] == report['marker_ack'] == read(capture / 'marker-calibration_01.json')
    assert (capture / 'marker.request').read_bytes() == b'calibration_01'
    assert len(frames) == captured['complete_frames'] == captured['frames_received'] == checkpoint['frames'] == 103
    times = [int(row['display_mach_ticks']) for row in frames]
    assert all(b > a for a, b in zip(times, times[1:]))
    numerator, denominator = captured['timebase_numerator'], captured['timebase_denominator']
    assert (numerator, denominator) == (125, 3)
    errors = []
    images = {}
    for row in frames:
        assert row['status'] == 0 and not row['eligible_for_latency']
        ticks = int(row['display_mach_ticks'])
        assert int(row['received_mach_ticks']) >= ticks
        errors.append(abs(row['presentation_seconds'] - ticks * numerator / denominator / 1e9))
        if ticks >= int(markers[0]['mach_ticks']):
            expected = (ticks - int(markers[0]['mach_ticks'])) * numerator / denominator / 1e6
            assert math.isclose(row['since_acknowledged_marker_ms'], expected, abs_tol=1e-9)
        images[row['png']] = row['png_sha256']
    assert max(errors) == checkpoint['presentation_to_mach_max_abs_difference_s']
    assert (times[-1] - times[0]) * numerator / denominator / 1e9 == checkpoint['first_to_last_display_span_s']
    assert len(images) == captured['unique_images'] == checkpoint['unique_images'] == 18
    image_bytes = 0
    for name, expected in images.items():
        data = (capture / name).read_bytes()
        assert hashlib.sha256(data).hexdigest() == expected
        assert data[:8] == b'\x89PNG\r\n\x1a\n' and struct.unpack('>II', data[16:24]) == (1276, 1146)
        image_bytes += len(data)
    assert image_bytes == captured['image_bytes'] == 3698345
    assert checkpoint['runtime_capture_directory_mode'] == '0o700'
    assert set(checkpoint['runtime_capture_file_modes'].values()) == {'0o600'}
    first = read(folder / 'first-readiness/report.json')
    crashed = read(folder / 'filter-crash/report.json')
    assert first['status'] == crashed['status'] == 'failed'
    assert not first['capture_executed'] and not crashed['capture_executed']
    assert first['forced_cleanup'] and crashed['forced_cleanup']
    assert crashed['final_recorder_exit_code'] == -6 and crashed['final_native_exit_code'] == -15
    assert crashed['owned_process_absent_after_cleanup']
    assert b'CGS_REQUIRE_INIT' in gzip.decompress((folder / 'filter-crash/recorder-stderr.txt.gz').read_bytes())
    assert checkpoint['all_owned_pids_absent_after_runs'] == [22540, 22687, 22961]
    assert all(not (folder / omission['path']).exists() for omission in checkpoint['excluded_artifacts'])
    print(json.dumps(dict(calibration_integrity_verified=True, timestamped_frames=103,
                          unique_images=18, eligible_latency_observations=0, latency_gate_passed=None)))


if __name__ == '__main__':
    main()
