#!/usr/bin/env python3
"""Replay interactive night-light and thermostat examples over the public agent API."""
import argparse
import importlib.util
import json
import math
from pathlib import Path

spec = importlib.util.spec_from_file_location('control_client', Path(__file__).with_name('controls-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def run(agent, assets):
    reports = []
    for example in ('night-light', 'thermostat'):
        client = helper.Client(agent)
        try:
            client.mutate('example.load', {'id': example})
            client.mutate('simulation.pause')
            client.mutate('simulation.restart')
            client.ask('simulation.wait')
            state = client.step(.01)
            thermal = example == 'thermostat'
            command = 'sensor.temperature.set' if thermal else 'sensor.light.set'
            name, key = ('RT1', 'temperature_c') if thermal else ('LDR1', 'illuminance_lux')
            cases = [(30, False), (25, False), (20, True), (25, True)] if thermal else [(1, True), (10, True), (100, False), (10, False), (0, True), (10000, False)]
            checked = []
            for value, on in cases:
                before = state
                client.mutate(command, {'id': name, key: value})
                state = client.ask('simulation.wait')
                assert state['input_work']['command'] == command
                assert state['mixed']['time_s'] == before['mixed']['time_s'] and state['run_id'] == before['run_id']
                if thermal:
                    old = next(c for c in before['mixed']['controls'] if c['id'] == 'K1')
                    now = next(c for c in state['mixed']['controls'] if c['id'] == 'K1')
                    assert old['current_a'] == now['current_a']
                sensor = state['mixed']['sensors'][0]
                resistance = 10000 * math.exp(3950 * (1 / (value + 273.15) - 1 / 298.15)) if thermal else (1e6 if value == 0 else max(100, min(1e6, 10000 * (value / 10)**(-.7))))
                assert abs(sensor['resistance_ohm'] - resistance) < 1e-8 * resistance
                state = client.step(.01)
                comparator = next(c for c in state['mixed']['controls'] if c['id'] == 'U1')
                assert comparator['memory']['high'] == on, (example, value, on, state)
                if thermal:
                    assert next(c for c in state['mixed']['controls'] if c['id'] == 'K1')['memory']['high'] == on
                assert (state['mixed']['leds'][0]['current_a'] > .001) == on
                assert state['mixed']['kcl_residual_a'] < 1e-8 and abs(state['mixed']['power_residual_w']) < 1e-7
                checked.append(dict(environment=value, resistance_ohm=resistance, output_on=on))
            probes = ['v:' + name, 'i:' + name, 'v:U1', 'i:D1']
            exports = helper.capture_and_reopen(client, agent, assets, example, probes)
            reports.append(dict(example=example, states=checked, continuity_preserved=True, exports=exports))
        finally:
            client.close()
    return dict(passed=True, transport='headless-stdio', examples=reports)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--agent', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    assets = args.artifacts.resolve()
    assets.mkdir(parents=True, exist_ok=False)
    report = run(args.agent, assets)
    args.output.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report))


if __name__ == '__main__':
    main()
