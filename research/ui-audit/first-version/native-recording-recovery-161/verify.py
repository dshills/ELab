"""Recheck the retained synthetic recovery evidence without launching the app."""
import gzip
import hashlib
import importlib.util
import json
import math
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


def law(page):
    assert page['total'] == len(page['rows']) == 21
    assert [probe['id'] for probe in page['probes']] == ['v:C1.1', 'v:V1.positive']
    for index, row in enumerate(page['rows']):
        assert abs(row['time_s'] - index*.001) < 1e-15
        assert len(row['values']) == 2
        for actual, expected in zip(row['values'], [5*(1-math.exp(-row['time_s']/.01)), 5]):
            assert abs(actual-expected) <= max(.001, .001*abs(expected))


def main():
    checkpoint = read('checkpoint.json')
    for name, digest in checkpoint['evidence_sha256'].items():
        assert hashlib.sha256((BASE/name).read_bytes()).hexdigest() == digest, name
    assert validate_release(ROOT, ROOT/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json') == checkpoint['release']
    report = read('session.json.gz')
    assert report['passed'] and 'error' not in report and read('preflight.json.gz')['passed']
    events = {event['name']: event['state'] for event in report['events']}
    protocol = read('protocol.json')
    assert set(protocol['barriers']).issubset(events)
    assert set(protocol['signals']) == {'allow-crash','allow-relaunch','finish'}
    spec = importlib.util.spec_from_file_location('parity', ROOT/'scripts/first-version-parity.py')
    parity = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(parity)
    for name, expected in report['reference'].items():
        actual = events[name]
        for key in ['components','wires','nets','selected','drawing','paused','transient_settings','undo_name','redo_name']:
            assert actual.get(key) == expected.get(key), (name,key)
        for key in ['result','npn','mixed','transient','scope','trace_focus']:
            parity.compare(actual.get(key), expected.get(key), name+'.'+key)
        if name in ['recorded-view','recorded-next','recorded-end']:
            parity.compare(actual['recording'], expected['recording'], name+'.recording')
    before = events['baseline-ready']
    for name in ['private-duration','duration-committed','capture-ready']:
        for key in ['components','wires','nets','selected','paused','run_id','electrical_revision','transient','scope','result','mixed','npn']:
            assert events[name].get(key) == before.get(key), (name,key)
    dirty = events['capture-ready']['document']
    assert dirty['recordings_dirty'] and not dirty['schematic_dirty'] and not dirty['preferences_dirty']
    assert report['original_capture']['operation']['actor'] == 'user'
    page = report['original_page']
    law(page)
    metadata = report['original_capture']['metadata']
    exclusions = {'operation_id','created_at','input_revision','electrical_revision','document_generation'}
    assert set(report['capture_provenance_fields_excluded_only_for_independent_first_capture']) == exclusions
    parity.compare({k:v for k,v in metadata.items() if k not in exclusions},
                   {k:v for k,v in report['reference_capture']['metadata'].items() if k not in exclusions})
    raw = (BASE/'Original recorded snapshot.elab-recovery').read_bytes()
    assert hashlib.sha256(raw).hexdigest() == report['choice']['sha256'] == report['capsule_sha256']
    capsule = json.loads(raw)
    # Verify the original embedded Go JSON bytes, not Python re-encoding.
    text = raw.decode()
    start = re.search(r'"document"\s*:\s*', text).end()
    document, end = json.JSONDecoder().raw_decode(text,start)
    assert hashlib.sha256(text[start:end].encode()).hexdigest() == capsule['document_sha256']
    entry = document['recordings']['entries'][0]
    assert entry['data']['metadata'] == metadata and entry['data']['rows'] == page['rows']
    assert read('crashed-owner-lease.json') == {'session':capsule['session'], 'pid':report['hosts'][0]['pid']}
    assert [host['exit_code'] for host in report['hosts']] == [-9,0]
    assert report['hosts'][0]['process_absent'] and report['hosts'][1]['normal_closure']
    assert report['inactive_choice']['owner_status'] == 'inactive' and not report['inactive_choice']['active_session']
    for name, offset in [('recovery-page-first',0),('recovery-page-last',20)]:
        check = report['checks'][name]
        assert check['work']['status'] == 'completed'
        assert check['page']['rows'] == page['rows'][offset:offset+20]
        assert check['page']['probes'] == page['probes'] and check['page']['total'] == 21
        for key in ['components','wires','document','transient','scope','paused']:
            assert events[name].get(key) == events['recovery-selected'].get(key), (name,key)
    restored = events['recovery-restored']
    assert restored['document']['dirty'] and not restored['document']['path']
    assert restored['document']['recovered_from'] == capsule['source_path']
    assert restored['components'] == document['graph']['components'] and restored['wires'] == document['graph']['wires']
    assert restored['paused'] and restored['transient']['sample']['time_s'] == 0
    for name in ['recorded-view','recorded-next','private-seek','recorded-end','view-live']:
        for key in ['components','wires','nets','selected','paused','run_id','electrical_revision','transient','scope','result','mixed','npn']:
            assert events[name].get(key) == restored.get(key), (name,key)
    assert events['recorded-next']['recording']['sample_index'] == 1
    assert events['private-seek']['recording'] == events['recorded-next']['recording']
    assert events['recorded-end']['recording']['sample_index'] == 20
    assert events['recorded-end']['recording']['time_s'] == .02
    assert not events['view-live']['recording']['viewing']
    assert hashlib.sha256((BASE/'Manual RC.elab').read_bytes()).hexdigest() == report['manual_sha256']
    for native_name, agent_name in [('Manual RC.elab','Agent baseline.elab'),('Recovered RC.elab','Agent recovered RC.elab')]:
        native, agent = read(native_name), read(agent_name)
        assert {k:v for k,v in native.items() if k != 'view'} == {k:v for k,v in agent.items() if k != 'view'}
    assert read('Recovered RC.elab')['recordings']['entries'] == [entry]
    fresh = read('fresh-agent-reopen.json.gz')
    assert fresh['passed'] and fresh['fresh_reopen_processes'] == 4
    for label, state in fresh['states'].items():
        assert state['paused'] and not state['document']['dirty']
        for key in ['components','wires','nets','transient_settings']:
            assert state[key] == events['restarted'][key], (label,key)
        for key in ['result','transient','npn','mixed','scope']:
            parity.compare(state.get(key),events['restarted'].get(key),label+'.'+key)
        has_recording = 'recovered' in label
        assert state['document']['recording_count'] == int(has_recording)
        if has_recording:
            assert state['recording']['sample_index'] == 0 and not state['recording']['viewing']
    assert len(fresh['independent_rc']) == 2 and all(item['passed'] for item in fresh['independent_rc'])
    cleanup = read('cleanup.json')
    assert cleanup['passed'] and cleanup['native_pids_absent'] == [h['pid'] for h in report['hosts']]
    assert cleanup['native_exit_codes'] == [-9,0] and cleanup['fresh_agent_reopens'] == 4
    assert cleanup['stopped_bundle_restored'] == read('identity.json')['harness_before']
    assert all(cleanup[k] for k in ['manual_save_preserved','original_capsule_preserved','embedded_recording_preserved_exactly'])
    with tarfile.open(BASE/'ui-captures.tar.gz') as archive:
        index = read('ui-capture-index.json')['captures']
        assert len(index) == len(archive.getmembers())
        for item in index:
            assert hashlib.sha256(archive.extractfile(item['archive_name']).read()).hexdigest() == item['sha256']
        assert not any('endpoint' in member.name for member in archive.getmembers())
    assert hashlib.sha256((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes()).hexdigest() == read('preservation.json')['canonical_native_sha256']
    for target in re.findall(r'!?\[[^\]]*\]\(([^)]+)\)', (BASE/'README.md').read_text()):
        assert (BASE/target.strip('<>')).exists(),target
    print(json.dumps(dict(passed=True,evidence_files=len(checkpoint['evidence_sha256']),ui_observations=len(index),recorded_rows=21,probes=2,fresh_agent_reopens=4,original_manual_and_capture_preserved=True)))


if __name__ == '__main__':
    main()
