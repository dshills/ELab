"""Read-only replay of one visible RC recording, including refusal controls."""
from pathlib import Path
from collections import Counter
from fractions import Fraction
import copy
import gzip
import hashlib
import io
import json
import math
import sys
import tarfile
import types

from PIL import Image

if sys.flags.optimize:raise SystemExit('Verification requires assertions')
folder = Path(sys.argv[1])
proof = Path('/Users/dshills/Development/projects/ELab/docs/validation/latency-followup-173/rc')


def contents(path):
    return path.read_bytes() if path.exists() else gzip.decompress(path.with_name(path.name + '.gz').read_bytes())


def read(path):
    return json.loads(contents(path))


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


sys.path.insert(0,str(Path(__file__).resolve().parent))
from frame_store import load_images
capture=folder/'capture'
images=load_images(capture)
names = ['capture/report.json', 'capture/configuration.json', 'capture-action.json',
         'before-apply.json', 'after-apply.json', 'agent-matched-time.json', 'inspection.json', 'outcome.json']
inputs = {name: read(folder / name) for name in names}
inputs['marker'] = read(capture / ('marker-' + folder.name.replace('-', '_') + '.json'))
inputs['rows'] = [json.loads(line) for line in contents(capture / 'frames.jsonl').splitlines()]
inputs['annotations'] = read(proof / 'image-annotations.json')
reference = types.ModuleType('recorded_rc_reference')
exec(compile(gzip.decompress((proof / 'session/producers/reference.py.gz').read_bytes()),
             'recorded_reference.py', 'exec'), reference.__dict__)


def exact_template(image, template):
    """Find exact RGB tau text; translation accommodates one/two-line captions."""
    box = (260, 500, 408, 540)
    raw = image.crop(box).tobytes()
    width, height = template.size
    needle = template.crop((0, 2, width, 3)).tobytes()
    expected = template.tobytes()
    position = raw.find(needle)
    while position >= 0:
        y, x = divmod(position, (box[2] - box[0]) * 3)
        if (x % 3 == 0 and x // 3 + width <= box[2] - box[0] and y >= 2
                and y - 2 + height <= box[3] - box[1]):
            at = (box[0] + x // 3, box[1] + y - 2)
            if image.crop((*at, at[0] + width, at[1] + height)).tobytes() == expected:
                return True
        position = raw.find(needle, position + 1)
    return False


def validate(data):
    report, config = data['capture/report.json'], data['capture/configuration.json']
    action, marker = data['capture-action.json'], data['marker']
    frames = [r for r in data['rows'] if r['event'] == 'frame']
    markers = [r for r in data['rows'] if r['event'] == 'marker']
    assert markers == [marker] and marker == action['marker_acknowledgment']
    assert marker['id'] == folder.name.replace('-', '_')
    assert marker['frame_count_before'] == sum(r['event'] == 'frame' for r in data['rows'][:data['rows'].index(marker)])
    assert action['input_call_invoked'] and action['input_call_completed'] and action['after_ax_has_expected_readout']
    assert not action.get('error')
    assert [e['name'] for e in action['events']] == ['capture_start_requested', 'baseline_image_received',
        'marker_requested', 'complete_acknowledgment_validated', 'input_call_will_invoke',
        'input_call_completed', 'after_input_ax_observed']
    assert [e['sequence'] for e in action['events']] == list(range(1, 8))
    assert action['events'][1]['frame_index'] <= marker['frame_count_before']
    assert report['marker_count'] == 1 and report['capture_executed'] and report['geometry_rechecked_after_stop']
    assert report['collector_error'] is report['command_error'] is report['journal_close_error'] is None
    assert report['eligible_latency_observations'] == 0 and report['latency_gate_passed'] is None
    assert not any(report[k] for k in ['pixel_mapping_calibrated', 'display_visibility_calibrated', 'clock_calibrated_for_latency'])
    assert data['outcome.json']['recorder_exit_code'] == data['inspection.json']['exit_code'] == 0
    inspected = json.loads(data['inspection.json']['stdout'])
    bounds = dict(x=2034, y=303, width=1053, height=802)
    assert config['capture_plan'] == inspected['capture_plan']
    assert config['capture_plan']['window_global_points'] == bounds
    assert config['capture_plan']['point_pixel_scale'] == 1 and config['seconds'] == 3
    assert config['target_executable_sha256'] == 'ecd24e458e0db4a0e580747b37145c7d30e9c8f4be686731df78cb690b38a330'
    assert config['tool_executable_sha256'] == 'a8cec4e8f9d16b2b5ad6b751f5657b537c5ed6ea956e7695403168b445bb923f'
    assert config['filter'] == 'display-excluding-no-windows'
    assert not any(config[k] for k in ['audio', 'microphone', 'cursor', 'permission_requested'])
    scale = Fraction(report['timebase_numerator'], report['timebase_denominator'])
    start, deadline, tick = int(config['start_mach_ticks']), int(config['deadline_mach_ticks']), int(marker['mach_ticks'])
    assert start < tick < deadline and deadline - start == int(Fraction(3_000_000_000, 1) / scale)
    assert len(frames) == report['frames_received_within_deadline'] <= 300
    assert [r['index'] for r in frames] == list(range(1, len(frames) + 1))
    annotation = data['annotations']
    templates = [Image.open(proof / t['image']).convert('RGB').crop(t['box']) for t in annotation['tau_templates']]
    roles, cache, first, used = Counter(), {}, {}, {}
    last = complete = future = outside = 0
    committed = None
    for row in frames:
        assert row['eligible_for_latency'] is False and row['target_geometry_matches_at_receipt']
        received = int(row['received_mach_ticks'])
        assert start <= received <= deadline and row['status'] in [0, 1, 3]
        displayed = int(row['display_mach_ticks']) if row.get('display_mach_ticks') else None
        if displayed is not None:
            future += displayed > received
        if row['status'] != 0:
            assert not row.get('png')
            continue
        assert displayed and displayed >= last
        last = displayed
        assert row['presentation_timescale'] == 1_000_000_000
        assert int(row['presentation_value']) == math.floor(displayed * scale)
        assert row['presentation_matches_converted_display_time']
        assert row['screen_rect_points'] == bounds and row['content_rect_points'] == dict(x=0, y=0, width=1053, height=802)
        assert row['scale_factor'] == row['content_scale'] == 1
        assert (row['width_pixels'], row['height_pixels']) == (1053, 802)
        if displayed > deadline:
            assert row['outside_presentation_window'] and not row.get('png')
            outside += 1
            continue
        assert row['frame_geometry_validated']
        raw = images[row['png']]
        assert digest(raw) == row['png_sha256'] and row['png'] == 'frame-' + digest(raw) + '.png'
        used[row['png']] = len(raw)
        if row['png'] not in cache:
            image = Image.open(io.BytesIO(raw)).convert('RGB')
            assert image.size == (1053, 802)
            for region in annotation['static_regions']:
                assert digest(image.crop(region['box']).tobytes()) == region['rgb_sha256'], 'Occluded or unknown UI region'
            cap_hash = digest(image.crop(annotation['capacitor_box']).tobytes())
            assert annotation['baseline_capacitor_sha256'] != annotation['committed_capacitor_sha256']
            has_new_tau = any(exact_template(image, template) for template in templates)
            if cap_hash == annotation['baseline_capacitor_sha256']:
                assert not has_new_tau
                role = 'private-draft-original-result'
            else:
                assert cap_hash == annotation['committed_capacitor_sha256'] and has_new_tau, 'Unknown committed/result state'
                role = 'committed-value-fresh-result'
            # Track acknowledgment from the accepted schematic label itself;
            # the independent tau check establishes result freshness.
            cache[row['png']] = (role, cap_hash == annotation['committed_capacitor_sha256'])
        role, has_committed_parameter = cache[row['png']]
        if has_committed_parameter and committed is None:
            committed = row
        roles[role] += 1
        first.setdefault(role, row)
        complete += 1
        if row['index'] > marker['frame_count_before'] and displayed > tick:
            assert abs(row['since_acknowledged_marker_ms'] - float((displayed - tick) * scale / 1_000_000)) < 1e-9
        else:
            assert 'since_acknowledged_marker_ms' not in row
    assert complete == report['complete_frames']
    assert future == report['frames_with_display_time_after_receipt'] and outside == report['frames_outside_presentation_window']
    assert set(used) == set(images) and sum(used.values()) == report['image_bytes']
    assert len(used) == report['unique_images']
    assert set(first) == {'private-draft-original-result', 'committed-value-fresh-result'}
    fresh = first['committed-value-fresh-result']
    assert committed is not None
    assert tick < int(committed['display_mach_ticks']) <= int(fresh['display_mach_ticks'])
    assert committed['index'] > marker['frame_count_before']
    before, after, agent = [data[k] for k in ['before-apply.json', 'after-apply.json', 'agent-matched-time.json']]
    cap = lambda state: next(p['value'] for p in state['components'] if p['id'] == 'Storage')
    assert cap(before) == 1e-6 and cap(after) == cap(agent) == 2e-6
    assert after['electrical_revision'] == before['electrical_revision'] + 1 and after['run_id'] == before['run_id'] + 1
    assert after['command_revision'] == before['command_revision'] + 3
    assert after['wires'] == before['wires']
    for key in ['components', 'nets', 'model', 'transient_settings']:
        assert after[key] == agent[key]
    semantics = lambda values: [{k: v for k, v in wire.items() if k != 'id'} for wire in values]
    assert semantics(after['wires']) == semantics(agent['wires'])
    for key in ['engine', 'source_id', 'resistor_id', 'capacitor_id', 'output_net', 'time_constant_s', 'settings', 'input_mode']:
        assert after['transient'][key] == agent['transient'][key]
    for key in ['time_s', 'input_v', 'voltage_v', 'current_a', 'energy_j']:
        assert abs(after['transient']['sample'][key] - agent['transient']['sample'][key]) < 1e-10
    reference.independent(before)
    reference.independent(after)
    reference.independent(agent)
    acknowledgment = float((int(committed['display_mach_ticks']) - tick) * scale / 1_000_000)
    interval = float((int(fresh['display_mach_ticks']) - tick) * scale / 1_000_000)
    return dict(trial=folder.name, measurement_status='available', complete_frames=complete, unique_images=len(used),
                roles=dict(roles), first_committed_parameter_frame=committed['index'], first_fresh_frame=fresh['index'],
                marker_to_committed_ms=acknowledgment, marker_to_fresh_ms=interval,
                eligible_latency_observations=0, latency_gate_passed=None)


result = validate(inputs)
rejected = []
first_image = next(i for i, row in enumerate(inputs['rows']) if row.get('png'))


def refuses(name, change):
    altered = copy.deepcopy(inputs)
    change(altered)
    try:
        validate(altered)
    except (AssertionError, KeyError, ValueError, FileNotFoundError):
        rejected.append(name)
        return
    raise AssertionError('Corruption admitted: ' + name)


refuses('shifted_display_clock', lambda d: d['rows'][first_image].__setitem__('display_mach_ticks', str(int(d['rows'][first_image]['display_mach_ticks']) + 24_000_000)))
refuses('changed_crop', lambda d: d['capture/configuration.json']['capture_plan']['window_global_points'].__setitem__('x', 2035))
refuses('missing_image', lambda d: d['rows'][first_image].__setitem__('png', 'missing.png'))
refuses('forged_eligibility', lambda d: d['rows'][first_image].__setitem__('eligible_for_latency', True))
refuses('unfinished_input', lambda d: d['capture-action.json'].__setitem__('input_call_completed', False))
refuses('wrong_native_run', lambda d: d['after-apply.json'].__setitem__('run_id', d['after-apply.json']['run_id'] + 1))
refuses('old_parameter_labeled_committed', lambda d: d['annotations'].__setitem__('committed_capacitor_sha256', d['annotations']['baseline_capacitor_sha256']))
refuses('old_tau_labeled_fresh', lambda d: d['annotations'].__setitem__('tau_templates', [dict(image='baseline.png', box=[329, 504, 383, 517])]))
result['rejected_mutations'] = rejected
if '--verify' in sys.argv:
    assert result == read(folder / 'analysis.json')
else:
    with (folder / 'analysis.json').open('x') as stream:
        json.dump(result, stream, indent=2)
        stream.write('\n')
print(json.dumps(result))
