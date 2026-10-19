"""Read-only verification; no CoreImage, capture, window or privacy operations."""
from pathlib import Path
import hashlib
import json

folder = Path(__file__).resolve().parent
root = folder.parents[2]


def require(condition, message):
    if not condition:
        raise SystemExit(message)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


checkpoint = json.loads((folder / 'checkpoint.json').read_text())
for name, digest in checkpoint['evidence_sha256'].items():
    require(sha(folder / name) == digest, f'Evidence changed: {name}')
for name, digest in checkpoint['source_sha256'].items():
    require(sha(root / name) == digest, f'Instrumentation source changed: {name}')
report = json.loads((folder / 'final-check-stdout.txt').read_text())
require(report['self_tests_passed'] and not report['capture_executed'], 'Offline checks did not pass')
require(len(report['journal_fault_cases']) == 7 and len(report['sample_fault_cases']) == 8,
        'Fault case inventory differs')
require(report['eligible_latency_observations'] == 0, 'Synthetic samples became eligible')
policy = json.loads((folder / 'prior-policy-report.json').read_text())
require(policy['expected_failure'] and policy['exit_code'] == 1, 'Prior error policy did not fail')
require(not checkpoint['new_probe_runtime_capture_qualified'] and
        checkpoint['eligible_latency_observations'] == 0 and checkpoint['latency_gate_passed'] is None,
        'Runtime/latency scope changed')
require(not checkpoint['app_rebuilt'] and not checkpoint['canonical_promoted'], 'App scope changed')
print(json.dumps(dict(passed=True, journal_fault_cases=7, sample_fault_cases=8,
                      capture_executed=False, eligible_latency_observations=0)))
