import gzip, importlib.util, json, sys
from pathlib import Path
from references import npn_check, mosfet_check, mosfet_trace_check

root = Path('/Users/dshills/Development/projects/ELab')
base = Path(__file__).parent
sys.path.insert(0, str(root/'scripts'))
from wiring_replay import validate_release
release = validate_release(root, root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
spec = importlib.util.spec_from_file_location('p', root/'scripts/first-version-parity.py')
p = importlib.util.module_from_spec(spec)
spec.loader.exec_module(p)
reference = json.loads((base/'independent-references.json').read_text())
c = p.helper.Client(command_preconditions=True, agent_binary=Path(release['agent']))
report = dict(passed=False, release=release, cases={})
try:
    for kind, duration in [('npn', .155), ('nmos', .555), ('pmos', .555)]:
        c.ask('example.npn-pulse' if kind == 'npn' else 'example.load', {} if kind == 'npn' else dict(id=kind), True)
        c.ask('simulation.pause', {}, True)
        c.ask('simulation.restart', {}, True)
        initial = c.ask('simulation.wait')
        if kind != 'npn':
            for probe in ['v:M1', 'v:M1.gate-source', 'i:M1.drain']:
                ids = c.ask('simulation.wait')['scope']['selected'][:]
                if probe not in ids:
                    if len(ids) == 6:
                        ids[-1] = probe
                    else:
                        ids.append(probe)
                c.ask('scope.probes.set', dict(ids=ids), True)
        if kind != 'npn':
            ids = c.ask('simulation.wait')['scope']['selected']+['i:M1.gate']
            c.ask('scope.probes.set', dict(ids=ids), True)
        startup = c.ask('simulation.wait')
        c.ask('simulation.step.start', dict(duration_s=duration), True)
        live = c.ask('simulation.wait')
        observations = [npn_check(live, reference) if kind == 'npn' else mosfet_check(live, kind)]
        if kind != 'npn':
            observations.append(mosfet_trace_check(live, kind))
        for mode in ['low', 'high', 'auto']:
            before = c.ask('simulation.wait')
            c.ask('simulation.input.set', dict(mode=mode), True)
            queued = c.ask('simulation.wait')
            assert queued['transient']['sample'] == before['transient']['sample']
            assert queued['transient']['pending_input_mode'] == mode
            c.ask('simulation.step.start', dict(duration_s=queued['transient_settings']['step_s']), True)
            stepped = c.ask('simulation.wait')
            observations.append(npn_check(stepped, reference) if kind == 'npn' else mosfet_check(stepped, kind))
            if kind != 'npn':
                observations.append(mosfet_trace_check(stepped, kind))
        path = base/('Preflight '+kind+'.elab')
        c.ask('project.save', dict(path=str(path), overwrite=True), True)
        fresh = p.helper.Client(command_preconditions=True, agent_binary=Path(release['agent']))
        try:
            fresh.ask('project.load', dict(path=str(path)), True)
            reopened = fresh.ask('simulation.wait')
            assert reopened['paused'] and not reopened['document']['dirty']
            for key in ['components','wires','nets','transient_settings']:
                assert reopened[key] == startup[key], key
            for key in ['transient','npn','mixed','scope','result']:
                p.compare(reopened.get(key), startup.get(key), kind+'.reopen.'+key)
        finally:
            fresh.close()
        report['cases'][kind] = dict(passed=True, initial=initial, startup=startup, final=stepped,
            observations=observations, saved_reopen=reopened)
    report['passed'] = True
finally:
    c.close()
    with (base/'preflight.json.gz').open('wb') as f:
        with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:
            z.write((json.dumps(report,indent=2)+'\n').encode())
print('Three-case reference sequence, independent phase checks and fresh saved-file reopening pass')
