#!/usr/bin/env python3
"""Exercise discoverable component domains and exact value entry on either transport."""
import argparse
import importlib.util
import json
import math
from pathlib import Path


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)

    def refuse(command, params):
        before = client.ask('project.inspect')
        client.ask(command, params, True, 'invalid-params')
        assert client.ask('project.inspect') == before

    try:
        parts = client.ask('parts.list')
        library = {part['kind']: part for part in parts}
        assert library['dc-source']['value_domain']['minimum'] == 0
        assert library['capacitor']['value_domain']['maximum'] == 1
        assert library['net-label']['value_type'] == 'label' and library['npn']['value_type'] == 'none'
        details = {}
        for command in ('component.place', 'component.set', 'capacitor.initial.set'):
            schema = client.ask('capabilities.list', {'command': command})['capabilities'][0]['input_schema']
            assert schema['x-command-contract'] and schema['properties']['value']['x-max-utf8-bytes'] == 128
            details[command] = schema
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        for command, params in [
            ('component.set', dict(id='V1', value='1e-999 V')),
            ('capacitor.initial.set', dict(id='C1', value='1e-999 V')),
            ('component.place', dict(id='V2', kind='dc-source', position=dict(x=400, y=200), value='1e-999 V')),
            ('component.set', dict(id='C1', value='1e-320pF')),
            ('component.set', dict(id='V1', value='13000 mV')),
        ]:
            refuse(command, params)
        client.ask('component.set', dict(id='V1', value='500 mV'), True)
        client.ask('component.set', dict(id='R1', value='10K ohm'), True)
        client.ask('component.set', dict(id='C1', value='1 µF'), True)
        client.ask('capacitor.initial.set', dict(id='C1', value='-250 mV'), True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        stepped = client.ask('simulation.step', dict(duration_s='10 ms'), True)
        expected = .5 - .75 * math.exp(-1)
        assert math.isclose(stepped['transient']['sample']['voltage_v'], expected, abs_tol=1e-12)
        graph = dict(components=stepped['components'], wires=stepped['wires'])
        project = directory / 'millivolt-rc.elab'
        client.ask('project.save', dict(path=str(project)), True)
        client.ask('project.load', dict(path=str(project)), True)
        reopened = client.ask('simulation.wait')
        assert reopened['components'] == graph['components'] and reopened['wires'] == graph['wires']
        assert reopened['transient']['sample']['voltage_v'] == -.25
        client.ask('project.clear', dict(discard_changes=True), True)
        refuse('component.place', dict(id='Pulse', kind='finite-pulse-source', position=dict(x=0, y=0), value='-1000 mV'))
        client.ask('component.place', dict(id='Pulse', kind='finite-pulse-source', position=dict(x=0, y=0), value='1000 mV'), True)
        client.ask('source.pulse.set', dict(id='Pulse', waveform=dict(high_v=1, low_v=-2, high_s=.05, low_s=.05, rise_s=.001, fall_s=.001)), True)
        client.ask('component.set', dict(id='Pulse', value='-1000 mV'), True)
        refuse('component.set', dict(id='Pulse', value='-3000 mV'))
        client.ask('component.place', dict(id='Sine', kind='sine-source', position=dict(x=100, y=0)), True)
        client.ask('source.sine.set', dict(id='Sine', waveform=dict(amplitude_v=1, offset_v=3, frequency_hz=10, phase_degrees=0)), True)
        client.ask('component.set', dict(id='Sine', value='9000 mV'), True)
        refuse('component.set', dict(id='Sine', value='9001 mV'))
        client.ask('component.place', dict(id='Label', kind='net-label', position=dict(x=200, y=0), value='VCC/+'), True)
        refuse('component.set', dict(id='Label', value='VCC +'))
        client.ask('component.place', dict(id='D1', kind='led', position=dict(x=300, y=0)), True)
        refuse('component.set', dict(id='D1', value='1'))
        final = client.ask('project.inspect')
        pulse = next(c for c in final['components'] if c['id'] == 'Pulse')
        sine = next(c for c in final['components'] if c['id'] == 'Sine')
        assert pulse['pulse']['high_v'] == -1 and pulse['pulse']['low_v'] == -2
        assert sine['sine']['amplitude_v'] == 9 and sine['sine']['offset_v'] == 3
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio', parts=parts,
                      detailed_inputs=details, rc_graph=graph, rc_voltage_v=stepped['transient']['sample']['voltage_v'], expected_voltage_v=expected,
                      coupled_components=final['components'], checks=['Discoverable per-kind domains and complete value-command contracts',
                      'Unrepresentable and out-of-range inputs preserve all public state', 'Millivolt source and signed capacitor startup match analytic RC result',
                      'Saved numeric SI graph reopens unchanged', 'Current/factory waveform constraints preserved', 'Labels and fixed symbols retain their value semantics'],
                      limits=['Existing numerical/model/work limits unchanged', 'Not full command coverage, physical-device or accessibility acceptance'])
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps(dict(passed=True, transport=report['transport'])))
    finally:
        client.close()


if __name__ == '__main__':
    main()
