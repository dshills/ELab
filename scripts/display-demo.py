#!/usr/bin/env python3
"""Validate persisted display preferences and shared illustrative frame geometry."""
import argparse
import importlib.util
import json
import math
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
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    fresh = None
    try:
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.resume', {}, True)
        client.ask('simulation.wait')
        before = client.ask('project.inspect')
        prefs = dict(voltage_color=True, voltage_range_v=5, current_cues=True,
                     current_motion=True, reduced_motion=False, direction='conventional')
        changed = client.ask('display.preferences.set', prefs, True)
        assert changed['run_id'] == before['run_id'] and changed['result'] == before['result']
        assert changed['display']['animate'] and changed['document']['preferences_dirty']
        assert all(abs(w['current_a']-before['result']['current_a']) < 1e-12 for w in changed['display']['wires'])
        stable = client.ask('project.inspect')
        a = client.ask('display.inspect', {'phase': .1})
        b = client.ask('display.inspect', {'phase': .35})
        assert len(a['markers']) == 12 and a['markers'] != b['markers']
        assert client.ask('project.inspect') == stable
        prefs['direction'] = 'electron'
        client.ask('display.preferences.set', prefs, True)
        electron = client.ask('display.inspect', {'phase': .1})
        assert electron['markers'][0]['dx'] == -a['markers'][0]['dx'] and electron['markers'][0]['dy'] == -a['markers'][0]['dy']
        assert electron['display']['wires'][0]['current_a'] == a['display']['wires'][0]['current_a']
        prefs['reduced_motion'] = True
        client.ask('display.preferences.set', prefs, True)
        reduced = client.ask('display.inspect', {'phase': .35})
        assert not reduced['display']['animate'] and len(reduced['markers']) == 4
        assert all(m['kind'] == 'arrow' for m in reduced['markers'])
        client.ask('display.preferences.set', dict(prefs, direction='guess'), True, 'invalid-params')
        directory = args.artifacts.resolve()
        directory.mkdir(parents=True, exist_ok=True)
        client.ask('export.schematic.svg', {'path': str(directory/'led-display.svg'), 'overwrite': True}, True)
        svg = (directory/'led-display.svg').read_text()
        assert 'stroke="#d56a39"' in svg and 'data-current-wire="W1"' in svg and 'electron direction' in svg
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        prefs['direction'] = 'conventional'
        client.ask('display.preferences.set', prefs, True)
        client.ask('capacitor.initial.set', {'id': 'C1', 'value': '-2 V'}, True)
        client.ask('simulation.restart', {}, True)
        negative = client.ask('display.inspect')
        assert next(w for w in negative['display']['wires'] if w['id'] == 'W2')['voltage_v'] == -2
        client.ask('export.schematic.svg', {'path': str(directory/'negative-start.svg'), 'overwrite': True}, True)
        client.ask('capacitor.initial.set', {'id': 'C1', 'value': 'default'}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('measurement.pin.add', {'id': 'v:C1', 'position': {'x': 350, 'y': 200}}, True)
        client.ask('measurement.pin.add', {'id': 'i:C1', 'position': {'x': 350, 'y': 260}}, True)
        client.ask('simulation.step', {'duration_s': .01}, True)
        charge = client.ask('display.inspect')
        assert all(abs(w['current_a']-5*math.exp(-1)/10000) < 1e-12 for w in charge['display']['wires'])
        client.ask('simulation.input.set', {'mode': 'discharge'}, True)
        client.ask('simulation.step', {'duration_s': .001}, True)
        discharge = client.ask('display.inspect')
        expected = -5*(1-math.exp(-1))*math.exp(-.1)/10000
        assert all(abs(w['current_a']-expected) < 1e-12 for w in discharge['display']['wires'])
        assert not discharge['display']['animate']
        client.ask('export.schematic.svg', {'path': str(directory/'rc-display.svg'), 'overwrite': True}, True)
        path = str(directory/'display-rc.elab')
        client.ask('project.save', {'path': path, 'overwrite': True}, True)
        document = json.loads(Path(path).read_text())
        assert document['display_preferences'] == prefs and 'wires' not in document['display_preferences']
        changed = client.ask('display.preferences.set', dict(prefs, voltage_range_v=10), True)
        assert changed['document']['preferences_dirty'] and not changed['document']['schematic_dirty']
        client.ask('edit.undo', {}, True)
        assert not client.ask('project.inspect')['document']['dirty']
        fresh = helper.Client(command_preconditions=True)
        fresh.ask('project.load', {'path': path}, True)
        reopened = fresh.ask('simulation.wait')
        assert reopened['paused'] and not reopened['document']['dirty'] and reopened['display']['preferences'] == prefs
        assert reopened['transient']['sample']['time_s'] == 0 and len(reopened['pinned_readings']) == 2
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                      charge_current_a=charge['display']['wires'][0]['current_a'],
                      discharge_current_a=discharge['display']['wires'][0]['current_a'],
                      checks=['preferences preserve physics', 'read-only deterministic illustrative phases',
                              'electron arrows reverse while signed amperes remain conventional', 'reduced motion retains static arrows',
                              'independent unique LED/RC wire cut currents and reverse discharge',
                              'SVG palette/arrows/provenance parity', 'strict preference validation',
                              'saved preference ownership and undo-to-clean', 'fresh paused startup restores prefs/pins',
                              'owned project ends clean and paused'],
                      limitations=['ideal-wire loops may have no unique current; they receive no guessed arrows',
                                   'dots have illustrative wall speed and density, not calibrated physical transport',
                                   'app reduced-motion setting is explicit; OS preference integration remains unqualified',
                                   'optional version-4 extension is refused by older readers lacking it',
                                   'native GPU motion/focus and complete accessibility are separate checks'])
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()
        if fresh:
            fresh.close()


if __name__ == '__main__':
    main()
