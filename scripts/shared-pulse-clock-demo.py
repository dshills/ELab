#!/usr/bin/env python3
"""Exercise ideal-square clock controls and a blinking LED through public commands."""
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
COMMAND = 'simulation.pulse.timing.set'


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)

    def inspect():
        return client.ask('project.inspect')

    def clock(state, identity='VIN'):
        return next(c for c in state['pulse_timings'] if c['id'] == identity)

    def refuse(params, code='invalid-params'):
        before = inspect()
        client.ask(COMMAND, params, True, code)
        assert inspect() == before

    try:
        client.ask('example.npn-pulse', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        client.ask('component.select', {'id': 'VIN'}, True)
        original = inspect()
        schema = client.ask('capabilities.list', {'command': COMMAND})['capabilities'][0]['input_schema']
        assert schema['x-command-contract']['scope'] == 'ideal-square-shared-clock'
        assert len(schema['oneOf']) == 2 and schema['properties']['frequency_hz']['type'] == 'string'
        assert inspect() == original
        state = client.ask(COMMAND, dict(id='VIN', frequency_hz='20 Hz', duty_percent='25 %'), True)
        assert state['components'] == original['components']
        for key in ('step_s', 'initial_voltage_v', 'seconds_per_wall_second'):
            assert state['transient_settings'][key] == original['transient_settings'][key]
        configured = clock(state)
        assert configured['shared'] and configured['source_kind'] == 'pulse-source'
        assert configured['frequency_hz'] == 20 and configured['duty_percent'] == 25
        same = client.ask(COMMAND, dict(id='VIN', frequency_hz=configured['frequency_text'], duty_percent=configured['duty_text']), True)
        for key in ('transient_settings', 'document', 'transient', 'undo_entry', 'run_id'):
            assert same.get(key) == state.get(key), key
        for params in [dict(id='VIN', frequency_hz='1 kHz', duty_percent='50 %'),
                       dict(id='VIN', frequency_hz='20', period_s='.05 s', duty_percent='25'),
                       dict(id='VIN', frequency_hz=20, duty_percent='25'),
                       dict(id='VIN', duty_percent='25'),
                       dict(id='VIN', frequency_hz='1e-999 Hz', duty_percent='25'),
                       dict(id='VCC', frequency_hz='20', duty_percent='25')]:
            refuse(params)
        refuse(dict(id='missing', frequency_hz='20', duty_percent='25'), 'not-found')
        client.ask('simulation.restart', {}, True)
        on = client.ask('simulation.wait')
        assert on['npn']['state'] == 'saturation' and on['transient']['sample']['input_v'] == 5
        client.ask('simulation.step', {'duration_s': '12.5 ms'}, True)
        off = client.ask('simulation.wait')
        assert off['npn']['state'] == 'cutoff' and off['transient']['sample']['input_v'] == 0
        client.ask('simulation.step', {'duration_s': '37.5 ms'}, True)
        again = client.ask('simulation.wait')
        assert again['npn']['state'] == 'saturation' and again['transient']['sample']['input_v'] == 5
        admitted = client.ask('simulation.verify', dict(duration_s='100 ms', sample_step_s='1 ms',
                                                       probe_ids=['v:VIN.positive', 'i:D1']), True)
        identity = admitted['last_operation']
        deadline = time.monotonic() + 10
        while True:
            operation = client.ask('operation.inspect', {'id': identity})['operation']
            assert operation['status'] not in ('failed', 'canceled'), operation
            if operation['status'] == 'completed':
                break
            assert time.monotonic() < deadline
            time.sleep(.01)
        rows = client.ask('operation.samples', {'id': identity, 'limit': 200})['rows']
        interior = 0
        for row in rows:
            phase = row['time_s'] % .05
            if min(abs(phase), abs(phase-.0125), abs(phase-.05)) < 1e-12:
                continue  # Exact corners retain both left and right values.
            high = phase < .0125
            assert row['values'][0] == (5 if high else 0), row
            assert row['values'][1] > .002 if high else abs(row['values'][1]) < 1e-8, row
            interior += 1
        assert interior >= 90
        for edge in (.0125, .05, .0625, .1):
            values = {row['values'][0] for row in rows if abs(row['time_s']-edge) < 1e-12}
            assert values == {0, 5}, (edge, values)
        timing = client.ask('recording.timing', dict(id=identity, probe_id='v:VIN.positive', level='2.5 V',
                                                   direction='both', start_s=0, end_s='100 ms'))
        high = timing['first_high_pulse']
        assert math.isclose(high['minimum_s'], .0125, abs_tol=1e-15)
        assert math.isclose(high['maximum_s'], .0125, abs_tol=1e-15)
        client.ask(COMMAND, dict(id='VIN', period_s='100 ms', duty_percent='25 %'), True)
        saved_settings = inspect()['transient_settings']
        project = assets / 'shared-clock-blink.elab'
        client.ask('project.save', {'path': str(project)}, True)
        document = json.loads(project.read_text())
        assert 'pulse_timings' not in document
        digest = hashlib.sha256(project.read_bytes()).hexdigest()
        client.ask('project.load', {'path': str(project)}, True)
        reopened = client.ask('simulation.wait')
        assert reopened['transient_settings'] == saved_settings
        assert hashlib.sha256(project.read_bytes()).hexdigest() == digest
        assert clock(reopened)['period_s'] == .1 and clock(reopened)['duty_percent'] == 25
        client.ask('project.clear', {'discard_changes': True}, True)
        for i, kind in enumerate(('pulse-source', 'pulse-source', 'finite-pulse-source')):
            client.ask('component.place', dict(id=f'S{i}', kind=kind, position=dict(x=200*i, y=0)), True)
        before = inspect()
        shared = client.ask(COMMAND, dict(id='S1', period_s='200 ms', duty_percent='60 %'), True)
        assert shared['components'] == before['components']
        affected = [c['id'] for c in shared['pulse_timings'] if c['shared']]
        assert affected == ['S0', 'S1']
        for identity in affected:
            assert clock(shared, identity)['frequency_hz'] == 5
            assert math.isclose(clock(shared, identity)['duty_percent'], 60, abs_tol=1e-12)
        assert clock(shared, 'S2') == clock(before, 'S2')
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', detailed_schema=schema,
                      observations=dict(configured_clock=configured, source_samples_checked=len(rows), interior_samples_checked=interior,
                                        high_width_bounds_s=[high['minimum_s'], high['maximum_s']], affected_shared_ids=affected,
                                        reopened_settings=saved_settings),
                      checks=['Shared clock discovery and direct frequency/period/percent input preserve sampling, playback and component definitions',
                              'Displayed timing reentry preserves phase bits, history and live state',
                              'Sampling/type/alternative/underflow/wrong-kind/missing-source refusals preserve state',
                              'Live NPN LED switches off at 12.5 ms and on at 50 ms for a 20 Hz/25-percent clock',
                              'Captured source and LED current agree with each interior phase and retain both sides of exact edges',
                              'Threshold measurement resolves a 12.5 ms high pulse; settings and recording save/reopen',
                              'Both ideal-square IDs share one clock in an incomplete drawing while finite-pulse definition remains unchanged'],
                      limits=['Ideal-square timing is global; selecting one source does not give it independent timing',
                              'Configured duty and measured threshold timing remain separate',
                              'Existing models, kernels, output sampling, work limits and project format remain unchanged',
                              'No physical device or complete physical typing/focus/accessibility/visible-latency qualification'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as output:
            json.dump(report, output, indent=2)
            output.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], source_samples=len(rows))))
    finally:
        try:
            if args.connect:
                inspect()
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()
