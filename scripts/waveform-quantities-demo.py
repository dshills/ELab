#!/usr/bin/env python3
"""Verify exact waveform quantities, independent RC references and saved numeric data."""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path

spec = importlib.util.spec_from_file_location('waveforms', Path(__file__).with_name('waveforms-demo.py'))
waveforms = importlib.util.module_from_spec(spec)
spec.loader.exec_module(waveforms)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = waveforms.helper.Client(args.connect, command_preconditions=True)

    def inspect():
        return client.ask('project.inspect')

    def part(state, identity):
        return next(c for c in state['components'] if c['id'] == identity)

    def refuse(command, params):
        before = inspect()
        client.ask(command, params, True, 'invalid-params')
        assert inspect() == before

    def capture(duration, source_id):
        state = client.ask('simulation.verify', dict(duration_s=duration, sample_step_s='500 us',
                                                    probe_ids=[f'v:{source_id}.positive', 'v:C1']), True)
        identity = state['last_operation']
        waveforms.wait(client, identity)
        return client.ask('operation.samples', {'id': identity, 'limit': 200})['rows']

    def save_reopen(identity, field, expected, filename):
        path = assets / filename
        client.ask('project.save', {'path': str(path)}, True)
        stored = json.loads(path.read_text())
        source = next(c for c in stored['graph']['components'] if c['id'] == identity)
        assert source[field] == expected and all(isinstance(v, (int, float)) for v in source[field].values())
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        client.ask('project.load', {'path': str(path)}, True)
        state = client.ask('simulation.wait')
        assert part(state, identity)[field] == expected
        assert hashlib.sha256(path.read_bytes()).hexdigest() == digest

    try:
        schemas = {name: client.ask('capabilities.list', {'command': name})['capabilities'][0]['input_schema']
                   for name in ('source.sine.set', 'source.pulse.set')}
        registry = client.ask('capabilities.list')['schemas']
        quantity_schemas = {key: value for key, value in registry.items() if key.startswith('urn:elab:waveform-quantity-')}
        assert len(quantity_schemas) == 3
        for name, schema in schemas.items():
            for field in ('amplitude_v', 'offset_v', 'frequency_hz', 'phase_degrees') if name == 'source.sine.set' else ('high_v', 'low_v'):
                ref = schema['properties']['waveform']['properties'][field]['$ref']
                assert ref in quantity_schemas and len(quantity_schemas[ref]['anyOf']) == 2
        waveforms.build_rc(client, 'sine-source', 'Wave')
        exact = dict(amplitude_v=1.23456789012345, offset_v=-.123456789012345,
                     frequency_hz=123.456789012345, phase_degrees=-123.456789012345)
        text = dict(amplitude_v='1234.56789012345 mV', offset_v='-123456.789012345 uV',
                    frequency_hz='.123456789012345 kHz', phase_degrees='-123.456789012345 °')
        configured = client.ask('source.sine.set', {'id': 'Wave', 'waveform': text}, True)
        assert part(configured, 'Wave')['sine'] == exact
        before = inspect()
        same = client.ask('source.sine.set', {'id': 'Wave', 'waveform': exact}, True)
        for key in ('components', 'document', 'undo_entry', 'run_id', 'transient'):
            assert before.get(key) == same.get(key), key
        save_reopen('Wave', 'sine', exact, 'exact-sine-rc.elab')
        for field, value in [('frequency_hz', '20 V'), ('amplitude_v', '1e-999 V'),
                             ('offset_v', '1 MV'), ('phase_degrees', '3.14 rad'), ('amplitude_v', None)]:
            invalid = dict(text)
            invalid[field] = value
            refuse('source.sine.set', {'id': 'Wave', 'waveform': invalid})
        refuse('source.sine.set', dict(id='Wave', waveform=dict(amplitude_v='11 V', offset_v='2 V',
                                                             frequency_hz='20 Hz', phase_degrees='30 deg')))
        sine_text = dict(amplitude_v='2000 mV', offset_v='1000000 µV', frequency_hz='.02 kHz', phase_degrees='30 °')
        settings = inspect()['transient_settings']
        client.ask('source.sine.set', {'id': 'Wave', 'waveform': sine_text}, True)
        assert inspect()['transient_settings'] == settings
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': '30 ms'}, True)
        live = client.ask('simulation.wait')
        sine_voltage = live['mixed']['capacitors'][0]['voltage_v']
        sine_reference = waveforms.sine_reference(.03)
        assert abs(sine_voltage-sine_reference) < .003
        sine_rows = capture('30 ms', 'Wave')
        for row in sine_rows:
            expected = 1+2*math.sin(2*math.pi*20*row['time_s']+math.pi/6)
            assert abs(row['values'][0]-expected) < 1e-12, row
        waveforms.build_rc(client, 'finite-pulse-source', 'Ramp')
        pulse_text = dict(high_v='4123.456789012345 mV', low_v='-123456.789012345 μV',
                          high_s='7 ms', low_s='13 ms', rise_s='2 ms', fall_s='3 ms')
        expected_pulse = dict(high_v=4.123456789012345, low_v=-.123456789012345,
                              high_s=.007, low_s=.013, rise_s=.002, fall_s=.003)
        client.ask('source.pulse.set', dict(id='Ramp', waveform=pulse_text), True)
        assert part(inspect(), 'Ramp')['pulse'] == expected_pulse
        invalid = dict(pulse_text, low_v='5 V')
        refuse('source.pulse.set', dict(id='Ramp', waveform=invalid))
        client.ask('simulation.sampling.fit', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.step', {'duration_s': '5 ms'}, True)
        live = client.ask('simulation.wait')
        pulse_voltage = live['mixed']['capacitors'][0]['voltage_v']
        low, high, tau, at, rise = expected_pulse['low_v'], expected_pulse['high_v'], .01, .005, .002
        pulse_reference = low*(1-math.exp(-at/tau))+(high-low)*(1-tau/rise*(math.exp(-(at-rise)/tau)-math.exp(-at/tau)))
        assert abs(pulse_voltage-pulse_reference) < .003
        pulse_rows = capture('12 ms', 'Ramp')
        for row in pulse_rows:
            t = row['time_s']
            value = low+(high-low)*t/.002 if t < .002 else high if t < .007 else high-(high-low)*(t-.007)/.003 if t < .01 else low
            assert abs(row['values'][0]-value) < 1e-12, row
        save_reopen('Ramp', 'pulse', expected_pulse, 'exact-ramp-rc.elab')
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', detailed_schemas=schemas, quantity_schemas=quantity_schemas,
                      observations=dict(exact_sine=exact, exact_pulse=expected_pulse, sine_capacitor_at_30ms_v=sine_voltage,
                                        sine_reference_v=sine_reference, sine_source_samples=len(sine_rows),
                                        ramp_capacitor_at_5ms_v=pulse_voltage, ramp_reference_v=pulse_reference,
                                        ramp_source_samples=len(pulse_rows)),
                      checks=['Exact decimal waveform quantities retain binary64 values; numeric re-entry preserves history/physics',
                              'Voltage, frequency, phase and time input share public commands; numeric compatibility and typed local references retained',
                              'Wrong units, underflow, null and coupled model violations refuse without mutation',
                              'Sine and finite-ramp RC transients agree with independent first-order references within existing 3 mV envelope',
                              'Every captured sine/ramp source sample agrees with independent analytic waveform',
                              'Saved waveform fields stay numeric SI and save/reopen preserves exact configured values and bytes'],
                      limits=['No device/model/kernel/sampling/file-version changes',
                              'Quantity text is command-only; persisted data and outputs stay numeric SI',
                              'Existing model and integration/work limits still apply; no physical device or complete mounted typing/IME/accessibility/latency qualification'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as output:
            json.dump(report, output, indent=2)
            output.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], sine_samples=len(sine_rows), ramp_samples=len(pulse_rows))))
    finally:
        try:
            if args.connect:
                inspect()
                client.ask('app.close', {'discard_changes': True}, True)
        finally:
            client.close()


if __name__ == '__main__':
    main()
