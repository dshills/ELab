"""Independent rational replay of retained clock data; no native clock/UI call."""
from copy import deepcopy
from fractions import Fraction
from pathlib import Path
import gzip
import hashlib
import json
import math

folder = Path(__file__).resolve().parent
root = folder.parents[2]


def require(condition, message):
    if not condition:
        raise ValueError(message)


def read(name):
    path = folder / name
    data = path.read_bytes()
    return json.loads(gzip.decompress(data) if name.endswith('.gz') else data)


def validate(report):
    require(report['passed'] and report['unbracketed_observations'] == 0, 'Clock check failed')
    require(len(report['observations']) == 200 and len(report['conversions']) == 4, 'Incomplete clock data')
    require(not report['capture_executed'] and not report['input_operated'] and
            not report['physical_display_timing_calibrated'] and not report['pixel_mapping_calibrated'] and
            not report['display_visibility_validated'] and report['eligible_latency_observations'] == 0 and
            report['latency_gate_passed'] is None, 'Clock evidence scope changed')
    ratio = Fraction(report['timebase_numerator'], report['timebase_denominator'] * 10**9)
    require(ratio > 0, 'Invalid Mach timebase')
    for row in report['observations']:
        def host(prefix):
            scale = row[prefix + '_timescale']
            require(scale > 0, 'Invalid host timescale')
            return Fraction(int(row[prefix + '_value']), scale)
        before, value, after = host('before_host'), host('host'), host('after_host')
        require(before <= value <= after and row['inside_host_time_bracket'], 'Host time outside exact bracket')
        require(math.isfinite(row['host_seconds']) and abs(float(value) - row['host_seconds']) < 1e-8,
                'Serialized host seconds differ')
        raw = [int(row[key + '_mach_ticks']) for key in ('before', 'host', 'after')]
        require(raw[0] <= raw[2] and row['roundtrip_ticks_inside_mach_bracket'] == (raw[0] <= raw[1] <= raw[2]),
                'Raw round-trip diagnostic changed')
        for prefix, ticks in (('before_host', raw[0]), ('after_host', raw[2])):
            require(abs(host(prefix) - ticks * ratio) <= Fraction(1, row[prefix + '_timescale']),
                    'Mach endpoint conversion differs beyond one representable host unit')
    for row in report['conversions']:
        ticks = int(row['mach_ticks'])
        require(abs(int(row['restored_mach_ticks']) - ticks) <= 1, 'Conversion lost more than one tick')
        require(abs(row['host_seconds'] - float(ticks * ratio)) < 1e-8, 'Conversion differs beyond 10 ns')


checkpoint = read('checkpoint.json')
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
for name, digest in checkpoint['evidence_sha256'].items():
    require(sha(folder / name) == digest, f'Evidence changed: {name}')
for name, digest in checkpoint['source_sha256'].items():
    require(sha(root / name) == digest, f'Clock source changed: {name}')
report = read('final-report.json.gz')
validate(report)
for mutation in ('one-second-shift', 'forged-eligibility'):
    altered = deepcopy(report)
    if mutation == 'one-second-shift':
        row = altered['observations'][0]
        row['host_value'] = str(int(row['host_value']) + row['host_timescale'])
    else:
        altered['eligible_latency_observations'] = 1
    try:
        validate(altered)
    except ValueError:
        continue
    raise ValueError(f'Bad evidence admitted: {mutation}')
print(json.dumps(dict(passed=True, exact_host_brackets=200, conversion_cases=4,
                      rejected_mutations=2, capture_executed=False, eligible_latency_observations=0)))
