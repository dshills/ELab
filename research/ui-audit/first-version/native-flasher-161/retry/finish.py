import gzip, hashlib, importlib.util, json, math, shutil, subprocess, sys
from pathlib import Path

root = Path('/Users/dshills/Development/projects/ELab')
base = root / 'build/native-flasher-161-retry'
out = base / 'observations'
sys.path.insert(0, str(root / 'scripts'))
from wiring_replay import validate_release

release = validate_release(root, root / 'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json')
report = json.loads((out / 'report.json').read_text())
assert report['passed'] and report['normal_closure'] and report['final_host_exit_code'] == 0
spec = importlib.util.spec_from_file_location('parity', root / 'scripts/first-version-parity.py')
parity = importlib.util.module_from_spec(spec)
spec.loader.exec_module(parity)
expected = json.loads(json.dumps(report['reference']['restart-20']))
# Cursor selection is session state, absent from the serialized project.
expected['scope']['active_cursor'] = 'a'
states = {}
for label, filename in [('native', 'Native timed flasher.elab'), ('agent', 'Agent timed flasher.elab')]:
    client = parity.helper.Client(command_preconditions=True, agent_binary=Path(release['agent']))
    try:
        blank = client.ask('simulation.wait')
        assert not blank['components'] and not blank['wires']
        client.ask('project.load', dict(path=str(out / filename)), True)
        state = client.ask('simulation.wait')
        for key in ['components', 'wires', 'nets', 'transient_settings']:
            assert state[key] == expected[key], key
        assert state['scope']['selected'] == expected['scope']['selected']
        assert state['paused'] and not state['document']['dirty']
        assert not state['transient']['stale'] and not state['mixed']['stale']
        assert state['transient']['sample']['time_s'] == 0
        assert 'a' not in state['scope'] and 'b' not in state['scope']
        caps = {cap['id']: cap for cap in state['mixed']['capacitors']}
        assert math.isclose(caps['C1']['voltage_v'], .1, abs_tol=1e-12)
        assert caps['C2']['voltage_v'] == 0
        for key in ['transient', 'mixed', 'scope', 'result']:
            parity.compare(state.get(key), expected.get(key), key)
        states[label] = state
    finally:
        client.close()
for key in ['transient', 'mixed', 'scope', 'result']:
    parity.compare(states['native'].get(key), states['agent'].get(key), key)
with (out / 'fresh-agent-reopen.json.gz').open('wb') as f:
    with gzip.GzipFile(filename='', mode='wb', fileobj=f, mtime=0) as z:
        z.write((json.dumps(dict(passed=True, blank_start=True, saved_paused_startup=True,
            scope='Two fresh exact packaged agents reopen saved native/reference graphs, settings, probes and startup; live waveforms/cursors are not persisted; native save/open panels are not exercised here.',
            states=states), indent=2) + '\n').encode())

ps = subprocess.run(['/bin/ps', '-p', str(report['pid']), '-o', 'pid=,comm='], text=True, capture_output=True)
assert ps.returncode == 1 and not ps.stdout.strip(), (ps.returncode, ps.stdout, ps.stderr)
assert not [line for line in subprocess.check_output(['/bin/ps', '-axo', 'pid=,comm='], text=True).splitlines() if 'ELab-value-recovery' in line]
identity = json.loads((base / 'identity.json').read_text())
bundle = root / 'build/native-files-156-ui/ELab-value-recovery.app'
restored = {}
for relative, digest in identity['harness_before'].items():
    backup = base / 'harness-before' / Path(relative).name
    assert hashlib.sha256(backup.read_bytes()).hexdigest() == digest
    shutil.copyfile(backup, bundle / relative)
    restored[relative] = hashlib.sha256((bundle / relative).read_bytes()).hexdigest()
    assert restored[relative] == digest
cleanup = dict(pid=report['pid'], exit_code=0, process_absent=True, ps_return_code=ps.returncode,
               stopped_bundle_restored=restored, fresh_agent_reopen_passed=True,
               release_reverified=validate_release(root, root / 'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json'))
(out / 'cleanup.json').write_text(json.dumps(cleanup, indent=2) + '\n')
print('Both fresh saved-file reopens pass; original host exited normally and is absent; stopped custom bundle restored; sealed /161 reverified.')
