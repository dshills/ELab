#!/usr/bin/env python3
"""Exercise discovered Verify contracts through the public protocol."""
import argparse
import copy
import importlib.util
import json
import math
from pathlib import Path
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def completed(client, params):
    state = client.ask('simulation.verify', params, True)
    identity = state['last_operation']
    deadline = time.monotonic() + 15
    while True:
        view = client.ask('operation.inspect', {'id': identity})
        status = view['operation']['status']
        assert status not in ('failed', 'canceled'), view
        if status == 'completed':
            return identity, view['metadata']
        assert time.monotonic() < deadline
        time.sleep(.01)


def rows(client, identity):
    result, offset = [], 0
    while True:
        page = client.ask('operation.samples', {'id': identity, 'offset': offset, 'limit': 7})
        result.extend(page['rows'])
        offset = page['next_offset']
        if offset == page['total']:
            end = client.ask('operation.samples', {'id': identity, 'offset': offset, 'limit': 0})
            assert end['rows'] == [] and end['next_offset'] == offset
            return result


def live(state):
    return {k: copy.deepcopy(state.get(k)) for k in
            ('components', 'wires', 'result', 'npn', 'mixed', 'transient', 'run_id', 'electrical_revision', 'undo_entry')}


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        before = client.ask('simulation.wait')
        full = client.ask('capabilities.list')
        details = {}
        for command in ('verification.settings.set', 'simulation.verify', 'operation.inspect',
                        'operation.samples', 'operation.select', 'operation.cancel'):
            cap = client.ask('capabilities.list', {'command': command})['capabilities'][0]
            assert cap['input_schema']['x-command-contract']['version'] == 1
            assert cap['output_schema'] == next(c['output_schema'] for c in full['capabilities'] if c['id'] == command)
            details[command] = cap['input_schema']
        assert before == client.ask('project.inspect')
        assert before['verification_input_modes'] == ['auto', 'charge', 'discharge']
        cfg = copy.deepcopy(details['verification.settings.set']['examples'][0])
        cfg.update(duration_s='20 ms', sample_step_s='1 ms', input_mode='discharge',
                   initial_voltage_v=2, capacitor_initials=[{'id': 'C1', 'voltage_v': 1}])
        client.ask('verification.settings.set', cfg, True)
        before = client.ask('project.inspect')
        for params in ({'probe_ids': None}, {'capacitor_initials': None}, {'sample_step_s': '0.1 us'},
                       {'input_mode': 'high'}, {'duration_s': '2 s', 'sample_step_s': '1 ms'},
                       {'capacitor_initials': [{'id': 'C1', 'voltage_v': 0}, {'id': 'C1', 'voltage_v': 1}]}):
            client.ask('simulation.verify', params, True, 'invalid-params')
            assert before == client.ask('project.inspect')
        identity, rc = completed(client, {'initial_voltage_v': None, 'input_mode': None,
                                          'capacitor_initials': [], 'probe_ids': ['v:C1', 'i:C1']})
        captured_rc = rows(client, identity)
        assert rc['input_mode'] == 'discharge' and rc['settings']['initial_voltage_v'] == 2
        assert captured_rc[0]['values'][0] == 2
        assert abs(captured_rc[-1]['values'][0] - 2 * math.exp(-2)) < 1e-12
        assert live(before) == live(client.ask('project.inspect'))
        client.ask('operation.cancel', {'id': identity}, True, 'unavailable')
        client.ask('operation.select', {'id': ''}, True)
        assert client.ask('project.inspect')['selected_operation'] == ''
        empty_id, empty = completed(client, {'duration_s': '2 ms', 'capacitor_initials': [], 'probe_ids': []})
        assert empty['probes'] == [] and all(r['values'] == [] for r in rows(client, empty_id))

        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        cfg.update(duration_s='20 ms', sample_step_s='2 ms', input_mode='charge', capacitor_initials=[])
        client.ask('verification.settings.set', cfg, True)
        before = client.ask('project.inspect')
        assert before['verification_input_modes'] == ['auto']
        client.ask('simulation.verify', {}, True, 'invalid-params')
        client.ask('simulation.verify', {'initial_voltage_v': None, 'input_mode': 'auto'}, True, 'invalid-params')
        assert before == client.ask('project.inspect')
        cfg.pop('initial_voltage_v')
        cfg.update(input_mode='auto', sample_step_s=0)
        client.ask('verification.settings.set', cfg, True)
        assert client.ask('project.inspect')['verification_settings'].get('initial_voltage_v') is None
        dc_id, dc = completed(client, {})
        captured_dc = rows(client, dc_id)
        assert dc['duration_s'] == 0 and dc['sample_count'] == 1 and dc['input_mode'] == 'auto'
        assert live(before) == live(client.ask('project.inspect'))

        client.ask('example.npn-pulse', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        before = client.ask('simulation.wait')
        assert before['verification_input_modes'] == ['auto', 'high', 'low']
        pulse_id, pulse = completed(client, {'duration_s': '100 ms', 'probe_ids': ['v:VIN.positive', 'i:Q1.collector']})
        captured_pulse = rows(client, pulse_id)
        edges = [r for r in captured_pulse if abs(r['time_s'] - .05) < 1e-12]
        assert any(r['values'][0] == 5 for r in edges) and any(r['values'][0] == 0 for r in edges)
        assert live(before) == live(client.ask('project.inspect'))
        client.ask('project.clear', {'discard_changes': True}, True)
        report = dict(passed=True, transport='native-socket' if args.connect else 'headless-stdio',
                      detailed_schemas=details, rc_rows=captured_rc, dc_rows=captured_dc, pulse_rows=captured_pulse,
                      checks=['discovery purity and unchanged output schemas', 'current analysis mode inventory',
                              'unit time inputs and admission/refusal purity', 'null inheritance and explicit empty arrays',
                              'RC discharge analytic endpoint', 'DC inherited-startup refusal and settings reset recovery',
                              'exact sample paging, inclusive end and duplicate pulse edges', 'terminal cancel refusal and clear selection',
                              'live graph, physics/time and history preserved'],
                      limits=['Six command contracts; no complete command/native interaction coverage claim',
                              'Pinned educational-model workflows; physical device and printing acceptance remain separate'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        with args.output.open('x') as target:
            target.write(json.dumps(report, indent=2) + '\n')
        print(json.dumps(dict(passed=True, commands=len(details), transport=report['transport'])))
    finally:
        client.close()


if __name__ == '__main__':
    main()
