#!/usr/bin/env python3
"""Build a finite-pulse RC circuit and exercise shared frequency/period/duty controls."""
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
COMMAND = 'source.pulse.timing.set'


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect, command_preconditions=True)

    def inspect():
        return client.ask('project.inspect')

    def clock(state):
        return next(c for c in state['pulse_timings'] if c['id'] == 'Clock')

    def source(state, identity='Clock'):
        return next(c['pulse'] for c in state['components'] if c['id'] == identity)

    def refuse(params, code='invalid-params'):
        before = inspect()
        client.ask(COMMAND, params, True, code)
        assert inspect() == before

    def keyed(params, expect=None):
        state = inspect()
        client.sequence += 1
        request = dict(version=1, id=f'clock-key-{client.sequence}', command=COMMAND, params=params,
                       expected_command_revision=state['command_revision'], key='clock-unit-retry')
        client.process.stdin.write(json.dumps(request) + '\n')
        client.process.stdin.flush()
        response = json.loads(client.process.stdout.readline())
        if expect:
            assert response.get('error', {}).get('code') == expect, response
        else:
            assert 'error' not in response, response
        inspect()  # A retry returns the historical admission, not the current snapshot.
        return response

    try:
        client.ask('project.clear', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        before = inspect()
        discovery = client.ask('capabilities.list', {'command': COMMAND})
        schema = discovery['capabilities'][0]['input_schema']
        assert schema['x-command-contract']['scope'] == 'finite-pulse-frequency-period-duty'
        assert len(schema['oneOf']) == 2 and schema['properties']['frequency_hz']['type'] == 'string'
        assert inspect() == before
        client.ask('component.place', dict(id='Clock', kind='finite-pulse-source', position=dict(x=-200, y=0)), True)
        incomplete = client.ask(COMMAND, dict(id='Clock', frequency_hz='20 Hz', duty_percent='25 %'), True)
        assert clock(incomplete)['frequency_hz'] == 20 and clock(incomplete)['duty_percent'] == 25
        for params in [dict(id='R1', kind='resistor', value='10k', position=dict(x=0, y=0)),
                       dict(id='C1', kind='capacitor', value='1uF', position=dict(x=200, y=0)),
                       dict(id='GND', kind='ground', position=dict(x=100, y=200))]:
            client.ask('component.place', params, True)
        for start, end in [('Clock.positive', 'R1.1'), ('R1.2', 'C1.1'),
                           ('C1.2', 'GND.reference'), ('Clock.negative', 'GND.reference')]:
            client.ask('wire.connect', {'from': start, 'to': end}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        settings = inspect()['transient_settings']
        original = source(inspect())
        first = keyed(dict(id='Clock', frequency_hz='10 Hz', duty_percent='50 %'))
        replay = keyed(dict(id='Clock', frequency_hz='.01 kHz', duty_percent='5e1 %'))
        assert replay.get('duplicate') and replay['result'] == first['result']
        keyed(dict(id='Clock', frequency_hz='10', duty_percent='40'), 'key-reused')
        client.ask(COMMAND, dict(id='Clock', period_s='50 ms', duty_percent='25 %'), True)
        before = inspect()
        current = clock(before)
        same = client.ask(COMMAND, dict(id='Clock', frequency_hz=current['frequency_text'], duty_percent=current['duty_text']), True)
        for key in ('components', 'transient', 'document', 'undo_entry', 'run_id'):
            assert before.get(key) == same.get(key), key
        assert same['transient_settings'] == settings
        for params in [dict(id='Clock', frequency_hz='1 kHz', duty_percent='50'),
                       dict(id='Clock', frequency_hz='20', period_s='.05 s', duty_percent='25'),
                       dict(id='Clock', frequency_hz=20, duty_percent='25'),
                       dict(id='Clock', frequency_hz='20', duty_percent='0'),
                       dict(id='Clock', frequency_hz='1e-999 Hz', duty_percent='25')]:
            refuse(params)
        refuse(dict(id='missing', frequency_hz='20', duty_percent='25'), 'not-found')
        client.ask('simulation.restart', {}, True)
        stepped = client.ask('simulation.step', {'duration_s': '5 ms'}, True)
        voltage = stepped['mixed']['capacitors'][0]['voltage_v']
        expected = 5 * (1 - 10 * (math.exp(-.004/.01) - math.exp(-.005/.01)))
        assert stepped['transient']['sample']['input_v'] == 5 and abs(voltage-expected) < .003
        wave = source(stepped)
        assert wave['rise_s'] == original['rise_s'] and wave['fall_s'] == original['fall_s']
        assert wave['high_v'] == original['high_v'] and wave['low_v'] == original['low_v']
        captured = client.ask('simulation.verify', dict(duration_s='50 ms', sample_step_s='500 us',
                                                       probe_ids=['v:Clock.positive', 'v:C1']), True)
        identity = captured['last_operation']
        deadline = time.monotonic() + 10
        while True:
            operation = client.ask('operation.inspect', {'id': identity})['operation']
            assert operation['status'] not in ('failed', 'canceled'), operation
            if operation['status'] == 'completed':
                break
            assert time.monotonic() < deadline
            time.sleep(.01)
        rows = client.ask('operation.samples', {'id': identity, 'limit': 200})['rows']
        for row in rows:
            phase = row['time_s'] % .05
            if phase >= .05-1e-12:
                phase = 0
            value = 5*phase/.001 if phase < .001 else 5 if phase < .0125 else 5*(1-(phase-.0125)/.001) if phase < .0135 else 0
            assert abs(row['values'][0]-value) < 1e-9, row
        timing = client.ask('recording.timing', dict(id=identity, probe_id='v:Clock.positive', level='2.5 V',
                                                   direction='both', start_s=0, end_s='.05 s'))
        high = timing['first_high_pulse']
        assert high['minimum_s'] <= .0125 <= high['maximum_s']
        client.ask(COMMAND, dict(id='Clock', period_s='100 ms', duty_percent='25 %'), True)
        configured = clock(inspect())
        project = assets / 'pulse-clock-rc.elab'
        client.ask('project.save', {'path': str(project)}, True)
        document = json.loads(project.read_text())
        assert 'pulse_timings' not in document
        digest = hashlib.sha256(project.read_bytes()).hexdigest()
        client.ask('project.load', {'path': str(project)}, True)
        reopened = client.ask('simulation.wait')
        assert clock(reopened) == configured and hashlib.sha256(project.read_bytes()).hexdigest() == digest
        before = inspect()
        historical = keyed(dict(id='Clock', frequency_hz='10 Hz', duty_percent='50 %'))
        assert historical.get('duplicate') and historical['result'] == first['result'] and inspect() == before
        assert hashlib.sha256(project.read_bytes()).hexdigest() == digest
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', detailed_schema=schema,
                      observations=dict(configured_clock=configured, capacitor_at_5ms_v=voltage, independent_reference_v=expected,
                                        source_samples_checked=len(rows), high_width_bounds_s=[high['minimum_s'], high['maximum_s']]),
                      checks=['Constructed pulse-driven RC circuit from blank through public placement/named wiring commands',
                              'Configured incomplete source, frequency or unit-period plus percent duty, retaining ramps/levels/settings',
                              'Equivalent unit-text retries replay original admission; changed intent refuses; late replay cannot overwrite reopened source',
                              'Displayed timing reentry preserves source bits, physical state and history',
                              'Incompatible ramps, alternative/type/range/underflow and missing-source requests refuse without state change',
                              'Live finite-ramp RC agrees with independent first-order solution within existing 3 mV envelope',
                              'Captured source rows follow independent 20 Hz/25-percent piecewise ramp; measured high-width brackets contain 12.5 ms',
                              'Ordinary saved phase fields restore derived timing; no new persisted field or file version'],
                      limits=['Configured high-phase percent includes rise, not a measured threshold duty; accepted source phase lengths are authoritative',
                              'Ideal square pulse-source retains separate shared experiment timing',
                              'Rise/fall are never silently scaled, output sampling is not automatically changed, and ordinary model/step/work limits still apply',
                              'No physical device, complete physical typing/focus/accessibility or visible-latency qualification'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as output:
            json.dump(report, output, indent=2)
            output.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], source_samples=len(rows))))
    finally:
        try:
            if args.connect:
                client.ask('project.inspect')
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()
