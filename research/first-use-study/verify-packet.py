"""Check study-packet integrity and fixture preparation, not participant outcomes."""
import gzip
import hashlib
import json
import math
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT/'scripts'))
from study_packet_origin import validate_origin


def read(path):
    return json.loads(gzip.decompress(path.read_bytes()) if path.suffix == '.gz' else path.read_text())


def main():
    folder = Path(__file__).resolve().parent
    checkpoint = read(folder/'checkpoint.json')
    assert checkpoint['status'] == 'prepared_unrun' and checkpoint['protocol_revision'] == 1
    assert checkpoint['planned_participants'] == 10 and checkpoint['planned_strata'] == dict(new_to_simulators=5, circuit_building=5)
    assert checkpoint['measured_participants'] == 0 and checkpoint['study_outcomes'] is checkpoint['study_gate_passed'] is None
    assert checkpoint['owner_acceptance'] is False
    names = {str(p.relative_to(folder)) for p in folder.rglob('*') if p.is_file()}
    assert names == set(checkpoint['files_sha256']) | {'checkpoint.json'}
    for name, expected in checkpoint['files_sha256'].items():
        assert hashlib.sha256((folder/name).read_bytes()).hexdigest() == expected, name
    text = (ROOT/'research/07-product-requirements.md').read_text()
    section = text.split('### Formative hobbyist test\n', 1)[1].split('\n## 12 Delivery sequence', 1)[0]
    assert hashlib.sha256(section.encode()).hexdigest() == checkpoint['source_section_sha256']
    assert (folder/'prd-formative-excerpt.md').read_text() == ('### Formative hobbyist test\n'+section).rstrip('\n')+'\n'
    archived = read(folder/'fixtures/preflight.json')['release']
    receipt_bytes = (ROOT/'research/ui-audit/first-version/recorded-timing-feedback-review-checkpoint.json').read_bytes()
    identity = validate_origin(ROOT, checkpoint, archived, receipt_bytes)
    assert identity['receipt_sha256'] == checkpoint['review_receipt_sha256']
    assert identity['release'] == checkpoint['build'] == '0.148.0 /165'
    assert identity['agent_sha256'] == checkpoint['agent_sha256']
    fixture = folder/'fixtures/RC-no-traces.elab'
    preflight = read(folder/'fixtures/preflight.json')
    assert preflight['passed'] and preflight['participants'] == 0 and preflight['release'] == identity
    assert hashlib.sha256(fixture.read_bytes()).hexdigest() == preflight['fixture_sha256'] == preflight['fixture_sha256_after_rehearsal']
    document = read(fixture)
    assert document['paused'] is False and document['scope_probes']['ids'] == []
    parts = {part['id']: part for part in document['graph']['components']}
    assert parts['R1']['value'] == 10000 and parts['C1']['value'] == 1e-6
    assert parts['V1']['kind'] == 'pulse-source' and parts['V1']['value'] == 5
    settings = document['transient_settings']
    assert settings['step_s'] == .001 and settings['high_s'] == settings['low_s'] == .05
    assert settings['seconds_per_wall_second'] == .02 and settings['initial_voltage_v'] == 0
    raw = read(folder/'fixtures/raw-preflight.json.gz')
    state = raw['fresh_reopen']
    assert not state['document']['dirty'] and not state['paused'] and not state['selected']
    assert state['scope']['selected'] == state['scope']['traces'] == [] and state['scope_visible']
    assert 'a' not in state['scope'] and 'b' not in state['scope']
    assert not state['transient']['stale'] and state['transient']['sample']['time_s'] == 0
    assert state['components'] == document['graph']['components'] and state['wires'] == document['graph']['wires']
    sample = raw['analytic_rehearsal']['transient']['sample']
    assert sample['time_s'] == preflight['analytic_charge_time_s'] == .01
    assert math.isclose(sample['voltage_v'], 5*(1-math.exp(-1)), rel_tol=0, abs_tol=1e-9)
    assert sample['voltage_v'] == preflight['analytic_charge_voltage_v']
    conditions = read(folder/'fixtures/state-preflight.json.gz')
    assert conditions['passed'] and conditions['participants'] == 0 and conditions['release'] == identity
    states = conditions['states']
    assert set(states) == {'running', 'paused', 'stale'}
    assert not states['running']['paused'] and not states['running']['transient']['stale']
    assert states['paused']['paused'] and not states['paused']['transient']['stale']
    assert states['stale']['paused'] and states['stale']['transient']['stale']
    assert all(state['transient']['input_mode'] == 'auto' for state in states.values())
    print(json.dumps(dict(packet_integrity_verified=True, fixture_preflight_passed=True,
                          planned_participants=10, measured_participants=0, study_run=False,
                          study_gate_passed=None, owner_acceptance=False,
                          scope="archived packet and recorded preflight integrity",
                          current_runtime_qualified=False)))


if __name__ == '__main__':
    main()
