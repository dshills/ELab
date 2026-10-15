#!/usr/bin/env python3
"""Reconstruct RC/LED and high-pass circuits; --connect requires an owned blank session."""
import argparse
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

    def ask(command, params=None):
        return client.ask(command, params, True)

    def restart():
        ask('simulation.restart')
        return client.ask('simulation.wait')

    def step(duration):
        ask('simulation.step', {'duration_s': duration})
        return client.ask('simulation.wait')

    def place(identity, kind, x, y, value=''):
        ask('component.place', dict(id=identity, kind=kind, position=dict(x=x, y=y), value=value))

    def connect(a, b):
        ask('wire.terminals.set', {'from': a, 'to': b})
        ask('wire.connect', {'from': a, 'to': b})

    try:
        assert not client.ask('project.inspect')['components'], 'use an owned blank session'
        catalog = client.ask('examples.list')
        info = next(e for e in catalog if e['id'] == 'led-fade')
        schemas = client.ask('capabilities.list', {'command': 'example.load'})
        ask('simulation.pause')
        for identity, kind, x, y, value in [
            ('V1', 'pulse-source', -230, 20, '5 V'), ('R1', 'resistor', 0, -110, '1k'),
            ('D1', 'led', 120, 20, ''), ('GND', 'ground', 0, 110, ''), ('C1', 'capacitor', 330, 20, '100uF'),
        ]:
            place(identity, kind, x, y, value)
        for a, b in [('V1.positive', 'R1.1'), ('R1.2', 'D1.anode'), ('D1.cathode', 'GND.reference'),
                     ('GND.reference', 'V1.negative'), ('D1.anode', 'C1.1'), ('C1.2', 'GND.reference')]:
            connect(a, b)
        settings = dict(step_s=.001, high_s=.2, low_s=.3, initial_voltage_v=0, seconds_per_wall_second=.1)
        ask('simulation.transient.set', settings)
        startup = restart()
        assert startup['analysis'] == 'Network transient' and startup['mixed']['capacitors'][0]['voltage_v'] == 0
        edge = step('200 ms')
        assert edge['transient']['sample']['input_v'] == 0
        assert edge['mixed']['capacitors'][0]['voltage_v'] > 2 and edge['mixed']['leds'][0]['current_a'] > .002
        trace = next(t for t in edge['scope']['traces'] if t['probe']['id'] == 'v:C1')
        corners = [(a, b) for a, b in zip(trace['points'], trace['points'][1:])
                   if a['time_s'] == b['time_s'] and abs(a['time_s']-.2) < 1e-12]
        assert corners and all(a['value'] == b['value'] for a, b in corners)
        manual_tail = step('10 ms')
        tail_current = manual_tail['mixed']['leds'][0]['current_a']
        assert 0 < tail_current < edge['mixed']['leds'][0]['current_a']
        ask('example.load', {'id': 'led-fade', 'discard_changes': True})
        ask('simulation.pause')
        template = restart()
        assert template['components'] == startup['components'] and template['wires'] == startup['wires']
        template_tail = step('210 ms')
        assert abs(template_tail['mixed']['leds'][0]['current_a']-tail_current) < 1e-9
        assert abs(template_tail['mixed']['capacitors'][0]['voltage_v']-manual_tail['mixed']['capacitors'][0]['voltage_v']) < 1e-9
        admitted = ask('simulation.verify', {'duration_s': '210 ms', 'probe_ids': ['v:C1', 'i:D1', 'v:V1.positive']})
        identity = admitted['last_operation']
        deadline = time.monotonic()+15
        while True:
            view = client.ask('operation.inspect', {'id': identity})
            status = view['operation']['status']
            if status == 'completed':
                break
            assert status in ('queued', 'running') and time.monotonic() < deadline, view
            time.sleep(.01)
        before = client.ask('project.inspect')
        metrics = client.ask('recording.statistics', dict(id=identity, probe_id='i:D1', start_s='200 ms', end_s='210 ms'))
        assert metrics['time_mean'] > 0 and metrics['time_rms'] > 0
        assert client.ask('project.inspect') == before
        ask('component.set', {'id': 'C1', 'value': '220uF'})
        assert client.ask('project.inspect')['mixed']['stale']
        restart()
        larger = step('210 ms')
        larger_current = larger['mixed']['leds'][0]['current_a']
        assert larger_current > tail_current
        saved = assets/'fade-220u.elab'
        ask('project.save', {'path': str(saved)})
        ask('project.load', {'path': str(saved)})
        loaded = client.ask('simulation.wait')
        assert loaded['example']['info']['id'] == 'led-fade' and loaded['example']['modified']
        assert loaded['transient']['sample']['time_s'] == 0 and loaded['mixed']['capacitors'][0]['voltage_v'] == 0
        ask('project.clear', {'discard_changes': True})
        ask('simulation.pause')
        for identity, kind, x, y, value in [('HV', 'dc-source', -200, 0, '5 V'),
                                          ('HC', 'capacitor', 0, -100, '1uF'),
                                          ('HR', 'resistor', 200, 0, '10k'), ('HG', 'ground', 0, 120, '')]:
            place(identity, kind, x, y, value)
        for a, b in [('HV.positive', 'HC.1'), ('HC.2', 'HR.1'), ('HR.2', 'HG.reference'), ('HG.reference', 'HV.negative')]:
            connect(a, b)
        high_start = restart()
        assert high_start['mixed']['capacitors'][0]['voltage_v'] == 0
        high = step('10 ms')
        output = high['mixed']['resistors'][0]['voltage_v']
        assert abs(output-5/math.e) < .003
        assert abs(high['mixed']['capacitors'][0]['voltage_v']-(5-5/math.e)) < .003
        assert high['mixed']['kcl_residual_a'] < 1e-9 and abs(high['mixed']['power_residual_w']) < 1e-8
        ask('project.save', {'path': str(assets/'high-pass.elab')})
        ask('export.schematic.svg', {'path': str(assets/'high-pass.svg')})
        ask('export.schematic.png', {'path': str(assets/'high-pass.png')})
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      example=info, load_contract=schemas,
                      observations=dict(manual_matches_template=True, capacitor_edge_continuity=True,
                                        led_current_100uf_10ms_after_edge_a=tail_current,
                                        led_current_220uf_10ms_after_edge_a=larger_current,
                                        tail_statistics=metrics, high_pass_output_10ms_v=output,
                                        high_pass_reference_10ms_v=5/math.e),
                      checks=['Blank placement/named wiring equals the guided LED fade graph',
                              'Stored voltage and LED current persist across ideal falling source edge',
                              'Verify/measurement preserve live input; larger capacitor lengthens this fixture tail',
                              'Save/reopen preserves edited origin and original recording, restarting live charge/time',
                              'Series-capacitor high-pass matches independent analytic exponential within 3 mV',
                              'Existing SVG/PNG output accepts the newly routed networks'],
                      limits=['Pinned generic models; brightness illustrative; no physical part, ESR, heating or ratings claim',
                              'Existing bounded network/grounded-source/ideal-constraint limits remain; no numerical kernel or file-version change'])
        with args.output.open('x') as output_file:
            json.dump(report, output_file, indent=2)
            output_file.write('\n')
        print(json.dumps(dict(passed=True, transport=report['transport'], **report['observations']), default=str))
    finally:
        if args.connect:
            client.ask('project.inspect')
            ask('app.close', {'discard_changes': True})
        client.close()


if __name__ == '__main__':
    main()
