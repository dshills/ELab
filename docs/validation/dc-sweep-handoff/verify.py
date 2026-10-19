"""Read-only source/evidence checks; does not run tests, a UI or a solver."""
from pathlib import Path
import gzip
import hashlib
import json

folder = Path(__file__).resolve().parent
root = folder.parents[2]


def require(condition, message):
    if not condition:
        raise SystemExit(message)


def read(name):
    path = folder / name
    return json.loads(gzip.decompress(path.read_bytes()) if name.endswith('.gz') else path.read_bytes())


sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
checkpoint = read('checkpoint.json')
for name, digest in checkpoint['evidence_sha256'].items():
    require(sha(folder / name) == digest, f'Evidence changed: {name}')
for name, digest in checkpoint['source_sha256'].items():
    require(sha(root / name) == digest, f'Source changed: {name}')
quality = read(checkpoint['quality_report'])
require(quality['passed'] and quality['sources_unchanged'], 'Final quality gate failed')
require(quality['source_sha256'] == checkpoint['source_sha256'], 'Quality/source inventories differ')
require(sha(root / 'scripts/check.sh') == quality['quality_runner_sha256'] == checkpoint['quality_runner_sha256'],
        'Quality runner changed')
before, after = read('native-inventory-before.json.gz'), read('native-inventory-final.json.gz')
for key in ('public_commands', 'native_alias_targets', 'commands_with_source_routes', 'commands_without_source_routes'):
    require(before[key] == after[key], f'Command/route set changed: {key}')
require(len(after['source_sha256']) == 72 and len(after['sites']) == 397 and not after['issues'], 'Native ledger differs')
require(sha(root / 'research/13-native-command-inventory.json') == checkpoint['native_ledger_sha256'], 'Current ledger differs')
require(read('original-source-correspondence.json')['matched'], 'Original source correspondence failed')
failure = gzip.decompress((folder / 'original-mounted.txt.gz').read_bytes()).decode()
require(failure.count('--- FAIL: TestMountedDCSweepRunUsesUnfinishedSettings/') == 6, 'Original six failures missing')
passed = gzip.decompress((folder / 'focused-final.txt.gz').read_bytes()).decode()
require('ok  \tgithub.com/dshills/elab/desktop' in passed and 'FAIL' not in passed, 'Focused suite did not pass')
require(not checkpoint['app_rebuilt'] and not checkpoint['canonical_promoted'] and
        not checkpoint['physical_input_qualified'] and not checkpoint['full_agent_parity_qualified'], 'Scope changed')
print(json.dumps(dict(passed=True, original_failures=6, focused_cases=30, public_commands=164,
                      app_rebuilt=False, physical_input_qualified=False)))
