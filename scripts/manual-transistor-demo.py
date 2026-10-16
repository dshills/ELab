#!/usr/bin/env python3
"""Exercise the manual transistor example; --connect uses an owned session."""
import argparse
import importlib.util
import json
from pathlib import Path


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--artifacts', type=Path, required=True)
    args = parser.parse_args()
    directory = args.artifacts.resolve()
    directory.mkdir(parents=True, exist_ok=False)
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    fresh = None
    try:
        info = next(i for i in client.ask('examples.list') if i['id'] == 'switch-npn')
        client.ask('example.switch-npn', {'discard_changes': True}, True)
        original = client.ask('simulation.wait')
        assert original['selected'] == 'S1' and not original['example']['modified']
        assert abs(original['mixed']['leds'][0]['current_a']) < 1e-8
        currents = []
        for closed in [True, False]:
            client.ask('switch.set', {'id': 'S1', 'closed': closed}, True)
            state = client.ask('simulation.wait')
            assert not state['mixed']['stale'] and state['switch_states'][0]['desired_closed'] == closed
            current = state['mixed']['leds'][0]['current_a']
            assert current > .002 if closed else abs(current) < 1e-8
            assert state['mixed']['kcl_residual_a'] < 1e-9
            assert abs(state['mixed']['power_residual_w']) < 1e-8
            currents.append(current)
        client.ask('edit.undo', {}, True)
        lit = client.ask('simulation.wait')
        assert lit['mixed']['leds'][0]['current_a'] == currents[0]
        client.ask('simulation.pause', {}, True)
        client.ask('example.current.reset', {}, True, 'unsaved-changes')
        path = directory / 'manual-transistor.elab'
        client.ask('project.save', {'path': str(path)}, True)
        client.ask('export.schematic.svg', {'path': str(directory / 'manual-transistor.svg')}, True)
        saved_bytes = path.read_bytes()
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': str(path)}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['example']['info'] == info and reopened['example']['modified']
        assert reopened['paused'] and not reopened['document']['dirty']
        assert reopened['mixed']['leds'][0]['current_a'] == currents[0]
        client.ask('example.current.reset', {'expected_example_id': 'switch-npn'}, True)
        reset = client.ask('simulation.wait')
        assert reset['paused'] and not reset['example']['modified'] and reset['selected'] == 'S1'
        assert reset['components'] == original['components'] and reset['wires'] == original['wires']
        assert path.read_bytes() == saved_bytes
        report = {'passed': True, 'transport': 'native-socket' if args.connect else 'headless-stdio',
                  'closed_led_current_a': currents[0], 'open_led_current_a': currents[1],
                  'closed_base_current_a': lit['mixed']['transistors'][0]['base_current_a'],
                  'checks': ['catalog and named load', 'immediately selected switch', 'closed/open current and conservation',
                             'undo restores lit circuit', 'unsaved reset protection', 'SVG output',
                             'fresh paused save/reopen', 'reset preserves saved original bytes'],
                  'limitations': ['educational models; no physical device or rating qualification',
                                  'DC actuation recomputes the operating point; no transistor storage delay']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2) + '\n')
        print(json.dumps(report))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()
