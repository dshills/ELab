"""Read-only replay of the /178 incomplete-document extension corpus."""
from pathlib import Path
import gzip
import hashlib
import json
import sys

if sys.flags.optimize:
    raise SystemExit('Verification requires assertions; optimized Python is refused')

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]


def sha(data):
    return hashlib.sha256(data).hexdigest()


def raw(name):
    return gzip.decompress((HERE / (name + '.gz')).read_bytes())


def read(name):
    return json.loads(raw(name))


def quantity(text):
    number, unit = text.split()
    scale = 1000 if number.endswith('k') else 1
    if scale != 1:
        number = number[:-1]
    return float(number) * scale * {'V': 1, 'ohm': 1, 'F': 1, 'pF': 1e-12,
                          'uF': 1e-6, 'H': 1, 'uH': 1e-6, 'mH': 1e-3,
                          'A': 1, 'mA': 1e-3}[unit]


def check_state(state, document):
    assert state['components'] == document['graph']['components']
    assert state['wires'] == document['graph']['wires'] == []
    assert state['view'] == document['view']
    assert state['paused'] == document['paused'] is True
    assert state['transient_settings'] == document['transient_settings']
    assert state['scope']['selected'] == document['scope_probes']['ids']
    assert state['display']['preferences'] == document['display_preferences']
    assert [p['definition'] for p in state['pinned_readings']] == document['pinned_readings']['items']
    assert len(state['pinned_readings']) == 8
    assert all(p['state'] == 'unavailable' and 'value' not in p for p in state['pinned_readings'])
    assert not state['document']['dirty']


def main():
    checkpoint = json.loads((HERE / 'checkpoint.json').read_text())
    files = {str(p.relative_to(HERE)) for p in HERE.rglob('*') if p.is_file() and p.name != 'checkpoint.json'}
    assert files == set(checkpoint['files'])
    for relative, digest in checkpoint['files'].items():
        path = HERE / relative
        assert not path.is_symlink() and path.resolve().is_relative_to(HERE)
        assert sha(path.read_bytes()) == digest, relative
    report = json.loads((HERE / 'report.json').read_text())
    release_path = ROOT / 'research/ui-audit/first-version/pin-reader-review-checkpoint.json'
    assert sha(release_path.read_bytes()) == report['release_receipt_sha256']
    release = json.loads(release_path.read_text())
    assert report['source_commit'] == release['source_commit'] == '3250604feee11e5edd06ff3ead21fe36e288376a'
    assert len(release['quality']['source_sha256']) == report['source_files_verified'] == 652
    for relative, digest in release['quality']['source_sha256'].items():
        assert sha((ROOT / relative).read_bytes()) == digest, relative
    manifest = Path(release['review_path']) / 'RELEASE_MANIFEST.json'
    assert sha(manifest.read_bytes()) == checkpoint['release_manifest_sha256']
    sys.path.insert(0, str(ROOT / 'scripts'))
    from package_review import members
    actual = members(manifest.parent)
    actual.pop('RELEASE_MANIFEST.json')
    assert actual == json.loads(manifest.read_text())['files']
    assert sha((Path(release['review_path']) / 'elab-agent').read_bytes()) == release['agent_sha256']
    plan, collected = read('plan.json'), read('report.json')
    assert plan['agent_sha256'] == collected['agent_sha256'] == release['agent_sha256']
    assert collected['passed'] and not collected['failure'] and not collected['cleanup_errors']
    assert collected['original_release_agent_unchanged']
    assert plan['release'] == report['release'] == '0.161.0 /178'
    assert len(plan['kinds']) == len(set(plan['kinds'])) == len(plan['ids']) == 18
    assert len(plan['cases']) == len(collected['results']) == 4
    expected_probes = ['v:VPULSE', 'i:VSIN', 'v:U1.input-difference', 'v:MN.gate-source', 'i:MP.drain', 'v:P1.wiper']
    counts = 0
    expected_sessions = set()
    for index, case in enumerate(plan['cases']):
        name = case['name']
        result = read(name + '-result.json')
        assert result['passed'] and result['name'] == name
        original = raw(name + ' Ω schematic.elab')
        document = json.loads(original)
        assert sha(original) == result['sha256'] == collected['results'][index]['sha256']
        assert document == result['document'] and document['version'] == 4
        parts = document['graph']['components']
        assert [p['kind'] for p in parts] == plan['kinds']
        assert [p['id'] for p in parts] == plan['ids']
        inventory = read(name + '-parts.json')
        assert [p['kind'] for p in inventory] == plan['kinds']
        for i, (part, definition) in enumerate(zip(parts, inventory)):
            assert part['terminals'] == definition['terminals'] and part['model'] == definition['model']
            assert part['position'] == dict(x=(i % 6) * 200 - 500, y=(i // 6) * 200 - 200)
            assert part['rotation'] == ((i + index) % 4) * 90
        assert parts[5]['sine'] == case['sine'] and parts[6]['pulse'] == case['pulse']
        for component, text in zip((parts[i] for i in (0, 1, 3, 4, 15)), case['values']):
            assert component['value'] == quantity(text)
        assert parts[5]['value'] == case['sine']['amplitude_v']
        assert parts[6]['value'] == case['pulse']['high_v']
        for part, field, text in ((parts[3], 'initial_voltage_v', case['cap']),
                                  (parts[4], 'initial_current_a', case['coil'])):
            if text == 'default':
                assert field not in part
            else:
                assert part[field] == quantity(text)
        assert parts[14]['label'] == case['label']
        assert parts[15]['wiper_position'] == case['wiper']
        assert parts[17]['switch_closed'] == case['closed']
        assert document['scope_probes']['ids'] == expected_probes
        assert document['pinned_readings']['items'] == [
            dict(id=probe, position=dict(x=i * 100 - 400, y=500))
            for i, probe in enumerate(expected_probes + ['v:L1', 'i:C1'])]
        preferences = document['display_preferences']
        assert preferences == dict(voltage_color=True, voltage_range_v=case['voltage_range'], current_cues=True,
                                   current_motion=case['motion'], reduced_motion=case['reduced'], direction=case['direction'])
        before = result['before']
        assert before['components'] == parts and before['wires'] == []
        assert before['document']['dirty'] and before['view'] == document['view']
        assert len(result['reopened']) == 2
        for attempt, state in enumerate(result['reopened']):
            check_state(state, document)
            assert raw(name + '-resave-' + str(attempt) + '.elab') == original
        for suffix in ('writer', 'reader-0', 'reader-1'):
            session = name + '-' + suffix
            expected_sessions.add(session)
            exchanges = [json.loads(line) for line in raw(session + '-rpc.jsonl').splitlines()]
            revision = 0
            for number, exchange in enumerate(exchanges, 1):
                request, response = exchange['request'], exchange['response']
                assert request['version'] == 1 and request['id'] == response['id'] == session + '-' + str(number)
                assert not response.get('error')
                if 'key' in request:
                    assert request['key'] == request['id'] and request['expected_command_revision'] == revision
                state = response.get('result')
                if isinstance(state, dict) and 'components' in state:
                    revision = state['command_revision']
            final = exchanges[-1]['response']['result']
            work = final['shutdown'] if 'shutdown' in final else final
            assert exchanges[-1]['request']['command'] in ('app.close', 'app.close.inspect')
            assert work['status'] == 'completed' and work['workers_joined'] and work['recovery_finalized'] and work['exit_ready']
            exit_record = next(e for e in collected['exits'] if e['session'] == session)
            assert exit_record['exit_code'] == 0 and exit_record['requests'] == len(exchanges)
            assert exit_record['shutdown'] == work
            assert raw(session + '-stderr.txt') == b''
            counts += len(exchanges)
    assert {e['session'] for e in collected['exits']} == expected_sessions
    assert len(collected['exits']) == report['sessions'] == 12
    assert counts == report['requests'] == 344
    assert report['passed'] and report['fresh_reopens'] == report['byte_identical_resaves'] == 8
    assert not any(report[k] for k in ('application_changed', 'new_native_input', 'new_visible_latency', 'owner_acceptance', 'full_mvp_complete'))
    print('Verified /178 document extensions: four 18-part drawings, eight fresh exact reopens/resaves, 344 exchanges and 12 joined normal exits; source unchanged.')


if __name__ == '__main__':
    main()
