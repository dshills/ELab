#!/usr/bin/env python3
"""Replay hobbyist control examples through the public agent transport."""
import argparse
import hashlib
import importlib.util
import json
import math
from pathlib import Path
import subprocess
import time

spec = importlib.util.spec_from_file_location('viewport_client', Path(__file__).with_name('viewport-fit-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


class Client(helper.Client):
    def __init__(self, agent):
        self.process = subprocess.Popen([str(agent.resolve())], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
        self.sequence = 0

    def mutate(self, command, params=None):
        return self.ask(command, params, True)

    def step(self, seconds):
        self.mutate('simulation.step', {'duration_s': seconds})
        return self.ask('simulation.wait')


def capture_and_reopen(client, agent, assets, example, probes):
    client.mutate('simulation.verify', {'duration_s': .02, 'sample_step_s': .0005, 'probe_ids': probes})
    operation = client.ask('project.inspect')['last_operation']
    deadline = time.monotonic() + 30
    while True:
        view = client.ask('operation.inspect', {'id': operation})
        if view['operation']['status'] not in ('queued', 'running'):
            break
        assert time.monotonic() < deadline
        time.sleep(.01)
    assert view['operation']['status'] == 'completed', view
    client.mutate('verification.export', {'id': operation, 'format': 'json', 'path': str(assets / (example + '-capture.json'))})
    path = assets / (example + '.elab')
    client.mutate('project.save', {'path': str(path)})
    fresh = Client(agent)
    try:
        fresh.mutate('project.load', {'path': str(path)})
        state = fresh.ask('simulation.wait')
        assert state['document']['recording_count'] == 1 and not state['document']['dirty']
    finally:
        fresh.close()
    exports = {}
    for fmt in ('svg', 'png', 'pdf'):
        output = assets / (example + '.' + fmt)
        client.mutate('export.schematic.' + fmt, {'path': str(output)})
        exports[fmt] = hashlib.sha256(output.read_bytes()).hexdigest()
    return exports


def buttons(agent, assets):
    reports = []
    for example in ('button-counter', 'button-debounce'):
        client = Client(agent)
        try:
            client.mutate('example.load', {'id': example})
            client.mutate('simulation.pause')
            client.mutate('simulation.restart')
            initial = client.ask('simulation.wait')
            client.mutate('button.click', {'id': 'SCOUNT'})
            pressed = client.ask('simulation.wait')
            assert pressed['input_work']['command'] == 'button.click'
            assert pressed['input_work']['requested_button']['id'] == 'SCOUNT'
            state = client.step(.25)
            counts = {u['id']: u['memory']['value'] for u in state['mixed']['sequential']}
            assert counts['U2'] == (5 if example == 'button-debounce' else 1), counts
            if example == 'button-debounce':
                assert counts['U4'] == 1
            assert not state['document']['dirty'] and state['run_id'] == initial['run_id']
            client.mutate('button.set', {'id': 'SRESET', 'pressed': True})
            reset = client.ask('simulation.wait')
            assert all(u['memory']['value'] == 0 for u in reset['mixed']['sequential'])
            client.mutate('button.set', {'id': 'SRESET', 'pressed': False})
            client.ask('simulation.wait')
            state = client.step(.005)
            assert state['mixed']['kcl_residual_a'] < 1e-8 and abs(state['mixed']['power_residual_w']) < 1e-7
            exports = capture_and_reopen(client, agent, assets, example, ['v:U2.clk', 'v:U2.q0', 'i:SCOUNT'])
            reports.append(dict(example=example, counts_after_click=counts, asynchronous_reset=True, project_clean_after_runtime_actions=True, exports=exports))
        finally:
            client.close()
    return reports


def comparators(agent, assets):
    reports = []
    phase = math.asin(.25) / (2 * math.pi)
    for example in ('comparator-threshold', 'comparator-counter'):
        client = Client(agent)
        try:
            client.mutate('example.load', {'id': example})
            client.mutate('simulation.pause')
            client.mutate('simulation.restart')
            client.ask('simulation.wait')
            checked = 0
            for _ in range(25):
                state = client.step(.1)
                at = state['mixed']['time_s']
                comparator = next(u for u in state['mixed']['controls'] if u['id'] == 'U1')
                high = phase <= at % 1 < .5 + phase
                assert comparator['memory']['high'] == high, (example, at, comparator)
                assert (comparator['voltage_v'] > 3.5) == high
                if example == 'comparator-counter':
                    expected = 0 if at < phase else (math.floor(at - phase) + 1) % 10
                    assert state['mixed']['sequential'][0]['memory']['value'] == expected
                    assert next(d for d in state['mixed']['displays'] if d['id'] == 'DS1')['digit'] == expected
                assert state['mixed']['kcl_residual_a'] < 1e-8 and abs(state['mixed']['power_residual_w']) < 1e-7
                checked += 1
            exports = capture_and_reopen(client, agent, assets, example, ['v:U1', 'v:U1.input-difference', 'i:U1'])
            reports.append(dict(example=example, independently_checked_states=checked, exports=exports))
        finally:
            client.close()
    return reports


def relays(agent, assets):
    reports = []
    for example in ('relay-switch', 'relay-flasher'):
        client = Client(agent)
        try:
            client.mutate('example.load', {'id': example})
            client.mutate('simulation.pause')
            client.mutate('simulation.restart')
            client.ask('simulation.wait')
            if example == 'relay-switch':
                client.mutate('button.click', {'id': 'S1'})
                client.ask('simulation.wait')
            states = []
            flyback = False
            for _ in range(80):
                state = client.step(.001)
                relay = next(u for u in state['mixed']['controls'] if u['id'] == 'K1')
                states.append(relay['memory']['high'])
                flyback |= relay['voltage_v'] < -.1 and relay['current_a'] > .001
                assert state['mixed']['kcl_residual_a'] < 1e-8 and abs(state['mixed']['power_residual_w']) < 1e-6
            assert any(states) and not states[-1] and flyback, (example, states, flyback)
            exports = capture_and_reopen(client, agent, assets, example, ['i:K1', 'v:K1', 'i:D2', 'i:D1'])
            reports.append(dict(example=example, pickup_and_release=True, flyback_current=True, exports=exports))
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
    report = dict(passed=True, transport='headless-stdio', buttons=buttons(args.agent, assets), comparators=comparators(args.agent, assets), relays=relays(args.agent, assets))
    args.output.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report))


if __name__ == '__main__':
    main()
