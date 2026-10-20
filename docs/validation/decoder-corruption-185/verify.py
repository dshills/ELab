"""Read-only metadata/source/corpus replay; does not execute a diagnostic."""
from pathlib import Path
import base64
import gzip
import hashlib
import json
import sys

if sys.flags.optimize:
    raise SystemExit('Assertions required')
P = Path(__file__).resolve().parent
R = P.parents[2]


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def raw(name):
    return gzip.decompress((P / (name + '.gz')).read_bytes())


def read(name):
    return json.loads(raw(name))


checkpoint = json.loads((P / 'checkpoint.json').read_text())['files']
assert set(checkpoint) == {
    str(f.relative_to(P)) for f in P.rglob('*')
    if f.is_file() and f != P / 'checkpoint.json'
}
for name, expected in checkpoint.items():
    path = P / name
    assert not path.is_symlink() and path.resolve().is_relative_to(P)
    assert digest(path.read_bytes()) == expected, name
for name, info in json.loads((P / 'manifest.json').read_text()).items():
    stored = (P / name).read_bytes()
    original = gzip.decompress(stored)
    assert digest(stored) == info['stored_sha256']
    assert digest(original) == info['original_sha256']
    assert len(original) == info['original_bytes']

plan = read('declaration.json')
artifacts = json.loads((P / 'artifacts.json').read_text())
terminal = read('terminal-results.json')
stats = read('structured-result.json')
assert stats == dict(mutations=20028, accepted_round_trips=886, refused=19142,
                     public_refusals=18, dirty_work_preserved=True,
                     manual_save_unchanged=True)
assert len(plan['application_go_sources']) == 659
for name, expected in plan['application_go_sources'].items():
    assert digest((R / name).read_bytes()) == expected, name
assert not (R / 'workbench/decoder_corruption_diagnostic_test.go').exists()
assert digest(raw('diagnostic_test.go.txt')) == plan['harness_sha256'] == artifacts['harness_sha256']
assert digest(raw('overlay.json')) == plan['overlay_sha256'] == artifacts['overlay_sha256']
assert digest((R / 'build/decoder-corruption-185-work/workbench-diagnostic.test').read_bytes()) == plan['test_binary_sha256'] == artifacts['current_test_binary_sha256']
assert digest((R / 'research/ui-audit/first-version/inspector-width-review-checkpoint.json').read_bytes()) == artifacts['current_receipt_sha256']
assert digest((R / 'build/ELab.app/Contents/MacOS/ELab').read_bytes()) == artifacts['canonical_sha256']
assert artifacts['production_go_sources_unchanged'] == 659
assert not any(artifacts[k] for k in ('app_opened', 'production_source_changed',
                                    'new_app_release', 'overall_corruption_gate_pass',
                                    'owner_acceptance', 'full_mvp_complete'))

for name in ('build', 'prepare', 'structured', 'unguided_fuzz', 'guided_fuzz',
             'vet', 'formatting', 'final_race'):
    assert terminal[name]['exit_code'] == 0, name
assert terminal['initial_race']['exit_code'] == 1
assert terminal['final_race']['handle'] == 89343
assert terminal['final_race']['package_elapsed_seconds'] == 195.582
assert terminal['unguided_fuzz']['executions'] == 175612
assert terminal['unguided_fuzz']['coverage_guided'] is False
assert terminal['guided_fuzz']['executions'] == 24200
assert terminal['guided_fuzz']['coverage_guided'] is True

unguided = raw('fuzz-output.txt')
guided = raw('guided-fuzz-output.txt')
initial_race = raw('race-output.txt')
final_race = raw('race-final-output.txt')
assert unguided.startswith(b'warning: the test binary was not built with coverage instrumentation')
assert b'execs: 175612 ' in unguided and unguided.endswith(b'PASS\n')
assert b'gathering baseline coverage: 72/72 completed' in guided
assert b'execs: 24200 ' in guided and b'new interesting: 475 (total: 547)' in guided
assert b'PASS\n' in guided and b'coverage instrumentation' not in guided
assert initial_race.startswith(b'panic: test timed out after 2m0s')
assert b'WARNING: DATA RACE' not in initial_race
assert b'195.582s' in final_race and b'FAIL' not in final_race
assert digest(unguided) == read('guided-declaration.json')['initial_fuzz_log_sha256']
assert digest(initial_race) == read('race-final-declaration.json')['original_race_log_sha256']
assert read('race-final-declaration.json')['reduced_cases'] is False
assert read('race-final-declaration.json')['product_limits_changed'] is False

fixtures = read('fixtures.json')
assert len(fixtures) == 6 and sorted(f['kind'] for f in fixtures) == [
    'project', 'project', 'recording', 'recording', 'recording', 'sweep']
for fixture in fixtures:
    data = raw('fixtures/' + fixture['name'] + '.json')
    assert json.loads(data) is not None
refused_files = [name for name in checkpoint if name.startswith('refused-') and name.endswith('.json.gz')]
assert len(refused_files) == 18
corpus = read('guided-corpus.json')
assert len(corpus) == artifacts['corpus_files'] == 475
assert len({c['path'] for c in corpus}) == len(corpus)
for entry in corpus:
    path = Path(entry['path'])
    assert not path.is_absolute() and '..' not in path.parts
    data = base64.b64decode(entry['base64'], validate=True)
    assert len(data) == entry['bytes'] and digest(data) == entry['sha256']
    assert data.startswith(b'go test fuzz v1\n')
assert sum(c['bytes'] for c in corpus) == artifacts['corpus_bytes'] == 30598
assert digest((R / 'research/34-engineering-checkpoint-history.md').read_bytes()) == json.loads((P / 'plan-relocation.json').read_text())['history_sha256']
print(json.dumps(dict(passed=True, metadata_and_source_only=True,
                      structured_mutations=20028, public_refusals=18,
                      guided_fuzz_executions=24200, retained_corpus_inputs=475,
                      unguided_run_separate=True, original_race_timeout_retained=True,
                      final_full_race_pass=True, application_go_sources_unchanged=659,
                      overall_corruption_gate_pass=False, app_opened=False)))
