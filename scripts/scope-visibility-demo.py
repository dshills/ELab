#!/usr/bin/env python3
"""Verify that scope presentation stays independent of live and recorded data."""
import argparse
import importlib.util
import json
from pathlib import Path
import time


spec = importlib.util.spec_from_file_location('helper', Path(__file__).with_name('persistence-demo.py'))
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


def main():
    from demo_session import connect_demo_client

    parser = argparse.ArgumentParser()
    parser.add_argument('--connect')
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    client = connect_demo_client(helper.Client, args.connect, command_preconditions=True)
    try:
        client.ask('example.rc', {'discard_changes': True}, True)
        client.ask('simulation.pause', {}, True)
        client.ask('simulation.restart', {}, True)
        client.ask('simulation.wait')
        before = client.ask('simulation.step', {'duration_s': '20 ms'}, True)
        client.ask('scope.cursor.set', {'cursor': 'a', 'time_s': '10 ms'}, True)
        before = client.ask('scope.window.set', {'mode': 'fixed', 'start_s': 0, 'end_s': '.02 s'}, True)
        for visible in [False, True, False]:
            after = client.ask('scope.visibility.set', {'visible': visible}, True)
            assert after['scope_visible'] is visible
            for field in ['scope', 'transient', 'document', 'run_id', 'electrical_revision', 'view', 'undo_name']:
                assert after[field] == before[field], field
        client.ask('scope.visibility.set', {'visible': None}, True, expect='invalid-params')
        client.ask('simulation.resume', {}, True)
        time.sleep(.05)
        acquired = client.ask('project.inspect')
        assert not acquired['scope_visible'] and acquired['transient']['sample']['time_s'] > .02
        live_paused = client.ask('simulation.pause', {}, True)
        client.ask('scope.window.set', {'mode': 'live'}, True)
        admitted = client.ask('simulation.verify', {'duration_s': '20 ms', 'sample_step_s': '1 ms'}, True)
        operation = admitted['last_operation']
        deadline = time.monotonic()+10
        while True:
            result = client.ask('operation.inspect', {'id': operation})
            if result['operation']['status'] == 'completed':
                break
            assert result['operation']['status'] not in ['failed', 'canceled'] and time.monotonic() < deadline
            time.sleep(.01)
        client.ask('operation.select', {'id': operation}, True)
        client.ask('recording.view.set', {'mode': 'recorded'}, True)
        recorded = client.ask('recording.cursor.set', {'cursor': 'a', 'time_s': '10 ms'}, True)
        shown = client.ask('scope.visibility.set', {'visible': True}, True)
        assert shown['recording'] == recorded['recording']
        hidden = client.ask('scope.visibility.set', {'visible': False}, True)
        assert hidden['recording'] == shown['recording']
        client.ask('recording.play', {'seconds_per_wall_second': .1}, True)
        deadline = time.monotonic()+2
        while True:
            playback = client.ask('project.inspect')
            if playback['recording']['sample_index'] > 0:
                break
            assert time.monotonic() < deadline
            time.sleep(.01)
        assert not playback['scope_visible']
        client.ask('recording.pause', {}, True)
        final = client.ask('scope.visibility.set', {'visible': True}, True)
        assert final['transient']['sample'] == live_paused['transient']['sample']
        report = {'passed': True, 'transport': 'native-private-socket' if args.connect else 'headless-stdio',
            'checks': ['paused physical/scope/cursor/window/history/document/view invariants',
                'invalid parameters refuse', 'live acquisition continues while collapsed',
                'recorded inspection survives collapse/expand', 'recorded playback continues while collapsed'],
            'rc_20ms_v': before['transient']['sample']['voltage_v']}
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2)+'\n')
        print(json.dumps(report))
    finally:
        client.close()


if __name__ == '__main__':
    main()
