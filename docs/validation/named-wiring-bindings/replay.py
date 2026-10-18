#!/usr/bin/env python3
"""Compare captured wiring controls with a fresh exact /149 agent process.

No connection/host argument is accepted. Only a newly owned stdio process runs.
Use --observations from the verbose focused Go test and a new --output path.
"""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def observation(state):
    result = state.get('result')
    stale = not result or result['stale']
    return dict(components=state['components'], wires=state['wires'], nets=state['nets'],
                drawing=state['drawing'], selected=state['selected'], undo_name=state['undo_name'],
                result_stale=stale, current_a=None if stale else result['current_a'])


def compare(expected, actual):
    expected = {k: v for k, v in expected.items() if k != 'case'}
    assert expected.keys() == actual.keys()
    for key, value in expected.items():
        if key == 'current_a' and value is not None:
            assert math.isclose(value, actual[key], rel_tol=1e-10, abs_tol=5e-12), key
        else:
            assert value == actual[key], key


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--observations', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    lines = args.observations.read_text().splitlines()
    assert 'PASS' in lines and not any(line.startswith('FAIL') or '--- FAIL:' in line for line in lines)
    captured = {}
    for line in lines:
        if 'named-wiring-observation ' in line:
            item = json.loads(line.split('named-wiring-observation ', 1)[1])
            assert item['case'] not in captured
            captured[item['case']] = item
    assert set(captured) == {'mounted-complete', 'mounted-undo', 'native-adapter-complete'}
    receipt_path = ROOT/'research/ui-audit/first-version/placement-handoff-review-checkpoint.json'
    receipt = json.loads(receipt_path.read_text())
    binary_root = ROOT/'build/first-version-149-tools'
    binary = binary_root/'build/elab-agent'
    assert digest(binary) == receipt['agent_sha256']
    ledger = json.loads((ROOT/'research/13-native-command-inventory.json').read_text())
    for filename, expected in ledger['source_sha256'].items():
        assert digest(ROOT/filename) == expected, filename
    assert json.loads((ROOT/'third_party/vellum/sdk.lock.json').read_text()) == receipt['sdk']
    spec = importlib.util.spec_from_file_location('owned_wiring_client', ROOT/'scripts/persistence-demo.py')
    helper = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(helper)
    helper.ROOT = binary_root
    client = helper.Client(command_preconditions=True)
    try:
        initial = client.ask('project.inspect')
        assert initial['components'] == [] and initial['wires'] == [] and initial['document']['generation'] == 1
        for part in [
            dict(id='V1', kind='dc-source', value='5V', position=dict(x=-200, y=0)),
            dict(id='R1', kind='resistor', value='1k', position=dict(x=0, y=-120)),
            dict(id='D1', kind='led', position=dict(x=200, y=0)),
            dict(id='GND', kind='ground', position=dict(x=-200, y=120)),
        ]:
            client.ask('component.place', part, True)
            client.ask('simulation.wait')
        client.ask('drawing.tool.set', dict(tool='wire'), True)
        for source, destination in [('V1.positive', 'R1.1'), ('R1.2', 'D1.anode'),
                                    ('D1.cathode', 'GND.reference'), ('GND.reference', 'V1.negative')]:
            pair = {'from': source, 'to': destination}
            client.ask('wire.terminals.set', pair, True)
            client.ask('wire.connect', pair, True)
            client.ask('simulation.wait')
        complete = observation(client.ask('simulation.wait'))
        for name in ['mounted-complete', 'native-adapter-complete']:
            compare(captured[name], complete)
        assert math.isclose(complete['current_a'], .002916796577441, rel_tol=0, abs_tol=5e-12)
        client.ask('edit.undo', {}, True)
        undone = observation(client.ask('simulation.wait'))
        compare(captured['mounted-undo'], undone)
        assert len(undone['wires']) == 3 and undone['result_stale']
        report = dict(passed=True, transport='owned headless stdio', release='0.132.0 /149',
                      agent_sha256=digest(binary), production_source_files_verified=len(ledger['source_sha256']),
                      observations_sha256=digest(args.observations), compared_cases=sorted(captured),
                      complete=complete, undone=undone,
                      scope='mounted selectors/buttons and macOS adapter callback bindings against exact public agent binary',
                      limits=['No physical OS menu/IME/VoiceOver or event-to-pixel measurement',
                              'Current review bytes preserved; no product version increment or owner acceptance'])
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps(dict(passed=True, current_a=complete['current_a'], compared_cases=sorted(captured))))
    finally:
        client.close()


if __name__ == '__main__':
    main()
