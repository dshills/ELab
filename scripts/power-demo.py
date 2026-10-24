#!/usr/bin/env python3
"""Replay stored-energy and isolation examples through ELab's public commands."""
import argparse
import importlib.util
import json
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


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--agent', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    report = dict(passed=True, transport='headless-stdio', batteries=batteries(args.agent, assets))
    args.output.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report))


if __name__ == '__main__':
    main()
