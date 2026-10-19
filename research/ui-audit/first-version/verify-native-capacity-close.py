"""Read-only bounded /169 close replay; the original timeout remains open."""

import gzip
import hashlib
import json
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / 'scripts'))
from wiring_replay import validate_release


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    data = path.read_bytes()
    return json.loads(gzip.decompress(data) if path.suffix == '.gz' else data)


def main():
    folder = ROOT / 'docs/validation/native-capacity-close-169'
    cp = read(folder / 'checkpoint.json')
    assert cp['status'] == 'bounded_non_reproduction_original_failure_open'
    for key in ['production_changed', 'new_release_created', 'original_failure_resolved', 'owner_acceptance', 'canonical_promoted']:
        assert cp[key] is False
    assert {p.relative_to(folder).as_posix() for p in folder.rglob('*') if p.is_file()} == set(cp['evidence_sha256']) | {'checkpoint.json'}
    for name, expected in cp['evidence_sha256'].items():
        assert sha(folder / name) == expected, name
    identity = validate_release(ROOT, ROOT / 'research/ui-audit/first-version/component-startup-review-checkpoint.json')
    assert identity == cp['source_release_identity']
    assert identity['source_commit'] == 'aa3b18b23a3cbc1863f097722b96a3263394edca'
    assert identity['manifest_file_entries_verified'] == 1393 and identity['runtime_inputs_verified'] == 349
    first = ROOT / 'docs/validation/component-startup-review-169/First-public-report.json.gz'
    assert sha(first) == cp['original_report_sha256']
    original = read(first)
    assert not original['passed']
    case = next(c for a in original['adapters'] if a['name'] == 'native-socket' for c in a['capacity']['cases'] if c['name'] == 'capacity-64-parts')
    assert len(case['calls']) == 162
    report = read(folder / 'report.json')
    assert report['passed'] and report['original_failure_reproduced'] is False
    assert report['release_identity'] == identity
    assert report['geometry_unchanged'] and report['geometry_before'] == report['geometry_after']
    assert read(folder / 'run.json')['exit_code'] == 0
    preflight = (folder / 'first-preflight-replay-output.txt').read_text()
    assert "KeyError: 'native_sha256'" in preflight and read(folder / 'first-preflight-run.json')['exit_code'] == 1
    isolation = read(folder / 'isolation.json')
    policy = gzip.decompress((folder / 'runtime.sb.gz').read_bytes())
    assert hashlib.sha256(policy).hexdigest() == isolation['policy_sha256']
    receipt = read(ROOT / 'research/ui-audit/first-version/component-startup-review-checkpoint.json')
    review = Path(receipt['review_path'])
    assert policy == gzip.decompress((review / isolation['sealed_policy_member']).read_bytes())
    assert isolation['native_sha256'] == receipt['native_sha256'] and isolation['agent_sha256'] == receipt['agent_sha256']
    for name in ['native', 'agent']:
        assert sha(folder / (name + '-launcher.txt')) == isolation['launcher_sha256'][name]
    sessions = report['sessions']
    assert {s['name'] for s in sessions} == {f'trial-{n}{suffix}' for n in range(1, 4) for suffix in ['', '-valid', '-excess']}
    assert len(sessions) == cp['native_sessions'] == 9 and cp['replays'] == 3
    assert sum(s['commands_completed'] for s in sessions) == cp['original_public_main_commands'] + cp['nested_reopen_commands'] == 510
    for session in sessions:
        assert session['closed'] and session['exit_code'] == 0 and session['owned_process_absent']
        assert session['commands_completed'] == session['commands_expected'] == (4 if session['name'].endswith(('valid', 'excess')) else 162)
        assert not any(session.get(key) for key in ['forced_cleanup', 'graceful_exit_timeout', 'diagnostic_signal'])
        assert 0 <= session['response_to_exit_s'] < 5 and session['close_response_s'] >= 0
        directory = folder / session['name']
        before = read(directory / 'before-close.json.gz')
        closed = read(directory / 'close-response.json.gz')
        graph = case['graph'] if session['name'].endswith('valid') else case['boundary']['graph']
        assert before['components'] == graph['components'] and before['wires'] == graph['wires']
        assert not before['closed'] and before['paused'] and not before['document']['dirty'] and before['document']['recording_count'] == 1
        shutdown = closed['shutdown']
        assert closed['closed'] and shutdown['status'] == 'completed' and shutdown['exit_ready'] and shutdown['workers_joined'] and shutdown['recovery_finalized']
        assert not any(shutdown.get(key) for key in ['error', 'recovery_error', 'appearance_error'])
        assert closed['components'] == before['components'] and closed['wires'] == before['wires']
    for trial in range(1, 4):
        base = folder / f'trial-{trial}'
        recording = gzip.decompress((base / 'recording.json.gz').read_bytes())
        for suffix in ['valid', 'excess']:
            assert recording == gzip.decompress((folder / f'trial-{trial}-{suffix}/recording-copy.json.gz').read_bytes())
        for filename, graph in [('project.elab', case['graph']), ('excess.elab', case['boundary']['graph'])]:
            assert read(base / (filename + '.gz'))['graph'] == graph
    assert max(s['close_response_s'] for s in sessions) == cp['max_close_response_s']
    assert max(s['response_to_exit_s'] for s in sessions) == cp['max_response_to_exit_s']
    print(json.dumps(dict(bounded_replay_verified=True, native_sessions=9, original_failure_resolved=False, production_changed=False, new_release_created=False, owner_acceptance=False)))


if __name__ == '__main__':
    main()
