"""Read-only command, metadata-selector, source and scoped-evidence review."""
from pathlib import Path
import ast
import gzip
import hashlib
import json
import lzma

folder = Path(__file__).resolve().parent
root = folder.parents[2]
read = lambda p: json.loads(p.read_text())
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
checkpoint = read(folder / 'checkpoint.json')
assert {p.relative_to(folder).as_posix(): sha(p) for p in folder.rglob('*')
        if p.is_file() and p.name != 'checkpoint.json'} == checkpoint['evidence_sha256']
assert all(sha(Path(p)) == digest for p, digest in checkpoint['preserved_artifacts'].items())
declaration = read(folder / 'declaration.json')
assert sha(folder / 'collect.py') == declaration['producer_sha256']
assert sha(root / 'scripts/release-project-jobs.py') == declaration['helper_sha256']
ast.parse((folder / 'collect.py').read_text())
assert all(sha(root / name) == digest for name, digest in read(folder / 'referenced-files.json').items())
quality = read(root / 'docs/validation/sweep-smoke-admission/quality-default.json')
assert quality['passed'] and len(quality['source_sha256']) == 650
assert all(sha(root / name) == digest for name, digest in quality['source_sha256'].items())
data = json.loads(gzip.decompress((folder / 'discovery.json.gz').read_bytes()))
storage = read(folder / 'transcript-storage.json')
packed = (folder / 'transcript.json.xz').read_bytes()
assert len(packed) == storage['xz_bytes'] < 8 << 20
assert hashlib.sha256(packed).hexdigest() == storage['xz_sha256']
decoded = lzma.decompress(packed, memlimit=64 << 20)
assert len(decoded) == storage['json_bytes'] < 128 << 20
assert hashlib.sha256(decoded).hexdigest() == storage['json_sha256']
rows = json.loads(decoded)
analysis = read(folder / 'analysis.json')
details = data['details']
ids = {cap['id'] for cap in data['catalog']['capabilities']}
assert len(ids) == len(details) == 164 and set(details) == ids
assert data['before'] == data['after']
assert data['before']['components'] == [] and data['before']['document']['generation'] == 1
assert len(rows) == analysis['request_count'] == 168
assert sum(row['request']['command'] == 'capabilities.list' for row in rows) == 165
assert all(not row['response'].get('error') for row in rows)
assert all(row['request']['command'] in ['project.inspect', 'capabilities.list', 'app.close'] for row in rows)
for name, cap in details.items():
    assert name == cap['id'] and cap['input_schema']['x-command-contract']['version'] == 1
    response, = [r['response'] for r in rows if r['request']['command'] == 'capabilities.list' and r['request']['params'].get('command') == name]
    assert response['result']['capabilities'] == [cap]
assert sum(bool(c['input_schema']['x-command-contract'].get('completion')) for c in details.values()) == analysis['completion_blocks'] == 156
nonliteral_observers = [(name, value) for name, cap in details.items()
                        for value in (cap['input_schema']['x-command-contract'].get('completion') or {}).get('observe') or []
                        if value not in ids]
assert nonliteral_observers == [(s['command'], s['observer']) for s in analysis['metadata_selectors']]
nonliteral_cancel = [(name, c['cancel']) for name, cap in details.items()
                     if (c := cap['input_schema']['x-command-contract'].get('completion'))
                     and c.get('cancel') and c['cancel'] not in ids]
assert nonliteral_cancel == [(h['command'], h['text']) for h in analysis['qualified_cancellation_hints']]
for selector in analysis['metadata_selectors']:
    assert selector['query'] in ids and selector['query'] == 'project.inspect'
    value = data['after']
    for key in selector['result_path']:
        value = value[key]
    assert selector['observer'] == selector['query'] + '.' + '.'.join(selector['result_path'])
    assert selector['observer'] in details[selector['command']]['input_schema']['x-command-contract']['completion']['observe']
assert len(analysis['metadata_selectors']) == len(analysis['qualified_cancellation_hints']) == 2
for hint in analysis['qualified_cancellation_hints']:
    assert hint['leading_command'] in ids and hint['text'].split()[0] == hint['leading_command']
    assert hint['text'] == details[hint['command']]['input_schema']['x-command-contract']['completion']['cancel']
shutdown = read(folder / 'shutdown.json')
assert rows[-1]['request']['command'] == 'app.close' and rows[-1]['response']['result'] == shutdown
assert shutdown['shutdown']['exit_ready'] and shutdown['shutdown']['workers_joined']
assert read(folder / 'result.json')['exit_code'] == 0
assert analysis['collector_exit_code'] == read(folder / 'summary-refusal.json')['collector_exit_code'] == 1
audit = read(root / 'research/40-background-job-coverage.json')
assert sha(root / audit['source_quality']) == audit['source_quality_sha256']
assert audit['discovery'] == analysis and len(audit['families']) == 12
assert not audit['overall_agent_gate_passed'] and not audit['overall_prd_passed']
for family in audit['families']:
    assert all(name in ids for name in family['entry_commands'])
    test = family['representative_source_test']
    assert 'func ' + test['name'] + '(' in (root / test['file']).read_text()
    assert sha(root / test['file']) == quality['source_sha256'][test['file']]
finding, = audit['nonblocking_findings']
native = (root / finding['source']).read_text()
classification = native[native.index('action := "open"'):native.index('title := "Project "')]
producer = (root / finding['producer_source']).read_text()
for name in finding['commands']:
    assert name in ids and name in producer and name not in classification
print(dict(passed=True, commands=164, semantic_contracts=164, completion_blocks=156,
           lifecycle_review_families=12, source_findings=1, collector_summary_failure_retained=True,
           owned_agent_exit_code=0, actual_native_ui=False, complete_agent_qualification=False))
