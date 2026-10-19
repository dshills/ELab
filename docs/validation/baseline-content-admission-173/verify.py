"""Read-only decision/source/provenance replay. No UI, capture or app process."""
from pathlib import Path
import hashlib
import gzip
import json
import subprocess

folder = Path(__file__).resolve().parent
root = folder.parents[2]
read = lambda path: json.loads(path.read_text())
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
checkpoint = read(folder / 'checkpoint.json')
assert {p.relative_to(folder).as_posix(): sha(p) for p in folder.rglob('*')
        if p.is_file() and p.name != 'checkpoint.json'} == checkpoint['evidence_sha256']
artifacts = read(folder / 'artifacts.json')
assert all(sha(Path(p)) == digest for p, digest in artifacts['preserved_artifacts'].items())
assert all(sha(root / p) == digest for p, digest in artifacts['source_sha256'].items())
prior = root / 'docs/validation/latency-followup-173'
assert sha(prior / 'checkpoint.json') == artifacts['prior_proof_checkpoint_sha256']
prior_check = read(prior / 'checkpoint.json')
assert all(sha(prior / p) == digest for p, digest in prior_check['evidence_sha256'].items())
app = read(root / 'docs/validation/sweep-smoke-admission/quality-default.json')
assert len(app['source_sha256']) == artifacts['application_go_sources_unchanged'] == 650
assert all(sha(root / p) == digest for p, digest in app['source_sha256'].items())
result = read(folder / 'result.json')
quality = read(folder / 'quality/report.json')
assert quality['passed'] and all(c['passed'] and c['exit_code'] == 0 for c in quality['checks'])
for report in [result, quality]:
    for path, digest in report['source_sha256'].items():
        if not path.endswith('.md'):  # Document-only wording update is explicit.
            assert sha(root / path) == digest
assert result['passed'] and result['cases'] == 45
assert (result['first_admitted'], result['first_refused'], result['fresh_state_refused']) == (22, 2, 21)
assert result['actual_capture'] is result['actual_input'] is False
assert result['overall_prd_performance_pass'] is None
assert sha(root / 'build/baseline-admission-tools/baseline-admission') == result['tool_sha256']
cases = read(folder / 'cases.json')
assert len(cases) == 45
for case in cases:
    expected, receipt = case['expected'], case['receipt']
    kind = expected['trial_id'].split('2-')[0]
    trial = prior / kind / 'session/trials' / expected['trial_id']
    # The source journal hashes refer to decompressed original bytes.
    journal = gzip.decompress((trial / 'capture/frames.jsonl.gz').read_bytes())
    assert hashlib.sha256(journal).hexdigest() == case['source_journal_sha256']
    assert sha(trial / 'capture/frame-store.json') == case['source_frame_store_sha256']
    rows = [json.loads(line) for line in journal.splitlines()]
    frame = next(r for r in rows if r.get('index') == expected['frame_index'] and r.get('png'))
    assert frame['png'] == expected['png'] and frame['png_sha256'] == expected['png_sha256']
    assert expected['tool_sha256'] == result['tool_sha256']
    assert receipt['admitted'] == case['should_admit']
process = subprocess.run(['/opt/homebrew/bin/node', str(root / 'tools/baseline-admission/replay_controller.mjs'),
                          str(folder / 'cases.json')], capture_output=True, text=True, check=True, timeout=10)
assert json.loads(process.stdout) == read(folder / 'controller-output.json')
assert read(folder / 'controller-output.json') == dict(cases=45, admitted=22, refused=23, actual_capture=False, actual_input=False)
print(dict(passed=True, offline_cases=45, controller_admitted=22, controller_refused=23,
           application_go_sources_unchanged=650, new_validation_go_files=3,
           actual_capture=False, actual_input=False, overall_prd_performance_pass=None))
