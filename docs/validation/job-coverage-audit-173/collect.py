"""Read every detailed contract from one frozen, empty owned headless agent."""
from pathlib import Path
import gzip
import hashlib
import importlib.util
import json

root = Path('/Users/dshills/Development/projects/ELab')
out = Path(__file__).resolve().parent
binary = Path('/private/tmp/elab-first-version-173-signed-runtime/elab-agent')
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
expected = '134f9863e2885e384035e30550adfc5fe0bea1bdee9f3d3cd42feaf802bb7de5'
assert sha(binary) == expected
helper_path = root / 'scripts/release-project-jobs.py'
declaration = dict(agent_sha256=expected, producer_sha256=sha(Path(__file__)), helper_sha256=sha(helper_path),
                   scope='164 read-only detailed discovery queries in owned empty headless session')
with (out / 'declaration.json').open('x') as stream:
    json.dump(declaration, stream, indent=2)
spec = importlib.util.spec_from_file_location('release_jobs', helper_path)
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
client = helper.Client(binary)
try:
    before = client.ask('project.inspect')
    assert before['components'] == [] and before['document']['generation'] == 1 and not before['document']['dirty']
    catalog = client.ask('capabilities.list')
    ids = [cap['id'] for cap in catalog['capabilities']]
    assert len(ids) == len(set(ids)) == 164
    details = {}
    for identity in ids:
        found = client.ask('capabilities.list', dict(command=identity))['capabilities']
        assert len(found) == 1 and found[0]['id'] == identity
        details[identity] = found[0]
    after = client.ask('project.inspect')
    assert after == before
    data = dict(catalog=catalog, details=details, before=before, after=after)
    (out / 'discovery.json.gz').write_bytes(gzip.compress(json.dumps(data, indent=2).encode(), mtime=0))
    result = client.ask('app.close', {}, True)
    assert result['shutdown']['exit_ready'] and result['shutdown']['workers_joined']
    (out / 'shutdown.json').write_text(json.dumps(result, indent=2) + '\n')
finally:
    exit_code = client.close()
    (out / 'transcript.json.gz').write_bytes(gzip.compress(json.dumps(client.transcript, indent=2).encode(), mtime=0))
with (out / 'result.json').open('x') as stream:
    json.dump(dict(passed=True, commands=164, discovery_preserves_snapshot=True,
                   exit_code=exit_code, actual_native_ui=False, actual_capture=False), stream, indent=2)
missing = []
invalid_observers = []
completion = {}
for identity, cap in details.items():
    contract = cap['input_schema'].get('x-command-contract')
    if not contract:
        missing.append(identity)
        continue
    if 'completion' in contract:
        completion[identity] = contract['completion']
        for observer in contract['completion'].get('observe', []):
            if observer not in ids:
                invalid_observers.append([identity, observer])
print(json.dumps(dict(commands=len(details), without_contract=missing, completion_contracts=len(completion),
                      invalid_observers=invalid_observers, completion=completion), indent=2))
