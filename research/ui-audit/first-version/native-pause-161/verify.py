"""Recheck retained evidence; does not launch or mutate the native app."""
import gzip
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import sys
import tarfile

ROOT = Path('/Users/dshills/Development/projects/ELab')
BASE = Path(__file__).parent
sys.path.insert(0, str(ROOT/'scripts'))
from wiring_replay import validate_release


def read(name):
    path = BASE/name
    return json.loads(gzip.decompress(path.read_bytes()) if path.suffix == '.gz' else path.read_text())


def main():
    checkpoint = read('checkpoint.json')
    assert checkpoint['passed'] and not checkpoint['historical_pause_cause_resolved']
    for name, expected in checkpoint['evidence_sha256'].items():
        assert hashlib.sha256((BASE/name).read_bytes()).hexdigest() == expected, name
    assert validate_release(ROOT, ROOT/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json') == checkpoint['release']
    assert hashlib.sha256((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes()).hexdigest() == read('preservation.json')['canonical_native_sha256']
    report = read('native-session.json.gz')
    assert report['passed'] and report['normal_closure'] and report['final_host_exit_code'] == 0
    assert 'error' not in report and 'completion_failure' not in report
    spec = importlib.util.spec_from_file_location('parity', ROOT/'scripts/first-version-parity.py')
    parity = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(parity)
    spec = importlib.util.spec_from_file_location('references', BASE/'references.py')
    references = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(references)
    independent = read('independent-references.json')
    assert references.prepare() == independent
    events = {event['name']: event['state'] for event in report['events']}
    for name, expected in report['reference'].items():
        state = events[name]
        for key in ['components','wires','nets','selected','drawing','paused','transient_settings','undo_name','redo_name']:
            assert state.get(key) == expected.get(key), (name, key)
        for key in ['npn','mixed','transient','scope','result','trace_focus']:
            parity.compare(state.get(key), expected.get(key), name+'.'+key)
        if name.startswith('npn-'):
            references.npn_check(state, independent)
        else:
            references.mosfet_check(state, 'nmos')
            if len(state['scope']['selected']) == 6:
                references.mosfet_trace_check(state, 'nmos')
    assert len(report['cycles']) == 6
    owner = events['nmos-gate-probe']['run_id']
    electrical = events['nmos-gate-probe']['electrical_revision']
    for cycle in report['cycles']:
        assert cycle['passed'] and cycle['command_revision_after'] == cycle['command_revision_before']+2
        name = 'cycle-'+str(cycle['number'])
        before, paused = events[name+'-before'], events[name+'-paused']
        assert before['paused'] and paused['paused']
        assert paused['transient']['sample']['time_s'] > before['transient']['sample']['time_s']
        assert before['run_id'] == paused['run_id'] == owner
        assert before['electrical_revision'] == paused['electrical_revision'] == electrical
        for action, expected in [('resume','simulation.pause'), ('pause','simulation.resume')]:
            completion = read(name+'-'+action+'-completion.json.gz')
            assert completion['passed'] and completion['clicks'] == 1
            assert completion['expected_control'] == expected and len(completion['observations']) == 1
    for kind in ['npn','nmos']:
        native, agent = read('Native '+kind+'.elab'), read('Agent '+kind+'.elab')
        assert {k:v for k,v in native.items() if k != 'view'} == {k:v for k,v in agent.items() if k != 'view'}
    reopening = read('fresh-agent-reopen.json.gz')
    assert reopening['passed'] and reopening['fresh_agent_processes'] == 4
    for kind, pair in reopening['states'].items():
        expected = report['reference']['npn-restart' if kind == 'npn' else 'nmos-gate-probe']
        for state in pair.values():
            assert state['paused'] and not state['document']['dirty']
            for key in ['npn','mixed','transient','scope','result']:
                parity.compare(state.get(key), expected.get(key), kind+'.reopen.'+key)
    with tarfile.open(BASE/'ui-captures.tar.gz') as archive:
        index = read('ui-capture-index.json')['captures']
        for item in index:
            assert hashlib.sha256(archive.extractfile(item['archive_name']).read()).hexdigest() == item['sha256']
        assert not any('endpoint' in item.name for item in archive.getmembers())
    assert read('cleanup.json')['process_absent'] and read('cleanup.json')['fresh_agent_reopen_passed']
    assert read('source-tests.json')['passed'] and '\nok ' in (BASE/'focused-source-tests.txt').read_text()
    for target in re.findall(r'!?\[[^\]]*\]\(([^)]+)\)', (BASE/'README.md').read_text()):
        assert (BASE/target).is_file(), target
    print(json.dumps(dict(passed=True, native_cycles=6, single_click_cycle_actions=12,
        full_reference_stages=len(report['reference']), captures=len(index),
        evidence_files=len(checkpoint['evidence_sha256']), historical_cause_resolved=False)))


if __name__ == '__main__':
    main()
