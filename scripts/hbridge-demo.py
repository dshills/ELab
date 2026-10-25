#!/usr/bin/env python3
"""Verify the reversible motor example using only ELab public agent commands."""
import argparse
import importlib.util
import json
from pathlib import Path

spec = importlib.util.spec_from_file_location('controls', Path(__file__).with_name('controls-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def run(agent, assets):
    modes = {}
    for mode in ('forward', 'reverse', 'brake low', 'brake high', 'coast'):
        client = helper.Client(agent)
        try:
            client.mutate('example.load', {'id': 'hbridge-motor'})
            client.mutate('simulation.pause')
            state = client.ask('simulation.wait')
            pulse = next(p['pulse'] for p in state['components'] if p['id'] == 'PWM').copy()
            pulse['high_v'] = pulse['low_v'] = 5
            client.mutate('source.pulse.set', {'id': 'PWM', 'waveform': pulse})
            client.mutate('simulation.restart')
            client.ask('simulation.wait')
            client.step(.03)
            before = client.step(.03)
            assert before['mixed']['motors'][0]['rpm'] > 4000
            edits = {'forward': [], 'reverse': [('S1', False), ('S2', True)],
                     'brake low': [('S1', False)], 'brake high': [('S2', True)],
                     'coast': [('SEN', False)]}[mode]
            for part, closed in edits:
                client.mutate('switch.set', {'id': part, 'closed': closed})
            assert client.ask('project.inspect')['mixed'] == before['mixed']
            state = client.step(.04)
            mixed = state['mixed']
            driver = mixed['hbridges'][0]
            assert driver['state'] == mode and state['run_id'] == before['run_id']
            assert mixed['kcl_residual_a'] < 1e-9 and abs(mixed['power_residual_w']) < 1e-6
            modes[mode] = dict(rpm=mixed['motors'][0]['rpm'], driver=driver)
        finally:
            client.close()
    assert modes['reverse']['rpm'] < 0
    for mode in ('brake low', 'brake high'):
        assert abs(modes[mode]['rpm']) < modes['coast']['rpm'] * .5
    client = helper.Client(agent)
    try:
        client.mutate('example.load', {'id': 'hbridge-motor'})
        client.mutate('simulation.pause')
        client.mutate('simulation.restart')
        client.ask('simulation.wait')
        state = client.step(.04)
        assert state['mixed']['motors'][0]['rpm'] > 500
        exports = helper.capture_and_reopen(client, agent, assets, 'hbridge-motor',
                                           ['v:U1', 'i:U1.vm', 'i:U1.out1', 'i:MTR1'],
                                           duration=.01, sample_step=.00005)
        return dict(passed=True, transport='headless-stdio', modes=modes,
                    pwm_motor_rpm=state['mixed']['motors'][0]['rpm'],
                    capture_fresh_reopen=True, exports=exports)
    finally:
        client.close()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--agent', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    args.artifacts.mkdir(parents=True, exist_ok=False)
    report = run(args.agent, args.artifacts.resolve())
    args.output.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report))


if __name__ == '__main__':
    main()
