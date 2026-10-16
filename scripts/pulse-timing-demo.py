#!/usr/bin/env python3
"""Qualify sampled pulse width/duty through public user/agent commands."""
import argparse
from fractions import Fraction
import hashlib
import importlib.util
import itertools
import json
import math
from pathlib import Path
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', required=True, type=Path)
    parser.add_argument('--artifacts', required=True, type=Path)
    args = parser.parse_args()
    root = args.artifacts.resolve()
    root.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    cases = []

    def capture(duration, probes, step=None):
        params = dict(duration_s=duration, probe_ids=probes)
        if step is not None:
            params['sample_step_s'] = step
        state = client.ask('simulation.verify', params, True)
        identity = state['last_operation']
        deadline = time.monotonic() + 15
        while True:
            operation = client.ask('operation.inspect', {'id': identity})['operation']
            assert operation['status'] not in ('failed', 'canceled'), operation
            if operation['status'] == 'completed':
                return identity
            assert time.monotonic() < deadline
            time.sleep(.01)

    def measure(params):
        before = client.ask('project.inspect')
        result = client.ask('recording.timing', params)
        after_query = client.ask('project.inspect')
        assert before == after_query, sorted(key for key in before if before[key] != after_query.get(key))
        configured = client.ask('recording.timing.set', params, True)
        assert configured['recording']['timing'] == result
        for key in ('document', 'components', 'wires', 'run_id', 'electrical_revision', 'transient', 'undo_entry'):
            assert before.get(key) == configured.get(key), key
        cycle = result['first_cycle']
        intervals = [(Fraction(cycle[edge]['before']['time_s']), Fraction(cycle[edge]['after']['time_s']))
                     for edge in ('rising', 'falling', 'next_rising')]
        assert cycle['period']['minimum_s'] > 0
        fractions = [(y-x)/(z-x) for x, y, z in itertools.product(*intervals)]
        assert math.isclose(cycle['high_fraction']['minimum'], float(min(fractions)), abs_tol=1e-12)
        assert math.isclose(cycle['high_fraction']['maximum'], float(max(fractions)), abs_tol=1e-12)
        filtered = client.ask('recording.timing', dict(params, direction='falling', limit=1))
        for key in ('first_high_pulse', 'first_low_pulse', 'first_cycle'):
            assert result[key] == filtered[key], key
        return result

    def semantic(result):
        result = json.loads(json.dumps(result))
        result['settings'].pop('id')
        result.pop('origin_id')
        return result

    try:
        schemas = {command: client.ask('capabilities.list', {'command': command})['capabilities'][0]['input_schema']
                   for command in ('recording.timing', 'recording.timing.set')}
        client.ask('example.npn-pulse', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        identity = capture('250 ms', ['v:VIN.positive'])
        params = dict(id=identity, probe_id='v:VIN.positive', level='2.5 V', direction='both', start_s=0, end_s='250 ms')
        result = measure(params)
        cycle = result['first_cycle']
        for key, expected in (('high', .05), ('low', .05), ('period', .1)):
            for value in cycle[key].values():
                assert math.isclose(value, expected, abs_tol=1e-12)
        assert math.isclose(cycle['high_fraction']['minimum'], .5, abs_tol=1e-12)
        cases.append(dict(case='ideal-npn-50-percent', result=semantic(result)))

        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('edit.batch', {'name': 'Finite ramp RC input', 'edits': [
            {'command': 'object.delete', 'params': {'id': 'V1'}},
            {'command': 'component.place', 'params': {'id': 'Ramp', 'kind': 'finite-pulse-source', 'position': {'x': -200, 'y': 0}}},
            {'command': 'wire.connect', 'params': {'from': 'Ramp.positive', 'to': 'R1.1'}},
            {'command': 'wire.connect', 'params': {'from': 'Ramp.negative', 'to': 'GND.reference'}},
        ]}, True)
        waveform = dict(high_v=5, low_v=0, high_s=.007, low_s=.013, rise_s=.002, fall_s=.003)
        client.ask('source.pulse.set', dict(id='Ramp', waveform=waveform), True)
        client.ask('simulation.wait')
        identity = capture('50 ms', ['v:Ramp.positive'], '100 us')
        params = dict(id=identity, probe_id='v:Ramp.positive', level='2.5 V', direction='both', start_s=0, end_s='50 ms')
        result = measure(params)
        cycle = result['first_cycle']
        for edge, expected in (('rising', .001), ('falling', .0085), ('next_rising', .021)):
            assert cycle[edge]['before']['time_s']-1e-12 <= expected <= cycle[edge]['after']['time_s']+1e-12
        for key, expected in (('high', .0075), ('low', .0125), ('period', .02)):
            assert cycle[key]['minimum_s']-1e-12 <= expected <= cycle[key]['maximum_s']+1e-12
        assert cycle['high_fraction']['minimum']-1e-12 <= .375 <= cycle['high_fraction']['maximum']+1e-12
        assert cycle['high_fraction']['minimum'] > .35 and cycle['high_fraction']['maximum'] < .4
        cases.append(dict(case='finite-ramp-analytic-37.5-percent', result=semantic(result)))

        client.ask('example.flasher', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        identity = capture('1.2 s', ['i:D1', 'i:D2'])
        for probe in ('i:D1', 'i:D2'):
            params = dict(id=identity, probe_id=probe, level='1 mA', direction='both', start_s='300 ms', end_s='1.2 s')
            result = measure(params)
            c = result['first_cycle']
            assert .47 < c['period']['minimum_s'] <= c['period']['maximum_s'] < .51
            assert 0 < c['high_fraction']['minimum'] <= c['high_fraction']['maximum'] < 1
            cases.append(dict(case='flasher-'+probe, result=semantic(result)))
        asset = root / 'flasher.json'
        client.ask('verification.export', dict(id=identity, path=str(asset), format='json'), True)
        client.ask('verification.import', dict(path=str(asset)), True)
        imported = client.ask('project.inspect')['recording']['id']
        assert semantic(client.ask('recording.timing', dict(params, id=imported))) == semantic(result)
        project = root / 'pulse-measurements.elab'
        client.ask('project.save', dict(path=str(project)), True)
        digest = hashlib.sha256(project.read_bytes()).hexdigest()
        client.ask('project.load', dict(path=str(project)), True)
        state = client.ask('simulation.wait')
        assert 'timing' not in state['recording']
        reopened = client.ask('recording.timing', dict(params, id=state['recording']['id']))
        assert semantic(reopened) == semantic(result)
        assert hashlib.sha256(project.read_bytes()).hexdigest() == digest
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', detailed_schemas=schemas, cases=cases,
                      checks=['Known ideal NPN high/low/period and 50% duty', 'Analytic finite-ramp 37.5% threshold duty differs from programmed 35%',
                              'Both flasher sides measured after startup', 'Exact rational corner enumeration checks shared-edge fraction bounds',
                              'Arrival direction/page leaves interval summaries unchanged', 'Query/set/live/history/file purity and identical import/reopen measurements'],
                      limits=['Observed saved brackets only; hidden transitions may be missed', 'First complete rising cycle is not a periodicity or steady-state certification',
                              'Zero minimum period omits duty fraction; no edge interpolation or device qualification'])
        with args.output.open('x') as target:
            json.dump(report, target, indent=2)
            target.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], cases=len(cases))))
        if args.connect:
            client.ask('app.close', {'discard_changes': True}, True)
    finally:
        client.close()


if __name__ == '__main__':
    main()
