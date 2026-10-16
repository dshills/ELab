#!/usr/bin/env python3
"""Verify themes and coalesced observation in a designated owned session."""
import argparse
import importlib.util
import json
from pathlib import Path
import threading


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def physical(snapshot):
    return {name: snapshot.get(name) for name in ['components', 'wires', 'run_id',
        'electrical_revision', 'transient', 'document', 'undo_name', 'redo_name']}


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect', required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--transport', default='packaged-native-private-socket',
        choices=['packaged-native-private-socket', 'headless-private-socket'])
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, str(Path(args.connect).resolve()), command_preconditions=True)
    observer = helper.Client(str(Path(args.connect).resolve()), command_preconditions=True)
    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.wait')
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        state = client.ask('simulation.step', {'duration_s': '10 ms'}, True)
        baseline = physical(state)
        catalog = client.ask('capabilities.list')
        assert catalog['events']['command'] == 'events.poll'
        replies, errors = [], []
        ready = threading.Event()
        def watch():
            ready.set()
            try:
                replies.append(observer.ask('events.poll', {'after_revision': state['revision'], 'timeout_ms': 1000}))
            except Exception as error:
                errors.append(str(error))
        worker = threading.Thread(target=watch)
        worker.start()
        ready.wait(timeout=2)
        dark = client.ask('appearance.theme.set', {'theme': 'dark'}, True)
        worker.join(timeout=3)
        assert not worker.is_alive() and not errors and replies, errors
        publication = replies[0]
        assert publication['changed'] and publication['snapshot']['theme'] == 'dark'
        assert any(item['topic'] == 'preferences' for item in publication['publications'])
        assert physical(dark) == baseline
        idle = observer.ask('events.poll', {'after_revision': publication['revision'], 'timeout_ms': 0})
        assert not idle['changed'] and 'snapshot' not in idle
        for mode in ['system', 'light', 'dark']:
            changed = client.ask('appearance.theme.set', {'theme': mode}, True)
            assert changed['theme'] == mode and physical(changed) == baseline
        client.ask('appearance.theme.set', {'theme': 'sepia'}, True, expect='invalid-params')
        assert physical(client.ask('project.inspect')) == baseline
        client.ask('view.fit', {'width_px': 480, 'height_px': 180}, True)
        report = {'passed': True, 'transport': args.transport,
            'checks': ['parallel observation/control', 'preference topic and independent snapshot',
                'idle snapshot omission', 'three shared theme modes preserve paused RC physics/history/document',
                'invalid theme refusal', 'circuit fit uses shared viewport command'],
            'rc_10ms_v': dark['transient']['sample']['voltage_v']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report))
    finally:
        observer.close()
        client.close()


if __name__ == '__main__':
    main()
