#!/usr/bin/env python3
"""Exercise exact history-tip guards over the public stdio/private socket API."""
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
    parser.add_argument('--output', type=Path)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect)
    try:
        client.ask('example.reset', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        first = client.ask('edit.batch', {'name': 'Planned resistor value', 'edits': [
            {'command': 'component.set', 'params': {'id': 'R1', 'value': '2k'}}]}, True)
        tip = first['undo_entry']
        assert tip['name'] == 'Planned resistor value'
        second = client.ask('component.set', {'id': 'R1', 'value': '3k'}, True)
        client.ask('edit.undo', {'expected_entry_revision': tip['command_revision']}, True, 'history-conflict')
        refused = client.ask('project.inspect')
        assert refused['components'] == second['components']
        assert refused['command_revision'] == second['command_revision']
        latest = second['undo_entry']
        restored = client.ask('edit.undo', {'expected_entry_revision': latest['command_revision']}, True)
        assert next(p['value'] for p in restored['components'] if p['id'] == 'R1') == 2000
        assert restored['redo_entry'] == latest
        redone = client.ask('edit.redo', {'expected_entry_revision': latest['command_revision']}, True)
        assert next(p['value'] for p in redone['components'] if p['id'] == 'R1') == 3000
        assert redone['undo_entry'] == latest
        report = dict(passed=True, transport='packaged-native-private-socket' if args.connect else 'headless-stdio',
                      actor=latest['actor'], checks=['named atomic batch exposes actor/tip',
                      'obsolete tip refuses without changing graph/command revision',
                      'guarded undo restores prior value', 'redo preserves original actor/tip identity'],
                      limitations=['cross-actor refusal/acknowledgment is exercised by core/native tests; this transport uses one actor'])
        if args.output:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report, indent=2))
    finally:
        client.close()


if __name__ == '__main__':
    main()
