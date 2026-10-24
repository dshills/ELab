"""Check affected shared commands and captured RC values in the clean package."""
import hashlib, importlib.util, json, math, time
from pathlib import Path

root = Path('/Users/dshills/Development/projects/ELab')
stage = Path(__file__).resolve().parent
build = json.loads((stage/'build-result-signed.json').read_text())
binary = Path(build['review'])/'elab-agent'
assert hashlib.sha256(binary.read_bytes()).hexdigest() == build['agent_sha256']
spec = importlib.util.spec_from_file_location('client', root/'scripts/persistence-demo.py')
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
c = helper.Client(command_preconditions=True, agent_binary=binary)
report = dict(source_commit=build['source_commit'], agent_sha256=build['agent_sha256'],
              transport='headless-stdio', fresh_refusal_cases=0, stale_valid_cases=0,
              fresh_corrections=0, scalar_readings_checked=0,
              max_voltage_error_v=0., max_current_error_a=0.)

def same_state_refusal(command, params, code, revision=None):
    before = c.ask('project.inspect')
    if revision is None:
        c.ask(command, params, True, code)
    else:
        c.sequence += 1
        request = dict(version=1, id=f'old-{c.sequence}', key=f'old-{c.sequence}',
                       command=command, params=params, expected_command_revision=revision)
        c.process.stdin.write(json.dumps(request)+'\n')
        c.process.stdin.flush()
        response = json.loads(c.process.stdout.readline())
        assert response['error']['code'] == code, response
    assert before == c.ask('project.inspect'), 'refusal changed accepted state'

try:
    c.ask('example.rc', {'discard_changes': True}, True)
    c.ask('simulation.pause', {}, True)
    state = c.ask('project.inspect')
    verification = dict(state['verification_settings'], duration_s='20 ms', sample_step_s='500 us', input_mode='charge')
    for field in ['duration_s', 'sample_step_s']:
        same_state_refusal('verification.settings.set', dict(verification, **{field:'oops'}), 'invalid-params')
        report['fresh_refusal_cases'] += 1
    revision = c.command_revision
    c.ask('editor.value.set', {'value':'new agent draft'}, True)
    same_state_refusal('verification.settings.set', verification, 'conflict', revision)
    report['stale_valid_cases'] += 1
    state = c.ask('verification.settings.set', verification, True)
    assert state['verification_settings']['duration_s'] == .02 and state['verification_settings']['sample_step_s'] == .0005
    assert state['drawing']['value'] == 'new agent draft'
    report['fresh_corrections'] += 1

    sweep = dict(state['dc_sweep']['settings'], start='1 V', stop='4 V', points=7)
    for field in ['start', 'stop', 'points']:
        same_state_refusal('sweep.settings.set', dict(sweep, **{field:'oops'}), 'invalid-params')
        report['fresh_refusal_cases'] += 1
    revision = c.command_revision
    c.ask('editor.value.set', {'value':'newer agent draft'}, True)
    same_state_refusal('sweep.settings.set', sweep, 'conflict', revision)
    report['stale_valid_cases'] += 1
    state = c.ask('sweep.settings.set', sweep, True)
    assert state['dc_sweep']['settings'] == sweep and state['drawing']['value'] == 'newer agent draft'
    report['fresh_corrections'] += 1

    identifier = c.ask('simulation.verify', {'probe_ids':['v:V1.positive','v:C1','i:C1']}, True)['last_operation']
    deadline = time.monotonic()+5
    while True:
        operation = c.ask('operation.inspect', {'id':identifier})
        status = operation['operation']['status']
        if status == 'completed':
            break
        assert status not in ['failed','canceled'] and time.monotonic() < deadline, operation
        time.sleep(.01)
    metadata = operation['metadata']
    page = c.ask('operation.samples', {'id':identifier,'offset':0,'limit':metadata['sample_count']})
    assert len(page['rows']) == 41
    for row in page['rows']:
        at = row['time_s']
        for probe, value in zip(page['probes'], row['values'], strict=True):
            if probe['id'] == 'v:V1.positive':
                expected, unit = 5., 'V'
            elif probe['id'] == 'v:C1':
                expected, unit = 5*(1-math.exp(-at/.01)), 'V'
            else:
                assert probe['id'] == 'i:C1'
                expected, unit = .0005*math.exp(-at/.01), 'A'
            assert probe['unit'] == unit
            error = abs(value-expected)
            key = 'max_voltage_error_v' if unit == 'V' else 'max_current_error_a'
            report[key] = max(report[key], error)
            assert error < (1e-10 if unit == 'V' else 1e-14)
            report['scalar_readings_checked'] += 1
    timing = dict(id=identifier,probe_id='v:C1',level='3 V',second_level='4 V',direction='both',start_s='1 ms',end_s='18 ms')
    for field in ['level','second_level','start_s','end_s']:
        same_state_refusal('recording.timing.set', dict(timing, **{field:'oops'}), 'invalid-params')
        report['fresh_refusal_cases'] += 1
    revision = c.command_revision
    c.ask('editor.value.set', {'value':'final agent draft'}, True)
    same_state_refusal('recording.timing.set', timing, 'conflict', revision)
    report['stale_valid_cases'] += 1
    state = c.ask('recording.timing.set', timing, True)
    measured = state['recording']['timing']
    assert measured['settings']['start_s'] == .001 and measured['settings']['end_s'] == .018
    assert state['drawing']['value'] == 'final agent draft' and measured['total_crossings'] == 1
    crossing = measured['crossings'][0]
    exact = -.01*math.log(1-3/5)
    assert crossing['before']['time_s'] <= exact <= crossing['after']['time_s']
    assert metadata == c.ask('operation.inspect', {'id':identifier})['metadata']
    report.update(fresh_corrections=report['fresh_corrections']+1,
                  captured_metadata_preserved=True, crossing_brackets_independent_rc_reference=True,
                  independent_reference='5 V / 10 kohm / 1 uF, zero startup: Vc=5*(1-exp(-t/.01)); Ic=.0005*exp(-t/.01)')
    job = c.ask('app.close', {'discard_changes':True}, True)['shutdown']
    job = c.wait_for_shutdown_outcome(job)
    assert job['status'] == 'completed' and job['workers_joined'] and job['exit_ready']
    report['shutdown'] = job
finally:
    c.close()
assert c.process.returncode == 0
report.update(passed=True,agent_exit_code=0)
(stage/'protocol-report.json').write_text(json.dumps(report,indent=2)+'\n')
print(json.dumps({k:report[k] for k in ['passed','fresh_refusal_cases','stale_valid_cases','fresh_corrections','scalar_readings_checked','agent_exit_code']}))
