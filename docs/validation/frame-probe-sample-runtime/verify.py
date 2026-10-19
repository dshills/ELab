"""Read-only data/source replay; never opens an app or requests capture access."""
from pathlib import Path
import hashlib
import json
import subprocess
import sys

folder = Path(__file__).resolve().parent
root = folder.parents[2]
checkpoint = json.loads((folder / 'checkpoint.json').read_text())
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
for name, digest in checkpoint['evidence_sha256'].items():
    if sha(folder / name) != digest:
        raise SystemExit(f'Evidence changed: {name}')
for name, digest in checkpoint['recorder_source_sha256'].items():
    if sha(root / name) != digest:
        raise SystemExit(f'Recorder source changed: {name}')
if (checkpoint['eligible_latency_observations'] != 0 or checkpoint['latency_gate_passed'] is not None
        or checkpoint['canonical_promoted'] or checkpoint['app_rebuilt']):
    raise SystemExit('Checkpoint scope changed')
subprocess.run([sys.executable, '-B', str(folder / 'analyze.py'), '--artifacts', str(folder), '--verify'], check=True)
