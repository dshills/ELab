"""Read-only integrity and raw-protocol replay for the Python client follow-up."""
from pathlib import Path
import gzip
import hashlib
import json

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]


def sha(data):
    return hashlib.sha256(data).hexdigest()


def raw(path):
    data = path.read_bytes()
    return gzip.decompress(data) if path.suffix == '.gz' else data


def read(path):
    return json.loads(raw(path))


def verify_case(scenario, expected_count, agent_sha):
    folder = HERE/'live'/scenario
    report = read(folder/'report.json.gz')
    assert report['passed'] and report['scenario'] == scenario
    rows = [json.loads(line) for line in raw(folder/'transcript.jsonl.gz').splitlines()]
    assert len(rows) == expected_count
    requests = [json.loads(row['request_json']) for row in rows]
    expected_errors = [('app.close', 'unsaved-changes'), ('app.close.start', 'conflict')]
    if scenario == 'failure':
        expected_errors += [('view.pan', 'closed'), ('app.close.acknowledge', 'not-found')]
    errors, ids = [], set()
    for req, row in zip(requests, rows):
        response = row['response']
        assert req['version'] == response['version'] == 1
        assert req['id'] == response['id'] and req['id'] not in ids
        ids.add(req['id'])
        assert req['command'] != 'events.poll'
        if response.get('error'):
            errors.append((req['command'], response['error']['code']))
    assert errors == expected_errors
    process = read(folder/'process-exit.json.gz')
    assert process['exit_code'] == 0 and process['requests'] == expected_count
    assert process['agent_sha256'] == agent_sha
    assert raw(folder/'agent-stderr.txt.gz') == b''
    before, final = rows[22]['response']['result'], rows[-1]['response']['result']
    command = 'app.close' if scenario == 'success' else 'app.close.start'
    assert requests[22]['command'] == 'project.inspect'
    assert requests[23]['command'] == command and requests[23]['params'] == {'discard_changes': True}
    assert rows[18]['response']['result'] == rows[20]['response']['result'] == before
    for key in ('document', 'components', 'wires', 'run_id', 'electrical_revision', 'transient', 'undo_entry'):
        assert final.get(key) == before.get(key), key
    job = final['shutdown']
    assert final['closed'] and job['workers_joined'] and job['exit_ready']
    assert job['input_revision'] == before['revision']
    assert job['document_generation'] == before['document']['generation']
    assert job['actor'] and not job['cancelable'] and job['discard_recovery']
    assert not job['checkpoint_recovery']
    successful_ack = [row for req, row in zip(requests, rows)
                        if req['command'] == 'app.close.acknowledge' and not row['response'].get('error')]
    inspections = [row['response']['result'] for req, row in zip(requests, rows)
                   if req['command'] == 'app.close.inspect']
    if scenario == 'success':
        assert job['status'] == 'completed' and job['recovery_finalized']
        assert not inspections and not successful_ack
    else:
        assert len(inspections) == 2 and inspections[0] == inspections[1]
        failed = inspections[0]
        assert failed['id'] == job['id'] and failed['status'] == job['status'] == 'failed'
        assert failed['workers_joined'] and not failed['exit_ready'] and not failed['recovery_finalized']
        assert failed['error'] == job['error'] and failed['error']['code'] == 'shutdown-failed'
        assert failed['recovery_error'] == job['recovery_error'] and not failed.get('appearance_error')
        assert rows[25]['response']['result'] == rows[29]['response']['result']
        assert len(successful_ack) == 1 and requests[-1]['params'] == {'id': job['id']}
        fixtures = list((folder/'artifacts/recovery').glob('session-*.json/retain.gz'))
        assert len(fixtures) == 1 and raw(fixtures[0]) == b'owned shutdown fault fixture'
    saved = raw(folder/'artifacts/saved.elab.gz')
    save_work = rows[9]['response']['result']['project_work']
    assert requests[9]['command'] == 'project.save' and save_work['status'] == 'completed'
    assert sha(saved) == save_work['artifact_sha256'] and len(saved) == save_work['bytes']
    capsules = list((folder/'artifacts/recovery').glob('*.elab-recovery.gz'))
    assert len(capsules) == 1
    capsule = read(capsules[0])
    candidate = next(c for c in rows[12]['response']['result']['recovery']['candidates'] if c['id'] == capsule['id'])
    assert sha(raw(capsules[0])) == candidate['sha256']
    current = [c for c in before['recovery']['candidates'] if c['id'] != capsule['id']]
    assert len(current) == 1 and not (folder/'artifacts/recovery'/(current[0]['id']+'.elab-recovery.gz')).exists()
    for graph, expected in [(json.loads(saved)['graph'], 10000), (capsule['document']['graph'], 20000),
                            ({'components': final['components']}, 30000)]:
        assert next(c for c in graph['components'] if c['id'] == 'R1')['value'] == expected


def main():
    manifest_bytes = (HERE/'manifest.json').read_bytes()
    assert sha(manifest_bytes) == read(HERE/'checkpoint.json')['manifest_sha256']
    manifest = json.loads(manifest_bytes)
    actual = {str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file()
              and p.name not in ('manifest.json', 'checkpoint.json')}
    assert actual == set(manifest)
    for relative, entry in manifest.items():
        path = HERE/relative
        assert not path.is_symlink() and path.is_file() and path.resolve().is_relative_to(HERE)
        data = path.read_bytes()
        assert len(data) == entry['bytes'] and sha(data) == entry['sha256'], relative
    report = read(HERE/'report.json')
    assert report['passed'] and report['python_tests'] == 65
    for relative, digest in report['final_live_source_sha256'].items():
        assert sha((HERE/'sources'/(Path(relative).name+'.txt')).read_bytes()) == digest
        assert sha((ROOT/relative).read_bytes()) == digest, relative
    final = (HERE/'sources/persistence-demo.py.txt').read_text()
    negative = (HERE/'sources/pace-removed.py.txt').read_text()
    assert negative.count('pass  # controlled removal of pacing') == 1
    assert negative.replace('pass  # controlled removal of pacing', 'time.sleep(delay)') == final
    assert '20 not less than 20' in raw(HERE/'checks/pace-removed.txt.gz').decode()
    assert 'FAILED (failures=1)' in raw(HERE/'checks/selection-before.txt.gz').decode()
    tests = raw(HERE/'checks/python-tests-final.txt.gz').decode()
    assert 'Ran 65 tests' in tests and tests.rstrip().endswith('OK')
    base_pin = report['base_receipt']
    assert sha((ROOT/base_pin['path']).read_bytes()) == base_pin['sha256']
    base = read(ROOT/base_pin['path'])
    assert len(base['quality']['source_sha256']) == report['application_go_files_unchanged'] == 650
    for relative, digest in base['quality']['source_sha256'].items():
        assert sha((ROOT/relative).read_bytes()) == digest, relative
    assert len(base['runtime_source_inputs']) == report['packaged_runtime_inputs_unchanged'] == 358
    for relative, digest in base['runtime_source_inputs'].items():
        data = (ROOT/relative).read_bytes()
        assert hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest() == digest, relative
    for relative, digest in report['preserved'].items():
        assert sha((ROOT/relative).read_bytes()) == digest, relative
    for case in report['final_live_cases']:
        assert case['passed'] and case['agent_exit_code'] == 0
        verify_case(case['scenario'], case['requests'], base['agent_sha256'])
    assert sum(c['requests'] for c in report['final_live_cases']) == 55
    assert not any(report[k] for k in ('app_version_changed', 'new_native_window', 'latency_acceptance', 'full_mvp_complete'))
    print('Verified 65-test record, controlled failures, 55 raw requests and unchanged /176 app inputs/artifacts.')


if __name__ == '__main__':
    main()
