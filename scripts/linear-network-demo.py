#!/usr/bin/env python3
"""Public high-voltage linear solve, capture/reopen and explicit failure recovery."""
import argparse
import importlib.util
import json
from pathlib import Path
import time

from validation_artifacts import discovery_summary

spec = importlib.util.spec_from_file_location('viewport', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', help='Fresh owned native endpoint; closes afterward')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect)
    mutate = lambda name, params=None: client.ask(name, params, True)

    def restart():
        mutate('simulation.restart')
        return client.ask('simulation.wait')

    def read(probe):
        result = client.ask('measurement.inspect', {'target': {'kind': 'probe', 'id': probe}})
        assert result['available'], result
        return next(r['value'] for r in result['readings'] if r['probe']['id'] == probe)

    def check(voltage):
        state = client.ask('project.inspect')
        r = state['mixed']
        assert r and not r['stale'] and state['status'] != 'error', state['diagnostics']
        assert abs(read('v:R1')-voltage) < 1e-10
        assert abs(read('i:R1')-voltage/1000) < 1e-12
        assert r['kcl_residual_a'] < 1e-10 and abs(r['power_residual_w']) < 1e-10
        return {'voltage_v': read('v:R1'), 'current_a': read('i:R1'),
                'power_w': r['resistors'][0]['power_w'], 'kcl_residual_a': r['kcl_residual_a'],
                'power_residual_w': r['power_residual_w']}

    try:
        assert not client.ask('project.inspect')['components'], 'requires fresh owned blank session'
        discovery = discovery_summary(client.ask('capabilities.list', {'command': 'component.place'}), assets/'discovery.json')
        numeric = discovery['capabilities'][0]['input_schema']['x-network-numerics']
        assert numeric['max_abs_node_voltage_v'] == 30 and numeric['max_nonlinear_node_update_v'] == .2
        assert numeric['max_newton_iterations'] == 100
        mutate('example.load', {'id': 'split-supply'})
        mutate('simulation.pause')
        mutate('object.delete', {'id': 'W5'})
        mutate('wire.connect', {'from': 'V2.negative', 'to': 'GND1.reference'})
        for identity in ('V1', 'V2'):
            mutate('component.set', {'id': identity, 'value': '12 V'})
        restart()
        twenty_four = check(24)
        for fmt in ('svg', 'png'):
            mutate(f'export.schematic.{fmt}', {'path': str(assets/f'stack24.{fmt}')})
        operation = mutate('simulation.verify', {'duration_s': 0, 'probe_ids': ['v:R1', 'i:R1']})['last_operation']
        deadline = time.monotonic()+20
        while True:
            captured = client.ask('operation.inspect', {'id': operation})
            if captured['operation']['status'] == 'completed':
                break
            assert captured['operation']['status'] not in ('failed', 'canceled') and time.monotonic() < deadline
            time.sleep(.01)
        assert abs(captured['metadata']['mixed']['resistors'][0]['voltage_v']-24) < 1e-10
        for fmt in ('json', 'csv'):
            mutate('verification.export', {'id': operation, 'format': fmt, 'path': str(assets/f'stack24.{fmt}')})
        mutate('project.save', {'path': str(assets/'stack24.elab')})
        # Add a third source within the existing three-source capacity.
        mutate('component.place', {'id': 'V3', 'kind': 'dc-source', 'value': '6 V', 'position': {'x': -400, 'y': 0}})
        mutate('object.delete', {'id': 'W1'})
        mutate('wire.connect', {'from': 'V1.positive', 'to': 'V3.negative'})
        mutate('wire.connect', {'from': 'V3.positive', 'to': 'R1.1'})
        restart()
        thirty = check(30)
        mutate('component.set', {'id': 'V3', 'value': '7 V'})
        refused = restart()
        assert refused['status'] == 'error' and any('±30 V' in d['message'] for d in refused['diagnostics'])
        assert refused['mixed']['stale'] and abs(refused['mixed']['resistors'][0]['voltage_v']-30) < 1e-10
        inspection = client.ask('measurement.inspect', {'target': {'kind': 'probe', 'id': 'v:R1'}})
        # Availability describes the source association; freshness belongs to
        # each reading. The existing probe still exists after a failed solve.
        assert inspection['available'], inspection
        assert len(inspection['readings']) == 1, inspection
        reading = inspection['readings'][0]
        assert reading['state'] == 'stale' and abs(reading['value']-30) < 1e-10, inspection
        mutate('edit.undo')
        restart()
        recovered = check(30)
        assert recovered == thirty
        mutate('project.load', {'path': str(assets/'stack24.elab'), 'discard_changes': True})
        client.ask('simulation.wait')
        assert check(24) == twenty_four
        reopened = client.ask('project.inspect')
        assert reopened['document']['recording_count'] == 1
        ground_wire = next(w['id'] for w in reopened['wires'] if 'GND1.reference' in (w['from'], w['to']))
        mutate('object.delete', {'id': ground_wire})
        mutate('wire.connect', {'from': 'V1.positive', 'to': 'GND1.reference'})
        restart()
        negative = read('v:V2.negative')
        assert abs(negative+24) < 1e-10 and abs(read('i:R1')-.024) < 1e-12
        report = dict(passed=True, transport='desktop-socket' if args.connect else 'headless-stdio',
                      discovery=discovery,
                      observations=dict(stack24=twenty_four, stack30=thirty, recovered30=recovered,
                          negative_rail_v=negative, range_diagnostics=refused['diagnostics'],
                          reopened_recordings=reopened['document']['recording_count']),
                      checks=['two12V sources reach24V with24mA and576mW', 'three sources reach exact30V limit',
                              '31V rejects explicitly; prior30V remains stale, never a valid zero',
                              'undo/restart recovers30V', 'captured JSON/CSV and saved24V recording reopen',
                              'reference shift produces negative24V rail', 'bounded numerical strategy is discoverable'])
        with args.output.open('x') as out:
            out.write(json.dumps(report, indent=2)+'\n')
        print(json.dumps({k: report[k] for k in ('passed', 'transport', 'observations')}, indent=2))
    finally:
        if args.connect:
            client.ask('app.close', {'discard_changes': True}, True)
        client.close()


if __name__ == '__main__':
    main()
