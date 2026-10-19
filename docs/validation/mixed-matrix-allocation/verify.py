"""Read-only verification of this source/backend checkpoint; no app launch."""
from pathlib import Path
import gzip
import hashlib
import json


folder = Path(__file__).resolve().parent
root = folder.parents[2]


def require(condition, message):
    if not condition:
        raise SystemExit(message)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(name):
    path = folder / name
    data = gzip.decompress(path.read_bytes()) if name.endswith('.gz') else path.read_bytes()
    return json.loads(data)


checkpoint = read('checkpoint.json')
for name, digest in checkpoint['evidence_sha256'].items():
    require(sha(folder / name) == digest, f'Evidence changed: {name}')
for name, digest in checkpoint['source_sha256'].items():
    require(sha(root / name) == digest, f'Source changed: {name}')
quality = read(checkpoint['final_quality_report'])
require(quality['passed'] and quality['sources_unchanged'], 'Final source gate failed')
require(quality['source_sha256'] == checkpoint['source_sha256'], 'Source inventories differ')

before, after = read('before-samples.json'), read('final-samples.json')
require(len(before) == len(after) == 2, 'Missing flasher variant')
for old, new in zip(before, after):
    require(old['samples'] == new['samples'] == 600, 'Missing accepted flasher outputs')
    for key in ('capacitance_scale', 'graph_sha256', 'accepted_memory_and_result_sha256',
                'total_attempts', 'peak_sample_attempts'):
        require(old[key] == new[key], f'Flasher comparison changed: {key}')
capacity_before = read('capacity-before.json')[0]
capacity_after = read('capacity-final.json')[0]
require(capacity_after['samples'] == 200 and capacity_after['matrix_size'] == 61,
        'Capacity fixture dimensions changed')
for key in ('accepted_sha256', 'attempts', 'peak_attempts', 'samples', 'matrix_size'):
    require(capacity_before[key] == capacity_after[key], f'Capacity comparison changed: {key}')

public = read('public-report.json.gz')
require(public['passed'], 'Public backend run failed')
require([adapter['name'] for adapter in public['adapters']] ==
        ['in-process', 'headless-stdio', 'headless-socket'], 'Unexpected adapter scope')
total = 0
for adapter in public['adapters']:
    require(adapter['equivalent_to_in_process'], 'Backend parity failed')
    timing = adapter['inductor_performance']
    require(timing['backend_targets_passed'], 'Backend coil target failed')
    for case in timing['cases']:
        require(len(case['samples']) == 20, 'Incomplete coil cohort')
        total += len(case['samples'])
require(total == 480, 'Unexpected coil sample count')

budget = read('final-production-audit.json')
require(budget['passed'] and budget['maximum_attempts'] == 512 and
        budget['maximum_wall_ms'] == 100, 'Production guard changed')
idle, pressure = budget['results']
require(idle['outcome'] == 'completed' and idle['last_accepted_time_s'] == 6,
        'Idle fixture did not finish')
require(pressure['code'] == 'integration-time-limit' and
        pressure['failed_sample_state_preserved'], 'Pressure failure lost accepted state')
scope = read('comparison.json')
require(scope['visible_latency_observations'] == 0 and
        not scope['native_current_source_qualified'] and not scope['new_release_created'] and
        not scope['canonical_promoted'], 'Checkpoint scope changed')
print(json.dumps(dict(passed=True, source_files=len(checkpoint['source_sha256']),
                      exact_accepted_records=1400, public_backends=3,
                      real_clock_coil_samples=total, visible_latency_observations=0)))
