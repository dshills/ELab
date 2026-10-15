#!/usr/bin/env python3
"""Qualify captured two-threshold rise/fall timing through public commands."""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import time

spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect, command_preconditions=True)
    cases = []

    def capture(duration, probes, **overrides):
        state = client.ask('simulation.verify', dict(duration_s=duration, probe_ids=probes, **overrides), True)
        identity = state['last_operation']
        deadline = time.monotonic() + 15
        while True:
            operation = client.ask('operation.inspect', {'id': identity})['operation']
            assert operation['status'] not in ('failed', 'canceled'), operation
            if operation['status'] == 'completed':
                return identity
            assert time.monotonic() < deadline
            time.sleep(.01)

    def semantic(result):
        result = json.loads(json.dumps(result))
        result['settings'].pop('id')
        result.pop('origin_id')
        return result

    def measure(params):
        before = client.ask('project.inspect')
        result = client.ask('recording.timing', params)
        assert before == client.ask('project.inspect'), 'query changed public state'
        state = client.ask('recording.timing.set', params, True)
        assert state['recording']['timing'] == result
        for key in ('document', 'components', 'wires', 'run_id', 'electrical_revision', 'transient', 'undo_entry'):
            assert before.get(key) == state.get(key), key
        other = client.ask('recording.timing', dict(params, level=params['second_level'], second_level=params['level'], direction='falling', limit=1))
        assert other['transitions'] == result['transitions'], 'entry order/filter/page changed transitions'
        disabled = dict(params)
        disabled.pop('second_level')
        legacy = client.ask('recording.timing', disabled)
        unchanged = json.loads(json.dumps(result))
        unchanged.pop('transitions')
        unchanged['settings'].pop('second_level')
        assert legacy == unchanged, 'second threshold changed original arrival/pulse summaries'
        return result

    def bounded(traversal, expected):
        assert traversal['minimum_s'] - 1e-12 <= expected <= traversal['maximum_s'] + 1e-12, (traversal, expected)

    try:
        schemas = {command: client.ask('capabilities.list', {'command': command})['capabilities'][0]['input_schema']
                   for command in ('recording.timing', 'recording.timing.set')}
        assert all('second_level' in value['properties'] for value in schemas.values())
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        for mode in ('charge', 'discharge'):
            overrides = dict(input_mode=mode)
            if mode == 'discharge':
                overrides['initial_voltage_v'] = 5
            identity = capture('40 ms', ['v:C1'], **overrides)
            params = dict(id=identity, probe_id='v:C1', level='500 mV', second_level='4.5 V', direction='both', start_s=0, end_s='40 ms')
            result = measure(params)
            edge = 'first_rise' if mode == 'charge' else 'first_fall'
            bounded(result['transitions'][edge], .01 * math.log(9))
            cases.append(dict(case='rc-10-90-'+mode, analytic_duration_s=.01 * math.log(9), result=semantic(result)))

        client.ask('edit.batch', {'name': 'Finite ramp input', 'edits': [
            {'command': 'object.delete', 'params': {'id': 'V1'}},
            {'command': 'component.place', 'params': {'id': 'Ramp', 'kind': 'finite-pulse-source', 'position': {'x': -200, 'y': 0}}},
            {'command': 'wire.connect', 'params': {'from': 'Ramp.positive', 'to': 'R1.1'}},
            {'command': 'wire.connect', 'params': {'from': 'Ramp.negative', 'to': 'GND.reference'}},
        ]}, True)
        client.ask('source.pulse.set', dict(id='Ramp', waveform=dict(high_v=5, low_v=0, high_s=.007, low_s=.013, rise_s=.002, fall_s=.003)), True)
        identity = capture('60 ms', ['v:Ramp.positive'], sample_step_s='100 us', input_mode='auto')
        params = dict(id=identity, probe_id='v:Ramp.positive', level='.5 V', second_level='4.5 V', direction='both', start_s=0, end_s='60 ms')
        result = measure(params)
        bounded(result['transitions']['first_rise'], .0016)
        bounded(result['transitions']['first_fall'], .0024)
        cases.append(dict(case='finite-ramp-10-90', analytic_rise_s=.0016, analytic_fall_s=.0024, result=semantic(result)))

        client.ask('example.npn-pulse', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        identity = capture('250 ms', ['v:VIN.positive'])
        params = dict(id=identity, probe_id='v:VIN.positive', level='.5 V', second_level='4.5 V', direction='both', start_s=0, end_s='250 ms')
        result = measure(params)
        for edge in ('first_rise', 'first_fall'):
            assert result['transitions'][edge]['minimum_s'] == result['transitions'][edge]['maximum_s'] == 0
        cases.append(dict(case='ideal-npn-duplicate-time-edges', result=semantic(result)))

        client.ask('example.flasher', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        identity = capture('1.2 s', ['i:D1', 'i:D2'])
        for probe in ('i:D1', 'i:D2'):
            params = dict(id=identity, probe_id=probe, level='.5 mA', second_level='2 mA', direction='both', start_s='300 ms', end_s='1.2 s')
            result = measure(params)
            for edge in ('first_rise', 'first_fall'):
                assert 0 <= result['transitions'][edge]['minimum_s'] <= result['transitions'][edge]['maximum_s'] < .1
            cases.append(dict(case='flasher-'+probe, result=semantic(result)))
        asset = assets / 'flasher-transitions.json'
        client.ask('verification.export', dict(id=identity, path=str(asset), format='json'), True)
        digest = hashlib.sha256(asset.read_bytes()).hexdigest()
        client.ask('verification.import', dict(path=str(asset)), True)
        imported = client.ask('project.inspect')['recording']['id']
        assert semantic(client.ask('recording.timing', dict(params, id=imported))) == semantic(result)
        project = assets / 'transitions.elab'
        client.ask('project.save', dict(path=str(project)), True)
        saved_digest = hashlib.sha256(project.read_bytes()).hexdigest()
        client.ask('project.load', dict(path=str(project)), True)
        state = client.ask('simulation.wait')
        assert 'timing' not in state['recording']
        reopened = client.ask('recording.timing', dict(params, id=state['recording']['id']))
        assert semantic(reopened) == semantic(result)
        assert hashlib.sha256(asset.read_bytes()).hexdigest() == digest
        assert hashlib.sha256(project.read_bytes()).hexdigest() == saved_digest
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', detailed_schemas=schemas, cases=cases,
                      checks=['RC 10–90% analytic tau*ln(9)', 'Finite ramp 80% rise/fall durations', 'Ideal duplicate-time zero transitions',
                              'Both flasher LED currents', 'Entry order/filter/page invariance', 'Single-threshold arrival/pulse results unchanged',
                              'Query/set preserve live source/physics/history/files', 'Raw import/save/reopen reproduces derived transitions'],
                      limits=['Saved-row brackets only; no interpolation, monotonicity, hidden-transition or device guarantee',
                              'Opposite starting-threshold arrivals reset unfinished traversals; interval boundaries create no arrivals'])
        with args.output.open('x') as target:
            json.dump(report, target, indent=2)
            target.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], cases=len(cases))))
    finally:
        try:
            if args.connect:
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()
