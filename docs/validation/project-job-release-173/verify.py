"""Read-only replay of public job responses and unchanged release identities."""
from pathlib import Path
import ast
import gzip
import hashlib
import json

folder = Path(__file__).resolve().parent
root = folder.parents[2]
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
read = lambda p: json.loads(p.read_text())
checkpoint = read(folder / 'checkpoint.json')
assert {p.relative_to(folder).as_posix(): sha(p) for p in folder.rglob('*')
        if p.is_file() and p.name != 'checkpoint.json'} == checkpoint['evidence_sha256']
assert all(sha(Path(p)) == digest for p, digest in checkpoint['preserved_artifacts'].items())
quality = read(root / 'docs/validation/sweep-smoke-admission/quality-default.json')
assert len(quality['source_sha256']) == 650
assert all(sha(root / p) == digest for p, digest in quality['source_sha256'].items())
for name, producer in [('initial-client-failure', 'initial-producer.py'),
                       ('corrected-client', 'producer.py'), ('final', 'producer.py')]:
    path = folder / name
    declaration = read(path / 'declaration.json')
    assert sha(path / producer) == declaration['producer_sha256']
    assert declaration['agent_sha256'] == '134f9863e2885e384035e30550adfc5fe0bea1bdee9f3d3cd42feaf802bb7de5'
    ast.parse((path / producer).read_text())
    rows = json.loads(gzip.decompress((path / 'transcript.json.gz').read_bytes()))
    if name == 'initial-client-failure':
        assert rows[-1]['request']['command'] == 'project.load.start'
        assert rows[-1]['request']['expected_command_revision'] == 7
        assert rows[-1]['response']['error']['code'] == 'conflict'
        assert 'current command revision is 8' in rows[-1]['response']['error']['message']
        continue
    result = read(path / 'result.json')
    assert result['passed'] and result['exit_code'] == 0 and len(result['checks']) == 6
    assert not result['actual_native_ui'] and not result['actual_capture'] and not result['actual_printing']
    assert len(rows) == 26
    assert all(row['request']['id'] == row['response']['id'] for row in rows)
    failed = [row for row in rows if row['request']['command'] == 'project.load' and row['response'].get('error')]
    assert len(failed) == 2 and all(row['response']['error']['code'] == 'invalid-document' for row in failed)
    assert failed[0]['request']['key'] == failed[1]['request']['key']
    assert failed[1]['response']['duplicate'] and failed[1]['response']['result'] == failed[0]['response']['result']
    first_id = failed[0]['response']['result']['project_work']['id']
    loaded = next(row for row in rows if row['request']['command'] == 'project.load' and not row['response'].get('error'))
    assert loaded['request']['key'] != failed[0]['request']['key']
    assert loaded['response']['result']['project_work']['id'] != first_id
    assert loaded['response']['result']['project_work']['status'] == 'completed'
    start = [row for row in rows if row['request']['command'] == 'project.load.start']
    assert len(start) == 2 and start[1]['response']['duplicate']
    assert start[0]['response']['result']['project_work'] == start[1]['response']['result']['project_work']
    assert start[0]['response']['result']['project_work']['status'] == 'queued'
    canceled = [row for row in rows if row['request']['command'] == 'project.work.cancel']
    assert len(canceled) == 2 and all(row['response']['error']['code'] == 'operation-unavailable' for row in canceled)
    final = rows[-1]['response']['result']
    assert rows[-1]['request']['command'] == 'app.close' and final['closed']
    assert final['shutdown']['exit_ready'] and final['shutdown']['workers_joined']
assert sha(root / 'scripts/release-project-jobs.py') == read(folder / 'final/declaration.json')['producer_sha256']
print(dict(passed=True, public_checks=6, final_requests=26, initial_client_failure_retained=True,
           application_go_sources_unchanged=650, actual_native_ui=False, actual_capture=False))
