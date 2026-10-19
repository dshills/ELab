"""Validate retained recovery evidence without launching or mutating the app."""
import gzip
import hashlib
import importlib.util
import json
from decimal import Decimal
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
    for name, digest in checkpoint['evidence_sha256'].items():
        assert hashlib.sha256((BASE/name).read_bytes()).hexdigest() == digest, name
    assert validate_release(ROOT, ROOT/'research/ui-audit/first-version/rc-input-frame-review-checkpoint.json') == checkpoint['release']
    report, reopen = read('original-session.json.gz'), read('reopen/session.json.gz')
    assert not report['passed'] and 'TimeoutExpired' in report['error']
    assert report['saved_fields_except_view_match'] and reopen['passed']
    assert reopen['native_exit_code'] == reopen['final_native_exit_code'] == 0
    assert read('corrected-close.json')['process_absent'] and read('corrected-close.json')['exit_code'] is None
    events = {event['name']: event['state'] for event in report['events']}
    spec = importlib.util.spec_from_file_location('parity', ROOT/'scripts/first-version-parity.py')
    parity = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(parity)
    for name, expected in report['reference'].items():
        actual = events[name]
        for key in ['components','wires','nets','selected','drawing','paused','transient_settings','undo_name','redo_name']:
            assert actual.get(key) == expected.get(key), (name, key)
        for key in ['result','npn','mixed','transient','scope','trace_focus']:
            parity.compare(actual.get(key), expected.get(key), name+'.'+key)
    for key in ['components','wires','nets','selected','result','scope','document','paused','run_id','electrical_revision','undo_entry','redo_entry']:
        assert events['private-value'].get(key) == events['selected-R1'].get(key), key
    manual = BASE/'first/Manual LED.elab'
    assert hashlib.sha256(manual.read_bytes()).hexdigest() == report['manual_save_sha256']
    capsule_raw = (BASE/'Original automatic snapshot.elab-recovery').read_bytes()
    assert hashlib.sha256(capsule_raw).hexdigest() == report['automatic_capsule_sha256']
    capsule = json.loads(capsule_raw)
    # Hash the exact embedded Go JSON bytes; Python re-encoding can differ.
    text = capsule_raw.decode()
    start = re.search(r'"document"\s*:\s*', text).end()
    document, end = json.JSONDecoder().raw_decode(text, start)
    assert document == capsule['document']
    assert hashlib.sha256(text[start:end].encode()).hexdigest() == capsule['document_sha256']
    assert capsule['source_path'] == events['baseline-saved']['document']['path']
    assert document['graph']['components'] == events['edited-fresh']['components']
    assert document['graph']['wires'] == events['edited-fresh']['wires'] and document['paused']
    assert read('crashed-owner-lease.json')['session'] == capsule['session']
    assert read('crashed-owner-lease.json')['pid'] == report['hosts'][0]['pid']
    assert report['hosts'][0]['exit_code'] == -9 and report['hosts'][0]['process_absent']
    assert report['inactive_candidate']['owner_status'] == 'inactive'
    restored = events['recovery-restored']
    assert restored['document']['dirty'] and not restored['document']['path']
    assert restored['document']['recovered_from'] == capsule['source_path']
    assert restored['components'] == document['graph']['components'] and restored['wires'] == document['graph']['wires']
    parity.compare(restored['result'], events['edited-fresh']['result'])
    assert report['inspection_read_work']['status'] == 'completed'
    assert report['inspection_read_work']['input']['id'] == report['automatic_candidate']['id']
    for native_name, agent_name in [('first/Manual LED.elab','Agent baseline.elab'), ('restored/Recovered LED.elab','Agent recovered.elab')]:
        native, agent = read(native_name), read(agent_name)
        assert {k:v for k,v in native.items() if k != 'view'} == {k:v for k,v in agent.items() if k != 'view'}
    reopened = reopen['events'][0]['state']
    assert not reopened['document']['dirty'] and reopened['project_work']['actor'] == 'user'
    for key in ['components','wires','nets','paused','transient_settings']:
        assert reopened[key] == restored[key], key
    for key in ['result','scope','transient','npn','mixed']:
        parity.compare(reopened.get(key), restored.get(key), 'native-reopen.'+key)
    fresh = read('fresh-agent-reopen.json.gz')
    assert fresh['passed'] and fresh['fresh_reopen_processes'] == 4
    assert all(row['passed'] and row['relative_error'] <= .01 for row in fresh['independent_led_checks'])
    generator = ROOT/'scripts/freeze-electrical-reference.py'
    assert hashlib.sha256(generator.read_bytes()).hexdigest() == read('preservation.json')['independent_generator_sha256']
    spec = importlib.util.spec_from_file_location('independent', generator)
    independent = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(independent)
    for label, state in fresh['states'].items():
        expected = events['loaded-paused'] if 'baseline' in label else reopen['reference']
        assert state['paused'] and not state['document']['dirty'] and not state['result']['stale']
        for key in ['components','wires','nets','transient_settings']:
            assert state[key] == expected[key], (label, key)
        for key in ['result','scope','transient','npn','mixed']:
            parity.compare(state.get(key), expected.get(key), label+'.'+key)
        resistance = 1000 if 'baseline' in label else 2200
        reference_current = float(independent.led(Decimal(resistance))['current_a'])
        assert abs(state['result']['current_a']-reference_current) <= .01*reference_current
    cleanup = read('cleanup.json')
    assert len(cleanup['native_pids_absent']) == 3 and cleanup['original_capsule_preserved'] and cleanup['manual_save_preserved']
    assert cleanup['restored_host_exit_code'] is None and cleanup['fresh_native_reopen_exit_code'] == 0
    with tarfile.open(BASE/'ui-captures.tar.gz') as archive:
        index = read('ui-capture-index.json')['captures']
        for item in index:
            assert hashlib.sha256(archive.extractfile(item['archive_name']).read()).hexdigest() == item['sha256']
        assert not any('endpoint' in item.name for item in archive.getmembers())
    assert hashlib.sha256((ROOT/'build/ELab.app/Contents/MacOS/ELab').read_bytes()).hexdigest() == read('preservation.json')['canonical_native_sha256']
    for target in re.findall(r'!?\[[^\]]*\]\(([^)]+)\)', (BASE/'README.md').read_text()):
        assert (BASE/target.strip('<>')).is_file(), target
    print(json.dumps(dict(passed=True, original_controller_passed=False, fresh_native_reopen_exit_code=0,
        fresh_agent_reopens=4, captures=len(index), evidence_files=len(checkpoint['evidence_sha256']),
        manual_save_and_original_capsule_preserved=True)))


if __name__ == '__main__':
    main()
