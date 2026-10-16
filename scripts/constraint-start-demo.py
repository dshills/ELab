#!/usr/bin/env python3
"""Public high-rail nonlinear startup and floating-gate refusal/recovery."""
import argparse
import importlib.util
import json
import math
from pathlib import Path
import time

from validation_artifacts import discovery_summary

spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def led_reference(voltage):
    lo, hi = 0., voltage/1000
    for _ in range(100):
        current = (lo+hi)/2
        drop = 2*1.380649e-23*300.15/1.602176634e-19*math.log1p(current/1e-20)+current
        if 1000*current+drop < voltage:
            lo = current
        else:
            hi = current
    return (lo+hi)/2


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native endpoint; closes afterward')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = helper.Client(args.connect)
    mutate = lambda command, params=None: client.ask(command, params, True)

    def restart():
        mutate('simulation.restart')
        return client.ask('simulation.wait')

    def reading(probe):
        inspection = client.ask('measurement.inspect', {'target': {'kind': 'probe', 'id': probe}})
        assert inspection['available'], inspection
        values = [r for r in inspection['readings'] if r['probe']['id'] == probe]
        assert len(values) == 1 and values[0]['state'] == 'live', inspection
        return values[0]['value']

    def check_led(volts):
        state = client.ask('project.inspect')
        assert state['status'] != 'error' and state['mixed'] and not state['mixed']['stale'], state['diagnostics']
        current = reading('i:D1')
        want = led_reference(volts)
        assert abs(current-want) < 1e-10
        assert abs(reading('v:R1')+reading('v:D1')-volts) < 1e-8
        assert state['mixed']['kcl_residual_a'] < 1e-10 and abs(state['mixed']['power_residual_w']) < 1e-8
        return dict(rail_v=reading('v:R1.1'), led_voltage_v=reading('v:D1'),
                    current_a=current, independent_current_a=want,
                    kcl_residual_a=state['mixed']['kcl_residual_a'], power_residual_w=state['mixed']['power_residual_w'])

    try:
        assert not client.ask('project.inspect')['components'], 'requires a fresh owned blank session'
        discovery = discovery_summary(client.ask('capabilities.list', {'command': 'component.place'}), assets/'discovery.json')
        numerics = discovery['capabilities'][0]['input_schema']['x-network-numerics']
        assert numerics['max_grounded_constraint_retries'] == 1
        assert numerics['max_newton_iterations'] == 100 and numerics['max_nonlinear_node_update_v'] == .2
        mutate('simulation.pause')
        for identity, kind, value, x, y in [
                ('V1', 'dc-source', '0 V', -300, 0), ('RL', 'resistor', '1k', -100, -100),
                ('RG', 'resistor', '1k', 100, -100), ('M1', 'nmos', '', 0, 50),
                ('M2', 'nmos', '', 200, 50), ('GND1', 'ground', '', 100, 250)]:
            mutate('component.place', dict(id=identity, kind=kind, value=value, position=dict(x=x, y=y)))
        for a, b in [('V1.positive', 'RL.1'), ('RL.2', 'M1.drain'), ('M1.drain', 'M2.drain'),
                     ('M1.source', 'GND1.reference'), ('M2.source', 'GND1.reference'), ('V1.negative', 'GND1.reference'),
                     ('M1.gate', 'RG.1'), ('M2.gate', 'RG.2')]:
            mutate('wire.connect', {'from': a, 'to': b})
        refused = restart()
        assert refused['status'] == 'error' and any('singular' in d['message'] for d in refused['diagnostics'])
        assert not refused.get('mixed') or refused['mixed']['stale']
        mutate('wire.connect', {'from': 'RG.2', 'to': 'GND1.reference'})
        fixed = restart()
        assert fixed['status'] != 'error' and not fixed['mixed']['stale'], fixed['diagnostics']
        assert reading('v:M1.gate-source') == 0 and reading('v:M2.gate-source') == 0
        mutate('example.load', {'id': 'split-supply', 'discard_changes': True})
        mutate('simulation.pause')
        mutate('object.delete', {'id': 'W5'})
        mutate('wire.connect', {'from': 'V2.negative', 'to': 'GND1.reference'})
        for identity in ('V1', 'V2'):
            mutate('component.set', {'id': identity, 'value': '12 V'})
        mutate('component.place', {'id': 'D1', 'kind': 'led', 'position': {'x': 350, 'y': 100}})
        mutate('object.delete', {'id': 'W2'})
        mutate('wire.connect', {'from': 'R1.2', 'to': 'D1.anode'})
        mutate('wire.connect', {'from': 'D1.cathode', 'to': 'V2.negative'})
        restart()
        twenty_four = check_led(24)
        for fmt in ('svg', 'png'):
            mutate('export.schematic.'+fmt, {'path': str(assets/f'led24.{fmt}')})
        mutate('component.place', {'id': 'V3', 'kind': 'dc-source', 'value': '6 V', 'position': {'x': -400, 'y': 0}})
        mutate('object.delete', {'id': 'W1'})
        mutate('wire.connect', {'from': 'V1.positive', 'to': 'V3.negative'})
        mutate('wire.connect', {'from': 'V3.positive', 'to': 'R1.1'})
        restart()
        thirty = check_led(30)
        assert thirty['rail_v'] == 30
        mutate('component.set', {'id': 'V3', 'value': '7 V'})
        over_range = restart()
        assert over_range['status'] == 'error' and any('±30 V' in d['message'] for d in over_range['diagnostics'])
        assert over_range['mixed']['stale']
        stale = client.ask('measurement.inspect', {'target': {'kind': 'probe', 'id': 'i:D1'}})['readings'][0]
        assert stale['state'] == 'stale' and abs(stale['value']-thirty['current_a']) < 1e-12
        mutate('edit.undo')
        restart()
        assert check_led(30) == thirty
        operation = mutate('simulation.verify', {'duration_s': 0, 'probe_ids': ['v:D1', 'i:D1']})['last_operation']
        deadline = time.monotonic()+20
        while True:
            capture = client.ask('operation.inspect', {'id': operation})
            if capture['operation']['status'] == 'completed':
                break
            assert capture['operation']['status'] not in ('failed', 'canceled') and time.monotonic() < deadline
            time.sleep(.01)
        assert abs(capture['metadata']['mixed']['leds'][0]['current_a']-thirty['current_a']) < 1e-12
        mutate('verification.export', {'id': operation, 'format': 'json', 'path': str(assets/'led30.json')})
        mutate('project.save', {'path': str(assets/'led30.elab')})
        mutate('project.load', {'path': str(assets/'led30.elab')})
        client.ask('simulation.wait')
        assert check_led(30) == thirty
        assert client.ask('project.inspect')['document']['recording_count'] == 1
        # An explicit capacitor's startup voltage remains an exact constraint;
        # it is not replaced by a guessed operating point in the seed.
        mutate('component.place', {'id': 'C1', 'kind': 'capacitor', 'value': '1uF', 'position': {'x': 500, 'y': 100}})
        mutate('capacitor.initial.set', {'id': 'C1', 'value': '1.25 V'})
        mutate('wire.connect', {'from': 'C1.1', 'to': 'D1.anode'})
        mutate('wire.connect', {'from': 'C1.2', 'to': 'V2.negative'})
        startup = restart()
        assert startup['status'] != 'error' and not startup['mixed']['stale'], startup['diagnostics']
        assert abs(reading('v:C1')-1.25) < 1e-10
        mutate('simulation.step', {'duration_s': .0001})
        advanced = client.ask('simulation.wait')
        cap = reading('v:C1')
        assert advanced['status'] != 'error' and advanced['transient']['sample']['time_s'] == .0001
        assert 1.25 < cap < 3 and advanced['mixed']['kcl_residual_a'] < 1e-10
        report = dict(passed=True, transport='desktop-socket' if args.connect else 'headless-stdio', discovery=discovery,
                      observations=dict(led24=twenty_four, led30=thirty, floating_diagnostics=refused['diagnostics'],
                                        over_range_diagnostics=over_range['diagnostics'], repaired_gate_v=0,
                                        reopened_recordings=1, capacitor_start_v=1.25, capacitor_after_100us_v=cap),
                      checks=['floating gates at zero excitation refuse; explicit ground repairs them',
                              '24/30 V LED currents match independent inverse-Shockley load lines and conservation',
                              '31 V refuses with stale prior current; undo/restart recovers',
                              'Verify and saved recording reopen preserve 30 V result',
                              'explicit capacitor startup and subsequent charge remain physical constraints'])
        with args.output.open('x') as output:
            output.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({k: report[k] for k in ('passed', 'transport', 'observations')}, indent=2))
    finally:
        if args.connect:
            client.ask('app.close', {'discard_changes': True}, True)
        client.close()


if __name__ == '__main__':
    main()
