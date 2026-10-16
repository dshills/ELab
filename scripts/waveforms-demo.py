#!/usr/bin/env python3
"""Public per-source waveform/RC references, Verify, sampling fit and portability."""
import argparse
import csv
import importlib.util
import json
import math
from pathlib import Path
import tempfile
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def build_rc(client, kind, source_id):
    client.ask('example.rc', {'discard_changes': True}, True)
    client.ask('simulation.pause', {}, True)
    client.ask('edit.batch', {'name': 'Per-source waveform RC', 'edits': [
        {'command': 'object.delete', 'params': {'id': 'V1'}},
        {'command': 'component.place', 'params': {'id': source_id, 'kind': kind, 'position': {'x': -200, 'y': 0}}},
        {'command': 'wire.connect', 'params': {'from': source_id+'.positive', 'to': 'R1.1'}},
        {'command': 'wire.connect', 'params': {'from': source_id+'.negative', 'to': 'GND.reference'}},
    ]}, True)


def sine_reference(at):
    omega, tau, amplitude, offset, phase = 2*math.pi*20, .01, 2., 1., math.pi/6
    x = omega*tau
    def steady(t):
        return offset+amplitude*(math.sin(omega*t+phase)-x*math.cos(omega*t+phase))/(1+x*x)
    return steady(at)-steady(0)*math.exp(-at/tau)


def wait(client, op):
    deadline = time.monotonic()+15
    while time.monotonic() < deadline:
        view = client.ask('operation.inspect', {'id': op})
        if view['operation']['status'] not in ['queued', 'running', 'canceling']:
            assert view['operation']['status'] == 'completed', view
            return view
        time.sleep(.02)
    raise AssertionError('Verify timeout')


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path)
    parser.add_argument('--artifacts', type=Path)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect)
    fresh = None
    try:
        catalog = client.ask('capabilities.list')
        kinds = client.ask('parts.list')
        build_rc(client, 'sine-source', 'Wave')
        sine = dict(amplitude_v=2., offset_v=1., frequency_hz=20., phase_degrees=30.)
        client.ask('source.sine.set', {'id': 'Wave', 'waveform': sine}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.step', {'duration_s': .03}, True)
        live_sine = client.ask('project.inspect')
        voltage = live_sine['mixed']['capacitors'][0]['voltage_v']
        reference = sine_reference(.03)
        assert abs(voltage-reference) < .003
        source_v = 1+2*math.sin(2*math.pi*20*.03+math.pi/6)
        assert abs(live_sine['mixed']['sources'][0]['voltage_v']-source_v) < 1e-9
        accepted = client.ask('simulation.verify', {'duration_s': .03, 'probe_ids': ['v:Wave.positive', 'v:C1', 'i:R1']}, True)
        sine_record = wait(client, accepted['last_operation'])
        assert abs(sine_record['metadata']['mixed']['capacitors'][0]['voltage_v']-voltage) < 1e-10
        build_rc(client, 'finite-pulse-source', 'Ramp')
        pulse = dict(high_v=5., low_v=0., high_s=.007, low_s=.013, rise_s=.002, fall_s=.003)
        client.ask('source.pulse.set', {'id': 'Ramp', 'waveform': pulse}, True)
        before = client.ask('project.inspect')
        assert before['source_warnings']
        client.ask('simulation.sampling.fit', {}, True)
        fitted = client.ask('project.inspect')
        assert fitted['transient_settings']['step_s'] == .0005 and not fitted.get('source_warnings')
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.step', {'duration_s': .005}, True)
        ramp_live = client.ask('project.inspect')
        ramp_voltage = ramp_live['mixed']['capacitors'][0]['voltage_v']
        ramp_reference = 5*(1-5*(math.exp(-.003/.01)-math.exp(-.005/.01)))
        assert abs(ramp_voltage-ramp_reference) < .003
        client.ask('simulation.step', {'duration_s': .007}, True)
        after = client.ask('project.inspect')
        for corner in [.002, .007, .01]:
            assert any(e['mode'] == 'pulse:Ramp:corner' and abs(e['time_s']-corner) < 1e-12 for e in after['transient']['events'])
        assert abs(after['mixed']['sources'][0]['voltage_v']) < 1e-12
        accepted = client.ask('simulation.verify', {'duration_s': .012, 'probe_ids': ['v:Ramp.positive', 'v:C1', 'i:R1']}, True)
        op = accepted['last_operation']
        pulse_record = wait(client, op)
        assert abs(pulse_record['metadata']['mixed']['capacitors'][0]['voltage_v']-after['mixed']['capacitors'][0]['voltage_v']) < 1e-10
        with tempfile.TemporaryDirectory(prefix='elab-waveforms-') as temporary:
            directory = args.artifacts.resolve() if args.artifacts else Path(temporary)
            directory.mkdir(parents=True, exist_ok=True)
            for kind in ['json', 'csv']:
                client.ask('verification.export', {'id': op, 'path': str(directory/('waveform-record.'+kind)), 'format': kind, 'overwrite': True}, True)
            with (directory/'waveform-record.csv').open() as source:
                rows = list(csv.DictReader(source))
            assert rows[-1]['waveform_model'] == 'elab.source-waveforms.v1'
            client.ask('verification.import', {'path': str(directory/'waveform-record.json')}, True)
            path = directory/'waveform-rc.elab'
            client.ask('project.save', {'path': str(path), 'overwrite': True}, True)
            client.ask('export.schematic.svg', {'path': str(directory/'waveform-rc.svg'), 'overwrite': True}, True)
            fresh = helper.Client()
            fresh.ask('project.load', {'path': str(path)}, True)
            restored = fresh.ask('project.inspect')
            assert restored['document']['recording_count'] == 2 and restored['paused'] and not restored['document']['dirty']
            assert next(p for p in restored['components'] if p['id'] == 'Ramp')['pulse'] == pulse
            assert restored['waveform_model'] == after['waveform_model']
            report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                          capabilities=len(catalog['capabilities']), library_kinds=len(kinds), sine_capacitor_voltage_v=voltage,
                          sine_reference_v=reference, sine_source_voltage_v=source_v,
                          ramp_capacitor_voltage_v=ramp_voltage, ramp_reference_v=ramp_reference,
                          fitted_step_s=.0005,
                          checks=['sine offset/amplitude/phase exact RC reference', 'endpoint source readings', 'sine live/Verify equivalence',
                                  'per-source finite ramp exact RC reference', 'sampling warning/fit command', 'three exact continuous corners',
                                  'pulse live/Verify equivalence', 'CSV model attribution and JSON import', 'two embedded records/source settings reopen', 'SVG waveform symbol'])
            if args.output:
                args.output.parent.mkdir(parents=True, exist_ok=True)
                args.output.write_text(json.dumps(report, indent=2)+'\n')
            print(json.dumps(report, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()
