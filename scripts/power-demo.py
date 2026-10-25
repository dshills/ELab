#!/usr/bin/env python3
"""Replay stored-energy and isolation examples through ELab's public commands."""
import argparse
import importlib.util
import json
import time
from pathlib import Path

spec = importlib.util.spec_from_file_location('control_client', Path(__file__).with_name('controls-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def batteries(agent, assets):
    reports = []
    for example in ('battery-led', 'battery-regulator'):
        client = helper.Client(agent)
        try:
            client.mutate('example.load', {'id': example})
            client.mutate('simulation.pause')
            client.mutate('simulation.restart')
            client.ask('simulation.wait')
            initial = client.step(.01)
            for _ in range(80):
                state = client.step(.1)
                battery = state['mixed']['batteries'][0]
                part = next(p for p in state['components'] if p['id'] == 'B1')
                parameters = part['power']['battery']
                assert abs(battery['terminal_voltage_v'] - (battery['open_circuit_voltage_v'] - battery['delivered_current_a'] * parameters['internal_resistance_ohm'])) < 1e-8
                assert abs(battery['stored_energy_j'] - .5 * parameters['capacity_ah'] * 3600 / part['value'] * battery['open_circuit_voltage_v']**2) < 1e-8
                assert state['mixed']['kcl_residual_a'] < 1e-8 and abs(state['mixed']['power_residual_w']) < 1e-7
            assert battery['charge_fraction'] < initial['mixed']['batteries'][0]['charge_fraction'] * .8
            if example == 'battery-led':
                assert state['mixed']['leds'][0]['current_a'] < initial['mixed']['leds'][0]['current_a'] * .5
                client.mutate('simulation.restart')
                client.ask('simulation.wait')
                before = client.step(.01)
                client.mutate('switch.set', {'id': 'S1', 'closed': True})
                queued = client.ask('project.inspect')
                assert queued['mixed']['batteries'][0]['open_circuit_voltage_v'] == before['mixed']['batteries'][0]['open_circuit_voltage_v']
                sag = client.step(.005)
                assert sag['mixed']['batteries'][0]['terminal_voltage_v'] < before['mixed']['batteries'][0]['terminal_voltage_v'] * .8
            else:
                assert initial['mixed']['regulators'][0]['state'] == 'regulating'
                assert state['mixed']['regulators'][0]['state'] == 'dropout'
            exports = helper.capture_and_reopen(client, agent, assets, example, ['v:B1', 'i:B1'])
            reports.append(dict(example=example, final_charge_fraction=battery['charge_fraction'], discharge_and_conservation=True, captures_and_fresh_reopen=True, exports=exports))
        finally:
            client.close()
    return reports


def motors(agent, assets):
    reports = []
    for example in ('motor-dc', 'motor-pwm'):
        client = helper.Client(agent)
        try:
            client.mutate('example.load', {'id': example})
            client.mutate('simulation.pause')
            client.mutate('simulation.restart')
            client.ask('simulation.wait')
            state = client.step(.04)
            m = state['mixed']['motors'][0]
            assert m['rpm'] > 500
            assert abs(m['back_emf_v'] - m['speed_rad_s'] * .02) < 1e-8
            client.mutate('motor.load.set', {'id': 'MTR1', 'load_n_m_s_per_rad': .0001})
            client.ask('simulation.wait')
            loaded = client.ask('project.inspect')
            assert loaded['mixed']['motors'][0]['rpm'] == m['rpm']
            assert loaded['mixed']['motors'][0]['armature_current_a'] == m['armature_current_a']
            assert loaded['mixed']['time_s'] == state['mixed']['time_s']
            slow = client.step(.04)
            assert slow['mixed']['motors'][0]['rpm'] < m['rpm']
            exports = helper.capture_and_reopen(client, agent, assets, example, ['v:MTR1', 'i:MTR1'])
            reports.append(dict(example=example, startup_and_live_load=True, captures_and_fresh_reopen=True, exports=exports))
        finally:
            client.close()
    return reports


def ac_sweeps(agent, assets):
    reports = []
    for example in ('ac-rc', 'ac-rlc', 'ac-opamp'):
        client = helper.Client(agent)
        try:
            client.mutate('example.load', {'id': example})
            client.mutate('simulation.pause')
            before = client.ask('simulation.wait')
            client.mutate('sweep.start')
            state = client.ask('project.inspect')
            op = state['dc_sweep']['selected_operation']
            deadline = time.monotonic() + 30
            while True:
                view = client.ask('sweep.inspect', {'id': op})
                if view['operation']['status'] not in ('queued', 'running', 'canceling'):
                    break
                assert time.monotonic() < deadline
                time.sleep(.01)
            state = client.ask('project.inspect')
            assert view['operation']['status'] == 'completed', view
            assert view['metadata']['format'] == 'elab.ac-sweep'
            assert before['run_id'] == state['run_id'] and before.get('mixed') == state.get('mixed')
            result = view['result']
            assert len(result['points']) == 201
            assert result['ac_bands'][0]['half_power_estimates_hz']
            if example == 'ac-rc':
                assert abs(result['ac_bands'][0]['half_power_estimates_hz'][0] - 159.154943) < .1
            client.mutate('sweep.trace.select', {'id': result['probes'][1]['id']})
            client.mutate('sweep.cursor.set', {'operation_id': op, 'index': 80})
            for fmt in ('json', 'csv'):
                client.mutate('sweep.export', {'id': op, 'format': fmt, 'path': str(assets / (example + '-sweep.' + fmt))})
            exports = helper.capture_and_reopen(client, agent, assets, example, [view['metadata']['settings']['probe_ids'][0]])
            fresh = helper.Client(agent)
            try:
                fresh.mutate('project.load', {'path': str(assets / (example + '.elab'))})
                reopened = fresh.ask('simulation.wait')
                assert reopened['dc_sweep']['result'] == result
                assert reopened['dc_sweep']['cursor_index'] == 80
            finally:
                fresh.close()
            reports.append(dict(example=example, bands=result['ac_bands'], no_live_mutation=True, gain_phase_cursor_and_exports=True, fresh_saved_capture=True, exports=exports))
        finally:
            client.close()
    return reports


def optocouplers(agent, assets):
    client = helper.Client(agent)
    try:
        client.mutate('example.load', {'id': 'opto-switch'})
        client.mutate('simulation.pause')
        off = client.ask('simulation.wait')
        assert off['mixed']['optocouplers'][0]['led_current_a'] < 1e-6
        client.mutate('switch.set', {'id': 'S1', 'closed': True})
        client.mutate('simulation.restart')
        on = client.ask('simulation.wait')
        assert on['mixed']['leds'][0]['current_a'] > .001
        client.mutate('optocoupler.configure', {'id': 'OC1', 'current_transfer_ratio': .5})
        client.mutate('simulation.restart')
        state = client.ask('simulation.wait')
        u = state['mixed']['optocouplers'][0]
        assert u['collector_current_a'] < on['mixed']['optocouplers'][0]['collector_current_a']
        assert abs(state['mixed']['power_residual_w']) < 1e-8
        assert state['mixed']['kcl_residual_a'] < 1e-9
        exports = helper.capture_and_reopen(client, agent, assets, 'opto-switch', ['v:OC1', 'v:OC1.output', 'i:OC1.anode', 'i:OC1.collector'], duration=0)
        return [dict(example='opto-switch', independent_ports_and_ctr=True, output_led_switch=True, captures_and_fresh_reopen=True, exports=exports)]
    finally:
        client.close()


def transformers(agent, assets):
    reports = []
    for example in ('coupled-windings', 'transformer-supply'):
        client = helper.Client(agent)
        try:
            client.mutate('example.load', {'id': example})
            client.mutate('simulation.pause')
            client.mutate('simulation.restart')
            client.ask('simulation.wait')
            state = client.step(.05)
            state = client.step(.05)
            assert state['mixed']['kcl_residual_a'] < 1e-9
            assert abs(state['mixed']['power_residual_w']) < 1e-7
            winding = state['mixed']['transformers'][0]
            assert winding['magnetic_energy_j'] >= 0 and winding['winding_loss_w'] >= 0
            assert not state['mixed'].get('inductors')
            if example == 'transformer-supply':
                cap = next(c for c in state['mixed']['capacitors'] if c['id'] == 'C1')
                assert cap['voltage_v'] > 4
                assert 3.29 < state['mixed']['regulators'][0]['output_voltage_v'] < 3.31
                probes = ['v:T1.secondary', 'v:C1', 'v:U1', 'i:T1.primary1']
            else:
                client.mutate('simulation.restart')
                client.ask('simulation.wait')
                normal = client.step(.003)['mixed']['transformers'][0]['secondary_voltage_v']
                parameters = next(p for p in state['components'] if p['id'] == 'T1')['power']['transformer']
                parameters['reversed_secondary'] = True
                client.mutate('transformer.configure', {'id': 'T1', 'parameters': parameters})
                client.mutate('simulation.restart')
                client.ask('simulation.wait')
                reversed_v = client.step(.003)['mixed']['transformers'][0]['secondary_voltage_v']
                assert abs(normal + reversed_v) < 1e-8
                probes = ['v:T1', 'v:T1.secondary', 'i:T1.primary1', 'i:T1.secondary1']
            exports = helper.capture_and_reopen(client, agent, assets, example, probes, sample_step=.0001)
            reports.append(dict(example=example, winding_energy_and_conservation=True, ratio_and_polarity=example == 'coupled-windings', rectification_and_regulation=example == 'transformer-supply', captures_and_fresh_reopen=True, exports=exports))
        finally:
            client.close()
    return reports


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--agent', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--only', choices=('batteries', 'motors', 'ac_sweeps', 'optocouplers', 'transformers'))
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    groups = dict(batteries=batteries, motors=motors, ac_sweeps=ac_sweeps, optocouplers=optocouplers, transformers=transformers)
    report = dict(passed=True, transport='headless-stdio')
    for name, replay in groups.items():
        if args.only is None or args.only == name:
            report[name] = replay(args.agent, assets)
    args.output.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report))


if __name__ == '__main__':
    main()
