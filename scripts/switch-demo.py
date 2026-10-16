#!/usr/bin/env python3
"""Validate SPST DC/leakage, transient actuation and portable records via public commands."""
import argparse
import importlib.util
import json
import math
from pathlib import Path
import tempfile
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def add_switch(client, closed, switch_id="S1"):
    first, second = switch_id+".1", switch_id+".2"
    client.ask('edit.batch', {'name': 'Series SPST', 'edits': [
        {'command': 'object.delete', 'params': {'id': 'W1'}},
        {'command': 'component.place', 'params': {'kind': 'switch', 'id': switch_id, 'position': {'x': -100, 'y': -60}}},
        {'command': 'wire.connect', 'params': {'from': 'V1.positive', 'to': first}},
        {'command': 'wire.connect', 'params': {'from': second, 'to': 'R1.1'}},
        {'command': 'switch.set', 'params': {'id': switch_id, 'closed': closed}},
    ]}, True)


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
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.resume', {}, True)
        add_switch(client, True)
        on = client.ask('simulation.wait')
        assert on['analysis'] == 'Network DC operating point'
        assert on['mixed']['leds'][0]['current_a'] > .0028
        client.ask('switch.set', {'id': 'S1', 'closed': False}, True)
        off = client.ask('simulation.wait')
        leakage = off['mixed']['switches'][0]['current_a']
        assert 0 < leakage < 5e-9
        assert abs(off['mixed']['power_residual_w']) < 1e-8
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        add_switch(client, True, "S2")
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.input.set', {'mode': 'high'}, True)
        client.ask('simulation.step', {'duration_s': .02}, True)
        before = client.ask('project.inspect')
        voltage = before['mixed']['capacitors'][0]['voltage_v']
        reference = 5 * (1-math.exp(-.02/(10001e-6)))
        assert abs(voltage-reference) < .003
        client.ask('switch.set', {'id': 'S2', 'closed': False}, True)
        opened = client.ask('project.inspect')
        assert opened['run_id'] == before['run_id']
        assert opened['transient']['sample']['time_s'] == before['transient']['sample']['time_s']
        assert abs(opened['mixed']['capacitors'][0]['voltage_v']-voltage) < 1e-10
        assert opened['electrical_revision'] == before['electrical_revision']
        assert opened['actuation_revision'] == before['actuation_revision']+1
        assert opened['switch_states'][0]['pending']
        assert opened['transient']['events'] == before['transient']['events']
        client.ask('simulation.step', {'duration_s': .001}, True)
        applied = client.ask('project.inspect')
        assert not applied['switch_states'][0]['pending']
        assert not applied['switch_states'][0]['applied_closed']
        assert applied['transient']['events'][-1]['mode'] == 'switch:S2:open'
        assert abs(applied['transient']['events'][-1]['time_s']-.021) < 1e-12
        applied_voltage = applied['mixed']['capacitors'][0]['voltage_v']
        client.ask('simulation.step', {'duration_s': .019}, True)
        held = client.ask('project.inspect')['mixed']['capacitors'][0]['voltage_v']
        expected_hold = 5+(applied_voltage-5)*math.exp(-.019/((1e9+10000)*1e-6))
        assert abs(held-expected_hold) < 2e-8
        client.ask('switch.set', {'id': 'S2', 'closed': True}, True)
        accepted = client.ask('simulation.verify', {'duration_s': .01, 'probe_ids': ['i:S2', 'v:C1']}, True)
        operation_id = accepted['last_operation']
        deadline = time.monotonic()+15
        while time.monotonic() < deadline:
            record = client.ask('operation.inspect', {'id': operation_id})
            if record['operation']['status'] not in ['queued', 'running', 'canceling']:
                break
            time.sleep(.02)
        assert record['operation']['status'] == 'completed', record
        assert record['metadata']['switch_model']['id'] == 'elab.spst-resistive.v1'
        with tempfile.TemporaryDirectory(prefix='elab-switch-') as temporary:
            directory = args.artifacts.resolve() if args.artifacts else Path(temporary)
            directory.mkdir(parents=True, exist_ok=True)
            path = directory / 'switched-rc.elab'
            client.ask('project.save', {'path': str(path), 'overwrite': True}, True)
            client.ask('export.schematic.svg', {'path': str(directory/'switched-rc.svg'), 'overwrite': True}, True)
            fresh = helper.Client()
            fresh.ask('project.load', {'path': str(path)}, True)
            restored = fresh.ask('project.inspect')
            assert restored['document']['recording_count'] == 1 and restored['paused']
            assert not restored['document']['dirty']
            assert restored['transient']['sample']['time_s'] == 0
            assert restored['switch_model'] == before['switch_model']
            report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                          capabilities=len(catalog['capabilities']), library_kinds=len(kinds),
                          on_current_a=on['mixed']['leds'][0]['current_a'], off_leakage_a=leakage,
                          capacitor_voltage_v=voltage, analytical_reference_v=reference,
                          checks=['atomic switched LED construction', 'DC on/off finite leakage', 'power conservation',
                                  'RC analytical reference', 'paused actuation queues without changing physics or electrical revision',
                                  'next accepted step applies the final desired switch state',
                                  'actuation revision captures Verify provenance', 'finite off-state hold',
                                  'independent Verify with switch probes/model', 'embedded recording/save/reopen',
                                  'fresh paused startup', 'SVG switch symbol'])
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
