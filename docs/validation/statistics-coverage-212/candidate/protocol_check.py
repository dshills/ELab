"""Check complete saved-time coverage through the exact packaged agent."""
import hashlib, importlib.util, json, math
from pathlib import Path

root = Path('/Users/dshills/Development/projects/ELab')
stage = Path(__file__).resolve().parent
build = json.loads((stage/'build-result-signed.json').read_text())
binary = Path(build['review'])/'elab-agent'
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
assert sha(binary) == build['agent_sha256']
fixture = root/'workbench/testdata/recorded-statistics-short-tail.json'
fixture_sha = sha(fixture)
original = json.loads(fixture.read_text())
spec = importlib.util.spec_from_file_location('client', root/'scripts/persistence-demo.py')
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
c = helper.Client(command_preconditions=True, agent_binary=binary)
report = dict(source_commit=build['source_commit'], agent_sha256=build['agent_sha256'],
              transport='headless-stdio', fixture_sha256=fixture_sha,
              refusal_cases=0, covered_cases=0, results=[])

def covered(p):
    before = c.ask('project.inspect')
    result = c.ask('recording.statistics', p)
    assert before == c.ask('project.inspect'), 'query changed state'
    assert result['settings'] == p and result['duration_s'] == p['end_s']-p['start_s']
    assert result['minimum'] == result['maximum'] == 5
    for key, expected in [('time_mean', 5), ('time_rms', 5), ('ac_rms', 0)]:
        assert math.isclose(result[key], expected, rel_tol=0, abs_tol=1e-13), result
    report['covered_cases'] += 1
    report['results'].append(result)

try:
    imported = c.ask('verification.import', {'path': str(fixture)}, True)
    selected = imported['recording']
    identifier, last = selected['id'], selected['last_sample_time_s']
    assert selected['duration_s'] == .02 and last == .02-5e-13
    metadata = c.ask('operation.inspect', {'id': identifier})['metadata']
    rows = c.ask('operation.samples', {'id': identifier, 'offset': 0, 'limit': metadata['sample_count']})['rows']
    assert rows == original['rows'], 'import normalized saved rows'
    endpoint = c.ask('operation.samples', {'id': identifier, 'offset': metadata['sample_count']-1, 'limit': 1})['rows'][0]['time_s']
    assert endpoint == last
    params = dict(id=identifier, probe_id='v:V1.positive', start_s=0, end_s=last)
    covered(params)
    cached = c.ask('recording.statistics.set', params, True)
    assert cached['recording']['interval_statistics']['settings'] == params
    for start, end in [(0, .02), (.02-1e-12, .02), (last, .02), (.02-1e-12, math.nextafter(last, math.inf))]:
        p = dict(params, start_s=start, end_s=end)
        for command in ['recording.statistics', 'recording.statistics.set']:
            before = c.ask('project.inspect')
            c.ask(command, p, mutate=command.endswith('.set'), expect='unavailable')
            assert before == c.ask('project.inspect'), 'refusal changed cached/live/document state'
            report['refusal_cases'] += 1
    for start in [.02-1e-12, math.nextafter(last, 0)]:
        covered(dict(params, start_s=start))
    export = stage/'roundtrip.json'
    assert not export.exists()
    c.ask('verification.export', {'id': identifier, 'path': str(export), 'format': 'json'}, True)
    saved = json.loads(export.read_text())
    assert saved['rows'] == original['rows'] and saved['metadata'] == original['metadata']
    reopened = c.ask('verification.import', {'path': str(export)}, True)['recording']
    assert reopened['last_sample_time_s'] == last and reopened['duration_s'] == .02
    covered(dict(params, id=reopened['id']))
    assert sha(fixture) == fixture_sha
    report.update(saved_rows_and_metadata_preserved=True, fixture_unchanged=True,
                  declared_duration_s=.02, actual_last_sample_time_s=last,
                  public_sample_reads_discover_actual_bound=True)
    job = c.ask('app.close', {'discard_changes': True}, True)['shutdown']
    job = c.wait_for_shutdown_outcome(job)
    assert job['status'] == 'completed' and job['workers_joined'] and job['exit_ready']
    report['shutdown'] = job
finally:
    c.close()
assert c.process.returncode == 0
report.update(passed=True, agent_exit_code=0)
(stage/'protocol-report.json').write_text(json.dumps(report, indent=2)+'\n')
print(json.dumps({k: report[k] for k in ['passed', 'refusal_cases', 'covered_cases', 'agent_exit_code']}))
