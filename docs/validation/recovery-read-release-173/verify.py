"""Read-only provenance, captured JSON, chunk and independent RC-law replay."""
from pathlib import Path
import ast
import base64
import hashlib
import json
import math

folder = Path(__file__).resolve().parent
root = folder.parents[2]
read = lambda p: json.loads(p.read_text())
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
checkpoint = read(folder / 'checkpoint.json')
assert {p.relative_to(folder).as_posix(): sha(p) for p in folder.rglob('*')
        if p.is_file() and p.name != 'checkpoint.json'} == checkpoint['evidence_sha256']
assert all(sha(Path(p)) == digest for p, digest in checkpoint['preserved_artifacts'].items())
quality = read(root / 'docs/validation/sweep-smoke-admission/quality-default.json')
assert len(quality['source_sha256']) == 650
assert all(sha(root / p) == digest for p, digest in quality['source_sha256'].items())
for name in ['preflight', 'final']:
    declaration = read(folder / name / 'declaration.json')
    ast.parse((folder / name / 'producer.py').read_text())
    assert sha(folder / name / 'producer.py') == declaration['source_sha256']['scripts/recovery-read-demo.py']
    assert declaration['agent_sha256'] == '134f9863e2885e384035e30550adfc5fe0bea1bdee9f3d3cd42feaf802bb7de5'
final = folder / 'final'
declaration = read(final / 'declaration.json')
assert all(sha(root / p) == digest for p, digest in declaration['source_sha256'].items())
report = read(final / 'result.json')
assert report['passed'] and report['client_exit_code'] == 0 and len(report['checks']) == 10
assert report['transport'] == 'headless-stdio' and report['agent_binary_sha256'] == declaration['agent_sha256']
assert read(final / 'invocation.json')['exit_code'] == 0 and (final / 'stderr.txt').read_bytes() == b''
source = read(final / 'files/source.elab')
assert sha(final / 'files/source.elab') == report['source_sha256']
capsule_path, = (final / 'files/recovery').glob('*.elab-recovery')
assert sha(capsule_path) == report['capsule_sha256']
capsule = read(capsule_path)
assert capsule['document']['graph'] == report['captured_graph'] == source['graph']
observed = report['observed_reads']
assert len(observed) == 11 and len({r['work']['id'] for r in observed}) == 11
completed = [r for r in observed if r['work']['status'] == 'completed']
failed = [r for r in observed if r['work']['status'] == 'failed']
assert len(completed) == 2 and len(failed) == 9
assert [r['work']['error']['code'] for r in failed] == ['file-conflict'] + ['invalid-recovery'] * 8
for entry in completed:
    work = entry['work']
    raw = base64.b64decode(entry['result_json_base64'], validate=True)
    assert len(raw) == work['result_bytes']
    assert hashlib.sha256(raw).hexdigest() == entry['reconstructed_sha256'] == work['result_sha256']
    assert json.loads(raw) == entry['structured_result']
    offset = 0
    for start, end in entry['chunk_ranges']:
        assert start == offset and start < end <= min(start + 512, len(raw))
        offset = end
    assert offset == len(raw)
    assert entry['eof'] == dict(id=work['id'], offset=len(raw), next_offset=len(raw), total=len(raw),
                                sha256=work['result_sha256'], json_base64='')
    assert entry['terminal_cancel_refusal'] == 'unavailable'
    assert entry['invalid_chunk_bounds_refusal'] == 'invalid-params'
    result = entry['structured_result']
    if work['input']['kind'] == 'inspection':
        assert result['inspection']['document']['graph'] == capsule['document']['graph']
        assert result['inspection']['document']['recordings'] == [entry['data']['metadata'] for entry in capsule['document']['recordings']['entries']]
    else:
        assert all(result['samples'][key] == value for key, value in report['sample_page'].items())
        stored_rows = capsule['document']['recordings']['entries'][0]['data']['rows']
        assert result['samples']['rows'] == stored_rows[result['samples']['offset']:result['samples']['next_offset']]
for entry in failed:
    assert entry['work']['result_bytes'] == 0 and 'result_sha256' not in entry['work']
    assert entry['structured_result_refusal'] == 'not-ready'
assert report['retained_jobs'] == [r['work'] for r in failed[-8:]]
assert report['evicted_read_id'] == completed[0]['work']['id']
components = {p['id']: p for p in source['graph']['components']}
tau = components['R1']['value'] * components['C1']['value']
voltage = components['V1']['value']
page = report['sample_page']
assert (page['offset'], page['next_offset'], page['total']) == (10, 12, 21)
assert [p['id'] for p in page['probes']] == ['v:C1.1', 'v:V1.positive']
for row in page['rows']:
    expected = voltage * -math.expm1(-row['time_s'] / tau)
    assert abs(row['values'][0] - expected) < 1e-12 and row['values'][1] == voltage
refusals = read(folder / 'preflight/refusals.json')
assert [r['name'] for r in refusals] == ['hash-mismatch', 'optimized', 'missing-pin']
assert all(r['exit_code'] == 2 and not r['artifacts_created'] and not r['stdout'] for r in refusals)
assert 'SHA-256 differs' in refusals[0]['stderr'] and 'without -O' in refusals[1]['stderr']
assert 'must be supplied together' in refusals[2]['stderr']
print(dict(passed=True, public_checks=10, observed_reads=11, completed=2, failed=9,
           retained_terminal_jobs=8, independent_rc_rows=2, preflight_refusals=3,
           application_go_sources_unchanged=650, actual_native_ui=False))
