import datetime, hashlib, importlib.util, json, subprocess, sys, time
from pathlib import Path
from references import npn_check, mosfet_check, mosfet_trace_check

root = Path('/Users/dshills/Development/projects/ELab')
base = Path(__file__).parent
out = base/'observations'
sys.path.insert(0, str(root/'scripts'))
from wiring_replay import validate_release
release = validate_release(root, root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
identity = json.loads((base/'identity.json').read_text())
binary = root/'build/native-files-156-ui/ELab-value-recovery.app/Contents/MacOS/ELab-value-recovery'
assert hashlib.sha256(binary.read_bytes()).hexdigest() == identity['native_sha256']
endpoint = out/'endpoint.json'
assert not endpoint.exists()
spec = importlib.util.spec_from_file_location('p', root/'scripts/first-version-parity.py')
p = importlib.util.module_from_spec(spec)
spec.loader.exec_module(p)
independent = json.loads((base/'independent-references.json').read_text())
report = dict(passed=False, release=release, identity=identity, events=[], reference={}, checks={},
    scope='Actual native Examples loads from blank, pulse-NPN and N/P MOSFET playback, paused Low/High/Auto plus Step, six signed MOSFET probes, independent references and saved reopening; public service only observes native state until explicit saves/close.',
    started_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
host = client = reference = None

def save():
    pending = out/'report.pending'
    pending.write_text(json.dumps(report, indent=2)+'\n')
    pending.replace(out/'report.json')

def acknowledge(name):
    (out/(name+'-verified.json')).write_text('{}\n')

def wait(name):
    deadline = time.monotonic()+900
    while not (out/name).exists():
        assert host.poll() is None, 'host exited before '+name
        assert time.monotonic() < deadline, 'UI lease expired '+name
        time.sleep(.05)
    state = client.ask('simulation.wait')
    report['events'].append(dict(name=name, utc=datetime.datetime.now(datetime.timezone.utc).isoformat(), state=state))
    save()
    return state

def equivalent(a, b):
    for key in ['components','wires','nets','selected','drawing','paused','transient_settings','undo_name','redo_name']:
        assert a.get(key) == b.get(key), key
    for key in ['npn','mixed','transient','scope','result','trace_focus']:
        p.compare(a.get(key), b.get(key), key)

def check(name, kind):
    expected = reference.ask('simulation.wait')
    report['reference'][name] = expected
    state = wait(name)
    equivalent(state, expected)
    report['checks'][name] = npn_check(state, independent) if kind == 'npn' else mosfet_check(state, kind)
    if kind != 'npn' and len(state['scope']['selected']) == 6:
        report['checks'][name]['trace'] = mosfet_trace_check(state, kind)
    save()
    acknowledge(name)
    return state

def advance(target):
    at = reference.ask('simulation.wait')['transient']['sample']['time_s']
    while target-at > 1e-10:
        reference.ask('simulation.step.start', dict(duration_s=min(1., target-at)), True)
        at = reference.ask('simulation.wait')['transient']['sample']['time_s']

def reach(kind, target):
    name = kind+'-running'
    state = wait(name)
    assert not state['paused']
    deadline = time.monotonic()+45
    while state['transient']['sample']['time_s'] < target:
        assert host.poll() is None and time.monotonic() < deadline
        state = client.ask('project.inspect')
        assert not state['paused'] and not state['transient']['stale'] and state['status'] != 'error'
        time.sleep(.05)
    report['checks'][kind+'-live-target'] = dict(time_s=state['transient']['sample']['time_s'], target_s=target)
    save()
    acknowledge(name)

def save_copies(kind):
    wait(kind+'-allow-save')
    for service, label in [(client,'Native'),(reference,'Agent')]:
        service.ask('project.save', dict(path=str(out/(label+' '+kind+'.elab'))), True)
    a = json.loads((out/('Native '+kind+'.elab')).read_text())
    b = json.loads((out/('Agent '+kind+'.elab')).read_text())
    assert {k:v for k,v in a.items() if k != 'view'} == {k:v for k,v in b.items() if k != 'view'}
    report['checks'][kind+'-save'] = dict(passed=True, all_fields_except_view_match=True)
    save()
    acknowledge(kind+'-allow-save')

with (out/'native-host.txt').open('x') as log:
    try:
        reference = p.helper.Client(command_preconditions=True, agent_binary=Path(release['agent']))
        host = subprocess.Popen([str(binary),'--agent-endpoint',str(endpoint),'--no-recovery','--no-appearance'],stdout=log,stderr=-2)
        report['pid'] = host.pid
        save()
        deadline = time.monotonic()+20
        while not endpoint.exists():
            assert host.poll() is None and time.monotonic() < deadline
            time.sleep(.05)
        client = p.helper.Client(str(endpoint),command_preconditions=True,agent_binary=Path(release['agent']))
        blank = client.ask('simulation.wait')
        assert not blank['components'] and not blank['wires']
        report['events'].append(dict(name='blank',state=blank))
        save()
        (out/'ready.json').write_text('{}\n')
        for kind, target in [('npn', .155), ('nmos', .555), ('pmos', .555)]:
            reference.ask('example.npn-pulse' if kind == 'npn' else 'example.load',{} if kind == 'npn' else dict(id=kind), True)
            reference.ask('simulation.pause',{},True)
            reference.ask('simulation.restart',{},True)
            initial = reference.ask('simulation.wait')
            loaded = wait(kind+'-loaded-paused')
            assert loaded['paused'] and not loaded['transient']['stale']
            for key in ['components','wires','nets','transient_settings']:
                assert loaded[key] == initial[key], key
            acknowledge(kind+'-loaded-paused')
            check(kind+'-restart',kind)
            if kind != 'npn':
                ids = initial['scope']['selected']+['i:M1.gate']
                reference.ask('scope.probes.set',dict(ids=ids),True)
                check(kind+'-gate-probe',kind)
            reach(kind,target)
            paused = wait(kind+'-paused')
            assert paused['paused']
            advance(paused['transient']['sample']['time_s'])
            expected = reference.ask('simulation.wait')
            report['reference'][kind+'-paused'] = expected
            equivalent(paused,expected)
            report['checks'][kind+'-paused'] = npn_check(paused,independent) if kind == 'npn' else mosfet_trace_check(paused,kind)
            source_points = next(t['points'] for t in paused['scope']['traces'] if t['probe']['id']=='v:VIN.positive')
            edge = .05 if kind == 'npn' else .2
            edge_values = [point['value'] for point in source_points if abs(point['time_s']-edge)<1e-12]
            assert any(abs(value)<1e-9 for value in edge_values) and any(abs(value-5)<1e-9 for value in edge_values)
            report['checks'][kind+'-paused']['exact_first_edge_both_sides'] = True
            save()
            acknowledge(kind+'-paused')
            for mode in ['low','high','auto']:
                before = reference.ask('simulation.wait')
                reference.ask('simulation.input.set',dict(mode=mode),True)
                queued = check(kind+'-queued-'+mode,kind)
                assert queued['transient']['pending_input_mode'] == mode
                assert queued['transient']['sample'] == before['transient']['sample']
                reference.ask('simulation.step.start',dict(duration_s=queued['transient_settings']['step_s']),True)
                stepped = check(kind+'-step-'+mode,kind)
                assert stepped['transient'].get('pending_input_mode','') == ''
            save_copies(kind)
        report['passed'] = True
        save()
        (out/'all-verified.json').write_text('{}\n')
        wait('finish')
        client.ask('app.close',{},True)
        client.close()
        client = None
        report['host_exit_code'] = host.wait(timeout=15)
        report['normal_closure'] = report['host_exit_code'] == 0
    except BaseException as error:
        report['error'] = type(error).__name__+': '+str(error)
        raise
    finally:
        if client:
            client.close()
        if reference:
            reference.close()
        if host and host.poll() is None:
            host.terminate()
            try:
                host.wait(timeout=10)
            except subprocess.TimeoutExpired:
                host.kill()
                host.wait(timeout=5)
        if host:
            report['final_host_exit_code'] = host.returncode
        report['ended_utc'] = datetime.datetime.now(datetime.timezone.utc).isoformat()
        save()
