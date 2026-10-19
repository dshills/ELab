import gzip, hashlib, importlib.util, json, shutil, subprocess, sys
from pathlib import Path

root = Path('/Users/dshills/Development/projects/ELab')
base = Path(__file__).parent
out = base/'observations'
first = root/'build/native-transistors-161/observations'
sys.path.insert(0,str(root/'scripts'))
from wiring_replay import validate_release
release = validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
report = json.loads((out/'report.json').read_text())
assert report['passed'] and report['normal_closure'] and report['final_host_exit_code'] == 0
ps = subprocess.run(['/bin/ps','-p',str(report['pid']),'-o','pid=,comm='],capture_output=True,text=True)
assert ps.returncode == 1 and not ps.stdout.strip()
assert not [line for line in subprocess.check_output(['/bin/ps','-axo','pid=,comm='],text=True).splitlines() if 'ELab-value-recovery' in line]
bundle = root/'build/native-files-156-ui/ELab-value-recovery.app'
identity = json.loads((base/'identity.json').read_text())
restored = {}
for relative,digest in identity['harness_before'].items():
    backup = base/'harness-before'/Path(relative).name
    assert hashlib.sha256(backup.read_bytes()).hexdigest() == digest
    shutil.copyfile(backup,bundle/relative)
    restored[relative] = hashlib.sha256((bundle/relative).read_bytes()).hexdigest()
    assert restored[relative] == digest
cleanup = dict(pid=report['pid'],exit_code=0,process_absent=True,stopped_bundle_restored=restored,
               release_reverified=release)
(out/'cleanup.json').write_text(json.dumps(cleanup,indent=2)+'\n')

spec = importlib.util.spec_from_file_location('p',root/'scripts/first-version-parity.py')
p = importlib.util.module_from_spec(spec)
spec.loader.exec_module(p)
npn_report = json.loads((first/'report.json').read_text())
assert not npn_report['passed'] and npn_report['checks']['npn-save']['passed']
states = {}
for kind in ['npn','nmos','pmos']:
    directory = first if kind == 'npn' else out
    expected = npn_report['reference']['npn-restart'] if kind == 'npn' else report['reference'][kind+'-gate-probe']
    states[kind] = {}
    for label in ['Native','Agent']:
        client = p.helper.Client(command_preconditions=True,agent_binary=Path(release['agent']))
        try:
            blank = client.ask('simulation.wait')
            assert not blank['components'] and not blank['wires']
            client.ask('project.load',dict(path=str(directory/(label+' '+kind+'.elab'))),True)
            state = client.ask('simulation.wait')
            assert state['paused'] and not state['document']['dirty']
            assert state['transient']['sample']['time_s'] == 0 and state['transient']['input_mode'] == 'auto'
            assert state['scope']['active_cursor'] == 'a' and 'a' not in state['scope'] and 'b' not in state['scope']
            for key in ['components','wires','nets','transient_settings']:
                assert state[key] == expected[key], (kind,key)
            for key in ['npn','mixed','transient','scope','result']:
                p.compare(state.get(key),expected.get(key),kind+'.reopen.'+key)
            states[kind][label] = state
        finally:
            client.close()
    for key in ['npn','mixed','transient','scope','result']:
        p.compare(states[kind]['Native'].get(key),states[kind]['Agent'].get(key),kind+'.saved-parity.'+key)
with (out/'fresh-agent-reopen.json.gz').open('wb') as f:
    with gzip.GzipFile(filename='',mode='wb',fileobj=f,mtime=0) as z:
        z.write((json.dumps(dict(passed=True,fresh_agent_processes=6,
            scope='Fresh packaged agents reopen three native/reference saved circuits at clean paused Auto startup, preserving graphs/settings/probes/models. Native file panels and live trace persistence are not claimed.',
            states=states),indent=2)+'\n').encode())
cleanup['fresh_agent_reopen_passed'] = True
cleanup['release_reverified'] = validate_release(root,root/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
(out/'cleanup.json').write_text(json.dumps(cleanup,indent=2)+'\n')
print('Six fresh saved-file reopens pass; original retry host exited normally and is absent; stopped custom bundle restored; sealed /161 reverified.')
